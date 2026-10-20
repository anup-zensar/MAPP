package settings

import (
	"net/http"
	"strconv"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/web"
)

// unreadableMessage answers a path or query value the API cannot read, such as an unknown level.
const unreadableMessage = "NorthStar couldn't read this request. Reload the page and try again."

// Handler answers the HTTP requests of Settings (feature 002).
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the Settings routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the Settings routes on r. All of them need a signed-in person, so all are wrapped in
// signedIn; every signed-in person may change Settings (FR-008, D-004).
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/settings/overview", signedIn(h.overview))

	r.Handle("GET /api/models", signedIn(h.listModels))
	r.Handle("POST /api/models", signedIn(h.createModel))
	r.Handle("GET /api/models/{modelId}", signedIn(h.getModel))
	r.Handle("PATCH /api/models/{modelId}", signedIn(h.updateModel))
	r.Handle("DELETE /api/models/{modelId}", signedIn(h.deleteModel))
	r.Handle("POST /api/models/{modelId}/impact", signedIn(h.modelImpact))
	r.Handle("POST /api/models/{modelId}/status", signedIn(h.setModelStatus))

	r.Handle("GET /api/activity-types", signedIn(h.listActivityTypes))
	r.Handle("POST /api/activity-types", signedIn(h.createActivityType))
	r.Handle("GET /api/activity-types/{activityTypeId}", signedIn(h.getActivityType))
	r.Handle("PATCH /api/activity-types/{activityTypeId}", signedIn(h.updateActivityType))
	r.Handle("DELETE /api/activity-types/{activityTypeId}", signedIn(h.deleteActivityType))
	r.Handle("POST /api/activity-types/{activityTypeId}/impact", signedIn(h.activityTypeImpact))
	r.Handle("POST /api/activity-types/{activityTypeId}/status", signedIn(h.setActivityTypeStatus))

	r.Handle("GET /api/shell-attributes", signedIn(h.listAttributes))
	r.Handle("POST /api/shell-attributes", signedIn(h.createAttribute))
	r.Handle("GET /api/shell-attributes/{attributeId}", signedIn(h.getAttribute))
	r.Handle("PATCH /api/shell-attributes/{attributeId}", signedIn(h.updateAttribute))
	r.Handle("DELETE /api/shell-attributes/{attributeId}", signedIn(h.deleteAttribute))
	r.Handle("POST /api/shell-attributes/{attributeId}/impact", signedIn(h.attributeImpact))
	r.Handle("POST /api/shell-attributes/{attributeId}/status", signedIn(h.setAttributeStatus))
	r.Handle("POST /api/shell-attributes/{attributeId}/move", signedIn(h.moveAttribute))
	r.Handle("GET /api/shell-attributes/{attributeId}/options", signedIn(h.listOptions))
	r.Handle("POST /api/shell-attributes/{attributeId}/options", signedIn(h.createOption))
	r.Handle("POST /api/shell-attributes/{attributeId}/options/import-check", signedIn(h.checkImport))
	r.Handle("POST /api/shell-attributes/{attributeId}/options/import", signedIn(h.importOptions))
	r.Handle("GET /api/shell-attribute-options/{optionId}", signedIn(h.getOption))
	r.Handle("PATCH /api/shell-attribute-options/{optionId}", signedIn(h.updateOption))
	r.Handle("DELETE /api/shell-attribute-options/{optionId}", signedIn(h.deleteOption))
	r.Handle("POST /api/shell-attribute-options/{optionId}/status", signedIn(h.setOptionStatus))
	r.Handle("POST /api/shell-attribute-options/{optionId}/move", signedIn(h.moveOption))
	r.Handle("GET /api/shell-attribute-forms/{level}", signedIn(h.getForm))
	r.Handle("POST /api/shell-attribute-forms/{level}/check", signedIn(h.checkValues))

	r.Handle("GET /api/currencies", signedIn(h.listCurrencies))
	r.Handle("POST /api/currencies", signedIn(h.createCurrency))
	r.Handle("GET /api/currencies/{code}", signedIn(h.getCurrency))
	r.Handle("PATCH /api/currencies/{code}", signedIn(h.updateCurrency))
	r.Handle("DELETE /api/currencies/{code}", signedIn(h.deleteCurrency))
	r.Handle("POST /api/currencies/{code}/impact", signedIn(h.currencyImpact))
	r.Handle("POST /api/currencies/{code}/status", signedIn(h.setCurrencyStatus))
	h.rateRoutes(r, signedIn)      // 016: rate history, current rates, GSAP rate import
	h.fieldHelpRoutes(r, signedIn) // 016: Field help

	r.Handle("GET /api/finance-budgets", signedIn(h.listFinanceBudgets))
	r.Handle("PATCH /api/finance-budgets/{budgetId}", signedIn(h.renameFinanceBudget))
	r.Handle("POST /api/finance-budget-items", signedIn(h.createFinanceBudgetItem))
	r.Handle("GET /api/finance-budget-items/{itemId}", signedIn(h.getFinanceBudgetItem))
	r.Handle("PATCH /api/finance-budget-items/{itemId}", signedIn(h.updateFinanceBudgetItem))
	r.Handle("DELETE /api/finance-budget-items/{itemId}", signedIn(h.deleteFinanceBudgetItem))
	r.Handle("POST /api/finance-budget-items/{itemId}/status", signedIn(h.setFinanceBudgetItemStatus))
}

