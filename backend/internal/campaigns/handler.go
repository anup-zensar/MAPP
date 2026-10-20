package campaigns

import (
	"net/http"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/web"
	"northstar/internal/rollup"
)

// Handler answers the HTTP requests about campaigns and initiatives.
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the campaign and initiative routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the campaign and initiative routes on r, every one wrapped in signedIn: they all need a
// signed-in person.
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/plans/{planId}/campaigns", signedIn(h.listPlanCampaigns))
	r.Handle("POST /api/plans/{planId}/campaigns", signedIn(h.createCampaign))
	r.Handle("GET /api/campaigns/{campaignId}", signedIn(h.getCampaign))
	r.Handle("PATCH /api/campaigns/{campaignId}", signedIn(h.updateCampaign))
	r.Handle("DELETE /api/campaigns/{campaignId}", signedIn(h.deleteCampaign))
	r.Handle("POST /api/campaigns/{campaignId}/initiatives", signedIn(h.createInitiative))
	r.Handle("GET /api/initiatives/{initiativeId}", signedIn(h.getInitiative))
	r.Handle("PATCH /api/initiatives/{initiativeId}", signedIn(h.updateInitiative))
	r.Handle("DELETE /api/initiatives/{initiativeId}", signedIn(h.deleteInitiative))
	r.Handle("GET /api/campaigns/{campaignId}/destinations", signedIn(h.campaignDestinations))
	r.Handle("POST /api/campaigns/{campaignId}/move", signedIn(h.moveCampaign))
	r.Handle("POST /api/campaigns/{campaignId}/copy", signedIn(h.copyCampaign))
	r.Handle("GET /api/initiatives/{initiativeId}/destinations", signedIn(h.initiativeDestinations))
	r.Handle("POST /api/initiatives/{initiativeId}/move", signedIn(h.moveInitiative))
	r.Handle("POST /api/initiatives/{initiativeId}/copy", signedIn(h.copyInitiative))
	r.Handle("PUT /api/campaigns/{campaignId}/budget-phasing", signedIn(h.setCampaignBudgetPhasing))       // 016 FR-020
	r.Handle("PUT /api/initiatives/{initiativeId}/budget-phasing", signedIn(h.setInitiativeBudgetPhasing)) // 016 FR-020
}

