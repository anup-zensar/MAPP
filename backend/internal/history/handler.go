package history

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

// defaultLimit is how many log entries GET /api/activity-log answers when the request does not say.
const defaultLimit = 50

// Handler answers the HTTP requests about the activity log and the recycle bin. Feature 011 builds the
// screens on them; feature 001 uses Undo after a delete and the developer sample page.
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the history routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the history routes on r. All of them need a signed-in person, so all are wrapped in
// signedIn.
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/activity-log", signedIn(h.listLog))
	r.Handle("GET /api/activity-log/summary", signedIn(h.logSummary))
	r.Handle("GET /api/recycle-bin", signedIn(h.listBin))
	r.Handle("GET /api/recycle-bin/{entryId}/restore-preview", signedIn(h.previewRestore))
	r.Handle("DELETE /api/recycle-bin/{entryId}", signedIn(h.purge))
	r.Handle("POST /api/recycle-bin/empty", signedIn(h.emptyBin))
	r.Handle("POST /api/recycle-bin/{entryId}/undo", signedIn(h.undo))
	r.Handle("POST /api/recycle-bin/{entryId}/restore", signedIn(h.restore))
}

// listLog answers GET /api/activity-log: the newest entries that match the filters (FR-003, FR-009) —
// actorId, recordType, action (repeated), planId, recordId with includeInside, from, to, q — older than
// beforeSeq, at most limit of them (1 to 200, 50 when not sent). Every problem with the parameters is
// answered at once.
func (h *Handler) listLog(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	var problems errs.Problems
	q := LogQuery{
		ActorIDs:    values(params, "actorId"),
		RecordTypes: values(params, "recordType"),
		Actions:     values(params, "action"),
		PlanIDs:     values(params, "planId"),
		RecordID:    strings.TrimSpace(params.Get("recordId")),
		Search:      params.Get("q"),
		Limit:       defaultLimit,
	}
	q.IncludeInside = boolParam(params, "includeInside", &problems)
	q.From = timeParam(params, "from", false, &problems)
	q.To = timeParam(params, "to", false, &problems)
	if params.Has("beforeSeq") {
		seq, err := strconv.Atoi(params.Get("beforeSeq"))
		if err != nil || seq < 1 {
			problems.Add("beforeSeq", reloadMessage)
		}
		q.BeforeSeq = max(seq, 0)
	}
	if params.Has("limit") {
		limit, err := strconv.Atoi(params.Get("limit"))
		if err != nil { // not a whole number; the service checks the range of the numbers
			problems.Add("limit", limitMessage)
			limit = defaultLimit
		}
		q.Limit = limit
	}
	if len(problems) > 0 {
		web.WriteError(w, r, withQueryProblems(problems, q))
		return
	}

	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	page, err := h.service.QueryLog(r.Context(), a, q)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, page)
}

// logParameters are the query parameters of GET /api/activity-log in the contract's order, the order in
// which their problems are listed.
var logParameters = []string{
	"actorId", "recordType", "action", "planId", "recordId", "includeInside", "from", "to", "q", "beforeSeq", "limit",
}

// withQueryProblems adds the problems of q to the problems of reading the parameters, so the answer lists
// them all at once, each field once, in the order of the parameters.
func withQueryProblems(problems errs.Problems, q LogQuery) error {
	for _, p := range q.problems() {
		if !slices.ContainsFunc(problems, func(known errs.FieldError) bool { return known.Field == p.Field }) {
			problems = append(problems, p)
		}
	}
	slices.SortStableFunc(problems, func(x, y errs.FieldError) int {
		return slices.Index(logParameters, x.Field) - slices.Index(logParameters, y.Field)
	})
	return problems.Err()
}

// logSummary answers GET /api/activity-log/summary: the summary cards and the filter choices of the
// activity log (FR-002, FR-003), for the todayStart and weekStart the browser sends.
func (h *Handler) logSummary(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	var problems errs.Problems
	todayStart := timeParam(params, "todayStart", true, &problems)
	weekStart := timeParam(params, "weekStart", true, &problems)
	if err := problems.Err(); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	summary, err := h.service.LogSummary(r.Context(), a, *todayStart, *weekStart)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, summary)
}

// values returns the values of the query parameter name that are not blank.
func values(params url.Values, name string) []string {
	var list []string
	for _, v := range params[name] {
		if v = strings.TrimSpace(v); v != "" {
			list = append(list, v)
		}
	}
	return list
}

// boolParam reads the query parameter name, which is "true" or "false" when sent, and false when not sent.
// Any other value is a problem: only the web app sends it.
func boolParam(params url.Values, name string, problems *errs.Problems) bool {
	if !params.Has(name) {
		return false
	}
	switch params.Get(name) {
	case "true":
		return true
	case "false":
		return false
	}
	problems.Add(name, reloadMessage)
	return false
}

// timeParam reads the query parameter name, a moment such as 2026-03-11T23:00:00Z, and returns nil when
// it is not sent or is not a moment. A required parameter that is missing is a problem, and so is a value
// that is not a moment.
func timeParam(params url.Values, name string, required bool, problems *errs.Problems) *time.Time {
	if !params.Has(name) {
		if required {
			problems.Add(name, reloadMessage)
		}
		return nil
	}
	moment, err := time.Parse(time.RFC3339, params.Get(name))
	if err != nil {
		problems.Add(name, dateMessage)
		return nil
	}
	return &moment
}

// listBin answers GET /api/recycle-bin: everything in the recycle bin, newest first, with restore states
// and the records each entry holds, without their data (011 FR-014 – FR-017).
func (h *Handler) listBin(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	items, err := h.service.ListBinItems(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, BinPage{Items: items, Total: len(items)})
}

// previewRestore answers GET /api/recycle-bin/{entryId}/restore-preview: what a restore would do, without
// doing it (011 FR-018 – FR-024).
func (h *Handler) previewRestore(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	preview, err := h.service.PreviewRestore(r.Context(), a, r.PathValue("entryId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, preview)
}

// undo answers POST /api/recycle-bin/{entryId}/undo: the Undo offered right after a delete (FR-026).
func (h *Handler) undo(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	result, err := h.service.Undo(r.Context(), a, r.PathValue("entryId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, result)
}

// restore answers POST /api/recycle-bin/{entryId}/restore: a restore from the recycle bin (FR-025), with
// withParents=true also of the entries holding its missing parents, parents first (011 FR-023).
func (h *Handler) restore(w http.ResponseWriter, r *http.Request) {
	var problems errs.Problems
	withParents := boolParam(r.URL.Query(), "withParents", &problems)
	if err := problems.Err(); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	restore := h.service.Restore
	if withParents {
		restore = h.service.RestoreWithParents
	}
	result, err := restore(r.Context(), a, r.PathValue("entryId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, result)
}

// purge answers DELETE /api/recycle-bin/{entryId}: Delete permanently (011 FR-028, FR-030).
func (h *Handler) purge(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	result, err := h.service.Purge(r.Context(), a, r.PathValue("entryId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, result)
}

// emptyBin answers POST /api/recycle-bin/empty: Empty recycle bin, for the entries the person saw in the
// confirmation (011 FR-029, FR-030, D-180).
func (h *Handler) emptyBin(w http.ResponseWriter, r *http.Request) {
	var request EmptyRequest
	if err := web.ReadJSON(w, r, &request); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	result, err := h.service.EmptyBin(r.Context(), a, request.EntryIDs)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, result)
}
