package budget_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
)

func TestGlobalBudgetNameIsOneLine_038_FR055_L11(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	for _, name := range []string{"FY2027\tbrand", "FY2027\nbrand", "FY2027 ‫brand"} {
		body := map[string]any{"name": name, "year": 2027, "amount": usd("900000.00"), "ownerId": "U-1005"}

		assertOneLineName(t, carlos.Do(t, http.MethodPost, "/api/v1/global-budgets", body))
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
