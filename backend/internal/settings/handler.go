package settings

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// unreadableMessage answers a path or query value the API cannot read, such as an unknown level.
const unreadableMessage = "MAPP couldn't read this request. Reload the page and try again."

// Handler answers the HTTP requests of Settings (feature 002).
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the Settings routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// RegisterRoutes registers the Settings routes on r.
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/settings/overview", h.overview)

	r.GET("/models", h.listModels)
	r.POST("/models", h.createModel)
	r.GET("/models/:modelId", h.getModel)
	r.PATCH("/models/:modelId", h.updateModel)
	r.DELETE("/models/:modelId", h.deleteModel)
	r.POST("/models/:modelId/impact", h.modelImpact)
	r.POST("/models/:modelId/status", h.setModelStatus)

	r.GET("/activity-types", h.listActivityTypes)
	r.POST("/activity-types", h.createActivityType)
	r.GET("/activity-types/:activityTypeId", h.getActivityType)
	r.PATCH("/activity-types/:activityTypeId", h.updateActivityType)
	r.DELETE("/activity-types/:activityTypeId", h.deleteActivityType)
	r.POST("/activity-types/:activityTypeId/impact", h.activityTypeImpact)
	r.POST("/activity-types/:activityTypeId/status", h.setActivityTypeStatus)

	h.attributeRoutes(r)

	r.GET("/currencies", h.listCurrencies)
	r.POST("/currencies", h.createCurrency)
	r.GET("/currencies/:code", h.getCurrency)
	r.PATCH("/currencies/:code", h.updateCurrency)
	r.DELETE("/currencies/:code", h.deleteCurrency)
	r.POST("/currencies/:code/impact", h.currencyImpact)
	r.POST("/currencies/:code/status", h.setCurrencyStatus)
	h.rateRoutes(r)      // 016: rate history, current rates, GSAP rate import
	h.fieldHelpRoutes(r) // 016: Field help

	r.GET("/finance-budgets", h.listFinanceBudgets)
	r.PATCH("/finance-budgets/:budgetId", h.renameFinanceBudget)
	r.POST("/finance-budget-items", h.createFinanceBudgetItem)
	r.GET("/finance-budget-items/:itemId", h.getFinanceBudgetItem)
	r.PATCH("/finance-budget-items/:itemId", h.updateFinanceBudgetItem)
	r.DELETE("/finance-budget-items/:itemId", h.deleteFinanceBudgetItem)
	r.POST("/finance-budget-items/:itemId/status", h.setFinanceBudgetItemStatus)
}

// attributeRoutes registers the Shell attribute, option and form routes.
func (h *Handler) attributeRoutes(r gin.IRoutes) {
	r.GET("/shell-attributes", h.listAttributes)
	r.POST("/shell-attributes", h.createAttribute)
	r.GET("/shell-attributes/:attributeId", h.getAttribute)
	r.PATCH("/shell-attributes/:attributeId", h.updateAttribute)
	r.DELETE("/shell-attributes/:attributeId", h.deleteAttribute)
	r.POST("/shell-attributes/:attributeId/impact", h.attributeImpact)
	r.POST("/shell-attributes/:attributeId/status", h.setAttributeStatus)
	r.POST("/shell-attributes/:attributeId/move", h.moveAttribute)
	r.GET("/shell-attributes/:attributeId/options", h.listOptions)
	r.POST("/shell-attributes/:attributeId/options", h.createOption)
	r.POST("/shell-attributes/:attributeId/options/import-check", h.checkImport)
	r.POST("/shell-attributes/:attributeId/options/import", h.importOptions)
	r.GET("/shell-attribute-options/:optionId", h.getOption)
	r.PATCH("/shell-attribute-options/:optionId", h.updateOption)
	r.DELETE("/shell-attribute-options/:optionId", h.deleteOption)
	r.POST("/shell-attribute-options/:optionId/status", h.setOptionStatus)
	r.POST("/shell-attribute-options/:optionId/move", h.moveOption)
	r.GET("/shell-attribute-forms/:level", h.getForm)
	r.POST("/shell-attribute-forms/:level/check", h.checkValues)
}

// overview answers GET /api/v1/settings/overview: the active records of each section (FR-001).
func (h *Handler) overview(c *gin.Context) {
	overview, err := h.service.Overview(c.Request.Context(), actorOf(c))
	respond(c, http.StatusOK, overview, err)
}

// actorOf returns the signed-in person, put in the request's context by the sign-in middleware.
func actorOf(c *gin.Context) actor.Actor {
	a, _ := actor.From(c.Request.Context())
	return a
}

// readBody decodes the request's JSON body into dst; when it cannot, it answers the error and returns
// false.
func readBody(c *gin.Context, dst any) bool {
	if err := web.BindJSON(c, dst); err != nil {
		web.RespondError(c, err)
		return false
	}
	return true
}

// respond answers body with status, or err when the service failed.
func respond(c *gin.Context, status int, body any, err error) {
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, status, body)
}

// respondDeleted answers a delete with the recycle-bin entry, so the web app can offer Undo.
func respondDeleted(c *gin.Context, entry history.BinEntry, err error) {
	respond(c, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()}, err)
}

// parseLevel reads a level from a path or query value.
func parseLevel(text string) (Level, bool) {
	for _, l := range Levels {
		if string(l) == text {
			return l, true
		}
	}
	return "", false
}

// wholeNumber reads a whole number from a query value; anything else is -1, which every range refuses.
func wholeNumber(text string) int {
	n, err := strconv.Atoi(text)
	if err != nil {
		return -1
	}
	return n
}
