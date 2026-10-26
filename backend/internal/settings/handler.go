package settings

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
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

// listModels answers GET /api/v1/models.
func (h *Handler) listModels(c *gin.Context) {
	models, err := h.service.ListModels(c.Request.Context(), actorOf(c))
	respond(c, http.StatusOK, ModelPage{Items: models, Total: len(models)}, err)
}

// createModel answers POST /api/v1/models with 201 and the new model.
func (h *Handler) createModel(c *gin.Context) {
	var input ModelInput
	if !readBody(c, &input) {
		return
	}
	model, err := h.service.CreateModel(c.Request.Context(), actorOf(c), input)
	respond(c, http.StatusCreated, model, err)
}

// getModel answers GET /api/v1/models/{modelId}.
func (h *Handler) getModel(c *gin.Context) {
	model, err := h.service.GetModel(c.Request.Context(), actorOf(c), c.Param("modelId"))
	respond(c, http.StatusOK, model, err)
}

// updateModel answers PATCH /api/v1/models/{modelId} with the model after the change.
func (h *Handler) updateModel(c *gin.Context) {
	var update ModelUpdate
	if !readBody(c, &update) {
		return
	}
	model, err := h.service.UpdateModel(c.Request.Context(), actorOf(c), c.Param("modelId"), update)
	respond(c, http.StatusOK, model, err)
}

// deleteModel answers DELETE /api/v1/models/{modelId} with the recycle-bin entry, so the web app can offer
// Undo.
func (h *Handler) deleteModel(c *gin.Context) {
	entry, err := h.service.DeleteModel(c.Request.Context(), actorOf(c), c.Param("modelId"))
	respondDeleted(c, entry, err)
}

// modelImpact answers POST /api/v1/models/{modelId}/impact with the warnings to confirm before saving.
func (h *Handler) modelImpact(c *gin.Context) {
	var update ModelUpdate
	if !readBody(c, &update) {
		return
	}
	impact, err := h.service.ModelImpact(c.Request.Context(), actorOf(c), c.Param("modelId"), update)
	respond(c, http.StatusOK, impact, err)
}

// setModelStatus answers POST /api/v1/models/{modelId}/status with the model in its new status.
func (h *Handler) setModelStatus(c *gin.Context) {
	var update StatusUpdate
	if !readBody(c, &update) {
		return
	}
	model, err := h.service.SetModelActive(c.Request.Context(), actorOf(c), c.Param("modelId"), update)
	respond(c, http.StatusOK, model, err)
}

// listActivityTypes answers GET /api/v1/activity-types.
func (h *Handler) listActivityTypes(c *gin.Context) {
	types, err := h.service.ListActivityTypes(c.Request.Context(), actorOf(c))
	respond(c, http.StatusOK, ActivityTypePage{Items: types, Total: len(types)}, err)
}

// getActivityType answers GET /api/v1/activity-types/{activityTypeId}.
func (h *Handler) getActivityType(c *gin.Context) {
	activityType, err := h.service.GetActivityType(c.Request.Context(), actorOf(c), c.Param("activityTypeId"))
	respond(c, http.StatusOK, activityType, err)
}

// createActivityType answers POST /api/v1/activity-types with 201 and the new activity type.
func (h *Handler) createActivityType(c *gin.Context) {
	var input ActivityTypeInput
	if !readBody(c, &input) {
		return
	}
	activityType, err := h.service.CreateActivityType(c.Request.Context(), actorOf(c), input)
	respond(c, http.StatusCreated, activityType, err)
}

// updateActivityType answers PATCH /api/v1/activity-types/{activityTypeId} with the activity type after the
// change.
func (h *Handler) updateActivityType(c *gin.Context) {
	var update ActivityTypeUpdate
	if !readBody(c, &update) {
		return
	}
	activityType, err := h.service.UpdateActivityType(c.Request.Context(), actorOf(c), c.Param("activityTypeId"), update)
	respond(c, http.StatusOK, activityType, err)
}

// deleteActivityType answers DELETE /api/v1/activity-types/{activityTypeId} with the recycle-bin entry.
func (h *Handler) deleteActivityType(c *gin.Context) {
	entry, err := h.service.DeleteActivityType(c.Request.Context(), actorOf(c), c.Param("activityTypeId"))
	respondDeleted(c, entry, err)
}

// activityTypeImpact answers POST /api/v1/activity-types/{activityTypeId}/impact with the warnings to confirm
// before saving.
func (h *Handler) activityTypeImpact(c *gin.Context) {
	var update ActivityTypeUpdate
	if !readBody(c, &update) {
		return
	}
	impact, err := h.service.ActivityTypeImpact(c.Request.Context(), actorOf(c), c.Param("activityTypeId"), update)
	respond(c, http.StatusOK, impact, err)
}

