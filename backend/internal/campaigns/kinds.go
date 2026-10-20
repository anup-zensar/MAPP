package campaigns

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"northstar/internal/history"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/rollup"
)

// The history kinds of this package (data model §7), and the kind of their parent, feature 004's plan.
const (
	planKind       = "plan"
	campaignKind   = "campaign"
	initiativeKind = "initiative"
)

// The mid-sentence names of the records, as the log, the recycle bin and the messages show them.
const (
	campaignSingular   = "campaign"
	initiativeSingular = "initiative"
)

// Messages of the plan's "History kinds" about bringing a record back.
const (
	undoNameTakenMessage = "Can't undo — there is now another %s called '%s'. Restore it from the recycle bin instead."
	renamedNote          = "Renamed to '%s' because the name was taken."
)

// Kinds returns the history kinds of campaigns and initiatives, the campaign first, for internal/app to
// register right after feature 004's plan kind (data model §7). With them, history deletes a campaign with
// its initiatives — and, once feature 006 registers its activity kind under the initiative, everything
// below — in one recycle-bin entry, brings them back, and logs their changes with their path.
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{
		{
			Name:           campaignKind,
			Singular:       campaignSingular,
			Plural:         "campaigns",
			Parent:         planKind,
			Snapshot:       s.campaignSnapshot,
			ChildIDs:       s.campaignIDs,
			Remove:         s.repo.DeleteCampaign,
			Put:            s.putCampaign,
			Describe:       s.describer("Plan"),
			PrepareRestore: s.prepareCampaignRestore,
		},
		{
			Name:           initiativeKind,
			Singular:       initiativeSingular,
			Plural:         "initiatives",
			Parent:         campaignKind,
			Snapshot:       s.initiativeSnapshot,
			ChildIDs:       s.initiativeIDs,
			Remove:         s.repo.DeleteInitiative,
			Put:            s.putInitiative,
			Describe:       s.describer("Campaign"),
			PrepareRestore: s.prepareInitiativeRestore,
		},
	}
}

// campaignSnapshot reads the campaign id as history keeps it: the stored campaign, its name, and its plan
// as its parent.
func (s *Service) campaignSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	c, found, err := s.repo.GetCampaign(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: c.ID, Name: c.Name, ParentID: c.PlanID, Data: data}, true, nil
}

// initiativeSnapshot reads the initiative id as history keeps it, with its campaign as its parent.
func (s *Service) initiativeSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	i, found, err := s.repo.GetInitiative(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(i)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: i.ID, Name: i.Name, ParentID: i.CampaignID, Data: data}, true, nil
}

// campaignIDs lists the ids of the campaigns of the plan planID.
func (s *Service) campaignIDs(ctx context.Context, planID string) ([]string, error) {
	campaigns, err := s.repo.CampaignsOfPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, len(campaigns))
	for _, c := range campaigns {
		list = append(list, c.ID)
	}
	return list, nil
}

// initiativeIDs lists the ids of the initiatives of the campaign campaignID.
func (s *Service) initiativeIDs(ctx context.Context, campaignID string) ([]string, error) {
	initiatives, err := s.repo.InitiativesOfCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, len(initiatives))
	for _, i := range initiatives {
		list = append(list, i.ID)
	}
	return list, nil
}

// putCampaign stores the campaign of a snapshot again, exactly as the snapshot holds it.
func (s *Service) putCampaign(ctx context.Context, snapshot history.Snapshot) error {
	var c Campaign
	if err := json.Unmarshal(snapshot.Data, &c); err != nil {
		return err
	}
	return s.repo.PutCampaign(ctx, c)
}

// putInitiative stores the initiative of a snapshot again, exactly as the snapshot holds it.
func (s *Service) putInitiative(ctx context.Context, snapshot history.Snapshot) error {
	var i Initiative
	if err := json.Unmarshal(snapshot.Data, &i); err != nil {
		return err
	}
	return s.repo.PutInitiative(ctx, i)
}

// prepareCampaignRestore applies the name rule of campaigns just before one comes back: no two campaigns of
// a plan share a name. When another campaign of the plan has meanwhile taken the name, an Undo, which must
// put the campaign back exactly, is refused with name_taken (FR-029); a Restore renames it to "<name>
// (restored)", "<name> (restored 2)", … and says so in a note (feature 011).
func (s *Service) prepareCampaignRestore(ctx context.Context, rc history.RestoreContext, snapshot *history.Snapshot) ([]string, error) {
	var c Campaign
	if err := json.Unmarshal(snapshot.Data, &c); err != nil {
		return nil, err
	}
	siblings, err := s.repo.CampaignsOfPlan(ctx, c.PlanID)
	if err != nil {
		return nil, err
	}
	other, taken := campaignNamed(siblings, c.Name, c.ID)
	if !taken {
		return nil, nil
	}
	if rc.Undo {
		return nil, errs.NameTaken(fmt.Sprintf(undoNameTakenMessage, campaignSingular, other.Name))
	}
	c.Name = restoredName(c.Name, func(name string) bool { _, taken := campaignNamed(siblings, name, c.ID); return taken })
	c.Touch(rc.Actor, rc.Now) // a changed record gets a new version
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	snapshot.Name, snapshot.Data = c.Name, data
	return []string{fmt.Sprintf(renamedNote, c.Name)}, nil
}