// overview answers GET /api/settings/overview: the active records of each section (FR-001).
func (h *Handler) overview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.service.Overview(r.Context(), actorOf(r))
	respond(w, r, http.StatusOK, overview, err)
}

// listModels answers GET /api/models.
func (h *Handler) listModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.service.ListModels(r.Context(), actorOf(r))
	respond(w, r, http.StatusOK, ModelPage{Items: models, Total: len(models)}, err)
}

// createModel answers POST /api/models with 201 and the new model.
func (h *Handler) createModel(w http.ResponseWriter, r *http.Request) {
	var input ModelInput
	if !readBody(w, r, &input) {
		return
	}
	model, err := h.service.CreateModel(r.Context(), actorOf(r), input)
	respond(w, r, http.StatusCreated, model, err)
}

// getModel answers GET /api/models/{modelId}.
func (h *Handler) getModel(w http.ResponseWriter, r *http.Request) {
	model, err := h.service.GetModel(r.Context(), actorOf(r), r.PathValue("modelId"))
	respond(w, r, http.StatusOK, model, err)
}

// updateModel answers PATCH /api/models/{modelId} with the model after the change.
func (h *Handler) updateModel(w http.ResponseWriter, r *http.Request) {
	var update ModelUpdate
	if !readBody(w, r, &update) {
		return
	}
	model, err := h.service.UpdateModel(r.Context(), actorOf(r), r.PathValue("modelId"), update)
	respond(w, r, http.StatusOK, model, err)
}

// deleteModel answers DELETE /api/models/{modelId} with the recycle-bin entry, so the web app can offer
// Undo.
func (h *Handler) deleteModel(w http.ResponseWriter, r *http.Request) {
	entry, err := h.service.DeleteModel(r.Context(), actorOf(r), r.PathValue("modelId"))
	respondDeleted(w, r, entry, err)
}

// modelImpact answers POST /api/models/{modelId}/impact with the warnings to confirm before saving.
func (h *Handler) modelImpact(w http.ResponseWriter, r *http.Request) {
	var update ModelUpdate
	if !readBody(w, r, &update) {
		return
	}
	impact, err := h.service.ModelImpact(r.Context(), actorOf(r), r.PathValue("modelId"), update)
	respond(w, r, http.StatusOK, impact, err)
}

// setModelStatus answers POST /api/models/{modelId}/status with the model in its new status.
func (h *Handler) setModelStatus(w http.ResponseWriter, r *http.Request) {
	var update StatusUpdate
	if !readBody(w, r, &update) {
		return
	}
	model, err := h.service.SetModelActive(r.Context(), actorOf(r), r.PathValue("modelId"), update)
	respond(w, r, http.StatusOK, model, err)
}

// listActivityTypes answers GET /api/activity-types.
func (h *Handler) listActivityTypes(w http.ResponseWriter, r *http.Request) {
	types, err := h.service.ListActivityTypes(r.Context(), actorOf(r))
	respond(w, r, http.StatusOK, ActivityTypePage{Items: types, Total: len(types)}, err)
}

// getActivityType answers GET /api/activity-types/{activityTypeId}.
func (h *Handler) getActivityType(w http.ResponseWriter, r *http.Request) {
	activityType, err := h.service.GetActivityType(r.Context(), actorOf(r), r.PathValue("activityTypeId"))
	respond(w, r, http.StatusOK, activityType, err)
}

