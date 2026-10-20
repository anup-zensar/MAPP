package allwork

import (
	"net/url"
	"slices"
	"testing"

	"northstar/internal/platform/errs"
)

// The Shell-attribute filters of 016 (FR-039) over the two years of twoYears: P-1001 (2026) › C-1 Europe ›
// I-1 › A-1 (Helix; Germany 60 % / France 40 %; FIT4 yes; L-1 Media, L-2 Event) and A-2 (Helix and Rimula;
// FIT4 no); P-2001 (2027) › C-2 Asia › I-2 › A-3 (no brand) and A-4 (L-3 Media).

// attrQuery is a Grid query with the attribute filters.
func attrQuery(filters ...AttrFilter) Query {
	return Query{Attributes: filters}
}

// attrActivities returns the activities the attribute filters show over 2026–2027.
func attrActivities(nodes []Node, filters ...AttrFilter) []string {
	return levelKeys(rangeView(nodes, attrQuery(filters...), 2026, 2027).result, LevelActivity)
}

// withAttr returns nodes with the attribute value of the record id replaced.
func withAttr(nodes []Node, id, attributeID string, v AttrValue) []Node {
	for i := range nodes {
		if nodes[i].ID == id {
			attrs := map[string]AttrValue{}
			for k, old := range nodes[i].Attrs {
				attrs[k] = old
			}
			attrs[attributeID] = v
			nodes[i].Attrs = attrs
		}
	}
	return nodes
}

// Each filter is tested on its attribute's level (D-113): a campaign attribute keeps the campaign's branch, the
// plans above keep only a match below; a line-item attribute is met by an activity one of whose line items
// meets it (FR-039).
func TestAttrFilter_TestedOnLevel_FR039(t *testing.T) {
	europe := rangeView(twoYears(), attrQuery(AttrFilter{AttributeID: "SA-CL", Values: []string{"SO-EU"}}), 2026, 2027).result
	event := attrActivities(twoYears(), AttrFilter{AttributeID: "SA-S", Values: []string{"SO-EV"}})
	media := attrActivities(twoYears(), AttrFilter{AttributeID: "SA-S", Values: []string{"SO-ME"}})

	if got := levelKeys(europe, LevelActivity); !slices.Equal(got, []string{"A-1", "A-2"}) {
		t.Errorf("Cluster Europe: activities %v, want A-1 and A-2", got)
	}
	if got := levelKeys(europe, LevelPlan, LevelCampaign); !slices.Equal(got, []string{"P-1001", "C-1"}) {
		t.Errorf("Cluster Europe: plans and campaigns %v, want P-1001 and C-1", got)
	}
	if !slices.Equal(event, []string{"A-1"}) || !slices.Equal(media, []string{"A-1", "A-4"}) {
		t.Errorf("Spend type Event %v (want A-1), Media %v (want A-1, A-4)", event, media)
	}
}

// Not filled in matches the records without a value, alone or beside chosen values (FR-039).
func TestAttrFilter_NotFilledIn_FR039(t *testing.T) {
	none := attrActivities(twoYears(), AttrFilter{AttributeID: "SA-B", Values: []string{AttrNotFilledIn}})
	rimulaOrNone := attrActivities(twoYears(), AttrFilter{AttributeID: "SA-B", Values: []string{"SO-RM", AttrNotFilledIn}})

	if !slices.Equal(none, []string{"A-3", "A-4"}) {
		t.Errorf("Brands Not filled in = %v, want A-3 and A-4", none)
	}
	if !slices.Equal(rimulaOrNone, []string{"A-2", "A-3", "A-4"}) {
		t.Errorf("Brands Rimula or Not filled in = %v, want A-2, A-3 and A-4", rimulaOrNone)
	}
}

// A % split value matches an option only while its share is above 0 (FR-039).
func TestAttrFilter_SplitShareAboveZero_FR039(t *testing.T) {
	france := AttrFilter{AttributeID: "SA-C", Values: []string{"SO-FR"}}
	zero := withAttr(twoYears(), "A-1", "SA-C", shares("SO-DE", "100", "SO-FR", "0"))

	if got := attrActivities(twoYears(), france); !slices.Equal(got, []string{"A-1"}) {
		t.Errorf("France at 40 %% = %v, want A-1", got)
	}
	if got := attrActivities(zero, france); len(got) != 0 {
		t.Errorf("France at 0 %% = %v, want none", got)
	}
}

