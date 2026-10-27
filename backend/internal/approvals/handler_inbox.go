package approvals

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

// inbox answers GET /api/v1/approvals?tab=waiting|open|decided (default waiting): every row of the tab.
func (h *Handler) inbox(c *gin.Context) {
	tab := c.Request.URL.Query().Get("tab")
	if tab == "" {
		tab = TabWaiting
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	rows, err := h.service.Rows(c.Request.Context(), a, tab)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, RowPage{Items: rows, Total: len(rows)})
}

// summary answers GET /api/v1/approvals/summary?monthStart&monthEnd: the counts of the summary cards.
func (h *Handler) summary(c *gin.Context) {
	query := c.Request.URL.Query()
	var problems errs.Problems
	monthStart := instant(query, "monthStart", &problems)
	monthEnd := instant(query, "monthEnd", &problems)
	if err := problems.Err(); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	counts, err := h.service.Summary(c.Request.Context(), a, monthStart, monthEnd)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, counts)
}

// instant reads the RFC 3339 time of the query parameter name: nil when it is absent; a value that is not
// a time adds a problem on name.
func instant(query url.Values, name string, problems *errs.Problems) *time.Time {
	text := query.Get(name)
	if text == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, text)
	if err != nil {
		problems.Add(name, reloadMessage)
		return nil
	}
	return &t
}

// preview answers GET /api/v1/approvals/activities/{activityId}: the preview panel of the Approvals page.
func (h *Handler) preview(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	preview, err := h.service.Preview(c.Request.Context(), a, c.Param("activityId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, preview)
}

// monitoring answers GET /api/v1/approvals/monitoring?monthStart&monthEnd: the Monitoring tab (017 FR-030).
func (h *Handler) monitoring(c *gin.Context) {
	query := c.Request.URL.Query()
	var problems errs.Problems
	monthStart := instant(query, "monthStart", &problems)
	monthEnd := instant(query, "monthEnd", &problems)
	if err := problems.Err(); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	m, err := h.service.Monitoring(c.Request.Context(), a, monthStart, monthEnd)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, m)
}

// runEscalations answers POST /api/v1/approvals/escalations/run: the reminders and escalations sent now (017
// FR-028).
func (h *Handler) runEscalations(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	run, err := h.service.RunEscalations(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, run)
}
