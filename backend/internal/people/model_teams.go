package people

import "mapp/internal/platform/record"

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

// TeamPage is the answer of GET /api/v1/teams (TeamPage).
type TeamPage struct {
	Items []TeamView `json:"items"`
	Total int        `json:"total"`
}

// TeamInput is the body of POST /api/v1/teams (TeamCreate); an empty colour means the least-used one.
type TeamInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Colour      string   `json:"colour"`
	MemberIDs   []string `json:"memberIds"`
}

// TeamUpdate is the body of PATCH /api/v1/teams/{teamId} and of its impact check (TeamUpdate): only the
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
