package people

import (
	"net/http"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// listPeople answers GET /api/people: everyone, with their teams, sorted by name.
func (h *Handler) listPeople(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	everyone, err := h.service.ListPeople(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, PersonPage{Items: everyone, Total: len(everyone)})
}

// createPerson answers POST /api/people with 201 and the new person.
func (h *Handler) createPerson(w http.ResponseWriter, r *http.Request) {
	var input PersonInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	person, err := h.service.CreatePerson(r.Context(), a, input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, person)
}

// getPerson answers GET /api/people/{personId}: the person with their teams and what they own or approve.
func (h *Handler) getPerson(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	person, err := h.service.GetPerson(r.Context(), a, r.PathValue("personId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, person)
}

// updatePerson answers PATCH /api/people/{personId} with the person after the change.
func (h *Handler) updatePerson(w http.ResponseWriter, r *http.Request) {
	var update PersonUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	person, err := h.service.UpdatePerson(r.Context(), a, r.PathValue("personId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, person)
}

// checkPersonImpact answers POST /api/people/{personId}/impact with the warnings to confirm before saving
// the body; it saves nothing.
func (h *Handler) checkPersonImpact(w http.ResponseWriter, r *http.Request) {
	var update PersonUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	impact, err := h.service.PersonImpact(r.Context(), a, r.PathValue("personId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, impact)
}

// setPersonStatus answers POST /api/people/{personId}/status with the person in their new status.
func (h *Handler) setPersonStatus(w http.ResponseWriter, r *http.Request) {
	var update StatusUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	person, err := h.service.SetPersonStatus(r.Context(), a, r.PathValue("personId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, person)
}
