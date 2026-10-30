package settings_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
)

func TestSettingsNamesAreOneLine_038_FR055_L11(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	for _, name := range []string{"Web\tinar", "Web\ninar", "\u202aWebinar"} {
		activityType := webinarBody("MOD-1001")
		activityType["name"] = name
		model := map[string]any{}
		for key, value := range lubricants2 {
			model[key] = value
		}
		model["name"] = name

		assertOneLineName(t, alex.Do(t, http.MethodPost, "/api/v1/activity-types", activityType), "name")
		assertOneLineName(t, alex.Do(t, http.MethodPost, "/api/v1/models", model), "name")
	}
}

func assertOneLineName(t *testing.T, response *apitest.Response, field string) {
	t.Helper()
	var answer struct {
		Error struct {
			Fields []struct{ Field, Message string } `json:"fields"`
		} `json:"error"`
	}
	response.Decode(t, &answer)
	for _, problem := range answer.Error.Fields {
		if problem.Field == field && problem.Message == "Names can't contain tabs, line breaks or text-direction characters." {
			return
		}
	}
	t.Errorf("answer %d %s, want the single-line problem on %s", response.Status, response.Body, field)
}
