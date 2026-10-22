package people

import (
	"net/http"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// listTeams answers GET /api/teams: every team with its members and the plans using it, sorted by name.
func (h *Handler) listTeams(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	teams, err := h.service.ListTeams(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, TeamPage{Items: teams, Total: len(teams)})
}

// createTeam answers POST /api/teams with 201 and the new team.
func (h *Handler) createTeam(w http.ResponseWriter, r *http.Request) {
	var input TeamInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	team, err := h.service.CreateTeam(r.Context(), a, input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, team)
}

// getTeam answers GET /api/teams/{teamId}: the team with its members and the plans using it.
func (h *Handler) getTeam(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	team, err := h.service.GetTeam(r.Context(), a, r.PathValue("teamId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, team)
}

// updateTeam answers PATCH /api/teams/{teamId} with the team after the change.
func (h *Handler) updateTeam(w http.ResponseWriter, r *http.Request) {
	var update TeamUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	team, err := h.service.UpdateTeam(r.Context(), a, r.PathValue("teamId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, team)
}

// deleteTeam answers DELETE /api/teams/{teamId} with the recycle-bin entry of the team and its
// memberships, so the web app can offer Undo.
func (h *Handler) deleteTeam(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	entry, err := h.service.DeleteTeam(r.Context(), a, r.PathValue("teamId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// checkTeamImpact answers POST /api/teams/{teamId}/impact with the warnings to confirm before saving the
// body; it saves nothing.
func (h *Handler) checkTeamImpact(w http.ResponseWriter, r *http.Request) {
	var update TeamUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	impact, err := h.service.TeamImpact(r.Context(), a, r.PathValue("teamId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, impact)
}

// setTeamStatus answers POST /api/teams/{teamId}/status with the team in its new status.
func (h *Handler) setTeamStatus(w http.ResponseWriter, r *http.Request) {
	var update StatusUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	team, err := h.service.SetTeamStatus(r.Context(), a, r.PathValue("teamId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, team)
}
