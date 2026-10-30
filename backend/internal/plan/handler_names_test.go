package plan_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"mapp/internal/platform/apitest"
)

func TestPlanNameIsOneLine_038_FR055_L11(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	for _, name := range []string{"FY2026\tLubricants", "FY2026\nLubricants", "FY2026 \u202eLubricants"} {
		body := newPlanBody()
		body["name"] = name

		assertOneLineName(t, priya.Do(t, http.MethodPost, "/api/v1/plans", body))
	}
}

func TestPlanNameRefusesUnpairedSurrogates_038_FR055_FR017_L11(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	for i, escaped := range []string{`~ud800`, `~uDC00`, `~udc00~ud800`, `~ud83d~ude00`, "😀"} {
		body := newPlanBody()
		body["name"] = "FY2026 Lubricants NAME"
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		escaped = strings.ReplaceAll(escaped, "~u", `\u`)
		raw = bytes.Replace(raw, []byte("NAME"), []byte(escaped+strconv.Itoa(i)), 1)
		response := priya.Do(t, http.MethodPost, "/api/v1/plans", nil, apitest.RawBody(raw), apitest.SkipRequestValidation())
		wantRefused := i < 3
		if refused := response.Status == http.StatusBadRequest &&
			strings.Contains(string(response.Body), `{"field":"name","message":"Remove the control characters from this text."}`); refused != wantRefused {
			t.Errorf("%s: answer %d %s, want refused %v", escaped, response.Status, response.Body, wantRefused)
		}
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
