package settings

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
