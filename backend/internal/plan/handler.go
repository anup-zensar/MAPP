package plan

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
	"mapp/internal/rollup"
)

// Handler answers the HTTP requests of plans (plan "API operations").
type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/plan-defaults", h.defaults)
	r.GET("/plans", h.list)
	r.POST("/plans", h.create)
	r.GET("/plans/:planId", h.get)
	r.PATCH("/plans/:planId", h.update)
	r.DELETE("/plans/:planId", h.delete)
	r.POST("/plans/:planId/change-effects", h.changeEffects)
	r.POST("/plans/:planId/publish", h.publish)
	r.POST("/plans/:planId/unpublish", h.unpublish)
	r.GET("/plans/:planId/copy-options", h.copyOptions)
	r.GET("/plans/:planId/search", h.search)
	r.POST("/plans/:planId/copy", h.copy)
	r.PUT("/plans/:planId/budget-phasing", h.setBudgetPhasing)
}

func (h *Handler) setBudgetPhasing(c *gin.Context) {
	var input rollup.PhasingInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	plan, err := h.service.SetBudgetPhasing(c.Request.Context(), a, c.Param("planId"), input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, plan)
}

func (h *Handler) defaults(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	defaults, err := h.service.Defaults(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, defaults)
}

func (h *Handler) create(c *gin.Context) {
	var input PlanInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	plan, err := h.service.Create(c.Request.Context(), a, input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, plan)
}

func (h *Handler) get(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	plan, err := h.service.Get(c.Request.Context(), a, c.Param("planId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, plan)
}

func (h *Handler) search(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	found, err := h.service.Search(c.Request.Context(), a, c.Param("planId"), c.Query("q"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, found)
}

// list answers GET /api/v1/plans: the plans matching the query's scope, status, years, plan types and search,
// with the summary cards; every wrong parameter is reported at once.
func (h *Handler) list(c *gin.Context) {
	filter, err := parseListFilter(c.Request.URL.Query())
	if err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	list, err := h.service.List(c.Request.Context(), a, filter)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, list)
}

func (h *Handler) update(c *gin.Context) {
	var update PlanUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	plan, err := h.service.Update(c.Request.Context(), a, c.Param("planId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, plan)
}

// delete answers DELETE /api/v1/plans/{planId} with the recycle-bin entry of the plan, so the web app can
// offer Undo.
func (h *Handler) delete(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	entry, err := h.service.Delete(c.Request.Context(), a, c.Param("planId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// changeEffects answers POST /api/v1/plans/{planId}/change-effects with what a new model and business unit
// would change; nothing is saved.
func (h *Handler) changeEffects(c *gin.Context) {
	var request ChangeEffectsRequest
	if err := web.BindJSON(c, &request); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	effects, err := h.service.ChangeEffects(c.Request.Context(), a, c.Param("planId"), request)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, effects)
}

// publish answers POST /api/v1/plans/{planId}/publish with the Published plan, or 409 not_ready listing
// every missing field.
func (h *Handler) publish(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	plan, err := h.service.Publish(c.Request.Context(), a, c.Param("planId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, plan)
}

func (h *Handler) unpublish(c *gin.Context) {
	var request UnpublishRequest
	if err := web.BindJSON(c, &request); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	plan, err := h.service.Unpublish(c.Request.Context(), a, c.Param("planId"), request)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, plan)
}

func (h *Handler) copyOptions(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	options, err := h.service.CopyOptions(c.Request.Context(), a, c.Param("planId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, options)
}

// copy answers POST /api/v1/plans/{planId}/copy with 201, the new plan and what the copy changed on purpose.
func (h *Handler) copy(c *gin.Context) {
	var request CopyRequest
	if err := web.BindJSON(c, &request); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	result, err := h.service.Copy(c.Request.Context(), a, c.Param("planId"), request)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, result)
}
