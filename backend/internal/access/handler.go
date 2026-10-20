package access

import (
	"net/http"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/web"
)

// Handler answers the HTTP requests of Settings › Roles and access (plan "API operations"). Every route needs
// "Give roles and change roles and access rules" (routes.go), which the access middleware checks.
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the access routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the access routes on r, each wrapped in signedIn (sign-in, then the access middleware).
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/access/permissions", signedIn(h.permissions))
	r.Handle("GET /api/access/people", signedIn(h.people))
	r.Handle("GET /api/access/changes", signedIn(h.changes))
	r.Handle("GET /api/roles", signedIn(h.listRoles))
	r.Handle("POST /api/roles", signedIn(h.createRole))
	r.Handle("GET /api/roles/{roleId}", signedIn(h.getRole))
	r.Handle("PATCH /api/roles/{roleId}", signedIn(h.updateRole))
	r.Handle("DELETE /api/roles/{roleId}", signedIn(h.deleteRole))
	r.Handle("GET /api/people/{personId}/roles", signedIn(h.personRoles))
	r.Handle("PUT /api/people/{personId}/roles", signedIn(h.setPersonRoles))
	r.Handle("POST /api/people/{personId}/roles/impact", signedIn(h.rolesImpact))
	r.Handle("GET /api/access-rules", signedIn(h.rules))
	r.Handle("PATCH /api/access-rules", signedIn(h.updateRules))
}

// permissions answers GET /api/access/permissions: Table A, grouped.
func (h *Handler) permissions(w http.ResponseWriter, r *http.Request) {
	web.WriteJSON(w, http.StatusOK, h.service.Permissions())
}

// people answers GET /api/access/people: everyone with their roles (People tab).
func (h *Handler) people(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	list, err := h.service.ListPeopleAccess(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, PeopleAccessPage{Items: list, Total: len(list)})
}

// changes answers GET /api/access/changes: the latest 20 access changes, newest first.
func (h *Handler) changes(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	entries, total, err := h.service.RecentChanges(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, history.LogPage{Items: entries, Total: total})
}

// listRoles answers GET /api/roles: every role, Administrator first.
func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	roles, err := h.service.ListRoles(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, RolePage{Items: roles, Total: len(roles)})
}

// createRole answers POST /api/roles with 201 and the new role.
func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	var input RoleInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	role, err := h.service.CreateRole(r.Context(), a, input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, role)
}

// getRole answers GET /api/roles/{roleId}.
func (h *Handler) getRole(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	role, err := h.service.GetRole(r.Context(), a, r.PathValue("roleId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, role)
}

// updateRole answers PATCH /api/roles/{roleId} with the role after the change.
func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	var update RoleUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	role, err := h.service.UpdateRole(r.Context(), a, r.PathValue("roleId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, role)
}

// deleteRole answers DELETE /api/roles/{roleId} with the recycle-bin entry, so the web app can offer Undo.
func (h *Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	entry, err := h.service.DeleteRole(r.Context(), a, r.PathValue("roleId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// personRoles answers GET /api/people/{personId}/roles.
func (h *Handler) personRoles(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	roles, err := h.service.PersonRoles(r.Context(), a, r.PathValue("personId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, roles)
}

// setPersonRoles answers PUT /api/people/{personId}/roles with the person's roles after the change.
func (h *Handler) setPersonRoles(w http.ResponseWriter, r *http.Request) {
	var update PersonRolesUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	roles, err := h.service.SetPersonRoles(r.Context(), a, r.PathValue("personId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, roles)
}

// rolesImpact answers POST /api/people/{personId}/roles/impact with the warnings before saving.
func (h *Handler) rolesImpact(w http.ResponseWriter, r *http.Request) {
	var update PersonRolesUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	impact, err := h.service.CheckRolesImpact(r.Context(), a, r.PathValue("personId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, impact)
}

// rules answers GET /api/access-rules.
func (h *Handler) rules(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	rules, err := h.service.GetRules(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, rules)
}

// updateRules answers PATCH /api/access-rules with the rules after the change.
func (h *Handler) updateRules(w http.ResponseWriter, r *http.Request) {
	var update RulesUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	rules, err := h.service.UpdateRules(r.Context(), a, update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, rules)
}