// createActivityType answers POST /api/activity-types with 201 and the new activity type.
func (h *Handler) createActivityType(w http.ResponseWriter, r *http.Request) {
	var input ActivityTypeInput
	if !readBody(w, r, &input) {
		return
	}
	activityType, err := h.service.CreateActivityType(r.Context(), actorOf(r), input)
	respond(w, r, http.StatusCreated, activityType, err)
}

// updateActivityType answers PATCH /api/activity-types/{activityTypeId} with the activity type after the
// change.
func (h *Handler) updateActivityType(w http.ResponseWriter, r *http.Request) {
	var update ActivityTypeUpdate
	if !readBody(w, r, &update) {
		return
	}
	activityType, err := h.service.UpdateActivityType(r.Context(), actorOf(r), r.PathValue("activityTypeId"), update)
	respond(w, r, http.StatusOK, activityType, err)
}

// deleteActivityType answers DELETE /api/activity-types/{activityTypeId} with the recycle-bin entry.
func (h *Handler) deleteActivityType(w http.ResponseWriter, r *http.Request) {
	entry, err := h.service.DeleteActivityType(r.Context(), actorOf(r), r.PathValue("activityTypeId"))
	respondDeleted(w, r, entry, err)
}

// activityTypeImpact answers POST /api/activity-types/{activityTypeId}/impact with the warnings to confirm
// before saving.
func (h *Handler) activityTypeImpact(w http.ResponseWriter, r *http.Request) {
	var update ActivityTypeUpdate
	if !readBody(w, r, &update) {
		return
	}
	impact, err := h.service.ActivityTypeImpact(r.Context(), actorOf(r), r.PathValue("activityTypeId"), update)
	respond(w, r, http.StatusOK, impact, err)
}

// setActivityTypeStatus answers POST /api/activity-types/{activityTypeId}/status with the activity type in
// its new status.
func (h *Handler) setActivityTypeStatus(w http.ResponseWriter, r *http.Request) {
	var update StatusUpdate
	if !readBody(w, r, &update) {
		return
	}
	activityType, err := h.service.SetActivityTypeActive(r.Context(), actorOf(r), r.PathValue("activityTypeId"), update)
	respond(w, r, http.StatusOK, activityType, err)
}

// listAttributes answers GET /api/shell-attributes?level=: the attributes of the level, or of every level,
// in display order.
func (h *Handler) listAttributes(w http.ResponseWriter, r *http.Request) {
	var level *Level
	if text := r.URL.Query().Get("level"); text != "" {
		l, ok := parseLevel(text)
		if !ok {
			web.WriteError(w, r, errs.BadRequest(unreadableMessage))
			return
		}
		level = &l
	}
	attributes, err := h.service.ListAttributes(r.Context(), actorOf(r), level)
	respond(w, r, http.StatusOK, ShellAttributePage{Items: attributes, Total: len(attributes)}, err)
}

// getAttribute answers GET /api/shell-attributes/{attributeId}.
func (h *Handler) getAttribute(w http.ResponseWriter, r *http.Request) {
	attribute, err := h.service.GetAttribute(r.Context(), actorOf(r), r.PathValue("attributeId"))
	respond(w, r, http.StatusOK, attribute, err)
}

// createAttribute answers POST /api/shell-attributes with 201 and the new attribute.
func (h *Handler) createAttribute(w http.ResponseWriter, r *http.Request) {
	var input ShellAttributeInput
	if !readBody(w, r, &input) {
		return
	}
	attribute, err := h.service.CreateAttribute(r.Context(), actorOf(r), input)
	respond(w, r, http.StatusCreated, attribute, err)
}

// updateAttribute answers PATCH /api/shell-attributes/{attributeId} with the attribute after the change.
func (h *Handler) updateAttribute(w http.ResponseWriter, r *http.Request) {
	var update ShellAttributeUpdate
	if !readBody(w, r, &update) {
		return
	}
	attribute, err := h.service.UpdateAttribute(r.Context(), actorOf(r), r.PathValue("attributeId"), update)
	respond(w, r, http.StatusOK, attribute, err)
}

// deleteAttribute answers DELETE /api/shell-attributes/{attributeId} with the recycle-bin entry.
func (h *Handler) deleteAttribute(w http.ResponseWriter, r *http.Request) {
	entry, err := h.service.DeleteAttribute(r.Context(), actorOf(r), r.PathValue("attributeId"))
	respondDeleted(w, r, entry, err)
}

