package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"mapp/internal/lineitem"
	"mapp/internal/platform/actor"
	"mapp/internal/stores"
	"mapp/internal/swbs"
)

const companyCodeAttribute = "activity/Company code"

// The demo data's run gives Venue hire its SWBS code (030 FR-021); Stand build stays ready to send.
const (
	swbsDemoActivity = "Social always-on"
	swbsDemoLineItem = "Venue hire"
)

func (d *demo) addSwbsRun(ctx context.Context) error {
	request, err := d.swbsRequest(ctx, swbsDemoActivity, swbsDemoLineItem)
	if err != nil {
		return err
	}
	_, err = swbs.LoadDemo(ctx, d.stores.SwbsRuns, d.ids, swbs.DemoLoad{
		Requests: []swbs.Request{request}, Items: demoReceiver{repo: d.stores.LineItems}, At: d.at,
	})
	return err
}

func (d *demo) swbsRequest(ctx context.Context, activityName, lineItemName string) (swbs.Request, error) {
	activityID, err := find(d.activities, "activity", lubricantsPlan+"/"+activityName)
	if err != nil {
		return swbs.Request{}, err
	}
	act, _, err := d.stores.Activities.Get(ctx, activityID)
	if err != nil {
		return swbs.Request{}, err
	}
	items, err := d.stores.LineItems.ListByActivity(ctx, activityID)
	if err != nil {
		return swbs.Request{}, err
	}
	i := slices.IndexFunc(items, func(li lineitem.LineItem) bool { return li.Name == lineItemName })
	if i < 0 {
		return swbs.Request{}, fmt.Errorf("seed: '%s' has no line item '%s'", activityName, lineItemName)
	}
	var optionID string
	if err := json.Unmarshal(act.Attributes[d.settings.Attributes[companyCodeAttribute]], &optionID); err != nil {
		return swbs.Request{}, fmt.Errorf("seed: the company code of '%s': %w", activityName, err)
	}
	return swbs.Request{LineItemID: items[i].ID, LineItemName: lineItemName, CompanyCode: d.optionLabel(optionID)}, nil
}

func (d *demo) optionLabel(optionID string) string {
	for key, id := range d.settings.Options {
		if id == optionID {
			return key[strings.LastIndex(key, "/")+1:]
		}
	}
	return ""
}

type demoReceiver struct {
	repo stores.LineItemStore
}

func (r demoReceiver) Receive(ctx context.Context, a actor.Actor, answers []swbs.Answer, at time.Time) error {
	received := make([]lineitem.SwbsCodeReceived, 0, len(answers))
	for _, answer := range answers {
		received = append(received, lineitem.SwbsCodeReceived{LineItemID: answer.LineItemID, Code: answer.SwbsCode})
	}
	return lineitem.ReceiveSwbsCodes(ctx, r.repo, lineitem.SwbsReceipt{By: a, At: at, Codes: received})
}