// listPlanCampaigns answers GET /api/plans/{planId}/campaigns: the plan's campaigns with their initiatives
// and the totals row.
func (h *Handler) listPlanCampaigns(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	list, err := h.service.ListPlanCampaigns(r.Context(), a, r.PathValue("planId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, list)
}

// createCampaign answers POST /api/plans/{planId}/campaigns with 201 and the new campaign.
func (h *Handler) createCampaign(w http.ResponseWriter, r *http.Request) {
	var input CampaignInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	campaign, err := h.service.CreateCampaign(r.Context(), a, r.PathValue("planId"), input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, campaign)
}

// getCampaign answers GET /api/campaigns/{campaignId}.
func (h *Handler) getCampaign(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	campaign, err := h.service.GetCampaign(r.Context(), a, r.PathValue("campaignId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, campaign)
}

// updateCampaign answers PATCH /api/campaigns/{campaignId} with the campaign after the save.
func (h *Handler) updateCampaign(w http.ResponseWriter, r *http.Request) {
	var update CampaignUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	campaign, err := h.service.UpdateCampaign(r.Context(), a, r.PathValue("campaignId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, campaign)
}

// deleteCampaign answers DELETE /api/campaigns/{campaignId} with the recycle-bin entry of the campaign and
// everything below it, so the web app can offer Undo.
func (h *Handler) deleteCampaign(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	entry, err := h.service.DeleteCampaign(r.Context(), a, r.PathValue("campaignId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// createInitiative answers POST /api/campaigns/{campaignId}/initiatives with 201 and the new initiative.
func (h *Handler) createInitiative(w http.ResponseWriter, r *http.Request) {
	var input InitiativeInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	initiative, err := h.service.CreateInitiative(r.Context(), a, r.PathValue("campaignId"), input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, initiative)
}

// getInitiative answers GET /api/initiatives/{initiativeId}.
func (h *Handler) getInitiative(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	initiative, err := h.service.GetInitiative(r.Context(), a, r.PathValue("initiativeId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, initiative)
}

// updateInitiative answers PATCH /api/initiatives/{initiativeId} with the initiative after the save.
func (h *Handler) updateInitiative(w http.ResponseWriter, r *http.Request) {
	var update InitiativeUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	initiative, err := h.service.UpdateInitiative(r.Context(), a, r.PathValue("initiativeId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, initiative)
}

// deleteInitiative answers DELETE /api/initiatives/{initiativeId} with the recycle-bin entry of the
// initiative and everything below it, so the web app can offer Undo.
func (h *Handler) deleteInitiative(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	entry, err := h.service.DeleteInitiative(r.Context(), a, r.PathValue("initiativeId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// campaignDestinations answers GET /api/campaigns/{campaignId}/destinations?purpose=move|copy: the plans
// the campaign can go to.
func (h *Handler) campaignDestinations(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	list, err := h.service.CampaignDestinations(r.Context(), a, r.PathValue("campaignId"), r.URL.Query().Get("purpose"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, list)
}

// moveCampaign answers POST /api/campaigns/{campaignId}/move with the campaign in its new plan.
func (h *Handler) moveCampaign(w http.ResponseWriter, r *http.Request) {
	var move CampaignMove
	if err := web.ReadJSON(w, r, &move); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	campaign, err := h.service.MoveCampaign(r.Context(), a, r.PathValue("campaignId"), move)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, campaign)
}

// copyCampaign answers POST /api/campaigns/{campaignId}/copy with 201 and the new campaign.
func (h *Handler) copyCampaign(w http.ResponseWriter, r *http.Request) {
	var request CampaignCopy
	if err := web.ReadJSON(w, r, &request); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	campaign, err := h.service.CopyCampaign(r.Context(), a, r.PathValue("campaignId"), request)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, campaign)
}

// initiativeDestinations answers GET /api/initiatives/{initiativeId}/destinations?purpose=move|copy: the
// campaigns of the plan the initiative can go to.
func (h *Handler) initiativeDestinations(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	list, err := h.service.InitiativeDestinations(r.Context(), a, r.PathValue("initiativeId"), r.URL.Query().Get("purpose"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, list)
}

// moveInitiative answers POST /api/initiatives/{initiativeId}/move with the initiative in its new campaign.
func (h *Handler) moveInitiative(w http.ResponseWriter, r *http.Request) {
	var move InitiativeMove
	if err := web.ReadJSON(w, r, &move); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	initiative, err := h.service.MoveInitiative(r.Context(), a, r.PathValue("initiativeId"), move)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, initiative)
}

// copyInitiative answers POST /api/initiatives/{initiativeId}/copy with 201 and the new initiative.
func (h *Handler) copyInitiative(w http.ResponseWriter, r *http.Request) {
	var request InitiativeCopy
	if err := web.ReadJSON(w, r, &request); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	initiative, err := h.service.CopyInitiative(r.Context(), a, r.PathValue("initiativeId"), request)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, initiative)
}

// setCampaignBudgetPhasing answers PUT /api/campaigns/{campaignId}/budget-phasing with the campaign after the
// save (016 FR-020).
func (h *Handler) setCampaignBudgetPhasing(w http.ResponseWriter, r *http.Request) {
	var input rollup.PhasingInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	campaign, err := h.service.SetCampaignBudgetPhasing(r.Context(), a, r.PathValue("campaignId"), input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, campaign)
}

// setInitiativeBudgetPhasing answers PUT /api/initiatives/{initiativeId}/budget-phasing with the initiative
// after the save (016 FR-020).
func (h *Handler) setInitiativeBudgetPhasing(w http.ResponseWriter, r *http.Request) {
	var input rollup.PhasingInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	initiative, err := h.service.SetInitiativeBudgetPhasing(r.Context(), a, r.PathValue("initiativeId"), input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, initiative)
}