// attributeImpact answers POST /api/shell-attributes/{attributeId}/impact with the warnings to confirm
// before saving.
func (h *Handler) attributeImpact(w http.ResponseWriter, r *http.Request) {
	var update ShellAttributeUpdate
	if !readBody(w, r, &update) {
		return
	}
	impact, err := h.service.AttributeImpact(r.Context(), actorOf(r), r.PathValue("attributeId"), update)
	respond(w, r, http.StatusOK, impact, err)
}

// setAttributeStatus answers POST /api/shell-attributes/{attributeId}/status with the attribute in its new
// status.
func (h *Handler) setAttributeStatus(w http.ResponseWriter, r *http.Request) {
	var update StatusUpdate
	if !readBody(w, r, &update) {
		return
	}
	attribute, err := h.service.SetAttributeActive(r.Context(), actorOf(r), r.PathValue("attributeId"), update)
	respond(w, r, http.StatusOK, attribute, err)
}

// moveAttribute answers POST /api/shell-attributes/{attributeId}/move with the level's attributes in their
// new order.
func (h *Handler) moveAttribute(w http.ResponseWriter, r *http.Request) {
	var in MoveInput
	if !readBody(w, r, &in) {
		return
	}
	attributes, err := h.service.MoveAttribute(r.Context(), actorOf(r), r.PathValue("attributeId"), in)
	respond(w, r, http.StatusOK, ShellAttributePage{Items: attributes, Total: len(attributes)}, err)
}

// createOption answers POST /api/shell-attributes/{attributeId}/options with 201 and the new option.
func (h *Handler) createOption(w http.ResponseWriter, r *http.Request) {
	var input OptionInput
	if !readBody(w, r, &input) {
		return
	}
	option, err := h.service.CreateOption(r.Context(), actorOf(r), r.PathValue("attributeId"), input)
	respond(w, r, http.StatusCreated, option, err)
}

// checkImport answers POST /api/shell-attributes/{attributeId}/options/import-check with every row's outcome.
// The request body holds a file, which is never logged.
func (h *Handler) checkImport(w http.ResponseWriter, r *http.Request) {
	var file ImportFile
	if !readBody(w, r, &file) {
		return
	}
	check, err := h.service.CheckImport(r.Context(), actorOf(r), r.PathValue("attributeId"), file)
	respond(w, r, http.StatusOK, check, err)
}

// importOptions answers POST /api/shell-attributes/{attributeId}/options/import with what was added and
// skipped. The request body holds a file, which is never logged.
func (h *Handler) importOptions(w http.ResponseWriter, r *http.Request) {
	var file ImportFile
	if !readBody(w, r, &file) {
		return
	}
	result, err := h.service.Import(r.Context(), actorOf(r), r.PathValue("attributeId"), file)
	respond(w, r, http.StatusOK, result, err)
}

// getOption answers GET /api/shell-attribute-options/{optionId} with the option as it is stored now.
func (h *Handler) getOption(w http.ResponseWriter, r *http.Request) {
	option, err := h.service.GetOption(r.Context(), actorOf(r), r.PathValue("optionId"))
	respond(w, r, http.StatusOK, option, err)
}

// updateOption answers PATCH /api/shell-attribute-options/{optionId} with the option after the change.
func (h *Handler) updateOption(w http.ResponseWriter, r *http.Request) {
	var update OptionUpdate
	if !readBody(w, r, &update) {
		return
	}
	option, err := h.service.UpdateOption(r.Context(), actorOf(r), r.PathValue("optionId"), update)
	respond(w, r, http.StatusOK, option, err)
}

// deleteOption answers DELETE /api/shell-attribute-options/{optionId} with the recycle-bin entry.
func (h *Handler) deleteOption(w http.ResponseWriter, r *http.Request) {
	entry, err := h.service.DeleteOption(r.Context(), actorOf(r), r.PathValue("optionId"))
	respondDeleted(w, r, entry, err)
}

// setOptionStatus answers POST /api/shell-attribute-options/{optionId}/status with the option in its new
// status.
func (h *Handler) setOptionStatus(w http.ResponseWriter, r *http.Request) {
	var update StatusUpdate
	if !readBody(w, r, &update) {
		return
	}
	option, err := h.service.SetOptionActive(r.Context(), actorOf(r), r.PathValue("optionId"), update)
	respond(w, r, http.StatusOK, option, err)
}

