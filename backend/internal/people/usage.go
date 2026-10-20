package people

import (
	"cmp"
	"context"
	"slices"
)

// The hooks below let later features tell people what they use (003 plan, "Constructor, hooks and
// wiring"), so that people never imports them. internal/app sets them before the API serves requests: 004
// UsePlans, 005/006/008 AddPersonUsage, 002 UseCurrencies. Every hook runs inside the people service's
// transaction, which its context carries.

// PlanUse is one plan as people needs it: its owner and teams, and whether it is in the recycle bin.
type PlanUse struct {
	ID, Name     string
	OwnerID      string   // "" when the plan has no owner
	TeamIDs      []string // the plan's teams
	InRecycleBin bool
}

// PlanSource lists every plan, stored and in the recycle bin (feature 004).
type PlanSource func(ctx context.Context) ([]PlanUse, error)

// PersonUsage counts the records of one type that name a person in one role ("owner" of activities,
// "approver" of approval workflows), records in the recycle bin not counted.
type PersonUsage struct {
	Role             string // "owner" or "approver"
	RecordType       string // "campaign", "initiative", "activity", "approvalWorkflow"
	Singular, Plural string // "activity" / "activities"
	Count            func(ctx context.Context, personID string) (int, error)
}

// Currency is a currency a person can show amounts in: its code, name, current rate in units per 1 USD as
// decimal text ("0.92"), and whether it is active (from feature 002).
type Currency struct {
	Code, Name, Rate string
	Active           bool
}

// CurrencySource lists every currency, active or not (feature 002, through an adapter in internal/app).
type CurrencySource func(ctx context.Context) ([]Currency, error)

// OnlyUSD is the CurrencySource until feature 002 is wired: US dollars only (spec Assumptions).
func OnlyUSD(context.Context) ([]Currency, error) {
	return []Currency{{Code: DefaultDisplayCurrency, Name: "US dollar", Rate: "1", Active: true}}, nil
}

// The roles of a usage count (UsageCount.role).
const (
	roleOwner    = "owner"
	roleApprover = "approver"
	roleManager  = "manager" // line manager of people (017 FR-016)
)

// usageOf counts what the person personID owns and approves, inside the caller's transaction (FR-004,
// D-067): first the plans they own (from the PlanSource, plans in the recycle bin left out), then each
// PersonUsage in the order it was added, owners before approvers. Only counts of at least 1 are listed. The
// label is the record type's singular for a count of 1 and its plural otherwise, like the contents of a
// recycle-bin entry; the web app writes the sentence ("Owner of 2 plans and 5 activities"). Never nil.
func (s *Service) usageOf(ctx context.Context, personID string, plans []PlanUse) ([]UsageCount, error) {
	usage := []UsageCount{}
	owned := 0
	for _, p := range plans {
		if p.OwnerID == personID && !p.InRecycleBin {
			owned++
		}
	}
	if owned > 0 {
		usage = append(usage, usageCount(roleOwner, "plan", "plan", "plans", owned))
	}

	for _, role := range []string{roleOwner, roleApprover} {
		for _, u := range s.usages {
			if u.Role != role {
				continue
			}
			count, err := u.Count(ctx, personID)
			if err != nil {
				return nil, err
			}
			if count > 0 {
				usage = append(usage, usageCount(u.Role, u.RecordType, u.Singular, u.Plural, count))
			}
		}
	}

	// Last, the people they are the line manager of, active or not: "Line manager of 2 people" in the
	// details and the Deactivate dialog (017 FR-016).
	everyone, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	reports := 0
	for _, p := range everyone {
		if p.ManagerID == personID {
			reports++
		}
	}
	if reports > 0 {
		usage = append(usage, usageCount(roleManager, "person", "person", "people", reports))
	}
	return usage, nil
}

// usageCount returns one usage count, labelled with singular for a count of 1 and plural otherwise.
func usageCount(role, recordType, singular, plural string, count int) UsageCount {
	label := plural
	if count == 1 {
		label = singular
	}
	return UsageCount{Role: role, RecordType: recordType, Label: label, Count: count}
}

// The lookup functions below are for other features' services (004–012). They take no actor, because the
// calling service already holds one (an exception to 001's rule, 003 plan "Lookup functions"), and they join
// the caller's transaction when its context carries one.

