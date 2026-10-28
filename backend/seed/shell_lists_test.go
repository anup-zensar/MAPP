package seed_test

import (
	"slices"
	"testing"

	"mapp/internal/settings"
)

// The five lists Shell's Hive9 tenant shows live (FA E3.0.2) are seeded with Shell's own values, in that
// order, character for character, with their labels, markings and sections unchanged and GL code depending
// on nothing (034 FR-006 – FR-012, D-325).
func TestShellLists_034_FR006_FR010(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	tests := []struct {
		level   settings.Level
		label   string
		marking settings.Marking
		want    []string
	}{
		{settings.LevelLineItem, "GL code", settings.MarkingNeededToSubmit, []string{
			"6140900 - PMTDR Credit", "7250050 - Advertising - Media Planning/Auditing",
			"7250060 - Advertising - Corp/Agency/Production", "7250070 - Advertising - Media Buying",
			"7250080 - Advertising - Digital", "7250100 - Public Relations", "7250150 - Sponsorship programmes",
			"7250190 - Loyalty Programmes - Materials", "7250200 - Loyalty Programmes - Services", "7250250 - Market Research",
			"7250300 - Promotions & POP - Incentives/Reimbursement", "7250310 - Promotions & POP - Agency/Print/Logistics",
			"7250350 - Brand Development & Coordination"}},
		{settings.LevelLineItem, "Spend type", settings.MarkingNeededToSubmit, []string{
			"ATL - Agency Fees - Creative", "ATL - Agency Fees - PR", "ATL - Digital", "ATL - Event",
			"ATL - Paid Media Campaign Activation", "ATL - Paid Media Strategy and Planning", "ATL - Public Relations",
			"ATL - Sponsorships", "Audits e.g. mystery shopper", "BTL - Agency Fees", "BTL - Branded Shop",
			"BTL - Capability building Training", "BTL - CRM Campaign", "BTL - Customer Meetings/Conference",
			"BTL - Distributor Marketing Budget", "BTL - Event", "BTL - Incentive Scheme", "BTL - Loyalty Operations Costs",
			"BTL - Market research/Insights", "BTL - Merchandise", "BTL - Pipeline", "BTL - Promotions", "BTL - Sponsorship",
			"Contractors or Field Support", "CRM - Agency Fee"}},
		{settings.LevelLineItem, "Spend category", settings.MarkingOptional, []string{
			"Not Applicable", "ABM (B2B Media)", "B2B Customer Specific", "Brand", "Canada Customer Specific",
			"Channel - DIFM", "Channel - DIY", "Channel - Indirect", "Consumer Media", "Content, Creative & Assets",
			"Influencer Marketing", "Innovation", "Insights", "MarTech", "Motorsports", "Program", "Promotions",
			"Sustainability", "Tradeshow"}},
		{settings.LevelLineItem, "Contracted spend", settings.MarkingOptional, []string{
			"Not Applicable", "No", "Yes - Both", "Yes - LT", "Yes - ST"}},
		{settings.LevelActivity, "Zero-based cost hierarchy", settings.MarkingOptional, []string{
			"LUBES - Growth drivers", "LUBES - Legal or regulatory requirements", "LUBES - Long term contractual obligations",
			"LUBES - Other", "LUBES - Protecting the core", "LUBES - Strategic Priorities", "LUBES - Tactical initiatives",
			"RETAIL - Long-Term Contracts", "RETAIL - Strategic Priorities", "RETAIL - Tactical Investments"}},
	}
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			t.Parallel()
			attribute := d.attributeByID[d.attributeID(t, tt.level, tt.label)]
			var got []string
			for _, o := range d.options {
				if o.AttributeID == attribute.ID {
					got = append(got, o.Label)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("%s options = %q\nwant %q", tt.label, got, tt.want)
			}
			if attribute.Marking != tt.marking || attribute.ParentAttributeID != "" || len(attribute.Condition) != 0 {
				t.Errorf("%s: marking %s, parent %q, condition %q; want %s, none, every business unit",
					tt.label, attribute.Marking, attribute.ParentAttributeID, attribute.Condition, tt.marking)
			}
		})
	}
}