// moveOption answers POST /api/shell-attribute-options/{optionId}/move with the option in its new place.
func (h *Handler) moveOption(w http.ResponseWriter, r *http.Request) {
	var in OptionMoveInput
	if !readBody(w, r, &in) {
		return
	}
	option, err := h.service.MoveOption(r.Context(), actorOf(r), r.PathValue("optionId"), in)
	respond(w, r, http.StatusOK, option, err)
}

// listOptions answers GET /api/shell-attributes/{attributeId}/options?q=&status=&limit=&offset=. A limit or
// offset that is not a whole number gets the message of a number out of range.
func (h *Handler) listOptions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := OptionFilter{Q: query.Get("q"), Status: query.Get("status"), Limit: defaultOptionLimit}
	if query.Has("limit") {
		filter.Limit = wholeNumber(query.Get("limit"))
	}
	if query.Has("offset") {
		filter.Offset = wholeNumber(query.Get("offset"))
	}
	options, err := h.service.ListOptions(r.Context(), actorOf(r), r.PathValue("attributeId"), filter)
	respond(w, r, http.StatusOK, options, err)
}

// getForm answers GET /api/shell-attribute-forms/{level}: everything the level's form needs.
func (h *Handler) getForm(w http.ResponseWriter, r *http.Request) {
	level, ok := parseLevel(r.PathValue("level"))
	if !ok {
		web.WriteError(w, r, errs.BadRequest(unreadableMessage))
		return
	}
	form, err := h.service.Form(r.Context(), actorOf(r), level)
	respond(w, r, http.StatusOK, form, err)
}

// checkValues answers POST /api/shell-attribute-forms/{level}/check: what a save would say.
func (h *Handler) checkValues(w http.ResponseWriter, r *http.Request) {
	level, ok := parseLevel(r.PathValue("level"))
	if !ok {
		web.WriteError(w, r, errs.BadRequest(unreadableMessage))
		return
	}
	var request CheckRequest
	if !readBody(w, r, &request) {
		return
	}
	result, err := h.service.CheckValues(r.Context(), actorOf(r), level, request)
	respond(w, r, http.StatusOK, result, err)
}

// listCurrencies answers GET /api/currencies: USD first, then by code.
func (h *Handler) listCurrencies(w http.ResponseWriter, r *http.Request) {
	currencies, err := h.service.ListCurrencies(r.Context(), actorOf(r))
	respond(w, r, http.StatusOK, CurrencyPage{Items: currencies, Total: len(currencies)}, err)
}

// getCurrency answers GET /api/currencies/{code}.
func (h *Handler) getCurrency(w http.ResponseWriter, r *http.Request) {
	currency, err := h.service.GetCurrency(r.Context(), actorOf(r), r.PathValue("code"))
	respond(w, r, http.StatusOK, currency, err)
}

// createCurrency answers POST /api/currencies with 201 and the new currency.
func (h *Handler) createCurrency(w http.ResponseWriter, r *http.Request) {
	var input CurrencyInput
	if !readBody(w, r, &input) {
		return
	}
	currency, err := h.service.CreateCurrency(r.Context(), actorOf(r), input)
	respond(w, r, http.StatusCreated, currency, err)
}

// updateCurrency answers PATCH /api/currencies/{code} with the currency after the change.
func (h *Handler) updateCurrency(w http.ResponseWriter, r *http.Request) {
	var update CurrencyUpdate
	if !readBody(w, r, &update) {
		return
	}
	currency, err := h.service.UpdateCurrency(r.Context(), actorOf(r), r.PathValue("code"), update)
	respond(w, r, http.StatusOK, currency, err)
}

// deleteCurrency answers DELETE /api/currencies/{code} with the recycle-bin entry.
func (h *Handler) deleteCurrency(w http.ResponseWriter, r *http.Request) {
	entry, err := h.service.DeleteCurrency(r.Context(), actorOf(r), r.PathValue("code"))
	respondDeleted(w, r, entry, err)
}

// currencyImpact answers POST /api/currencies/{code}/impact with the warnings to confirm before saving.
func (h *Handler) currencyImpact(w http.ResponseWriter, r *http.Request) {
	var update CurrencyUpdate
	if !readBody(w, r, &update) {
		return
	}
	impact, err := h.service.CurrencyImpact(r.Context(), actorOf(r), r.PathValue("code"), update)
	respond(w, r, http.StatusOK, impact, err)
}

