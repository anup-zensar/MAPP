package activities

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"northstar/internal/campaigns"
	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/record"
)

// Moving and copying an activity (US5, FR-029, FR-030): inside its plan only; copying into another plan
// happens through a plan copy (004) or bulk actions (012).

// Messages of the plan's "Move / copy refusals" row and the copy's log details.
const (
	datesDontFitFormat   = "'%s' runs %s, but '%s' runs %s. Change the activity's dates first, or pick another initiative."
	nameTakenThereFormat = "'%s' already has an activity called '%s'. Rename the activity first, or pick another initiative."
	alreadyHereMessage   = "It's already in this initiative."
	chooseAnotherMessage = "Choose another initiative of this plan."
	copiedFromFormat     = "Copied from '%s' (%s)" // + ", with 3 line items" when line items came along
	runsFromFormat       = "from %s"               // the run of an activity without an end date
	parentField          = "parent"                // the change of a move (data model §5)
	lineItemKind         = "lineItem"              // feature 007's kind: the count in a copy's details
	initiativeRefType    = "initiative"
	campaignRefType      = "campaign"
)

// Destinations returns every initiative of the plan of the activity id as a place to move or copy it to
// (FR-029, FR-030), grouped by campaign — campaigns, then their initiatives, by start date, then name —
// each saying why the activity cannot be moved or copied there, or nothing when it can: its own initiative
// is no place to move it to; an initiative whose dates do not contain the activity's takes neither (an
// ongoing activity: its start only); one that already has an activity with its name takes no move, while a
// copy gets a name of its own.
func (s *Service) Destinations(ctx context.Context, a actor.Actor, id string) ([]Destination, error) {
	var destinations []Destination
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		activity, pc, err := s.load(ctx, id)
		if err != nil {
			return err
		}
		campaignList, err := s.hierarchy.CampaignsOfPlan(ctx, pc.plan.ID)
		if err != nil {
			return err
		}
		slices.SortFunc(campaignList, func(x, y campaigns.Campaign) int { return inTree(x.StartDate, x.Name, x.ID, y.StartDate, y.Name, y.ID) })
		campaignIDs := make([]string, 0, len(campaignList))
		for _, c := range campaignList {
			campaignIDs = append(campaignIDs, c.ID)
		}
		initiatives, err := s.hierarchy.InitiativesOfCampaigns(ctx, campaignIDs)
		if err != nil {
			return err
		}
		slices.SortFunc(initiatives, func(x, y campaigns.Initiative) int {
			return inTree(x.StartDate, x.Name, x.ID, y.StartDate, y.Name, y.ID)
		})
		initiativeIDs := make([]string, 0, len(initiatives))
		for _, i := range initiatives {
			initiativeIDs = append(initiativeIDs, i.ID)
		}
		all, err := s.repo.ListByInitiatives(ctx, initiativeIDs)
		if err != nil {
			return err
		}

		destinations = make([]Destination, 0, len(initiatives))
		for _, c := range campaignList {
			for _, i := range initiatives {
				if i.CampaignID != c.ID {
					continue
				}
				d := Destination{
					Campaign:   errs.RecordRef{Type: campaignRefType, ID: c.ID, Name: c.Name},
					Initiative: errs.RecordRef{Type: initiativeRefType, ID: i.ID, Name: i.Name},
					StartDate:  i.StartDate, EndDate: i.EndDate, Current: i.ID == activity.InitiativeID,
				}
				if d.Current {
					d.MoveProblem = textPtr(alreadyHereMessage)
				} else {
					var moveProblems []string
					if !datesFit(activity, i) {
						dates := datesDontFit(activity, i)
						d.CopyProblem = textPtr(dates)
						moveProblems = append(moveProblems, dates)
					}
					siblings := slices.DeleteFunc(slices.Clone(all), func(x Activity) bool { return x.InitiativeID != i.ID })
					if other, taken := namedLike(siblings, activity.Name, activity.ID); taken {
						moveProblems = append(moveProblems, fmt.Sprintf(nameTakenThereFormat, i.Name, other.Name))
					}
					if len(moveProblems) > 0 {
						d.MoveProblem = textPtr(strings.Join(moveProblems, " "))
					}
				}
				destinations = append(destinations, d)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return destinations, nil
}

// Move moves the activity id to another initiative of its plan (FR-029): its status, values and line items
// stay as they are (D-023, D-033), and moving is not a change that restarts approval. The version must be
// the one the person saw. A target that is not another initiative of the plan, whose dates do not contain
// the activity's, or that already has an activity with its name is refused on initiativeId, every reason at
// once. One "moved" entry records the old and the new initiative.
func (s *Service) Move(ctx context.Context, a actor.Actor, id string, in MoveInput) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		before, pc, err := s.load(ctx, id)
		if err != nil {
			return err
		}
		if in.Version != nil && *in.Version >= 1 {
			if err := before.CheckVersion(*in.Version, noun); err != nil {
				return err
			}
		}
		var problems errs.Problems
		target, found, err := s.contextOf(ctx, strings.TrimSpace(in.InitiativeID))
		if err != nil {
			return err
		}
		if !found || target.plan.ID != pc.plan.ID || target.initiative.ID == before.InitiativeID {
			problems.Add("initiativeId", chooseAnotherMessage)
		} else {
			if !datesFit(before, target.initiative) {
				problems.Add("initiativeId", datesDontFit(before, target.initiative))
			}
			siblings, err := s.repo.ListByInitiative(ctx, target.initiative.ID)
			if err != nil {
				return err
			}
			if other, taken := namedLike(siblings, before.Name, before.ID); taken {
				problems.Add("initiativeId", fmt.Sprintf(nameTakenThereFormat, target.initiative.Name, other.Name))
			}
		}
		if in.Version == nil || *in.Version < 1 {
			problems.Add("version", reloadMessage)
		}
		if err := problems.Err(); err != nil {
			return err
		}

		after := before
		after.InitiativeID = target.initiative.ID
		after.Touch(a, s.now())
		if err := s.repo.Put(ctx, after); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: after.ID, RecordName: after.Name, Action: history.ActionMoved,
			Changes: []history.Change{{
				Field: parentField,
				Old:   mustJSON(errs.RecordRef{Type: initiativeRefType, ID: pc.initiative.ID, Name: pc.initiative.Name}),
				New:   mustJSON(errs.RecordRef{Type: initiativeRefType, ID: target.initiative.ID, Name: target.initiative.Name}),
			}},
		}); err != nil {
			return err
		}
		view, err = s.viewOf(ctx, after, target)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return s.shown(ctx, a, view)
}

