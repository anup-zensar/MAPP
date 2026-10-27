package people

import "mapp/internal/platform/record"

// The API types of feature 003 (contract schemas in brackets; JSON names as the contract). 001's
// CurrentUser, PreferencesView and PreferencesUpdate (model_me.go) have their 003 fields (D-191): the API always
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

// PersonListItem is a person in the people list (Person): the person's view with their line manager and
// how many people they are the line manager of (017 FR-016). The details answer (PersonDetails) has no
// manager fields: the web app reads the manager from the list and "Line manager of <n> people" from usage.
type PersonListItem struct {
	PersonView
	Manager     *PersonRef `json:"manager"`     // nil when the person has no line manager
	ReportCount int        `json:"reportCount"` // people, active or not, whose line manager this person is
}

// UsageCount is how many records of one type name the person in one role, records in the recycle bin not
// counted (UsageCount); only counts of at least 1 are returned.
type UsageCount struct {
	Role       string `json:"role"`       // "owner", "approver" or "manager" (017: line manager of people)
	RecordType string `json:"recordType"` // "plan", "campaign", "initiative", "activity", "approvalWorkflow", "person"
	Label      string `json:"label"`      // "2 plans", "1 approval workflow"
	Count      int    `json:"count"`
}

// PersonDetails is a person with what they own and approve: the answer of the details, create, update and
// status operations (PersonDetails).
type PersonDetails struct {
	PersonView
	Usage []UsageCount `json:"usage"` // owners first, then approvers
}

// PersonPage is the answer of GET /api/v1/people (PersonPage).
type PersonPage struct {
	Items []PersonListItem `json:"items"`
	Total int              `json:"total"`
}

// PersonInput is the body of POST /api/v1/people (PersonCreate).
type PersonInput struct {
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	JobTitle  string   `json:"jobTitle"`
	TeamIDs   []string `json:"teamIds"`
	ManagerID string   `json:"managerId"` // "" or left out for none (017 FR-016)
}

// PersonUpdate is the body of PATCH /api/v1/people/{personId} and of its impact check (PersonUpdate): only
// the fields to change, plus the version (D-203, D-210). A field that is not sent (nil) keeps its value;
// TeamIDs, when sent, is every team after the change; ManagerID "" clears the line manager.
type PersonUpdate struct {
	Name      *string   `json:"name"`
	Email     *string   `json:"email"`
	JobTitle  *string   `json:"jobTitle"`
	TeamIDs   *[]string `json:"teamIds"`
	ManagerID *string   `json:"managerId"`
	Version   *int      `json:"version"`
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
	if u.ManagerID != nil {
		form.ManagerID = *u.ManagerID
	}
	return form
}

// StatusUpdate is the body of POST /api/v1/people/{personId}/status and /api/v1/teams/{teamId}/status
// (feature 002's StatusUpdate).
type StatusUpdate struct {
	Active  *bool `json:"active"`
	Version *int  `json:"version"`
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

// ChangeImpact is the answer of POST /api/v1/people/{personId}/impact and /api/v1/teams/{teamId}/impact (feature
// 002's ChangeImpact).
type ChangeImpact struct {
	Warnings []ImpactWarning `json:"warnings"`
}
