package financials

import (
	"net/http"

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

// Routes registers the ten operations of feature 015 on r, each behind signedIn.
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/financials/records/{recordId}", signedIn(h.record))
	r.Handle("GET /api/financials/transactions", signedIn(h.transactions))
	r.Handle("GET /api/financials/transactions/{transactionId}", signedIn(h.transaction))
	r.Handle("POST /api/financials/transactions/{transactionId}/link", signedIn(h.link))
	r.Handle("POST /api/financials/transactions/{transactionId}/unlink", signedIn(h.unlink))
	r.Handle("GET /api/financials/overview", signedIn(h.overview))
	r.Handle("GET /api/financials/imports", signedIn(h.imports))
	r.Handle("POST /api/financials/imports", signedIn(h.importFile))
	r.Handle("POST /api/financials/imports/check", signedIn(h.checkFile))
	r.Handle("GET /api/financials/imports/template", signedIn(h.template))
}

// record answers GET /api/financials/records/{recordId}: a Financials card.
func (h *Handler) record(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	view, err := h.service.Record(r.Context(), a, r.PathValue("recordId"))
	answer(w, r, http.StatusOK, view, err)
}

// transactions answers GET /api/financials/transactions: one page of the Financials page. Every wrong
// parameter is reported at once.
func (h *Handler) transactions(w http.ResponseWriter, r *http.Request) {
	q, err := ParseQuery(r.URL.Query())
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	page, err := h.service.Transactions(r.Context(), a, q)
	answer(w, r, http.StatusOK, page, err)
}

// transaction answers GET /api/financials/transactions/{transactionId}: the details panel.
func (h *Handler) transaction(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	view, err := h.service.Transaction(r.Context(), a, r.PathValue("transactionId"))
	answer(w, r, http.StatusOK, view, err)
}

// link answers POST /api/financials/transactions/{transactionId}/link: a manual link.
func (h *Handler) link(w http.ResponseWriter, r *http.Request) {
	var req LinkRequest
	if err := web.ReadJSON(w, r, &req); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	view, err := h.service.Link(r.Context(), a, r.PathValue("transactionId"), req)
	answer(w, r, http.StatusOK, view, err)
}

// unlink answers POST /api/financials/transactions/{transactionId}/unlink: a manual link undone.
func (h *Handler) unlink(w http.ResponseWriter, r *http.Request) {
	var req UnlinkRequest
	if err := web.ReadJSON(w, r, &req); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	view, err := h.service.Unlink(r.Context(), a, r.PathValue("transactionId"), req)
	answer(w, r, http.StatusOK, view, err)
}

// overview answers GET /api/financials/overview: Home's Financials card.
func (h *Handler) overview(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	overview, err := h.service.Overview(r.Context(), a, r.URL.Query().Get("scope"))
	answer(w, r, http.StatusOK, overview, err)
}

// imports answers GET /api/financials/imports: the import history.
func (h *Handler) imports(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	page, err := h.service.Imports(r.Context(), a)
	answer(w, r, http.StatusOK, page, err)
}

// importFile answers POST /api/financials/imports: an import of a simulated GSAP extract.
func (h *Handler) importFile(w http.ResponseWriter, r *http.Request) {
	var req ImportRequest
	if err := web.ReadJSON(w, r, &req); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	result, err := h.service.Import(r.Context(), a, req)
	answer(w, r, http.StatusCreated, result, err)
}

// checkFile answers POST /api/financials/imports/check: what an import would do.
func (h *Handler) checkFile(w http.ResponseWriter, r *http.Request) {
	var req ImportRequest
	if err := web.ReadJSON(w, r, &req); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context())
	check, err := h.service.Check(r.Context(), a, req)
	answer(w, r, http.StatusOK, check, err)
}

// template answers GET /api/financials/imports/template: the sample file, as an attachment.
func (h *Handler) template(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context())
	body, err := h.service.Template(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="`+TemplateName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body) // a failed write means the client has gone; there is nobody left to tell
}

// answer writes v with status, or err when there is one.
func answer(w http.ResponseWriter, r *http.Request, status int, v any, err error) {
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, status, v)
}
