package activity_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
)

func TestActivityNameIsOneLine_038_FR055_L11(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	for _, name := range []string{"Launch\tvideo", "Launch\r\nvideo", "⁧Launch video"} {
		body := map[string]any{"name": name, "activityTypeId": "AT-1002"}

		assertOneLineName(t, priya.Do(t, http.MethodPost, "/api/v1/initiatives/I-1001/activities", body))
	}
}

func assertOneLineName(t *testing.T, response *apitest.Response) {
	t.Helper()
	var answer struct {
		Error struct {
			Fields []struct{ Field, Message string } `json:"fields"`
		} `json:"error"`
	}
	response.Decode(t, &answer)
	for _, problem := range answer.Error.Fields {
		if problem.Field == "name" && problem.Message == "Names can't contain tabs, line breaks or text-direction characters." {
			return
		}
	}
	t.Errorf("answer %d %s, want the single-line problem on name", response.Status, response.Body)
}
