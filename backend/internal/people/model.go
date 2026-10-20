// Package people holds the people who can sign in, "me" (the signed-in person) and each person's display
// preferences (data model §1–§2), and the teams people belong to (feature 003).
//
// The shared foundation added feature 003's data layer: Team, Membership, the new preferences fields, the
// API types (this file), the usage hooks (usage.go) and the repository methods. Feature 003 adds the
// service operations, the history kinds and the handlers for people, teams and My profile.
package people

import (
	"cmp"
	"strings"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/record"
)

// Person is someone who can sign in, stored in the table "people". In this feature only the seed creates
// people; feature 003 adds the screens that create and change them, with the checks of these rules.
type Person struct {
	ID       string `json:"id"`       // "U-1001", from the readable-ID counter U; never changes
	Name     string `json:"name"`     // the full name, 1 to 120 characters
	Email    string `json:"email"`    // the sign-in name: at most 254 characters, unique, stored in lower case
	JobTitle string `json:"jobTitle"` // 0 to 120 characters
	Active   bool   `json:"active"`   // only active people are listed on the sign-in page and can sign in
	record.Meta
}

// FirstName returns the first word of the name: "Priya" for "Priya Raman".
func (p Person) FirstName() string {
	words := strings.Fields(p.Name)
	if len(words) == 0 {
		return ""
	}
	return words[0]
}

// Initials returns the first letters of the first two words of the name, in upper case: "PR" for "Priya
// Raman", "MA" for "Mary Ann Smith". A one-word name gives its first two letters: "CH" for "Cher".
func (p Person) Initials() string {
	words := strings.Fields(p.Name)
	var letters []rune // runes, so that a letter such as "É" is never cut in half
	if len(words) == 1 {
		letters = []rune(words[0])
		letters = letters[:min(2, len(letters))]
	} else {
		for _, word := range words[:min(2, len(words))] {
			letters = append(letters, []rune(word)[0])
		}
	}
	return strings.ToUpper(string(letters))
}

// Actor returns the person as the actor of what they do: their id and their name.
func (p Person) Actor() actor.Actor {
	return actor.Actor{ID: p.ID, Name: p.Name}
}

// The themes a person can choose (FR-014). "system" follows the computer's light or dark setting.
const (
	themeSystem = "system"
	themeLight  = "light"
	themeDark   = "dark"
)

// Preferences are one person's display settings, stored in the table "preferences" under the person's id
// (data model §2; feature 003 data model §4). They are not business records: changes have no version and
// are not written to the activity log (D-038). Records saved by feature 001 have no display currency or
// date format: an empty value means the default (DefaultDisplayCurrency, DefaultDateFormat).
type Preferences struct {
	PersonID         string `json:"personId"`
	Theme            string `json:"theme"`            // themeSystem, themeLight or themeDark
	SidebarCollapsed bool   `json:"sidebarCollapsed"` // the sidebar shows icons only
	DisplayCurrency  string `json:"displayCurrency"`  // ISO code; "" = DefaultDisplayCurrency (003)
	DateFormat       string `json:"dateFormat"`       // one of DateFormats; "" = DefaultDateFormat (003)
	HomeScope        string `json:"homeScope"`        // HomeScopeEverything, or "" for the default HomeScopeMyTeams (009)
	// HistoryIncludeInside is the History section's Include items inside / Include line items choice on
	// record pages (011 FR-012); false by default.
	HistoryIncludeInside bool `json:"historyIncludeInside,omitempty"`
}

// The values of Home's My teams / Everything switch (feature 009, D-172): My teams is the default.
const (
	HomeScopeMyTeams    = "myTeams"
	HomeScopeEverything = "everything"
)

// The defaults of the preferences feature 003 adds.
const (
	DefaultDisplayCurrency = "USD"
	DefaultDateFormat      = "DD MMM YYYY"
)

// DateFormats are the date formats a person can choose (003 data model §4), the default first.
var DateFormats = []string{"DD MMM YYYY", "DD/MM/YYYY", "MM/DD/YYYY", "YYYY-MM-DD"}

