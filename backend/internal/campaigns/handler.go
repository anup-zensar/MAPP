package campaigns

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
	"mapp/internal/rollup"
)

// Handler answers the HTTP requests about campaigns and initiatives.
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the campaign and initiative routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// RegisterRoutes registers the campaign and initiative routes on r.
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/plans/:planId/campaigns", h.listPlanCampaigns)
	r.POST("/plans/:planId/campaigns", h.createCampaign)
	r.GET("/campaigns/:campaignId", h.getCampaign)
	r.PATCH("/campaigns/:campaignId", h.updateCampaign)
	r.DELETE("/campaigns/:campaignId", h.deleteCampaign)
	r.POST("/campaigns/:campaignId/initiatives", h.createInitiative)
	r.GET("/initiatives/:initiativeId", h.getInitiative)
	r.PATCH("/initiatives/:initiativeId", h.updateInitiative)
	r.DELETE("/initiatives/:initiativeId", h.deleteInitiative)
	r.GET("/campaigns/:campaignId/destinations", h.campaignDestinations)
	r.POST("/campaigns/:campaignId/move", h.moveCampaign)
	r.POST("/campaigns/:campaignId/copy", h.copyCampaign)
	r.GET("/initiatives/:initiativeId/destinations", h.initiativeDestinations)
	r.POST("/initiatives/:initiativeId/move", h.moveInitiative)
	r.POST("/initiatives/:initiativeId/copy", h.copyInitiative)
	r.PUT("/campaigns/:campaignId/budget-phasing", h.setCampaignBudgetPhasing)       // 016 FR-020
	r.PUT("/initiatives/:initiativeId/budget-phasing", h.setInitiativeBudgetPhasing) // 016 FR-020
}

// listPlanCampaigns answers GET /api/v1/plans/{planId}/campaigns: the plan's campaigns with their initiatives
// and the totals row.
func (h *Handler) listPlanCampaigns(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	list, err := h.service.ListPlanCampaigns(c.Request.Context(), a, c.Param("planId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, list)
}

// createCampaign answers POST /api/v1/plans/{planId}/campaigns with 201 and the new campaign.
func (h *Handler) createCampaign(c *gin.Context) {
	var input CampaignInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	campaign, err := h.service.CreateCampaign(c.Request.Context(), a, c.Param("planId"), input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, campaign)
}

// getCampaign answers GET /api/v1/campaigns/{campaignId}.
func (h *Handler) getCampaign(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	campaign, err := h.service.GetCampaign(c.Request.Context(), a, c.Param("campaignId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, campaign)
}

// updateCampaign answers PATCH /api/v1/campaigns/{campaignId} with the campaign after the save.
func (h *Handler) updateCampaign(c *gin.Context) {
	var update CampaignUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	campaign, err := h.service.UpdateCampaign(c.Request.Context(), a, c.Param("campaignId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, campaign)
}

// deleteCampaign answers DELETE /api/v1/campaigns/{campaignId} with the recycle-bin entry of the campaign and
// everything below it, so the web app can offer Undo.
func (h *Handler) deleteCampaign(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	entry, err := h.service.DeleteCampaign(c.Request.Context(), a, c.Param("campaignId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// createInitiative answers POST /api/v1/campaigns/{campaignId}/initiatives with 201 and the new initiative.
func (h *Handler) createInitiative(c *gin.Context) {
	var input InitiativeInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	initiative, err := h.service.CreateInitiative(c.Request.Context(), a, c.Param("campaignId"), input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, initiative)
}

// getInitiative answers GET /api/v1/initiatives/{initiativeId}.
func (h *Handler) getInitiative(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	initiative, err := h.service.GetInitiative(c.Request.Context(), a, c.Param("initiativeId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, initiative)
}

// updateInitiative answers PATCH /api/v1/initiatives/{initiativeId} with the initiative after the save.
func (h *Handler) updateInitiative(c *gin.Context) {
	var update InitiativeUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	initiative, err := h.service.UpdateInitiative(c.Request.Context(), a, c.Param("initiativeId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, initiative)
}

// deleteInitiative answers DELETE /api/v1/initiatives/{initiativeId} with the recycle-bin entry of the
// initiative and everything below it, so the web app can offer Undo.
func (h *Handler) deleteInitiative(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	entry, err := h.service.DeleteInitiative(c.Request.Context(), a, c.Param("initiativeId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// campaignDestinations answers GET /api/v1/campaigns/{campaignId}/destinations?purpose=move|copy: the plans
// the campaign can go to.
func (h *Handler) campaignDestinations(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	list, err := h.service.CampaignDestinations(c.Request.Context(), a, c.Param("campaignId"), c.Request.URL.Query().Get("purpose"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, list)
}

// moveCampaign answers POST /api/v1/campaigns/{campaignId}/move with the campaign in its new plan.
func (h *Handler) moveCampaign(c *gin.Context) {
	var move CampaignMove
	if err := web.BindJSON(c, &move); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	campaign, err := h.service.MoveCampaign(c.Request.Context(), a, c.Param("campaignId"), move)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, campaign)
}

// copyCampaign answers POST /api/v1/campaigns/{campaignId}/copy with 201 and the new campaign.
func (h *Handler) copyCampaign(c *gin.Context) {
	var request CampaignCopy
	if err := web.BindJSON(c, &request); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	campaign, err := h.service.CopyCampaign(c.Request.Context(), a, c.Param("campaignId"), request)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, campaign)
}

// initiativeDestinations answers GET /api/v1/initiatives/{initiativeId}/destinations?purpose=move|copy: the
// campaigns of the plan the initiative can go to.
func (h *Handler) initiativeDestinations(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	list, err := h.service.InitiativeDestinations(c.Request.Context(), a, c.Param("initiativeId"), c.Request.URL.Query().Get("purpose"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, list)
}

// moveInitiative answers POST /api/v1/initiatives/{initiativeId}/move with the initiative in its new campaign.
func (h *Handler) moveInitiative(c *gin.Context) {
	var move InitiativeMove
	if err := web.BindJSON(c, &move); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	initiative, err := h.service.MoveInitiative(c.Request.Context(), a, c.Param("initiativeId"), move)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, initiative)
}

// copyInitiative answers POST /api/v1/initiatives/{initiativeId}/copy with 201 and the new initiative.
func (h *Handler) copyInitiative(c *gin.Context) {
	var request InitiativeCopy
	if err := web.BindJSON(c, &request); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	initiative, err := h.service.CopyInitiative(c.Request.Context(), a, c.Param("initiativeId"), request)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, initiative)
}

// setCampaignBudgetPhasing answers PUT /api/v1/campaigns/{campaignId}/budget-phasing with the campaign after the
// save (016 FR-020).
func (h *Handler) setCampaignBudgetPhasing(c *gin.Context) {
	var input rollup.PhasingInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	campaign, err := h.service.SetCampaignBudgetPhasing(c.Request.Context(), a, c.Param("campaignId"), input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, campaign)
}

// setInitiativeBudgetPhasing answers PUT /api/v1/initiatives/{initiativeId}/budget-phasing with the initiative
// after the save (016 FR-020).
func (h *Handler) setInitiativeBudgetPhasing(c *gin.Context) {
	var input rollup.PhasingInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	initiative, err := h.service.SetInitiativeBudgetPhasing(c.Request.Context(), a, c.Param("initiativeId"), input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, initiative)
}
