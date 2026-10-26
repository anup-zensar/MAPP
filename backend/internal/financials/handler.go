package financials

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// Handler serves the operations of feature 015 (plan, "API operations"). Every route needs a signed-in
// person; the service checks access on every operation (FR-028).
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the financials service s.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// RegisterRoutes registers the ten operations of feature 015 on r.
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/financials/records/:recordId", h.record)
	r.GET("/financials/transactions", h.transactions)
	r.GET("/financials/transactions/:transactionId", h.transaction)
	r.POST("/financials/transactions/:transactionId/link", h.link)
	r.POST("/financials/transactions/:transactionId/unlink", h.unlink)
	r.GET("/financials/overview", h.overview)
	r.GET("/financials/imports", h.imports)
	r.POST("/financials/imports", h.importFile)
	r.POST("/financials/imports/check", h.checkFile)
	r.GET("/financials/imports/template", h.template)
}

// record answers GET /api/v1/financials/records/{recordId}: a Financials card.
func (h *Handler) record(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	view, err := h.service.Record(c.Request.Context(), a, c.Param("recordId"))
	answer(c, http.StatusOK, view, err)
}

// transactions answers GET /api/v1/financials/transactions: one page of the Financials page. Every wrong
// parameter is reported at once.
func (h *Handler) transactions(c *gin.Context) {
	q, err := ParseQuery(c.Request.URL.Query())
	if err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	page, err := h.service.Transactions(c.Request.Context(), a, q)
	answer(c, http.StatusOK, page, err)
}

// transaction answers GET /api/v1/financials/transactions/{transactionId}: the details panel.
func (h *Handler) transaction(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Transaction(c.Request.Context(), a, c.Param("transactionId"))
	answer(c, http.StatusOK, view, err)
}

// link answers POST /api/v1/financials/transactions/{transactionId}/link: a manual link.
func (h *Handler) link(c *gin.Context) {
	var req LinkRequest
	if err := web.BindJSON(c, &req); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Link(c.Request.Context(), a, c.Param("transactionId"), req)
	answer(c, http.StatusOK, view, err)
}

// unlink answers POST /api/v1/financials/transactions/{transactionId}/unlink: a manual link undone.
func (h *Handler) unlink(c *gin.Context) {
	var req UnlinkRequest
	if err := web.BindJSON(c, &req); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Unlink(c.Request.Context(), a, c.Param("transactionId"), req)
	answer(c, http.StatusOK, view, err)
}

// overview answers GET /api/v1/financials/overview: Home's Financials card.
func (h *Handler) overview(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	overview, err := h.service.Overview(c.Request.Context(), a, c.Request.URL.Query().Get("scope"))
	answer(c, http.StatusOK, overview, err)
}

// imports answers GET /api/v1/financials/imports: the import history.
func (h *Handler) imports(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	page, err := h.service.Imports(c.Request.Context(), a)
	answer(c, http.StatusOK, page, err)
}

// importFile answers POST /api/v1/financials/imports: an import of a simulated GSAP extract.
func (h *Handler) importFile(c *gin.Context) {
	var req ImportRequest
	if err := web.BindJSON(c, &req); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	result, err := h.service.Import(c.Request.Context(), a, req)
	answer(c, http.StatusCreated, result, err)
}

// checkFile answers POST /api/v1/financials/imports/check: what an import would do.
func (h *Handler) checkFile(c *gin.Context) {
	var req ImportRequest
	if err := web.BindJSON(c, &req); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	check, err := h.service.Check(c.Request.Context(), a, req)
	answer(c, http.StatusOK, check, err)
}

// template answers GET /api/v1/financials/imports/template: the sample file, as an attachment.
func (h *Handler) template(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	body, err := h.service.Template(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+TemplateName+`"`)
	c.Data(http.StatusOK, "text/csv", body)
}

// answer writes v with status, or err when there is one.
func answer(c *gin.Context, status int, v any, err error) {
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, status, v)
}