// setCurrencyStatus answers POST /api/currencies/{code}/status with the currency in its new status.
func (h *Handler) setCurrencyStatus(w http.ResponseWriter, r *http.Request) {
	var update StatusUpdate
	if !readBody(w, r, &update) {
		return
	}
	currency, err := h.service.SetCurrencyActive(r.Context(), actorOf(r), r.PathValue("code"), update)
	respond(w, r, http.StatusOK, currency, err)
}

// listFinanceBudgets answers GET /api/finance-budgets?year=: the budgets of the year (the current year
// when not sent).
func (h *Handler) listFinanceBudgets(w http.ResponseWriter, r *http.Request) {
	var year *int
	if text := r.URL.Query().Get("year"); text != "" {
		y, err := strconv.Atoi(text)
		if err != nil {
			web.WriteError(w, r, errs.BadRequest(unreadableMessage))
			return
		}
		year = &y
	}
	page, err := h.service.ListFinanceBudgets(r.Context(), actorOf(r), year)
	respond(w, r, http.StatusOK, page, err)
}

// renameFinanceBudget answers PATCH /api/finance-budgets/{budgetId} with the budget and its items after the
// rename.
func (h *Handler) renameFinanceBudget(w http.ResponseWriter, r *http.Request) {
	var input RenameInput
	if !readBody(w, r, &input) {
		return
	}
	budget, err := h.service.RenameFinanceBudget(r.Context(), actorOf(r), r.PathValue("budgetId"), input)
	respond(w, r, http.StatusOK, budget, err)
}

// createFinanceBudgetItem answers POST /api/finance-budget-items with 201 and the new item.
func (h *Handler) createFinanceBudgetItem(w http.ResponseWriter, r *http.Request) {
	var input ItemInput
	if !readBody(w, r, &input) {
		return
	}
	item, err := h.service.CreateFinanceBudgetItem(r.Context(), actorOf(r), input)
	respond(w, r, http.StatusCreated, item, err)
}

// getFinanceBudgetItem answers GET /api/finance-budget-items/{itemId}.
func (h *Handler) getFinanceBudgetItem(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetFinanceBudgetItem(r.Context(), actorOf(r), r.PathValue("itemId"))
	respond(w, r, http.StatusOK, item, err)
}

// updateFinanceBudgetItem answers PATCH /api/finance-budget-items/{itemId} with the item after the change.
func (h *Handler) updateFinanceBudgetItem(w http.ResponseWriter, r *http.Request) {
	var update ItemUpdate
	if !readBody(w, r, &update) {
		return
	}
	item, err := h.service.UpdateFinanceBudgetItem(r.Context(), actorOf(r), r.PathValue("itemId"), update)
	respond(w, r, http.StatusOK, item, err)
}

// deleteFinanceBudgetItem answers DELETE /api/finance-budget-items/{itemId} with the recycle-bin entry.
func (h *Handler) deleteFinanceBudgetItem(w http.ResponseWriter, r *http.Request) {
	entry, err := h.service.DeleteFinanceBudgetItem(r.Context(), actorOf(r), r.PathValue("itemId"))
	respondDeleted(w, r, entry, err)
}

// setFinanceBudgetItemStatus answers POST /api/finance-budget-items/{itemId}/status with the item in its new
// status.
func (h *Handler) setFinanceBudgetItemStatus(w http.ResponseWriter, r *http.Request) {
	var update StatusUpdate
	if !readBody(w, r, &update) {
		return
	}
	item, err := h.service.SetFinanceBudgetItemActive(r.Context(), actorOf(r), r.PathValue("itemId"), update)
	respond(w, r, http.StatusOK, item, err)
}

// actorOf returns the signed-in person, put in the request's context by the sign-in middleware.
func actorOf(r *http.Request) actor.Actor {
	a, _ := actor.From(r.Context())
	return a
}

// readBody decodes the request's JSON body into dst; when it cannot, it answers the error and returns
// false.
func readBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := web.ReadJSON(w, r, dst); err != nil {
		web.WriteError(w, r, err)
		return false
	}
	return true
}

// respond answers body with status, or err when the service failed.
func respond(w http.ResponseWriter, r *http.Request, status int, body any, err error) {
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, status, body)
}

// respondDeleted answers a delete with the recycle-bin entry, so the web app can offer Undo.
func respondDeleted(w http.ResponseWriter, r *http.Request, entry history.BinEntry, err error) {
	respond(w, r, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()}, err)
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
