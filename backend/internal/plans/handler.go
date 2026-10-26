package plans

import (
	"net/http"
	"strconv"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
	"mapp/internal/rollup"
)

// Handler answers the HTTP requests of plans (plan "API operations").
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the plan routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the plan routes on r. All of them need a signed-in person, so all are wrapped in
// signedIn.
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/plan-defaults", signedIn(h.defaults))
	r.Handle("GET /api/plans", signedIn(h.list))
	r.Handle("POST /api/plans", signedIn(h.create))
	r.Handle("GET /api/plans/{planId}", signedIn(h.get))
	r.Handle("PATCH /api/plans/{planId}", signedIn(h.update))
	r.Handle("DELETE /api/plans/{planId}", signedIn(h.delete))
	r.Handle("POST /api/plans/{planId}/change-effects", signedIn(h.changeEffects))
	r.Handle("POST /api/plans/{planId}/publish", signedIn(h.publish))
	r.Handle("POST /api/plans/{planId}/unpublish", signedIn(h.unpublish))
	r.Handle("GET /api/plans/{planId}/copy-options", signedIn(h.copyOptions))
	r.Handle("POST /api/plans/{planId}/copy", signedIn(h.copy))
	r.Handle("PUT /api/plans/{planId}/budget-phasing", signedIn(h.setBudgetPhasing)) // 016 FR-020
}

// setBudgetPhasing answers PUT /api/plans/{planId}/budget-phasing with the plan after the save (016 FR-020).
func (h *Handler) setBudgetPhasing(w http.ResponseWriter, r *http.Request) {
	var input rollup.PhasingInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	plan, err := h.service.SetBudgetPhasing(r.Context(), a, r.PathValue("planId"), input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, plan)
}

// defaults answers GET /api/plan-defaults: what a new plan starts with for the signed-in person.
func (h *Handler) defaults(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	defaults, err := h.service.Defaults(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, defaults)
}

// create answers POST /api/plans with 201 and the new plan.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input PlanInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	plan, err := h.service.Create(r.Context(), a, input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, plan)
}

// get answers GET /api/plans/{planId} with the plan and its calculated values.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	plan, err := h.service.Get(r.Context(), a, r.PathValue("planId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, plan)
}

// list answers GET /api/plans: the plans matching the query's scope, status, years (year, or yearFrom and
// yearTo, 016 FR-033), plan types and search, with the summary cards (FR-001 – FR-004). Every wrong
// parameter is reported at once.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := ListFilter{Scope: query.Get("scope"), Status: query.Get("status"), Q: query.Get("q"), Types: query["type"]}
	var yearProblems errs.Problems
	for _, year := range []struct {
		name   string
		target **int
	}{{"year", &filter.Year}, {"yearFrom", &filter.YearFrom}, {"yearTo", &filter.YearTo}} {
		name, target := year.name, year.target
		if text := query.Get(name); text != "" {
			year, err := strconv.Atoi(text)
			if err != nil {
				yearProblems.Add(name, msgChooseListYear)
				continue
			}
			*target = &year
		}
	}
	// In the order of the query: scope and status, then the years, then the range and the plan types.
	var problems errs.Problems
	others := filterProblems(filter)
	for _, p := range others {
		if p.Field == "scope" || p.Field == "status" {
			problems.Add(p.Field, p.Message)
		}
	}
	problems = append(problems, yearProblems...)
	for _, p := range others {
		if p.Field != "scope" && p.Field != "status" {
			problems.Add(p.Field, p.Message)
		}
	}
	if err := problems.Err(); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	list, err := h.service.List(r.Context(), a, filter)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, list)
}

// update answers PATCH /api/plans/{planId} with the plan after the save.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var update PlanUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	plan, err := h.service.Update(r.Context(), a, r.PathValue("planId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, plan)
}

// delete answers DELETE /api/plans/{planId} with the recycle-bin entry of the plan, so the web app can
// offer Undo.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	entry, err := h.service.Delete(r.Context(), a, r.PathValue("planId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// changeEffects answers POST /api/plans/{planId}/change-effects with what a new model and business unit
// would change; nothing is saved.
func (h *Handler) changeEffects(w http.ResponseWriter, r *http.Request) {
	var request ChangeEffectsRequest
	if err := web.ReadJSON(w, r, &request); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	effects, err := h.service.ChangeEffects(r.Context(), a, r.PathValue("planId"), request)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, effects)
}

// publish answers POST /api/plans/{planId}/publish with the Published plan, or 409 not_ready listing
// every missing field.
func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	plan, err := h.service.Publish(r.Context(), a, r.PathValue("planId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, plan)
}

// unpublish answers POST /api/plans/{planId}/unpublish with the plan back in Draft.
func (h *Handler) unpublish(w http.ResponseWriter, r *http.Request) {
	var request UnpublishRequest
	if err := web.ReadJSON(w, r, &request); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	plan, err := h.service.Unpublish(r.Context(), a, r.PathValue("planId"), request)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, plan)
}

// copyOptions answers GET /api/plans/{planId}/copy-options: what the copy dialog offers.
func (h *Handler) copyOptions(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	options, err := h.service.CopyOptions(r.Context(), a, r.PathValue("planId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, options)
}

// copy answers POST /api/plans/{planId}/copy with 201, the new plan and what the copy changed on purpose.
func (h *Handler) copy(w http.ResponseWriter, r *http.Request) {
	var request CopyRequest
	if err := web.ReadJSON(w, r, &request); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	result, err := h.service.Copy(r.Context(), a, r.PathValue("planId"), request)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, result)
}
