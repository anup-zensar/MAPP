// Package people holds the people who can sign in, "me" (the signed-in person) and each person's display
// preferences (data model §1–§2). Feature 003 adds teams and the screens that create and change people.
package people

import (
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
// (data model §2). They are not business records: changes have no version and are not written to the
// activity log (D-038).
type Preferences struct {
	PersonID         string `json:"personId"`
	Theme            string `json:"theme"`            // themeSystem, themeLight or themeDark
	SidebarCollapsed bool   `json:"sidebarCollapsed"` // the sidebar shows icons only
}

// defaultPreferences are the preferences of a person who has changed none: follow the computer's light or
// dark setting, with the sidebar expanded (FR-011, FR-014).
func defaultPreferences(personID string) Preferences {
	return Preferences{PersonID: personID, Theme: themeSystem, SidebarCollapsed: false}
}

// The API types below are what the API returns or receives. They are kept apart from the stored records
// above, so the stored data can change without changing the API (research R10).

// CurrentUser is the signed-in person with their preferences: the answer of GET /api/me and of a sign-in.
type CurrentUser struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	FirstName   string          `json:"firstName"`
	Email       string          `json:"email"`
	JobTitle    string          `json:"jobTitle"`
	Initials    string          `json:"initials"`
	Preferences PreferencesView `json:"preferences"`
}

// DemoPerson is a person as the sign-in page lists them under "Explore as a demo person" (FR-001).
type DemoPerson struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	JobTitle string `json:"jobTitle"`
	Initials string `json:"initials"`
}

// PreferencesView is a person's preferences as the API returns them.
type PreferencesView struct {
	Theme            string `json:"theme"`
	SidebarCollapsed bool   `json:"sidebarCollapsed"`
}

// PreferencesUpdate is the body of PATCH /api/me/preferences: only the preferences to change. A field that
// was not sent is nil and stays as it is.
type PreferencesUpdate struct {
	Theme            *string `json:"theme"`
	SidebarCollapsed *bool   `json:"sidebarCollapsed"`
}

// currentUser returns p as the signed-in person, with prefs.
func (p Person) currentUser(prefs Preferences) CurrentUser {
	return CurrentUser{
		ID:          p.ID,
		Name:        p.Name,
		FirstName:   p.FirstName(),
		Email:       p.Email,
		JobTitle:    p.JobTitle,
		Initials:    p.Initials(),
		Preferences: prefs.view(),
	}
}

// demoPerson returns p as the sign-in page lists them.
func (p Person) demoPerson() DemoPerson {
	return DemoPerson{ID: p.ID, Name: p.Name, Email: p.Email, JobTitle: p.JobTitle, Initials: p.Initials()}
}

// view returns the preferences as the API returns them.
func (p Preferences) view() PreferencesView {
	return PreferencesView{Theme: p.Theme, SidebarCollapsed: p.SidebarCollapsed}
}
