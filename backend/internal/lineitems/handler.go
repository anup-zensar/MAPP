package lineitems

import (
	"net/http"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

// Handler answers the HTTP requests about line items (contract tag "Line items").
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the line-item routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the line-item routes on r. All of them need a signed-in person, so all are wrapped in
// signedIn.
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/activities/{activityId}/line-items", signedIn(h.section))
	r.Handle("GET /api/activities/{activityId}/line-items/form-options", signedIn(h.formOptions))
	r.Handle("POST /api/activities/{activityId}/line-items", signedIn(h.create))
	r.Handle("GET /api/line-items/{lineItemId}", signedIn(h.get))
	r.Handle("PATCH /api/line-items/{lineItemId}", signedIn(h.update))
	r.Handle("DELETE /api/line-items/{lineItemId}", signedIn(h.delete))
	r.Handle("POST /api/line-items/{lineItemId}/duplicate", signedIn(h.duplicate))
}

// get answers GET /api/line-items/{lineItemId} with the line item, its activity and the balance.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	details, err := h.service.Get(r.Context(), a, r.PathValue("lineItemId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, details)
}

// update answers PATCH /api/line-items/{lineItemId} with the line item after the save.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var update Update
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	view, err := h.service.Update(r.Context(), a, r.PathValue("lineItemId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, view)
}

// delete answers DELETE /api/line-items/{lineItemId}?restartApproval=true with the recycle-bin entry, so the
// web app can offer Undo. The query value is true or false (missing: false); anything else is refused.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	var restart bool
	switch r.URL.Query().Get("restartApproval") {
	case "", "false":
	case "true":
		restart = true
	default:
		web.WriteError(w, r, errs.BadRequest(unreadableMessage))
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	entry, err := h.service.Delete(r.Context(), a, r.PathValue("lineItemId"), restart)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// unreadableMessage is the answer to a query value the API cannot read (001's bad_request text).
const unreadableMessage = "MAPP couldn't read this request. Reload the page and try again."

// duplicate answers POST /api/line-items/{lineItemId}/duplicate with 201 and the copy.
func (h *Handler) duplicate(w http.ResponseWriter, r *http.Request) {
	var input DuplicateInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	view, err := h.service.Duplicate(r.Context(), a, r.PathValue("lineItemId"), input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, view)
}

// section answers GET /api/activities/{activityId}/line-items with the activity's Line items section.
func (h *Handler) section(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	section, err := h.service.Section(r.Context(), a, r.PathValue("activityId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, section)
}

// formOptions answers GET /api/activities/{activityId}/line-items/form-options with the defaults and choices
// of a new line item.
func (h *Handler) formOptions(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	options, err := h.service.FormOptions(r.Context(), a, r.PathValue("activityId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, options)
}

// create answers POST /api/activities/{activityId}/line-items with 201 and the new line item.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input Input
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	view, err := h.service.Create(r.Context(), a, r.PathValue("activityId"), input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, view)
}
