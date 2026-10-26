package people

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// listPeople answers GET /api/v1/people: everyone, with their teams, sorted by name.
func (h *Handler) listPeople(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	everyone, err := h.service.ListPeople(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, PersonPage{Items: everyone, Total: len(everyone)})
}

// createPerson answers POST /api/v1/people with 201 and the new person.
func (h *Handler) createPerson(c *gin.Context) {
	var input PersonInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	person, err := h.service.CreatePerson(c.Request.Context(), a, input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, person)
}

// getPerson answers GET /api/v1/people/{personId}: the person with their teams and what they own or approve.
func (h *Handler) getPerson(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	person, err := h.service.GetPerson(c.Request.Context(), a, c.Param("personId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, person)
}

// updatePerson answers PATCH /api/v1/people/{personId} with the person after the change.
func (h *Handler) updatePerson(c *gin.Context) {
	var update PersonUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	person, err := h.service.UpdatePerson(c.Request.Context(), a, c.Param("personId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, person)
}

// checkPersonImpact answers POST /api/v1/people/{personId}/impact with the warnings to confirm before saving
// the body; it saves nothing.
func (h *Handler) checkPersonImpact(c *gin.Context) {
	var update PersonUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	impact, err := h.service.PersonImpact(c.Request.Context(), a, c.Param("personId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, impact)
}

// setPersonStatus answers POST /api/v1/people/{personId}/status with the person in their new status.
func (h *Handler) setPersonStatus(c *gin.Context) {
	var update StatusUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	person, err := h.service.SetPersonStatus(c.Request.Context(), a, c.Param("personId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, person)
}
