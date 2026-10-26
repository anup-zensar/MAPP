package authz

import (
	"fmt"
	"strings"
)

// The refusal messages of plan "User-visible messages" 1–3, 6a, 6b and 22, and the not-found answer for a record
// outside the person's access (constitution VII). "%s" is filled in at run time.
const (
	msgNoRole          = "%s needs a role, and you don't have one yet. Ask an administrator."                   // 1
	msgNotInRole       = "%s isn't part of %s. Ask an administrator if you need it."                            // 2
	msgOtherTeams      = "%s is allowed for you only in plans of %s. Ask an administrator if you need it here." // 3
	msgNotFound        = "This %s no longer exists — someone may have deleted it."
	msgOwnActivity     = "You own this activity, so someone else must approve it."                                       // 6a
	msgSubmittedByYou  = "You submitted this activity, so someone else must approve it."                                 // 6b
	msgNotInAnyTeamYet = "Plans belong to teams, and you are not in a team yet. Ask an administrator to add you to one." // 22
)

// Refusal returns the message that refuses p (plan messages 1–3): the person holds no role; no role of
// theirs includes p; or p is theirs only for other teams than the plan's. plan is nil for an action that
// is not about one plan. A person whose scope covers no team at all gets message 22.
func (g Grants) Refusal(p Permission, plan *PlanFacts) string {
	noun := info(p).Noun
	switch {
	case len(g.Roles) == 0:
		return fmt.Sprintf(msgNoRole, noun)
	case !g.Has(p):
		return fmt.Sprintf(msgNotInRole, noun, YourRoles(g.RoleNames()))
	}
	_, teams := g.Teams(p)
	names := g.teamNames(teams)
	if len(names) == 0 {
		return msgNotInAnyTeamYet
	}
	return fmt.Sprintf(msgOtherTeams, noun, strings.Join(names, ", "))
}

// YourRoles names roles as the messages do: "your role (Viewer)" for one, "your roles (Planner, Approver)"
// for several.
func YourRoles(names []string) string {
	if len(names) == 1 {
		return "your role (" + names[0] + ")"
	}
	return "your roles (" + strings.Join(names, ", ") + ")"
}

// NotFoundMessage is the answer for a record outside the person's access: the not-found message of a missing
// record, so the two cannot be told apart. recordType is mid-sentence, such as "plan" or "line item".
func NotFoundMessage(recordType string) string {
	return fmt.Sprintf(msgNotFound, recordType)
}

// SelfApprovalRefusal returns why the person may not approve or decline an activity because of segregation
// of duties (FR-026, messages 6a and 6b): they own it (ownerID) or submitted its open round (submitterID),
// while the access rule "Nobody approves an activity they own or submitted" is on. "" means nothing stops
// them. Feature 008 uses it now, feature 017 for every approver source (D-302).
func (g Grants) SelfApprovalRefusal(ownerID, submitterID string) string {
	if !g.Rules.NoSelfApproval || g.PersonID == "" {
		return ""
	}
	if ownerID == g.PersonID {
		return msgOwnActivity
	}
	if submitterID == g.PersonID {
		return msgSubmittedByYou
	}
	return ""
}