// Team is a group of people that plans are assigned to, stored in the table "teams" (003 data model §2).
// A plan's owner must belong to one of the plan's teams (R1).
type Team struct {
	ID          string `json:"id"`          // "T-1001", from the readable-ID counter T
	Name        string `json:"name"`        // 1–80 characters, unique among teams ignoring case
	Description string `json:"description"` // 0–300 characters
	Colour      string `json:"colour"`      // a key of Palette: "blue", …
	Active      bool   `json:"active"`
	record.Meta
}

// Membership is one person in one team, stored in the table "team_memberships" (003 data model §3). The
// pair (TeamID, PersonID) is unique. A membership is created and removed, never changed (version 1).
type Membership struct {
	ID       string `json:"id"`       // "TM-1001", internal, never shown
	TeamID   string `json:"teamId"`   // never changes
	PersonID string `json:"personId"` // never changes
	record.Meta
}

// Colour is one colour of the team palette (D-025): its key, stored on the team, and its label.
type Colour struct {
	Key, Label string
}

// Palette is the team colours in palette order (003 data model §2); the web app has a token
// --team-<key> for each.
var Palette = []Colour{
	{"blue", "Blue"}, {"violet", "Violet"}, {"teal", "Teal"}, {"magenta", "Magenta"},
	{"orange", "Orange"}, {"green", "Green"}, {"slate", "Slate"}, {"gold", "Gold"},
}

// defaultPreferences are the preferences of a person who has changed none: follow the computer's light or
// dark setting, with the sidebar expanded (FR-011, FR-014), amounts in US dollars and dates as "12 Mar
// 2026" (003 FR-023, FR-024).
func defaultPreferences(personID string) Preferences {
	return Preferences{PersonID: personID, Theme: themeSystem, SidebarCollapsed: false,
		DisplayCurrency: DefaultDisplayCurrency, DateFormat: DefaultDateFormat}
}

// withDefaults returns p with the default display currency and date format in place of empty ones: records
// saved by feature 001 have neither, and need no migration (003 data model §4).
func (p Preferences) withDefaults() Preferences {
	p.DisplayCurrency = cmp.Or(p.DisplayCurrency, DefaultDisplayCurrency)
	p.DateFormat = cmp.Or(p.DateFormat, DefaultDateFormat)
	return p
}

// The API types below are what the API returns or receives. They are kept apart from the stored records
// above, so the stored data can change without changing the API (research R10).

// CurrentUser is the signed-in person with their preferences: the answer of GET /api/me, PATCH /api/me and
// of a sign-in. Version is the person's, sent back by PATCH /api/me; Teams are their teams, active or not,
// by name — "My teams" (003 FR-026).
type CurrentUser struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	FirstName   string          `json:"firstName"`
	Email       string          `json:"email"`
	JobTitle    string          `json:"jobTitle"`
	Initials    string          `json:"initials"`
	Preferences PreferencesView `json:"preferences"`
	Version     int             `json:"version"`
	Teams       []TeamRef       `json:"teams"`
}

// DemoPerson is a person as the sign-in page lists them under "Explore as a demo person" (FR-001).
type DemoPerson struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	JobTitle string `json:"jobTitle"`
	Initials string `json:"initials"`
}

// PreferencesView is a person's preferences as the API returns them, every one with a value.
type PreferencesView struct {
	Theme            string `json:"theme"`
	SidebarCollapsed bool   `json:"sidebarCollapsed"`
	DisplayCurrency  string `json:"displayCurrency"`
	DateFormat       string `json:"dateFormat"`
	HomeScope        string `json:"homeScope"` // always sent (009, D-191)
	// HistoryIncludeInside is always sent (011, D-191).
	HistoryIncludeInside bool `json:"historyIncludeInside"`
}

// PreferencesUpdate is the body of PATCH /api/me/preferences: only the preferences to change. A field that
// was not sent is nil and stays as it is.
type PreferencesUpdate struct {
	Theme            *string `json:"theme"`
	SidebarCollapsed *bool   `json:"sidebarCollapsed"`
	DisplayCurrency  *string `json:"displayCurrency"`
	DateFormat       *string `json:"dateFormat"`
	HomeScope        *string `json:"homeScope"`
	// HistoryIncludeInside (011 FR-012) is saved only when it changes, and not logged.
	HistoryIncludeInside *bool `json:"historyIncludeInside"`
}