// setActivityTypeStatus answers POST /api/v1/activity-types/{activityTypeId}/status with the activity type in
// its new status.
func (h *Handler) setActivityTypeStatus(c *gin.Context) {
	var update StatusUpdate
	if !readBody(c, &update) {
		return
	}
	activityType, err := h.service.SetActivityTypeActive(c.Request.Context(), actorOf(c), c.Param("activityTypeId"), update)
	respond(c, http.StatusOK, activityType, err)
}

// listAttributes answers GET /api/v1/shell-attributes?level=: the attributes of the level, or of every level,
// in display order.
func (h *Handler) listAttributes(c *gin.Context) {
	var level *Level
	if text := c.Request.URL.Query().Get("level"); text != "" {
		l, ok := parseLevel(text)
		if !ok {
			web.RespondError(c, errs.BadRequest(unreadableMessage))
			return
		}
		level = &l
	}
	attributes, err := h.service.ListAttributes(c.Request.Context(), actorOf(c), level)
	respond(c, http.StatusOK, ShellAttributePage{Items: attributes, Total: len(attributes)}, err)
}

// getAttribute answers GET /api/v1/shell-attributes/{attributeId}.
func (h *Handler) getAttribute(c *gin.Context) {
	attribute, err := h.service.GetAttribute(c.Request.Context(), actorOf(c), c.Param("attributeId"))
	respond(c, http.StatusOK, attribute, err)
}

// createAttribute answers POST /api/v1/shell-attributes with 201 and the new attribute.
func (h *Handler) createAttribute(c *gin.Context) {
	var input ShellAttributeInput
	if !readBody(c, &input) {
		return
	}
	attribute, err := h.service.CreateAttribute(c.Request.Context(), actorOf(c), input)
	respond(c, http.StatusCreated, attribute, err)
}

// updateAttribute answers PATCH /api/v1/shell-attributes/{attributeId} with the attribute after the change.
func (h *Handler) updateAttribute(c *gin.Context) {
	var update ShellAttributeUpdate
	if !readBody(c, &update) {
		return
	}
	attribute, err := h.service.UpdateAttribute(c.Request.Context(), actorOf(c), c.Param("attributeId"), update)
	respond(c, http.StatusOK, attribute, err)
}

// deleteAttribute answers DELETE /api/v1/shell-attributes/{attributeId} with the recycle-bin entry.
func (h *Handler) deleteAttribute(c *gin.Context) {
	entry, err := h.service.DeleteAttribute(c.Request.Context(), actorOf(c), c.Param("attributeId"))
	respondDeleted(c, entry, err)
}

// attributeImpact answers POST /api/v1/shell-attributes/{attributeId}/impact with the warnings to confirm
// before saving.
func (h *Handler) attributeImpact(c *gin.Context) {
	var update ShellAttributeUpdate
	if !readBody(c, &update) {
		return
	}
	impact, err := h.service.AttributeImpact(c.Request.Context(), actorOf(c), c.Param("attributeId"), update)
	respond(c, http.StatusOK, impact, err)
}

// setAttributeStatus answers POST /api/v1/shell-attributes/{attributeId}/status with the attribute in its new
// status.
func (h *Handler) setAttributeStatus(c *gin.Context) {
	var update StatusUpdate
	if !readBody(c, &update) {
		return
	}
	attribute, err := h.service.SetAttributeActive(c.Request.Context(), actorOf(c), c.Param("attributeId"), update)
	respond(c, http.StatusOK, attribute, err)
}

// moveAttribute answers POST /api/v1/shell-attributes/{attributeId}/move with the level's attributes in their
// new order.
func (h *Handler) moveAttribute(c *gin.Context) {
	var in MoveInput
	if !readBody(c, &in) {
		return
	}
	attributes, err := h.service.MoveAttribute(c.Request.Context(), actorOf(c), c.Param("attributeId"), in)
	respond(c, http.StatusOK, ShellAttributePage{Items: attributes, Total: len(attributes)}, err)
}

// createOption answers POST /api/v1/shell-attributes/{attributeId}/options with 201 and the new option.
func (h *Handler) createOption(c *gin.Context) {
	var input OptionInput
	if !readBody(c, &input) {
		return
	}
	option, err := h.service.CreateOption(c.Request.Context(), actorOf(c), c.Param("attributeId"), input)
	respond(c, http.StatusCreated, option, err)
}

// checkImport answers POST /api/v1/shell-attributes/{attributeId}/options/import-check with every row's outcome.
// The request body holds a file, which is never logged.
func (h *Handler) checkImport(c *gin.Context) {
	var file ImportFile
	if !readBody(c, &file) {
		return
	}
	check, err := h.service.CheckImport(c.Request.Context(), actorOf(c), c.Param("attributeId"), file)
	respond(c, http.StatusOK, check, err)
}

