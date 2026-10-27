package settings_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
)

// formView is the part of a form these tests read.
type formView struct {
	Sections []struct {
		Name        string
		Conditional bool
		Attributes  []struct {
			ID, Label string
			Options   []struct{ ID, Label string }
		}
	}
	BusinessUnitAttributeID string
	BusinessUnitOptions     []struct{ ID, Label string }
}

// ids returns the ids of the form's attributes by label, and of their options by "<attribute>/<option>".
func (f formView) ids() map[string]string {
	ids := map[string]string{}
	for _, s := range f.Sections {
		for _, a := range s.Attributes {
			ids[a.Label] = a.ID
			for _, o := range a.Options {
				ids[a.Label+"/"+o.Label] = o.ID
			}
		}
	}
	for _, o := range f.BusinessUnitOptions {
		ids["Business unit/"+o.Label] = o.ID
	}
	return ids
}

// getForm reads the form of level.
func getForm(t *testing.T, c *apitest.Client, level string) formView {
	t.Helper()
	var form formView
	c.Do(t, http.MethodGet, "/api/v1/shell-attribute-forms/"+level, nil).Decode(t, &form)
	return form
}

// The check answers what a save would say: a Region that no longer fits a changed Cluster is cleared, a hidden
// Lubricants attribute is not missing for a Mobility plan, and an unknown level is refused (FR-025, FR-026, FR-020).
func TestFormsAPI_Check_FR025(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	campaign := getForm(t, alex, "campaign").ids()
	activity := getForm(t, alex, "activity").ids()
	cluster, region := campaign["Cluster"], campaign["Region"]
	europe, china := campaign["Cluster/Lubricants – Europe"], campaign["Cluster/Retail – China"]
	dach := campaign["Region/LUBES – DACH"]

	changed := alex.Do(t, http.MethodPost, "/api/v1/shell-attribute-forms/campaign/check", map[string]any{
		"values":         map[string]any{cluster: []string{china}, region: []string{dach}},
		"previousValues": map[string]any{cluster: []string{europe}, region: []string{dach}},
	})
	mobility := alex.Do(t, http.MethodPost, "/api/v1/shell-attribute-forms/activity/check", map[string]any{
		"values": map[string]any{}, "businessUnitOptionIds": []string{activity["Business unit/Mobility"]},
	})
	unknown := alex.Do(t, http.MethodGet, "/api/v1/shell-attribute-forms/program", nil, apitest.SkipRequestValidation())

	assertAnswer(t, changed, http.StatusOK, `{"problems":[],"missing":[{"field":"attributes.`+region+`","label":"Region"}],`+
		`"cleared":[{"attributeId":"`+region+`","label":"Region","optionIds":["`+dach+`"],"optionLabels":["LUBES – DACH"]}],`+
		`"values":{"`+cluster+`":["`+china+`"]}}`)
	var result struct{ Missing []struct{ Label string } }
	mobility.Decode(t, &result)
	for _, m := range result.Missing {
		if m.Label == "Brands" || m.Label == "Sector" {
			t.Errorf("hidden Lubricants attribute %s missing for a Mobility plan", m.Label)
		}
	}
	if len(result.Missing) != 9 { // the nine needed attributes of section Shell attributes
		t.Errorf("missing = %+v", result.Missing)
	}
	if unknown.Status != http.StatusBadRequest {
		t.Errorf("unknown level = %d %s", unknown.Status, unknown.Body)
	}
}