// currentUser returns p as the signed-in person, with prefs and their teams.
func (p Person) currentUser(prefs Preferences, teams []TeamRef) CurrentUser {
	return CurrentUser{
		ID:          p.ID,
		Name:        p.Name,
		FirstName:   p.FirstName(),
		Email:       p.Email,
		JobTitle:    p.JobTitle,
		Initials:    p.Initials(),
		Preferences: prefs.view(),
		Version:     p.Version,
		Teams:       teams,
	}
}

// demoPerson returns p as the sign-in page lists them.
func (p Person) demoPerson() DemoPerson {
	return DemoPerson{ID: p.ID, Name: p.Name, Email: p.Email, JobTitle: p.JobTitle, Initials: p.Initials()}
}

// view returns the preferences as the API returns them.
func (p Preferences) view() PreferencesView {
	homeScope := p.HomeScope
	if homeScope == "" {
		homeScope = HomeScopeMyTeams
	}
	return PreferencesView{Theme: p.Theme, SidebarCollapsed: p.SidebarCollapsed,
		DisplayCurrency: p.DisplayCurrency, DateFormat: p.DateFormat, HomeScope: homeScope,
		HistoryIncludeInside: p.HistoryIncludeInside}
}

// The API types of feature 003 (contract schemas in brackets; JSON names as the contract). 001's
// CurrentUser, PreferencesView and PreferencesUpdate above have their 003 fields (D-191): the API always
// sends CurrentUser.Version and Teams and the preferences' displayCurrency and dateFormat.

// PersonRef is a person as every other record shows them — owners, approvers, members (PersonRef). Names
// are read when the answer is made, so a rename shows everywhere. Features 004–008 return it.
type PersonRef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Initials string `json:"initials"`
	JobTitle string `json:"jobTitle"`
	Active   bool   `json:"active"`
}

// Ref returns p as other records show them.
func (p Person) Ref() PersonRef {
	return PersonRef{ID: p.ID, Name: p.Name, Initials: p.Initials(), JobTitle: p.JobTitle, Active: p.Active}
}

// TeamRef is a team as every other record shows it — plan teams, My teams (TeamRef). Colour is a palette
// key (TeamColour).
type TeamRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Colour string `json:"colour"`
	Active bool   `json:"active"`
}

// Ref returns t as other records show it.
func (t Team) Ref() TeamRef {
	return TeamRef{ID: t.ID, Name: t.Name, Colour: t.Colour, Active: t.Active}
}

// PersonView is a person in the people list and the pickers (Person).
type PersonView struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	JobTitle string    `json:"jobTitle"`
	Initials string    `json:"initials"`
	Active   bool      `json:"active"`
	Teams    []TeamRef `json:"teams"` // active or not, by name
	record.Meta
}

// UsageCount is how many records of one type name the person in one role, records in the recycle bin not
// counted (UsageCount); only counts of at least 1 are returned.
type UsageCount struct {
	Role       string `json:"role"`       // "owner" or "approver"
	RecordType string `json:"recordType"` // "plan", "campaign", "initiative", "activity", "approvalWorkflow"
	Label      string `json:"label"`      // "2 plans", "1 approval workflow"
	Count      int    `json:"count"`
}

// PersonDetails is a person with what they own and approve: the answer of the details, create, update and
// status operations (PersonDetails).
type PersonDetails struct {
	PersonView
	Usage []UsageCount `json:"usage"` // owners first, then approvers
}

// PersonPage is the answer of GET /api/people (PersonPage).
type PersonPage struct {
	Items []PersonView `json:"items"`
	Total int          `json:"total"`
}

// PersonInput is the body of POST /api/people (PersonCreate).
type PersonInput struct {
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	JobTitle string   `json:"jobTitle"`
	TeamIDs  []string `json:"teamIds"`
}

// PersonUpdate is the body of PATCH /api/people/{personId} and of its impact check (PersonUpdate): only
// the fields to change, plus the version (D-203, D-210). A field that is not sent (nil) keeps its value;
// TeamIDs, when sent, is every team after the change.
type PersonUpdate struct {
	Name     *string   `json:"name"`
	Email    *string   `json:"email"`
	JobTitle *string   `json:"jobTitle"`
	TeamIDs  *[]string `json:"teamIds"`
	Version  *int      `json:"version"`
}