// importOptions answers POST /api/v1/shell-attributes/{attributeId}/options/import with what was added and
// skipped. The request body holds a file, which is never logged.
func (h *Handler) importOptions(c *gin.Context) {
	var file ImportFile
	if !readBody(c, &file) {
		return
	}
	result, err := h.service.Import(c.Request.Context(), actorOf(c), c.Param("attributeId"), file)
	respond(c, http.StatusOK, result, err)
}

// getOption answers GET /api/v1/shell-attribute-options/{optionId} with the option as it is stored now.
func (h *Handler) getOption(c *gin.Context) {
	option, err := h.service.GetOption(c.Request.Context(), actorOf(c), c.Param("optionId"))
	respond(c, http.StatusOK, option, err)
}

// updateOption answers PATCH /api/v1/shell-attribute-options/{optionId} with the option after the change.
func (h *Handler) updateOption(c *gin.Context) {
	var update OptionUpdate
	if !readBody(c, &update) {
		return
	}
	option, err := h.service.UpdateOption(c.Request.Context(), actorOf(c), c.Param("optionId"), update)
	respond(c, http.StatusOK, option, err)
}

// deleteOption answers DELETE /api/v1/shell-attribute-options/{optionId} with the recycle-bin entry.
func (h *Handler) deleteOption(c *gin.Context) {
	entry, err := h.service.DeleteOption(c.Request.Context(), actorOf(c), c.Param("optionId"))
	respondDeleted(c, entry, err)
}

// setOptionStatus answers POST /api/v1/shell-attribute-options/{optionId}/status with the option in its new
// status.
func (h *Handler) setOptionStatus(c *gin.Context) {
	var update StatusUpdate
	if !readBody(c, &update) {
		return
	}
	option, err := h.service.SetOptionActive(c.Request.Context(), actorOf(c), c.Param("optionId"), update)
	respond(c, http.StatusOK, option, err)
}

// moveOption answers POST /api/v1/shell-attribute-options/{optionId}/move with the option in its new place.
func (h *Handler) moveOption(c *gin.Context) {
	var in OptionMoveInput
	if !readBody(c, &in) {
		return
	}
	option, err := h.service.MoveOption(c.Request.Context(), actorOf(c), c.Param("optionId"), in)
	respond(c, http.StatusOK, option, err)
}

// listOptions answers GET /api/v1/shell-attributes/{attributeId}/options?q=&status=&limit=&offset=. A limit or
// offset that is not a whole number gets the message of a number out of range.
func (h *Handler) listOptions(c *gin.Context) {
	query := c.Request.URL.Query()
	filter := OptionFilter{Q: query.Get("q"), Status: query.Get("status"), Limit: defaultOptionLimit}
	if query.Has("limit") {
		filter.Limit = wholeNumber(query.Get("limit"))
	}
	if query.Has("offset") {
		filter.Offset = wholeNumber(query.Get("offset"))
	}
	options, err := h.service.ListOptions(c.Request.Context(), actorOf(c), c.Param("attributeId"), filter)
	respond(c, http.StatusOK, options, err)
}

// getForm answers GET /api/v1/shell-attribute-forms/{level}: everything the level's form needs.
func (h *Handler) getForm(c *gin.Context) {
	level, ok := parseLevel(c.Param("level"))
	if !ok {
		web.RespondError(c, errs.BadRequest(unreadableMessage))
		return
	}
	form, err := h.service.Form(c.Request.Context(), actorOf(c), level)
	respond(c, http.StatusOK, form, err)
}

// checkValues answers POST /api/v1/shell-attribute-forms/{level}/check: what a save would say.
func (h *Handler) checkValues(c *gin.Context) {
	level, ok := parseLevel(c.Param("level"))
	if !ok {
		web.RespondError(c, errs.BadRequest(unreadableMessage))
		return
	}
	var request CheckRequest
	if !readBody(c, &request) {
		return
	}
	result, err := h.service.CheckValues(c.Request.Context(), actorOf(c), level, request)
	respond(c, http.StatusOK, result, err)
}

// listCurrencies answers GET /api/v1/currencies: USD first, then by code.
func (h *Handler) listCurrencies(c *gin.Context) {
	currencies, err := h.service.ListCurrencies(c.Request.Context(), actorOf(c))
	respond(c, http.StatusOK, CurrencyPage{Items: currencies, Total: len(currencies)}, err)
}

// getCurrency answers GET /api/v1/currencies/{code}.
func (h *Handler) getCurrency(c *gin.Context) {
	currency, err := h.service.GetCurrency(c.Request.Context(), actorOf(c), c.Param("code"))
	respond(c, http.StatusOK, currency, err)
}

// createCurrency answers POST /api/v1/currencies with 201 and the new currency.
func (h *Handler) createCurrency(c *gin.Context) {
	var input CurrencyInput
	if !readBody(c, &input) {
		return
	}
	currency, err := h.service.CreateCurrency(c.Request.Context(), actorOf(c), input)
	respond(c, http.StatusCreated, currency, err)
}

