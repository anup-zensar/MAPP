package settings

import (
	"net/http"
	"strconv"

	"northstar/internal/platform/errs"
	"northstar/internal/platform/web"
)

// The handlers of feature 016's exchange rates (FR-001 – FR-009) and Field help (FR-045, FR-046).

// rateRoutes registers the rate history, current rates and GSAP rate import routes.
func (h *Handler) rateRoutes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/currencies/{code}/rates", signedIn(h.listRates))
	r.Handle("POST /api/currencies/{code}/rates", signedIn(h.addRate))
	r.Handle("POST /api/currencies/{code}/rates/impact", signedIn(h.addRateImpact))
	r.Handle("PATCH /api/exchange-rates/{rateId}", signedIn(h.updateRate))
	r.Handle("DELETE /api/exchange-rates/{rateId}", signedIn(h.deleteRate))
	r.Handle("POST /api/exchange-rates/{rateId}/impact", signedIn(h.updateRateImpact))
	r.Handle("GET /api/exchange-rates/current", signedIn(h.currentRates))
	r.Handle("POST /api/exchange-rates/import-check", signedIn(h.checkRateImport))
	r.Handle("POST /api/exchange-rates/import", signedIn(h.importRates))
	r.Handle("GET /api/exchange-rates/imports", signedIn(h.listRateImports))
	r.Handle("GET /api/exchange-rates/import-sample", signedIn(h.rateImportSample))
}

// fieldHelpRoutes registers the Field help routes.
func (h *Handler) fieldHelpRoutes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/field-help", signedIn(h.listFieldHelp))
	r.Handle("PATCH /api/field-help/{helpKey}", signedIn(h.updateFieldHelp))
}

// listRates answers GET /api/currencies/{code}/rates: the rate history, newest first, and the current rate.
func (h *Handler) listRates(w http.ResponseWriter, r *http.Request) {
	page, err := h.service.ListRates(r.Context(), actorOf(r), r.PathValue("code"))
	respond(w, r, http.StatusOK, page, err)
}

// addRate answers POST /api/currencies/{code}/rates with 201 and the new rate.
func (h *Handler) addRate(w http.ResponseWriter, r *http.Request) {
	var input RateInput
	if !readBody(w, r, &input) {
		return
	}
	rate, err := h.service.AddRate(r.Context(), actorOf(r), r.PathValue("code"), input)
	respond(w, r, http.StatusCreated, rate, err)
}

// addRateImpact answers POST /api/currencies/{code}/rates/impact with what the new rate would change.
func (h *Handler) addRateImpact(w http.ResponseWriter, r *http.Request) {
	var input RateInput
	if !readBody(w, r, &input) {
		return
	}
	impact, err := h.service.AddRateImpact(r.Context(), actorOf(r), r.PathValue("code"), input)
	respond(w, r, http.StatusOK, impact, err)
}

// updateRate answers PATCH /api/exchange-rates/{rateId} with the rate after the change.
func (h *Handler) updateRate(w http.ResponseWriter, r *http.Request) {
	var update RateUpdate
	if !readBody(w, r, &update) {
		return
	}
	rate, err := h.service.UpdateRate(r.Context(), actorOf(r), r.PathValue("rateId"), update)
	respond(w, r, http.StatusOK, rate, err)
}

// deleteRate answers DELETE /api/exchange-rates/{rateId} with the recycle-bin entry.
func (h *Handler) deleteRate(w http.ResponseWriter, r *http.Request) {
	entry, err := h.service.DeleteRate(r.Context(), actorOf(r), r.PathValue("rateId"))
	respondDeleted(w, r, entry, err)
}

// updateRateImpact answers POST /api/exchange-rates/{rateId}/impact with what the change would change.
func (h *Handler) updateRateImpact(w http.ResponseWriter, r *http.Request) {
	var update RateUpdate
	if !readBody(w, r, &update) {
		return
	}
	impact, err := h.service.UpdateRateImpact(r.Context(), actorOf(r), r.PathValue("rateId"), update)
	respond(w, r, http.StatusOK, impact, err)
}

// currentRates answers GET /api/exchange-rates/current: every currency's current rate with its date and
// source (FR-009).
func (h *Handler) currentRates(w http.ResponseWriter, r *http.Request) {
	page, err := h.service.ListCurrentRates(r.Context(), actorOf(r))
	respond(w, r, http.StatusOK, page, err)
}

// checkRateImport answers POST /api/exchange-rates/import-check with every row's outcome. The body holds a
// file of up to 2 MB, which is never logged.
func (h *Handler) checkRateImport(w http.ResponseWriter, r *http.Request) {
	var upload RateImportUpload
	if !readRateUpload(w, r, &upload) {
		return
	}
	check, err := h.service.CheckRateImport(r.Context(), actorOf(r), upload)
	respond(w, r, http.StatusOK, check, err)
}

// importRates answers POST /api/exchange-rates/import with 201 and the import record.
func (h *Handler) importRates(w http.ResponseWriter, r *http.Request) {
	var upload RateImportUpload
	if !readRateUpload(w, r, &upload) {
		return
	}
	result, err := h.service.ApplyRateImport(r.Context(), actorOf(r), upload)
	respond(w, r, http.StatusCreated, result, err)
}

// listRateImports answers GET /api/exchange-rates/imports?limit=: the latest imports, newest first.
func (h *Handler) listRateImports(w http.ResponseWriter, r *http.Request) {
	limit := defaultRecentImports
	if text := r.URL.Query().Get("limit"); text != "" {
		n, err := strconv.Atoi(text)
		if err != nil || n < 1 || n > maxRecentImports {
			web.WriteError(w, r, errs.BadRequest(unreadableMessage))
			return
		}
		limit = n
	}
	page, err := h.service.ListRateImports(r.Context(), actorOf(r), limit)
	respond(w, r, http.StatusOK, page, err)
}

// rateImportSample answers GET /api/exchange-rates/import-sample with a CSV file of today's rates.
func (h *Handler) rateImportSample(w http.ResponseWriter, r *http.Request) {
	name, content, err := h.service.ImportSample(r.Context())
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content) // a failed write means the client has gone
}

// listFieldHelp answers GET /api/field-help: every catalogue entry with its text.
func (h *Handler) listFieldHelp(w http.ResponseWriter, r *http.Request) {
	page, err := h.service.ListFieldHelp(r.Context(), actorOf(r))
	respond(w, r, http.StatusOK, page, err)
}

// updateFieldHelp answers PATCH /api/field-help/{helpKey} with the entry after the change.
func (h *Handler) updateFieldHelp(w http.ResponseWriter, r *http.Request) {
	var update FieldHelpUpdate
	if !readBody(w, r, &update) {
		return
	}
	entry, err := h.service.UpdateFieldHelp(r.Context(), actorOf(r), r.PathValue("helpKey"), update)
	respond(w, r, http.StatusOK, entry, err)
}

// readRateUpload decodes a rate file upload of up to 3 MB; when it cannot, it answers the error and returns false.
func readRateUpload(w http.ResponseWriter, r *http.Request, dst *RateImportUpload) bool {
	if err := web.ReadJSONUpTo(w, r, dst, maxRateUploadBytes); err != nil {
		web.WriteError(w, r, err)
		return false
	}
	return true
}