// PersonRefs returns the references of the people with the given ids, by id. Unknown ids are left out.
func (s *Service) PersonRefs(ctx context.Context, personIDs []string) (map[string]PersonRef, error) {
	refs := map[string]PersonRef{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		for _, id := range personIDs {
			p, found, err := s.repo.Get(ctx, id)
			if err != nil {
				return err
			}
			if found {
				refs[id] = p.Ref()
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return refs, nil
}

// TeamRefs returns the references of the teams with the given ids, by id. Unknown ids are left out.
func (s *Service) TeamRefs(ctx context.Context, teamIDs []string) (map[string]TeamRef, error) {
	refs := map[string]TeamRef{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		for _, id := range teamIDs {
			t, found, err := s.repo.GetTeam(ctx, id)
			if err != nil {
				return err
			}
			if found {
				refs[id] = t.Ref()
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return refs, nil
}

// MemberIDs returns the ids of the members of each team with one of the given ids, active people and
// inactive ones, ordered by id. Unknown teams are left out; a team without members has an empty list.
func (s *Service) MemberIDs(ctx context.Context, teamIDs []string) (map[string][]string, error) {
	members := map[string][]string{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		d, err := s.readDirectory(ctx)
		if err != nil {
			return err
		}
		for _, id := range teamIDs {
			if _, found := d.teams[id]; found {
				members[id] = d.memberIDsOf(id)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return members, nil
}

// TeamIDsOf returns the ids of the person's teams, active teams and inactive ones, ordered by id: "My
// teams" of other features (FR-026). Never nil.
func (s *Service) TeamIDsOf(ctx context.Context, personID string) ([]string, error) {
	var teamIDs []string
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		memberships, err := s.repo.MembershipsOfPerson(ctx, personID)
		if err != nil {
			return err
		}
		teamIDs = make([]string, 0, len(memberships))
		for _, m := range memberships {
			teamIDs = append(teamIDs, m.TeamID)
		}
		slices.Sort(teamIDs)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return teamIDs, nil
}

// IsMemberOfAny reports whether the person belongs to one of the teams: the owner rule R1 of feature 004.
// Inactive teams count, because plans keep them (FR-014).
func (s *Service) IsMemberOfAny(ctx context.Context, personID string, teamIDs []string) (bool, error) {
	mine, err := s.TeamIDsOf(ctx, personID)
	if err != nil {
		return false, err
	}
	for _, id := range teamIDs {
		if slices.Contains(mine, id) {
			return true, nil
		}
	}
	return false, nil
}

// DisplayRateOf returns the person's display currency and its current rate in units per 1 USD, with which
// the APIs of 004–012 convert their USD amounts (D-061). A display currency set inactive later is still
// used with its rate (D-063); a code the currency source no longer knows falls back to US dollars.
func (s *Service) DisplayRateOf(ctx context.Context, personID string) (DisplayRate, error) {
	rate := DisplayRate{Code: DefaultDisplayCurrency, Rate: "1"}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		prefs, err := s.preferences(ctx, personID)
		if err != nil {
			return err
		}
		code := cmp.Or(prefs.DisplayCurrency, DefaultDisplayCurrency) // 001's records have none: USD
		currencies, err := s.currencies(ctx)
		if err != nil {
			return err
		}
		for _, c := range currencies {
			if c.Code == code {
				rate = DisplayRate{Code: c.Code, Rate: c.Rate}
			}
		}
		return nil
	})
	if err != nil {
		return DisplayRate{}, err
	}
	return rate, nil
}

// PeopleByDisplayCurrency returns, for each currency code, the ids of the people who show amounts in it,
// ordered by id: feature 002's "Display currency of 2 people". Everyone who never chose one shows USD.
func (s *Service) PeopleByDisplayCurrency(ctx context.Context) (map[string][]string, error) {
	byCurrency := map[string][]string{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		everyone, err := s.repo.List(ctx) // ordered by id
		if err != nil {
			return err
		}
		for _, p := range everyone {
			prefs, err := s.preferences(ctx, p.ID)
			if err != nil {
				return err
			}
			code := cmp.Or(prefs.DisplayCurrency, DefaultDisplayCurrency)
			byCurrency[code] = append(byCurrency[code], p.ID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return byCurrency, nil
}
