package budgets

import (
	"net/http"
	"strconv"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/web"
)

// Handler answers the HTTP requests of global budgets and allocations (014 plan "API operations").
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the global budget routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the nine global budget routes on r. All of them need a signed-in person, so all are
// wrapped in signedIn, which also applies their access rules (014 plan Table C).
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/global-budgets", signedIn(h.list))
	r.Handle("POST /api/global-budgets", signedIn(h.create))
	r.Handle("GET /api/global-budgets/{globalBudgetId}", signedIn(h.get))
	r.Handle("PATCH /api/global-budgets/{globalBudgetId}", signedIn(h.update))
	r.Handle("DELETE /api/global-budgets/{globalBudgetId}", signedIn(h.delete))
	r.Handle("GET /api/global-budgets/{globalBudgetId}/allocation-options", signedIn(h.allocationOptions))
	r.Handle("POST /api/global-budgets/{globalBudgetId}/allocations", signedIn(h.allocate))
	r.Handle("PATCH /api/budget-allocations/{allocationId}", signedIn(h.updateAllocation))
	r.Handle("DELETE /api/budget-allocations/{allocationId}", signedIn(h.removeAllocation))
}

// list answers GET /api/global-budgets: the global budgets of the query's year, or of every year (FR-036).
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	var year *int
	if text := r.URL.Query().Get("year"); text != "" {
		parsed, err := strconv.Atoi(text)
		if err != nil {
			web.WriteError(w, r, errs.Validation(errs.FieldError{Field: "year", Message: msgChooseListYear}))
			return
		}
		year = &parsed
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	page, err := h.service.List(r.Context(), a, year)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, page)
}

// create answers POST /api/global-budgets with 201 and the new global budget.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input GlobalBudgetInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	view, err := h.service.Create(r.Context(), a, input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, view)
}

// get answers GET /api/global-budgets/{globalBudgetId} with the global budget and its figures.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	view, err := h.service.Get(r.Context(), a, r.PathValue("globalBudgetId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, view)
}

// update answers PATCH /api/global-budgets/{globalBudgetId} with the global budget after the save.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var update GlobalBudgetUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	view, err := h.service.Update(r.Context(), a, r.PathValue("globalBudgetId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, view)
}

// delete answers DELETE /api/global-budgets/{globalBudgetId} with the recycle-bin entry, so the web app can
// offer Undo.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	entry, err := h.service.Delete(r.Context(), a, r.PathValue("globalBudgetId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// allocationOptions answers GET /api/global-budgets/{globalBudgetId}/allocation-options: the plans the
// Allocate dialog offers.
func (h *Handler) allocationOptions(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	page, err := h.service.AllocationOptions(r.Context(), a, r.PathValue("globalBudgetId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, page)
}

// allocate answers POST /api/global-budgets/{globalBudgetId}/allocations with 201 and the global budget
// after the allocation.
func (h *Handler) allocate(w http.ResponseWriter, r *http.Request) {
	var input AllocationInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	view, err := h.service.Allocate(r.Context(), a, r.PathValue("globalBudgetId"), input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, view)
}

// updateAllocation answers PATCH /api/budget-allocations/{allocationId} with the global budget after the
// change.
func (h *Handler) updateAllocation(w http.ResponseWriter, r *http.Request) {
	var update AllocationUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	view, err := h.service.UpdateAllocation(r.Context(), a, r.PathValue("allocationId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, view)
}

// removeAllocation answers DELETE /api/budget-allocations/{allocationId} with the global budget after the
// removal.
func (h *Handler) removeAllocation(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	view, err := h.service.RemoveAllocation(r.Context(), a, r.PathValue("allocationId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, view)
}
