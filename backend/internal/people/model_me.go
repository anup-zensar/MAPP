package people

import "mapp/internal/platform/authz"

// The API types below are what the API returns or receives. They are kept apart from the stored records
// of model.go, so the stored data can change without changing the API (research R10).

// CurrentUser is the signed-in person with their preferences: the answer of GET /api/v1/me, PATCH /api/v1/me and
// of a sign-in. Version is the person's, sent back by PATCH /api/v1/me; Teams are their teams, active or not,
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
	// Access is the person's roles, permissions, administrators and access rule (014 FR-024); nil only
	// while access control is not wired (package tests).
	Access *authz.MyAccess `json:"access,omitempty"`
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
	// PinnedPlanIDs is always sent and never null, in the order pinned (018 FR-004, D-317).
	PinnedPlanIDs []string `json:"pinnedPlanIds"`
}

// PreferencesUpdate is the body of PATCH /api/v1/me/preferences: only the preferences to change. A field that
// was not sent is nil and stays as it is.
type PreferencesUpdate struct {
	Theme            *string `json:"theme"`
	SidebarCollapsed *bool   `json:"sidebarCollapsed"`
	DisplayCurrency  *string `json:"displayCurrency"`
	DateFormat       *string `json:"dateFormat"`
	HomeScope        *string `json:"homeScope"`
	// HistoryIncludeInside (011 FR-012) is saved only when it changes, and not logged.
	HistoryIncludeInside *bool `json:"historyIncludeInside"`
	// PinnedPlanIDs, when sent, is every pinned plan after the change, in the order pinned (018 FR-005):
	// duplicates are dropped keeping that order, each id matches ^P-\d+$, at most 100 (D-317).
	PinnedPlanIDs *[]string `json:"pinnedPlanIds"`
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
	pinned := p.PinnedPlanIDs
	if pinned == nil {
		pinned = []string{} // "pinnedPlanIds": [], never null (018 FR-004)
	}
	return PreferencesView{Theme: p.Theme, SidebarCollapsed: p.SidebarCollapsed,
		DisplayCurrency: p.DisplayCurrency, DateFormat: p.DateFormat, HomeScope: homeScope,
		HistoryIncludeInside: p.HistoryIncludeInside, PinnedPlanIDs: pinned}
}

// ProfileUpdate is the body of PATCH /api/v1/me (ProfileUpdate): My profile's name and job title.
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

// DisplayCurrencyPage is the answer of GET /api/v1/me/display-currencies (DisplayCurrencyPage), by code.
type DisplayCurrencyPage struct {
	Items []DisplayCurrencyOption `json:"items"`
	Total int                     `json:"total"`
}

// DisplayRate is a person's display currency and its current rate in units per 1 USD as decimal text
// ("0.92"; "1" for USD) — Go only, for the APIs of 004–012 to convert their USD amounts (003 data model §4).
type DisplayRate struct {
	Code string
	Rate string
}
