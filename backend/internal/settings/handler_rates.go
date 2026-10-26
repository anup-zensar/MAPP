package settings

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

// The handlers of feature 016's exchange rates (FR-001 – FR-009) and Field help (FR-045, FR-046).

// rateRoutes registers the rate history, current rates and GSAP rate import routes.
func (h *Handler) rateRoutes(r gin.IRoutes) {
	r.GET("/currencies/:code/rates", h.listRates)
	r.POST("/currencies/:code/rates", h.addRate)
	r.POST("/currencies/:code/rates/impact", h.addRateImpact)
	r.PATCH("/exchange-rates/:rateId", h.updateRate)
	r.DELETE("/exchange-rates/:rateId", h.deleteRate)
	r.POST("/exchange-rates/:rateId/impact", h.updateRateImpact)
	r.GET("/exchange-rates/current", h.currentRates)
	r.POST("/exchange-rates/import-check", h.checkRateImport)
	r.POST("/exchange-rates/import", h.importRates)
	r.GET("/exchange-rates/imports", h.listRateImports)
	r.GET("/exchange-rates/import-sample", h.rateImportSample)
}

// fieldHelpRoutes registers the Field help routes.
func (h *Handler) fieldHelpRoutes(r gin.IRoutes) {
	r.GET("/field-help", h.listFieldHelp)
	r.PATCH("/field-help/:helpKey", h.updateFieldHelp)
}

// listRates answers GET /api/v1/currencies/{code}/rates: the rate history, newest first, and the current rate.
func (h *Handler) listRates(c *gin.Context) {
	page, err := h.service.ListRates(c.Request.Context(), actorOf(c), c.Param("code"))
	respond(c, http.StatusOK, page, err)
}

// addRate answers POST /api/v1/currencies/{code}/rates with 201 and the new rate.
func (h *Handler) addRate(c *gin.Context) {
	var input RateInput
	if !readBody(c, &input) {
		return
	}
	rate, err := h.service.AddRate(c.Request.Context(), actorOf(c), c.Param("code"), input)
	respond(c, http.StatusCreated, rate, err)
}

// addRateImpact answers POST /api/v1/currencies/{code}/rates/impact with what the new rate would change.
func (h *Handler) addRateImpact(c *gin.Context) {
	var input RateInput
	if !readBody(c, &input) {
		return
	}
	impact, err := h.service.AddRateImpact(c.Request.Context(), actorOf(c), c.Param("code"), input)
	respond(c, http.StatusOK, impact, err)
}

// updateRate answers PATCH /api/v1/exchange-rates/{rateId} with the rate after the change.
func (h *Handler) updateRate(c *gin.Context) {
	var update RateUpdate
	if !readBody(c, &update) {
		return
	}
	rate, err := h.service.UpdateRate(c.Request.Context(), actorOf(c), c.Param("rateId"), update)
	respond(c, http.StatusOK, rate, err)
}

// deleteRate answers DELETE /api/v1/exchange-rates/{rateId} with the recycle-bin entry.
func (h *Handler) deleteRate(c *gin.Context) {
	entry, err := h.service.DeleteRate(c.Request.Context(), actorOf(c), c.Param("rateId"))
	respondDeleted(c, entry, err)
}

// updateRateImpact answers POST /api/v1/exchange-rates/{rateId}/impact with what the change would change.
func (h *Handler) updateRateImpact(c *gin.Context) {
	var update RateUpdate
	if !readBody(c, &update) {
		return
	}
	impact, err := h.service.UpdateRateImpact(c.Request.Context(), actorOf(c), c.Param("rateId"), update)
	respond(c, http.StatusOK, impact, err)
}

// currentRates answers GET /api/v1/exchange-rates/current: every currency's current rate with its date and
// source (FR-009).
func (h *Handler) currentRates(c *gin.Context) {
	page, err := h.service.ListCurrentRates(c.Request.Context(), actorOf(c))
	respond(c, http.StatusOK, page, err)
}

// checkRateImport answers POST /api/v1/exchange-rates/import-check with every row's outcome. The body holds a
// file of up to 2 MB, which is never logged.
func (h *Handler) checkRateImport(c *gin.Context) {
	var upload RateImportUpload
	if !readBody(c, &upload) {
		return
	}
	check, err := h.service.CheckRateImport(c.Request.Context(), actorOf(c), upload)
	respond(c, http.StatusOK, check, err)
}

// importRates answers POST /api/v1/exchange-rates/import with 201 and the import record.
func (h *Handler) importRates(c *gin.Context) {
	var upload RateImportUpload
	if !readBody(c, &upload) {
		return
	}
	result, err := h.service.ApplyRateImport(c.Request.Context(), actorOf(c), upload)
	respond(c, http.StatusCreated, result, err)
}

// listRateImports answers GET /api/v1/exchange-rates/imports?limit=: the latest imports, newest first.
func (h *Handler) listRateImports(c *gin.Context) {
	limit := defaultRecentImports
	if text := c.Request.URL.Query().Get("limit"); text != "" {
		n, err := strconv.Atoi(text)
		if err != nil || n < 1 || n > maxRecentImports {
			web.RespondError(c, errs.BadRequest(unreadableMessage))
			return
		}
		limit = n
	}
	page, err := h.service.ListRateImports(c.Request.Context(), actorOf(c), limit)
	respond(c, http.StatusOK, page, err)
}

// rateImportSample answers GET /api/v1/exchange-rates/import-sample with a CSV file of today's rates.
func (h *Handler) rateImportSample(c *gin.Context) {
	name, content, err := h.service.ImportSample(c.Request.Context())
	if err != nil {
		web.RespondError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", content)
}

// listFieldHelp answers GET /api/v1/field-help: every catalogue entry with its text.
func (h *Handler) listFieldHelp(c *gin.Context) {
	page, err := h.service.ListFieldHelp(c.Request.Context(), actorOf(c))
	respond(c, http.StatusOK, page, err)
}

// updateFieldHelp answers PATCH /api/v1/field-help/{helpKey} with the entry after the change.
func (h *Handler) updateFieldHelp(c *gin.Context) {
	var update FieldHelpUpdate
	if !readBody(c, &update) {
		return
	}
	entry, err := h.service.UpdateFieldHelp(c.Request.Context(), actorOf(c), c.Param("helpKey"), update)
	respond(c, http.StatusOK, entry, err)
}
