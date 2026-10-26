package activities

import (
	"net/http"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// Handler answers the HTTP requests of activities.
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the activity routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the activity routes on r; every one needs a signed-in person, so all are wrapped in
// signedIn.
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/v1/initiatives/{initiativeId}/activity-form", signedIn(h.newForm))
	r.Handle("POST /api/v1/initiatives/{initiativeId}/activities", signedIn(h.create))
	r.Handle("GET /api/v1/plans/{planId}/activities", signedIn(h.listForPlan))
	r.Handle("GET /api/v1/activities/{activityId}", signedIn(h.get))
	r.Handle("PATCH /api/v1/activities/{activityId}", signedIn(h.update))
	r.Handle("DELETE /api/v1/activities/{activityId}", signedIn(h.delete))
	r.Handle("GET /api/v1/activities/{activityId}/form", signedIn(h.editForm))
	r.Handle("GET /api/v1/activities/{activityId}/destinations", signedIn(h.destinations))
	r.Handle("POST /api/v1/activities/{activityId}/move", signedIn(h.move))
	r.Handle("POST /api/v1/activities/{activityId}/copy", signedIn(h.copy))
	r.Handle("GET /api/v1/activities/{activityId}/split-options", signedIn(h.splitOptions))  // 016 FR-026
	r.Handle("POST /api/v1/activities/{activityId}/split-preview", signedIn(h.splitPreview)) // 016 FR-027
	r.Handle("POST /api/v1/activities/{activityId}/split", signedIn(h.split))                // 016 FR-028
}

// splitOptions answers GET /api/v1/activities/{activityId}/split-options: what the Split dialog offers.
func (h *Handler) splitOptions(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	options, err := h.service.SplitOptions(r.Context(), a, r.PathValue("activityId"))
	answer(w, r, http.StatusOK, options, err)
}

// splitPreview answers POST /api/v1/activities/{activityId}/split-preview: both activities as the split would
// leave them and its problems, in the body (200).
func (h *Handler) splitPreview(w http.ResponseWriter, r *http.Request) {
	var in SplitRequest
	if err := web.ReadJSON(w, r, &in); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	preview, err := h.service.PreviewSplit(r.Context(), a, r.PathValue("activityId"), in)
	answer(w, r, http.StatusOK, preview, err)
}

// split answers POST /api/v1/activities/{activityId}/split with 201, the original and the new activity.
func (h *Handler) split(w http.ResponseWriter, r *http.Request) {
	var in SplitRequest
	if err := web.ReadJSON(w, r, &in); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	result, err := h.service.Split(r.Context(), a, r.PathValue("activityId"), in)
	answer(w, r, http.StatusCreated, result, err)
}

// newForm answers GET /api/v1/initiatives/{initiativeId}/activity-form: the new-activity form with defaults.
func (h *Handler) newForm(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	form, err := h.service.NewForm(r.Context(), a, r.PathValue("initiativeId"))
	answer(w, r, http.StatusOK, form, err)
}

// create answers POST /api/v1/initiatives/{initiativeId}/activities with 201 and the new activity.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input Input
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	view, err := h.service.Create(r.Context(), a, r.PathValue("initiativeId"), input)
	answer(w, r, http.StatusCreated, view, err)
}

// listForPlan answers GET /api/v1/plans/{planId}/activities: the plan's activities as plan-tree rows.
func (h *Handler) listForPlan(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	summaries, err := h.service.ListForPlan(r.Context(), a, r.PathValue("planId"))
	page := SummaryPage{Items: summaries, Total: len(summaries)}
	if err == nil {
		page.Access, err = h.service.PlanContentAccess(r.Context(), r.PathValue("planId"))
	}
	answer(w, r, http.StatusOK, page, err)
}

// get answers GET /api/v1/activities/{activityId}.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	view, err := h.service.Get(r.Context(), a, r.PathValue("activityId"))
	answer(w, r, http.StatusOK, view, err)
}

// update answers PATCH /api/v1/activities/{activityId} with the activity after the save.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var update Update
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	view, err := h.service.Update(r.Context(), a, r.PathValue("activityId"), update)
	answer(w, r, http.StatusOK, view, err)
}

// delete answers DELETE /api/v1/activities/{activityId} with the recycle-bin entry, so the web app can offer
// Undo.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	entry, err := h.service.Delete(r.Context(), a, r.PathValue("activityId"))
	answer(w, r, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()}, err)
}

// editForm answers GET /api/v1/activities/{activityId}/form: the edit form, without defaults.
func (h *Handler) editForm(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	form, err := h.service.EditForm(r.Context(), a, r.PathValue("activityId"))
	answer(w, r, http.StatusOK, form, err)
}

// destinations answers GET /api/v1/activities/{activityId}/destinations: every initiative of the plan, with
// why the activity can't be moved or copied there.
func (h *Handler) destinations(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	destinations, err := h.service.Destinations(r.Context(), a, r.PathValue("activityId"))
	answer(w, r, http.StatusOK, DestinationPage{Items: destinations, Total: len(destinations)}, err)
}

// move answers POST /api/v1/activities/{activityId}/move with the activity in its new initiative.
func (h *Handler) move(w http.ResponseWriter, r *http.Request) {
	var in MoveInput
	if err := web.ReadJSON(w, r, &in); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	view, err := h.service.Move(r.Context(), a, r.PathValue("activityId"), in)
	answer(w, r, http.StatusOK, view, err)
}

// copy answers POST /api/v1/activities/{activityId}/copy with 201 and the copy.
func (h *Handler) copy(w http.ResponseWriter, r *http.Request) {
	var in CopyInput
	if err := web.ReadJSON(w, r, &in); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	view, err := h.service.Copy(r.Context(), a, r.PathValue("activityId"), in)
	answer(w, r, http.StatusCreated, view, err)
}

// answer writes v with status, or the error when there is one.
func answer(w http.ResponseWriter, r *http.Request, status int, v any, err error) {
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, status, v)
}
