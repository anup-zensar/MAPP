package financials

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
)

// Record returns everything a Financials card shows for the plan, campaign, initiative, activity or line
// item recordID (FR-011 – FR-019): its figures, a row per live child, its 5 latest transactions, the SWBS
// codes of a line item's transactions and where the data comes from. A record that is not live, or is
// outside a's access, is not_found (FR-028).
func (s *Service) Record(ctx context.Context, a actor.Actor, recordID string) (RecordView, error) {
	var view RecordView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		gone := errs.NotFound(fmt.Sprintf(msgRecordGone, levelWordOf(recordID)))
		planID, live, err := s.planning.PlanOf(ctx, recordID)
		if err != nil {
			return err
		}
		if !live {
			return gone
		}
		b, err := s.loadPlans(ctx, a.ID, []string{planID})
		if err != nil {
			return err
		}
		n := b.tree.byID[recordID]
		if n == nil {
			return gone
		}
		if err := s.access.SeeFinanceIn(ctx, a, n.PlanID); err != nil {
			return orGone(err, gone)
		}
		view = b.recordView(n)
		return nil
	})
	if err != nil {
		return RecordView{}, err
	}
	return view, nil
}

// recordView returns the card of n.
func (b *book) recordView(n *tnode) RecordView {
	v := RecordView{
		Record: n.ref(), PlanID: n.PlanID, Path: n.path(), Figures: n.figures(b.display), Breakdown: []BreakdownRow{},
		Latest: []TransactionRow{}, SwbsCodes: []string{}, Source: b.source(),
	}
	if level, found := childLevel[n.Level]; found {
		v.BreakdownLevel = &level
		for _, c := range n.children {
			v.Breakdown = append(v.Breakdown, BreakdownRow{Record: c.ref(), Figures: c.figures(b.display)})
		}
	}
	latest := b.below(n)
	for _, t := range latest[:min(latestCount, len(latest))] {
		v.Latest = append(v.Latest, b.row(t))
	}
	if n.Level == LevelLineItem {
		for _, t := range b.byLineItem[n.ID] {
			if t.SwbsCode != "" && !slices.Contains(v.SwbsCodes, t.SwbsCode) {
				v.SwbsCodes = append(v.SwbsCodes, t.SwbsCode)
			}
		}
		slices.Sort(v.SwbsCodes)
	}
	return v
}

// levelWordOf names the level of a readable ID mid-sentence: "activity" for A-1002.
func levelWordOf(recordID string) string {
	prefix, _, _ := strings.Cut(recordID, "-")
	for level, p := range map[Level]string{
		LevelPlan: ids.PrefixPlan, LevelCampaign: ids.PrefixCampaign, LevelInitiative: ids.PrefixInitiative,
		LevelActivity: ids.PrefixActivity, LevelLineItem: ids.PrefixLineItem,
	} {
		if p == prefix {
			return levelWords[level]
		}
	}
	return "record"
}