// A value hidden by its attribute's Business unit condition counts as not filled in (FR-039): Mobility channel
// shows only for Mobility plans, so A-1's value in the Lubricants plan is not filled in, A-3's in the Mobility
// plan is.
func TestAttrFilter_HiddenCountsAsNotFilled_FR039(t *testing.T) {
	nodes := withAttr(withAttr(twoYears(), "A-1", "SA-M", options("SO-X")), "A-3", "SA-M", options("SO-X"))

	fleet := attrActivities(nodes, AttrFilter{AttributeID: "SA-M", Values: []string{"SO-X"}})
	none := attrActivities(nodes, AttrFilter{AttributeID: "SA-M", Values: []string{AttrNotFilledIn}})

	if !slices.Equal(fleet, []string{"A-3"}) {
		t.Errorf("Fleet = %v, want only A-3 (A-1's value is hidden)", fleet)
	}
	if !slices.Equal(none, []string{"A-1", "A-2", "A-4"}) {
		t.Errorf("Not filled in = %v, want A-1, A-2 and A-4", none)
	}
}

// A yes/no attribute filters on Yes, No and Not filled in.
func TestAttrFilter_YesNo(t *testing.T) {
	yesAnswers := attrActivities(twoYears(), AttrFilter{AttributeID: "SA-F", Values: []string{AttrYes}})
	noAnswers := attrActivities(twoYears(), AttrFilter{AttributeID: "SA-F", Values: []string{AttrNo}})
	none := attrActivities(twoYears(), AttrFilter{AttributeID: "SA-F", Values: []string{AttrNotFilledIn}})

	if !slices.Equal(yesAnswers, []string{"A-1"}) || !slices.Equal(noAnswers, []string{"A-2"}) || !slices.Equal(none, []string{"A-3", "A-4"}) {
		t.Errorf("FIT4 yes %v, no %v, not filled in %v", yesAnswers, noAnswers, none)
	}
}

// Filters of an attribute that cannot be filtered (a number) or no longer exists are ignored; two filters of
// one attribute combine with OR, two attributes with AND.
func TestAttrFilter_IgnoredAndCombined(t *testing.T) {
	ignored := attrActivities(twoYears(), AttrFilter{AttributeID: "SA-N", Values: []string{"5"}}, AttrFilter{AttributeID: "SA-GONE", Values: []string{"x"}})
	both := attrActivities(twoYears(), AttrFilter{AttributeID: "SA-B", Values: []string{"SO-HX"}}, AttrFilter{AttributeID: "SA-F", Values: []string{AttrNo}})

	if !slices.Equal(ignored, []string{"A-1", "A-2", "A-3", "A-4"}) {
		t.Errorf("ignored filters = %v, want every activity", ignored)
	}
	if !slices.Equal(both, []string{"A-2"}) {
		t.Errorf("Helix and FIT4 no = %v, want A-2", both)
	}
}

// The attr parameter is "<attributeId>:<value>,<value>"; repeated for one attribute its values add up; any
// other form is refused on attr (FR-039, D-219).
func TestQuery_AttrParameterRefused(t *testing.T) {
	q, err := ParseQuery(url.Values{"attr": {"SA-B:SO-HX,none", "SA-F:yes", "SA-B:SO-RM,SO-HX"}})
	_, noColon := ParseQuery(url.Values{"attr": {"SA-B"}})
	_, noValues := ParseQuery(url.Values{"attr": {"SA-B: , "}})
	_, noAttribute := ParseQuery(url.Values{"attr": {":SO-HX"}})

	want := []AttrFilter{{AttributeID: "SA-B", Values: []string{"SO-HX", "none", "SO-RM"}}, {AttributeID: "SA-F", Values: []string{"yes"}}}
	if err != nil || len(q.Attributes) != 2 || !slices.Equal(q.Attributes[0].Values, want[0].Values) ||
		q.Attributes[1].AttributeID != "SA-F" || !slices.Equal(q.Attributes[1].Values, want[1].Values) {
		t.Errorf("attributes = %+v, %v; want %+v", q.Attributes, err, want)
	}
	problem := []errs.FieldError{{Field: "attr", Message: "Choose a Shell attribute and its values, such as SA-1022:SO-1301,none."}}
	assertProblems(t, noColon, problem)
	assertProblems(t, noValues, problem)
	assertProblems(t, noAttribute, problem)
}
