package people

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// listTeams answers GET /api/v1/teams: every team with its members and the plans using it, sorted by name.
func (h *Handler) listTeams(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	teams, err := h.service.ListTeams(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, TeamPage{Items: teams, Total: len(teams)})
}

// createTeam answers POST /api/v1/teams with 201 and the new team.
func (h *Handler) createTeam(c *gin.Context) {
	var input TeamInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	team, err := h.service.CreateTeam(c.Request.Context(), a, input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, team)
}

// getTeam answers GET /api/v1/teams/{teamId}: the team with its members and the plans using it.
func (h *Handler) getTeam(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	team, err := h.service.GetTeam(c.Request.Context(), a, c.Param("teamId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, team)
}

// updateTeam answers PATCH /api/v1/teams/{teamId} with the team after the change.
func (h *Handler) updateTeam(c *gin.Context) {
	var update TeamUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	team, err := h.service.UpdateTeam(c.Request.Context(), a, c.Param("teamId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, team)
}

// deleteTeam answers DELETE /api/v1/teams/{teamId} with the recycle-bin entry of the team and its
// memberships, so the web app can offer Undo.
func (h *Handler) deleteTeam(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	entry, err := h.service.DeleteTeam(c.Request.Context(), a, c.Param("teamId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// checkTeamImpact answers POST /api/v1/teams/{teamId}/impact with the warnings to confirm before saving the
// body; it saves nothing.
func (h *Handler) checkTeamImpact(c *gin.Context) {
	var update TeamUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	impact, err := h.service.TeamImpact(c.Request.Context(), a, c.Param("teamId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, impact)
}

// setTeamStatus answers POST /api/v1/teams/{teamId}/status with the team in its new status.
func (h *Handler) setTeamStatus(c *gin.Context) {
	var update StatusUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	team, err := h.service.SetTeamStatus(c.Request.Context(), a, c.Param("teamId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, team)
}
