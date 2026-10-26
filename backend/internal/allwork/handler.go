package allwork

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

// Handler answers the HTTP requests of All work (plan "API operations").
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the All work routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// RegisterRoutes registers the All work routes on r.
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/all-work", h.query)
	r.GET("/all-work/options", h.options)
	r.GET("/all-work/export", h.export)
	r.GET("/all-work/records/:recordId", h.locate)
	r.GET("/all-work/preferences", h.preferences)
	r.PUT("/all-work/preferences", h.updatePreferences)
	r.GET("/saved-views", h.listViews)
	r.POST("/saved-views", h.createView)
	r.PATCH("/saved-views/:viewId", h.updateView)
	r.DELETE("/saved-views/:viewId", h.deleteView)
	r.GET("/plans/:planId/budget-by-month", h.budgetByMonth) // 016 FR-024
	r.GET("/all-work/breakdown", h.breakdown)                // 016 FR-040
}

// breakdown answers GET /api/v1/all-work/breakdown: the Breakdown of the filter set of the query, by the
// grouping by and then by thenBy (016 FR-040 - FR-042). Every wrong parameter is reported at once.
func (h *Handler) breakdown(c *gin.Context) {
	q, err := ParseQuery(c.Request.URL.Query())
	if err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	result, err := h.service.Breakdown(c.Request.Context(), a, q)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, result)
}

// budgetByMonth answers GET /api/v1/plans/{planId}/budget-by-month: the phased budget against the planned cost
// of the plan and each of its campaigns and initiatives, in the plan currency (016 FR-024).
func (h *Handler) budgetByMonth(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	result, err := h.service.BudgetByMonth(c.Request.Context(), a, c.Param("planId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, result)
}

// listViews answers GET /api/v1/saved-views: the signed-in person's saved views.
func (h *Handler) listViews(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	page, err := h.service.ListViews(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, page)
}

// createView answers POST /api/v1/saved-views with 201 and the new view.
func (h *Handler) createView(c *gin.Context) {
	var in SavedViewCreate
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.CreateView(c.Request.Context(), a, in)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, view)
}

// updateView answers PATCH /api/v1/saved-views/{viewId} with the view after the change.
func (h *Handler) updateView(c *gin.Context) {
	var in SavedViewUpdate
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.UpdateView(c.Request.Context(), a, c.Param("viewId"), in)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}

// deleteView answers DELETE /api/v1/saved-views/{viewId} with the deleted view, so the web app can offer Undo.
func (h *Handler) deleteView(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.DeleteView(c.Request.Context(), a, c.Param("viewId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}

// preferences answers GET /api/v1/all-work/preferences: the signed-in person's remembered columns and sort.
func (h *Handler) preferences(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	p, err := h.service.Preferences(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, p)
}

// updatePreferences answers PUT /api/v1/all-work/preferences with the preferences after the change.
func (h *Handler) updatePreferences(c *gin.Context) {
	var p Preferences
	if err := web.BindJSON(c, &p); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	saved, err := h.service.UpdatePreferences(c.Request.Context(), a, p)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, saved)
}

// options answers GET /api/v1/all-work/options: the choices of the filter bar for the year of the query (the
// current year without one) and the column catalogue.
func (h *Handler) options(c *gin.Context) {
	var problems errs.Problems
	from, to := parseRange(c.Request.URL.Query(), &problems) // 016 FR-034: year, or yearFrom and yearTo
	if err := problems.Err(); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	options, err := h.service.Options(c.Request.Context(), a, from, to)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, options)
}

// query answers GET /api/v1/all-work: everything one view shows for the filter set of the query. Every wrong
// parameter is reported at once.
func (h *Handler) query(c *gin.Context) {
	q, err := ParseQuery(c.Request.URL.Query())
	if err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	result, err := h.service.Query(c.Request.Context(), a, q)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, result)
}

// export answers GET /api/v1/all-work/export with the view of the query as an .xlsx or CSV file, named in
// Content-Disposition (FR-032 – FR-034). Every wrong parameter is reported at once.
func (h *Handler) export(c *gin.Context) {
	q, format, unassigned, err := ParseExport(c.Request.URL.Query())
	if err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	file, err := h.service.Export(c.Request.Context(), a, q, format, unassigned)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	c.Header("Content-Disposition", ContentDisposition(file.Name))
	c.Data(http.StatusOK, file.ContentType, file.Body)
}

// locate answers GET /api/v1/all-work/records/{recordId}: the record a row opens, or 404 with what became of it.
func (h *Handler) locate(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	location, err := h.service.Locate(c.Request.Context(), a, c.Param("recordId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, location)
}