// Copy copies the activity id, with its line items, into its own initiative or another of its plan whose
// dates contain it (FR-030, R13): every value, the name "<name> (copy)", " (copy 2)", … free in that
// initiative, a new ID and the status Created — no approvals, comments, history or GSAP values. One
// "copied" entry on the copy says where it came from and how many line items came with it (D-110).
func (s *Service) Copy(ctx context.Context, a actor.Actor, id string, in CopyInput) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		source, pc, err := s.load(ctx, id)
		if err != nil {
			return err
		}
		target, found, err := s.contextOf(ctx, strings.TrimSpace(in.InitiativeID))
		if err != nil {
			return err
		}
		switch {
		case !found || target.plan.ID != pc.plan.ID:
			return errs.Validation(errs.FieldError{Field: "initiativeId", Message: chooseAnotherMessage})
		case target.initiative.ID != source.InitiativeID && !datesFit(source, target.initiative):
			return errs.Validation(errs.FieldError{Field: "initiativeId", Message: datesDontFit(source, target.initiative)})
		}

		siblings, err := s.repo.ListByInitiative(ctx, target.initiative.ID)
		if err != nil {
			return err
		}
		dup := copyOf(source, 0)
		dup.InitiativeID, dup.Name = target.initiative.ID, copyName(siblings, source.Name)
		if dup.ID, err = s.ids.Next(ctx, ids.PrefixActivity); err != nil {
			return err
		}
		dup.Status, dup.Meta = StatusCreated, record.NewMeta(a, s.now())
		if err := s.repo.Put(ctx, dup); err != nil {
			return err
		}
		lineItems, err := s.lineItems.CopyAll(ctx, a, source.ID, dup.ID, 0)
		if err != nil {
			return err
		}
		details := fmt.Sprintf(copiedFromFormat, source.Name, source.ID)
		if with := s.history.CountsText(map[string]int{lineItemKind: lineItems}); with != "" {
			details += ", " + with
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: dup.ID, RecordName: dup.Name, Action: history.ActionCopied, Details: details,
		}); err != nil {
			return err
		}
		view, err = s.viewOf(ctx, dup, target)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return s.shown(ctx, a, view)
}

// datesDontFit is the refusal of an initiative whose dates do not contain the activity's: "'Launch film'
// runs 01 Mar – 31 May 2026, but 'Retail events' runs 01 Jun – 31 Dec 2026. …". The initiative has both
// dates; an activity without an end date "runs from <start>".
func datesDontFit(a Activity, initiative campaigns.Initiative) string {
	runs := fmt.Sprintf(runsFromFormat, formatStart(a.StartDate))
	if a.StartDate != nil && a.EndDate != nil && !a.Ongoing {
		runs = date.FormatRange(*a.StartDate, *a.EndDate)
	}
	return fmt.Sprintf(datesDontFitFormat, a.Name, runs, initiative.Name, date.FormatRange(*initiative.StartDate, *initiative.EndDate))
}

// formatStart writes an optional start date as people read it.
func formatStart(d *date.Date) string {
	if d == nil {
		return ""
	}
	return date.Format(*d)
}

// inTree orders campaigns and initiatives as the plan tree does: by start date, records without one last,
// then by name ignoring letter case, then by id.
func inTree(xStart *date.Date, xName, xID string, yStart *date.Date, yName, yID string) int {
	return cmp.Or(compareStart(xStart, yStart), strings.Compare(strings.ToLower(xName), strings.ToLower(yName)), cmp.Compare(xID, yID))
}

// textPtr returns a pointer to s.
func textPtr(s string) *string {
	return &s
}