// applyTo returns form with the fields u sends put in place of the stored values.
func (u PersonUpdate) applyTo(form personForm) personForm {
	if u.Name != nil {
		form.Name = *u.Name
	}
	if u.Email != nil {
		form.Email = *u.Email
	}
	if u.JobTitle != nil {
		form.JobTitle = *u.JobTitle
	}
	if u.TeamIDs != nil {
		form.TeamIDs = *u.TeamIDs
	}
	return form
}

// TeamPlanUse is a plan that uses a team, in the recycle bin or not (TeamPlanUse).
type TeamPlanUse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	InRecycleBin bool   `json:"inRecycleBin"`
}

// TeamView is a team with its members and the plans that use it (Team).
type TeamView struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Colour      string        `json:"colour"`
	Active      bool          `json:"active"`
	Members     []PersonRef   `json:"members"` // active or not, by name
	Plans       []TeamPlanUse `json:"plans"`   // including plans in the recycle bin, by name
	record.Meta
}

// TeamPage is the answer of GET /api/teams (TeamPage).
type TeamPage struct {
	Items []TeamView `json:"items"`
	Total int        `json:"total"`
}

// TeamInput is the body of POST /api/teams (TeamCreate); an empty colour means the least-used one.
type TeamInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Colour      string   `json:"colour"`
	MemberIDs   []string `json:"memberIds"`
}

// TeamUpdate is the body of PATCH /api/teams/{teamId} and of its impact check (TeamUpdate): only the
// fields to change, plus the version (D-203, D-210). A field that is not sent (nil) keeps its value;
// MemberIDs, when sent, is every member after the change.
type TeamUpdate struct {
	Name        *string   `json:"name"`
	Description *string   `json:"description"`
	Colour      *string   `json:"colour"`
	MemberIDs   *[]string `json:"memberIds"`
	Version     *int      `json:"version"`
}

// applyTo returns form with the fields u sends put in place of the stored values.
func (u TeamUpdate) applyTo(form teamForm) teamForm {
	if u.Name != nil {
		form.Name = *u.Name
	}
	if u.Description != nil {
		form.Description = *u.Description
	}
	if u.Colour != nil {
		form.Colour = *u.Colour
	}
	if u.MemberIDs != nil {
		form.MemberIDs = *u.MemberIDs
	}
	return form
}

// StatusUpdate is the body of POST /api/people/{personId}/status and /api/teams/{teamId}/status
// (feature 002's StatusUpdate).
type StatusUpdate struct {
	Active  *bool `json:"active"`
	Version *int  `json:"version"`
}

// ProfileUpdate is the body of PATCH /api/me (ProfileUpdate): My profile's name and job title.
type ProfileUpdate struct {
	Name     string `json:"name"`
	JobTitle string `json:"jobTitle"`
	Version  *int   `json:"version"`
}

// DisplayCurrencyOption is a currency a person can show amounts in (DisplayCurrencyOption).
type DisplayCurrencyOption struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Active bool   `json:"active"` // false only for the person's current one after it was set inactive
}

// DisplayCurrencyPage is the answer of GET /api/me/display-currencies (DisplayCurrencyPage), by code.
type DisplayCurrencyPage struct {
	Items []DisplayCurrencyOption `json:"items"`
	Total int                     `json:"total"`
}

// The kind of the owner warning of the two impact checks (ImpactWarning.kind).
const ImpactOwnerNeedsAttention = "ownerNeedsAttention"

// ImpactWarning is one warning of a membership change (feature 002's ImpactWarning; people keeps its own
// Go type so it never imports settings).
type ImpactWarning struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// ChangeImpact is the answer of POST /api/people/{personId}/impact and /api/teams/{teamId}/impact (feature
// 002's ChangeImpact).
type ChangeImpact struct {
	Warnings []ImpactWarning `json:"warnings"`
}

// DisplayRate is a person's display currency and its current rate in units per 1 USD as decimal text
// ("0.92"; "1" for USD) — Go only, for the APIs of 004–012 to convert their USD amounts (003 data model §4).
type DisplayRate struct {
	Code string
	Rate string
}
