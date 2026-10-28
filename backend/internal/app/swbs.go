package app

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"time"

	"mapp/internal/activity"
	"mapp/internal/history"
	"mapp/internal/lineitem"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/settings"
	"mapp/internal/stores"
	"mapp/internal/swbs"
)

const companyCodeLabel = "company code"

// swbsLineItems implements swbs.LineItems over the line items of the activities the scope reaches (the SQL scope
// filter of ActivityStore.ListInScope); the company code is the activity's "Company code" Shell attribute.
type swbsLineItems struct {
	activities stores.ActivityStore
	lineItems  stores.LineItemStore
	settings   stores.SettingsStore
}

var _ swbs.LineItems = swbsLineItems{}

func (x swbsLineItems) Candidates(ctx context.Context, scope authz.Scope, ids []string) ([]swbs.Candidate, error) {
	activities, items, err := x.inScope(ctx, scope)
	if err != nil {
		return nil, err
	}
	codeOf, err := x.companyCodes(ctx)
	if err != nil {
		return nil, err
	}
	candidates := []swbs.Candidate{}
	for _, li := range items {
		if ids != nil && !slices.Contains(ids, li.ID) {
			continue
		}
		act := activities[li.ActivityID]
		status := lineitem.SwbsStatusOf(li, lineitem.ActivityFacts{Status: string(act.Status)})
		candidates = append(candidates, swbs.Candidate{
			Request: swbs.Request{LineItemID: li.ID, LineItemName: li.Name, CompanyCode: codeOf(act.Attributes)},
			Ready:   status == lineitem.SwbsReadyToSend,
		})
	}
	return candidates, nil
}

func (x swbsLineItems) inScope(ctx context.Context, scope authz.Scope) (map[string]activity.Activity, []lineitem.LineItem, error) {
	seen, err := x.activities.ListInScope(ctx, scope)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]activity.Activity, len(seen))
	for _, a := range seen {
		byID[a.ID] = a
	}
	if len(byID) == 0 {
		return byID, nil, nil
	}
	items, err := x.lineItems.ListByActivities(ctx, slices.Sorted(maps.Keys(byID)))
	return byID, items, err
}

func (x swbsLineItems) Receive(ctx context.Context, a actor.Actor, answers []swbs.Answer, at time.Time) error {
	received := make([]lineitem.SwbsCodeReceived, 0, len(answers))
	for _, answer := range answers {
		received = append(received, lineitem.SwbsCodeReceived{LineItemID: answer.LineItemID, Code: answer.SwbsCode})
	}
	return lineitem.ReceiveSwbsCodes(ctx, x.lineItems, lineitem.SwbsReceipt{By: a, At: at, Codes: received})
}

// companyCodes returns what reads the label of the option chosen for the activity-level "Company code"
// attribute in an activity's values; "" when there is no such attribute or nothing is chosen.
func (x swbsLineItems) companyCodes(ctx context.Context) (func(settings.AttributeValues) string, error) {
	attributes, err := x.settings.ListAttributes(ctx)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(attributes, func(a settings.Attribute) bool {
		return a.Level == settings.LevelActivity && strings.EqualFold(strings.TrimSpace(a.Label), companyCodeLabel)
	})
	if i < 0 {
		return func(settings.AttributeValues) string { return "" }, nil
	}
	options, err := x.settings.ListOptions(ctx)
	if err != nil {
		return nil, err
	}
	labels := map[string]string{}
	for _, o := range options {
		labels[o.ID] = o.Label
	}
	attributeID := attributes[i].ID
	return func(values settings.AttributeValues) string {
		var optionID string
		if raw, found := values[attributeID]; !found || json.Unmarshal(raw, &optionID) != nil {
			return ""
		}
		return strings.TrimSpace(labels[optionID])
	}, nil
}

func (w *wiring) swbsArea() error {
	s := w.set
	service := swbs.NewService(history.Tools{Tx: s.Tx, IDs: w.generator, Log: w.history, Now: w.now}, s.SwbsRuns,
		swbsLineItems{activities: s.Activities, lineItems: s.LineItems, settings: s.Settings}, swbs.NewSimulated(s.SwbsRuns))
	if err := w.register(service.Kinds()); err != nil {
		return err
	}
	mount(w.signedIn, swbs.NewHandler(service))
	return nil
}
