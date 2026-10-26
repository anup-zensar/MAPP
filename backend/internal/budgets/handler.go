package budgets

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

const yearField = "year"

// Handler answers the HTTP requests of global budgets and allocations (014 plan "API operations").
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the global budget routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// RegisterRoutes registers the nine global budget routes on r.
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/global-budgets", h.list)
	r.POST("/global-budgets", h.create)
	r.GET("/global-budgets/:globalBudgetId", h.get)
	r.PATCH("/global-budgets/:globalBudgetId", h.update)
	r.DELETE("/global-budgets/:globalBudgetId", h.delete)
	r.GET("/global-budgets/:globalBudgetId/allocation-options", h.allocationOptions)
	r.POST("/global-budgets/:globalBudgetId/allocations", h.allocate)
	r.PATCH("/budget-allocations/:allocationId", h.updateAllocation)
	r.DELETE("/budget-allocations/:allocationId", h.removeAllocation)
}

// list answers GET /api/v1/global-budgets: the global budgets of the query's year, or of every year (FR-036).
func (h *Handler) list(c *gin.Context) {
	var year *int
	if text := c.Request.URL.Query().Get(yearField); text != "" {
		parsed, err := strconv.Atoi(text)
		if err != nil {
			web.RespondError(c, errs.Validation(errs.FieldError{Field: yearField, Message: msgChooseListYear}))
			return
		}
		year = &parsed
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	page, err := h.service.List(c.Request.Context(), a, year)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, page)
}

// create answers POST /api/v1/global-budgets with 201 and the new global budget.
func (h *Handler) create(c *gin.Context) {
	var input GlobalBudgetInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Create(c.Request.Context(), a, input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, view)
}

// get answers GET /api/v1/global-budgets/{globalBudgetId} with the global budget and its figures.
func (h *Handler) get(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Get(c.Request.Context(), a, c.Param("globalBudgetId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}

// update answers PATCH /api/v1/global-budgets/{globalBudgetId} with the global budget after the save.
func (h *Handler) update(c *gin.Context) {
	var update GlobalBudgetUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Update(c.Request.Context(), a, c.Param("globalBudgetId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}

// delete answers DELETE /api/v1/global-budgets/{globalBudgetId} with the recycle-bin entry, so the web app can
// offer Undo.
func (h *Handler) delete(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	entry, err := h.service.Delete(c.Request.Context(), a, c.Param("globalBudgetId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// allocationOptions answers GET /api/v1/global-budgets/{globalBudgetId}/allocation-options: the plans the
// Allocate dialog offers.
func (h *Handler) allocationOptions(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	page, err := h.service.AllocationOptions(c.Request.Context(), a, c.Param("globalBudgetId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, page)
}

// allocate answers POST /api/v1/global-budgets/{globalBudgetId}/allocations with 201 and the global budget
// after the allocation.
func (h *Handler) allocate(c *gin.Context) {
	var input AllocationInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Allocate(c.Request.Context(), a, c.Param("globalBudgetId"), input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, view)
}

// updateAllocation answers PATCH /api/v1/budget-allocations/{allocationId} with the global budget after the
// change.
func (h *Handler) updateAllocation(c *gin.Context) {
	var update AllocationUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.UpdateAllocation(c.Request.Context(), a, c.Param("allocationId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}

// removeAllocation answers DELETE /api/v1/budget-allocations/{allocationId} with the global budget after the
// removal.
func (h *Handler) removeAllocation(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.RemoveAllocation(c.Request.Context(), a, c.Param("allocationId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}