// updateCurrency answers PATCH /api/v1/currencies/{code} with the currency after the change.
func (h *Handler) updateCurrency(c *gin.Context) {
	var update CurrencyUpdate
	if !readBody(c, &update) {
		return
	}
	currency, err := h.service.UpdateCurrency(c.Request.Context(), actorOf(c), c.Param("code"), update)
	respond(c, http.StatusOK, currency, err)
}

// deleteCurrency answers DELETE /api/v1/currencies/{code} with the recycle-bin entry.
func (h *Handler) deleteCurrency(c *gin.Context) {
	entry, err := h.service.DeleteCurrency(c.Request.Context(), actorOf(c), c.Param("code"))
	respondDeleted(c, entry, err)
}

// currencyImpact answers POST /api/v1/currencies/{code}/impact with the warnings to confirm before saving.
func (h *Handler) currencyImpact(c *gin.Context) {
	var update CurrencyUpdate
	if !readBody(c, &update) {
		return
	}
	impact, err := h.service.CurrencyImpact(c.Request.Context(), actorOf(c), c.Param("code"), update)
	respond(c, http.StatusOK, impact, err)
}

// setCurrencyStatus answers POST /api/v1/currencies/{code}/status with the currency in its new status.
func (h *Handler) setCurrencyStatus(c *gin.Context) {
	var update StatusUpdate
	if !readBody(c, &update) {
		return
	}
	currency, err := h.service.SetCurrencyActive(c.Request.Context(), actorOf(c), c.Param("code"), update)
	respond(c, http.StatusOK, currency, err)
}

// listFinanceBudgets answers GET /api/v1/finance-budgets?year=: the budgets of the year (the current year
// when not sent).
func (h *Handler) listFinanceBudgets(c *gin.Context) {
	var year *int
	if text := c.Request.URL.Query().Get("year"); text != "" {
		y, err := strconv.Atoi(text)
		if err != nil {
			web.RespondError(c, errs.BadRequest(unreadableMessage))
			return
		}
		year = &y
	}
	page, err := h.service.ListFinanceBudgets(c.Request.Context(), actorOf(c), year)
	respond(c, http.StatusOK, page, err)
}

// renameFinanceBudget answers PATCH /api/v1/finance-budgets/{budgetId} with the budget and its items after the
// rename.
func (h *Handler) renameFinanceBudget(c *gin.Context) {
	var input RenameInput
	if !readBody(c, &input) {
		return
	}
	budget, err := h.service.RenameFinanceBudget(c.Request.Context(), actorOf(c), c.Param("budgetId"), input)
	respond(c, http.StatusOK, budget, err)
}

// createFinanceBudgetItem answers POST /api/v1/finance-budget-items with 201 and the new item.
func (h *Handler) createFinanceBudgetItem(c *gin.Context) {
	var input ItemInput
	if !readBody(c, &input) {
		return
	}
	item, err := h.service.CreateFinanceBudgetItem(c.Request.Context(), actorOf(c), input)
	respond(c, http.StatusCreated, item, err)
}

// getFinanceBudgetItem answers GET /api/v1/finance-budget-items/{itemId}.
func (h *Handler) getFinanceBudgetItem(c *gin.Context) {
	item, err := h.service.GetFinanceBudgetItem(c.Request.Context(), actorOf(c), c.Param("itemId"))
	respond(c, http.StatusOK, item, err)
}

// updateFinanceBudgetItem answers PATCH /api/v1/finance-budget-items/{itemId} with the item after the change.
func (h *Handler) updateFinanceBudgetItem(c *gin.Context) {
	var update ItemUpdate
	if !readBody(c, &update) {
		return
	}
	item, err := h.service.UpdateFinanceBudgetItem(c.Request.Context(), actorOf(c), c.Param("itemId"), update)
	respond(c, http.StatusOK, item, err)
}

// deleteFinanceBudgetItem answers DELETE /api/v1/finance-budget-items/{itemId} with the recycle-bin entry.
func (h *Handler) deleteFinanceBudgetItem(c *gin.Context) {
	entry, err := h.service.DeleteFinanceBudgetItem(c.Request.Context(), actorOf(c), c.Param("itemId"))
	respondDeleted(c, entry, err)
}

// setFinanceBudgetItemStatus answers POST /api/v1/finance-budget-items/{itemId}/status with the item in its new
// status.
func (h *Handler) setFinanceBudgetItemStatus(c *gin.Context) {
	var update StatusUpdate
	if !readBody(c, &update) {
		return
	}
	item, err := h.service.SetFinanceBudgetItemActive(c.Request.Context(), actorOf(c), c.Param("itemId"), update)
	respond(c, http.StatusOK, item, err)
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
