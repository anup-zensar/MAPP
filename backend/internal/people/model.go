// Package people holds the people who can sign in, "me" (the signed-in person) and each person's display
// preferences (data model §1–§2), and the teams people belong to (feature 003).
//
// The shared foundation added feature 003's data layer: Team, Membership, the new preferences fields, the
// API types (the model files), the usage hooks (usage.go) and the repository methods. Feature 003 adds the
// service operations, the history kinds and the handlers for people, teams and My profile.
package people

import (
	"cmp"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/record"
)

// Person is someone who can sign in, stored in the table "people". In this feature only the seed creates
// people; feature 003 adds the screens that create and change them, with the checks of these rules.
type Person struct {
	ID       string `json:"id"`       // "U-1001", from the readable-ID counter U; never changes
	Name     string `json:"name"`     // the full name, 1 to 120 characters
	Email    string `json:"email"`    // the sign-in name: at most 254 characters, unique, stored in lower case
	JobTitle string `json:"jobTitle"` // 0 to 120 characters
	Active   bool   `json:"active"`   // only active people are listed on the sign-in page and can sign in
	// ManagerID is the person's line manager (017 FR-016): an active person when chosen, not the person, and no
	// loop of managers; "" = none. It is kept in MAPP until Microsoft Entra ID provides it (D-282).
	ManagerID string `json:"managerId"`
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
	// PinnedPlanIDs are the plans the person pinned on the Plans list (018 FR-004, D-317), in the order
	// they pinned them; empty — and omitted from the stored record — when they pinned none. An id of a
	// plan that was later deleted is kept: it is harmless and simply doesn't show (018 FR-007).
	PinnedPlanIDs []string `json:"pinnedPlanIds,omitempty"`
}

// maxPinnedPlans is the most plans one person can pin on the Plans list (018 FR-006, D-317).
const maxPinnedPlans = 100

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
		DisplayCurrency: DefaultDisplayCurrency, DateFormat: DefaultDateFormat, PinnedPlanIDs: []string{}}
}

// withDefaults returns p with the default display currency and date format in place of empty ones: records
// saved by feature 001 have neither, and need no migration (003 data model §4). Pinned plans read as an
// empty list, never nil, so records saved before 018 need none either (018 FR-004).
func (p Preferences) withDefaults() Preferences {
	p.DisplayCurrency = cmp.Or(p.DisplayCurrency, DefaultDisplayCurrency)
	p.DateFormat = cmp.Or(p.DateFormat, DefaultDateFormat)
	if p.PinnedPlanIDs == nil {
		p.PinnedPlanIDs = []string{}
	}
	return p
}

// equal reports whether two preference records hold the same choices. Preferences hold a slice since 018,
// so they can no longer be compared with ==.
func (p Preferences) equal(o Preferences) bool {
	return p.PersonID == o.PersonID && p.Theme == o.Theme && p.SidebarCollapsed == o.SidebarCollapsed &&
		p.DisplayCurrency == o.DisplayCurrency && p.DateFormat == o.DateFormat && p.HomeScope == o.HomeScope &&
		p.HistoryIncludeInside == o.HistoryIncludeInside && slices.Equal(p.PinnedPlanIDs, o.PinnedPlanIDs)
}