// prepareInitiativeRestore is prepareCampaignRestore for initiatives: names are unique within a campaign.
func (s *Service) prepareInitiativeRestore(ctx context.Context, rc history.RestoreContext, snapshot *history.Snapshot) ([]string, error) {
	var i Initiative
	if err := json.Unmarshal(snapshot.Data, &i); err != nil {
		return nil, err
	}
	siblings, err := s.repo.InitiativesOfCampaign(ctx, i.CampaignID)
	if err != nil {
		return nil, err
	}
	other, taken := initiativeNamed(siblings, i.Name, i.ID)
	if !taken {
		return nil, nil
	}
	if rc.Undo {
		return nil, errs.NameTaken(fmt.Sprintf(undoNameTakenMessage, initiativeSingular, other.Name))
	}
	i.Name = restoredName(i.Name, func(name string) bool { _, taken := initiativeNamed(siblings, name, i.ID); return taken })
	i.Touch(rc.Actor, rc.Now)
	data, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	snapshot.Name, snapshot.Data = i.Name, data
	return []string{fmt.Sprintf(renamedNote, i.Name)}, nil
}

// restoredName returns the first of "<name> (restored)", "<name> (restored 2)", … that is not taken. The
// suffix may make the name longer than 120 characters: nothing but a missing parent may block a restore.
func restoredName(name string, taken func(string) bool) string {
	candidate := name + " (restored)"
	for n := 2; taken(candidate); n++ {
		candidate = fmt.Sprintf("%s (restored %d)", name, n)
	}
	return candidate
}

// describer returns the Describe of a kind (data model §7): readable labels and values for the activity
// log, fixed at the time of the change. parentLabel names the parent in a move's "parent" change: "Plan"
// for campaigns, "Campaign" for initiatives.
func (s *Service) describer(parentLabel string) func(ctx context.Context, field string, value any) (string, string, error) {
	return func(ctx context.Context, field string, value any) (string, string, error) {
		text, _ := value.(string) // ids and dates are strings; other values are handled below
		switch field {
		case "name":
			return "Name", "", nil
		case "description":
			return "Description", "", nil
		case "ownerId":
			if text == "" {
				return "Owner", "", nil
			}
			person, found, err := s.deps.People.Get(ctx, text)
			if err != nil || !found {
				return "Owner", "", err
			}
			return "Owner", person.Name, nil
		case "startDate", "endDate":
			label := "Start date"
			if field == "endDate" {
				label = "End date"
			}
			if d, err := date.Parse(text); err == nil {
				return label, date.Format(d), nil
			}
			return label, "", nil
		case "budget":
			return "Budget", moneyText(value), nil
		case "parent":
			name, err := s.parentName(ctx, parentLabel, text)
			return parentLabel, name, err
		}
		if strings.HasPrefix(field, "attributes.") {
			return s.deps.Settings.DescribeAttribute(ctx, field, value)
		}
		if label, text, ok := rollup.DescribePhasing(field, value); ok { // 016 FR-023
			return label, text, nil
		}
		return "", "", nil
	}
}

// parentName returns the name of the plan or campaign id, as a move's log entry shows it; "" when it is
// gone (the log then shows the id's default text).
func (s *Service) parentName(ctx context.Context, parentLabel, id string) (string, error) {
	if id == "" {
		return "", nil
	}
	if parentLabel == "Plan" {
		plan, found, err := s.deps.Plans.Plan(ctx, id)
		if err != nil || !found {
			return "", err
		}
		return plan.Name, nil
	}
	c, found, err := s.repo.GetCampaign(ctx, id)
	if err != nil || !found {
		return "", err
	}
	return c.Name, nil
}

// moneyText writes a stored money value, decoded from JSON, as people read it: "400,000.00 USD"; "" when it
// is empty or not money.
func moneyText(value any) string {
	fields, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	text, _ := fields["amount"].(string)
	currency, _ := fields["currency"].(string)
	amount, err := money.ParseAmount(text)
	if err != nil {
		return ""
	}
	return money.Money{Amount: amount, Currency: currency}.Format()
}
