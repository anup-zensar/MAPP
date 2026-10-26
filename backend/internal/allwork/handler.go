package allwork

import (
	"net/http"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

// Handler answers the HTTP requests of All work (plan "API operations").
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the All work routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the All work routes on r. All of them need a signed-in person, so all are wrapped in
// signedIn.
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/v1/all-work", signedIn(h.query))
	r.Handle("GET /api/v1/all-work/options", signedIn(h.options))
	r.Handle("GET /api/v1/all-work/export", signedIn(h.export))
	r.Handle("GET /api/v1/all-work/records/{recordId}", signedIn(h.locate))
	r.Handle("GET /api/v1/all-work/preferences", signedIn(h.preferences))
	r.Handle("PUT /api/v1/all-work/preferences", signedIn(h.updatePreferences))
	r.Handle("GET /api/v1/saved-views", signedIn(h.listViews))
	r.Handle("POST /api/v1/saved-views", signedIn(h.createView))
	r.Handle("PATCH /api/v1/saved-views/{viewId}", signedIn(h.updateView))
	r.Handle("DELETE /api/v1/saved-views/{viewId}", signedIn(h.deleteView))
	r.Handle("GET /api/v1/plans/{planId}/budget-by-month", signedIn(h.budgetByMonth)) // 016 FR-024
	r.Handle("GET /api/v1/all-work/breakdown", signedIn(h.breakdown))                 // 016 FR-040
}

// breakdown answers GET /api/v1/all-work/breakdown: the Breakdown of the filter set of the query, by the
// grouping by and then by thenBy (016 FR-040 - FR-042). Every wrong parameter is reported at once.
func (h *Handler) breakdown(w http.ResponseWriter, r *http.Request) {
	q, err := ParseQuery(r.URL.Query())
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	result, err := h.service.Breakdown(r.Context(), a, q)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, result)
}

// budgetByMonth answers GET /api/v1/plans/{planId}/budget-by-month: the phased budget against the planned cost
// of the plan and each of its campaigns and initiatives, in the plan currency (016 FR-024).
func (h *Handler) budgetByMonth(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	result, err := h.service.BudgetByMonth(r.Context(), a, r.PathValue("planId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, result)
}

// listViews answers GET /api/v1/saved-views: the signed-in person's saved views.
func (h *Handler) listViews(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	page, err := h.service.ListViews(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, page)
}

// createView answers POST /api/v1/saved-views with 201 and the new view.
func (h *Handler) createView(w http.ResponseWriter, r *http.Request) {
	var in SavedViewCreate
	if err := web.ReadJSON(w, r, &in); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	view, err := h.service.CreateView(r.Context(), a, in)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, view)
}

// updateView answers PATCH /api/v1/saved-views/{viewId} with the view after the change.
func (h *Handler) updateView(w http.ResponseWriter, r *http.Request) {
	var in SavedViewUpdate
	if err := web.ReadJSON(w, r, &in); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	view, err := h.service.UpdateView(r.Context(), a, r.PathValue("viewId"), in)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, view)
}

// deleteView answers DELETE /api/v1/saved-views/{viewId} with the deleted view, so the web app can offer Undo.
func (h *Handler) deleteView(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	view, err := h.service.DeleteView(r.Context(), a, r.PathValue("viewId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, view)
}

// preferences answers GET /api/v1/all-work/preferences: the signed-in person's remembered columns and sort.
func (h *Handler) preferences(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	p, err := h.service.Preferences(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, p)
}

// updatePreferences answers PUT /api/v1/all-work/preferences with the preferences after the change.
func (h *Handler) updatePreferences(w http.ResponseWriter, r *http.Request) {
	var p Preferences
	if err := web.ReadJSON(w, r, &p); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	saved, err := h.service.UpdatePreferences(r.Context(), a, p)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, saved)
}

// options answers GET /api/v1/all-work/options: the choices of the filter bar for the year of the query (the
// current year without one) and the column catalogue.
func (h *Handler) options(w http.ResponseWriter, r *http.Request) {
	var problems errs.Problems
	from, to := parseRange(r.URL.Query(), &problems) // 016 FR-034: year, or yearFrom and yearTo
	if err := problems.Err(); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	options, err := h.service.Options(r.Context(), a, from, to)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, options)
}

// query answers GET /api/v1/all-work: everything one view shows for the filter set of the query. Every wrong
// parameter is reported at once.
func (h *Handler) query(w http.ResponseWriter, r *http.Request) {
	q, err := ParseQuery(r.URL.Query())
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	result, err := h.service.Query(r.Context(), a, q)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, result)
}

// export answers GET /api/v1/all-work/export with the view of the query as an .xlsx or CSV file, named in
// Content-Disposition (FR-032 – FR-034). Every wrong parameter is reported at once.
func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	q, format, unassigned, err := ParseExport(r.URL.Query())
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	file, err := h.service.Export(r.Context(), a, q, format, unassigned)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", ContentDisposition(file.Name))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Body) // a failed write means the client has gone; there is nobody left to tell
}

// locate answers GET /api/v1/all-work/records/{recordId}: the record a row opens, or 404 with what became of it.
func (h *Handler) locate(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	location, err := h.service.Locate(r.Context(), a, r.PathValue("recordId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, location)
}
