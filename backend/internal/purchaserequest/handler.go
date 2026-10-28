package purchaserequest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

const prPath = "/purchase-requests/:purchaseRequestId"

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/purchase-requests", h.list)
	r.POST("/purchase-requests", h.create)
	r.GET("/purchase-requests/new-defaults", h.defaults)
	r.GET("/purchase-requests/line-item-options", h.options)
	r.GET(prPath, h.get)
	r.PUT(prPath, h.save)
	r.DELETE(prPath, h.delete)
	r.GET(prPath+"/close-preview", h.closePreview)
	r.POST(prPath+"/close", h.close)
	r.POST(prPath+"/resend", h.resend)
	r.GET(prPath+"/routings/:routingId/form", h.form)
	r.GET("/line-items/:lineItemId/purchase-requests", h.lineItemSection)
	r.GET("/activities/:activityId/purchase-requests", h.activitySection)
	r.GET("/purchase-request-configuration", h.configuration)
	r.PUT("/purchase-request-configuration", h.saveConfiguration)
}

// answer writes what a service call returned: the value with status, or the error.
func answer[T any](c *gin.Context, status int, value T, err error) {
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, status, value)
}

func actorOf(c *gin.Context) actor.Actor {
	a, _ := actor.From(c.Request.Context())
	return a
}

func (h *Handler) list(c *gin.Context) {
	q, err := parseListQuery(c.Request.URL.Query())
	if err != nil {
		web.RespondError(c, err)
		return
	}
	page, err := h.service.List(c.Request.Context(), actorOf(c), q)
	answer(c, http.StatusOK, page, err)
}

func (h *Handler) create(c *gin.Context) {
	var in Input
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	view, err := h.service.Create(c.Request.Context(), actorOf(c), in)
	answer(c, http.StatusCreated, view, err)
}

func (h *Handler) defaults(c *gin.Context) {
	d, err := h.service.Defaults(c.Request.Context(), actorOf(c), c.Query("lineItemId"))
	answer(c, http.StatusOK, d, err)
}

func (h *Handler) options(c *gin.Context) {
	page, err := h.service.Options(c.Request.Context(), actorOf(c), OptionQuery{Search: c.Query("search"),
		PurchaseRequestID: c.Query("purchaseRequestId"), CompanyCode: c.Query("companyCode"), PlanID: c.Query("planId")})
	answer(c, http.StatusOK, page, err)
}

func (h *Handler) get(c *gin.Context) {
	view, err := h.service.Get(c.Request.Context(), actorOf(c), c.Param("purchaseRequestId"))
	answer(c, http.StatusOK, view, err)
}

func (h *Handler) save(c *gin.Context) {
	var in Input
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	view, err := h.service.Save(c.Request.Context(), actorOf(c), c.Param("purchaseRequestId"), in)
	answer(c, http.StatusOK, view, err)
}

func (h *Handler) delete(c *gin.Context) {
	entry, err := h.service.Delete(c.Request.Context(), actorOf(c), c.Param("purchaseRequestId"))
	answer(c, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()}, err)
}

func (h *Handler) closePreview(c *gin.Context) {
	preview, err := h.service.ClosePreview(c.Request.Context(), actorOf(c), c.Param("purchaseRequestId"))
	answer(c, http.StatusOK, preview, err)
}

func (h *Handler) close(c *gin.Context) {
	var in VersionInput
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	view, err := h.service.Close(c.Request.Context(), actorOf(c), c.Param("purchaseRequestId"), in)
	answer(c, http.StatusOK, view, err)
}

func (h *Handler) resend(c *gin.Context) {
	var in VersionInput
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	view, err := h.service.Resend(c.Request.Context(), actorOf(c), c.Param("purchaseRequestId"), in)
	answer(c, http.StatusOK, view, err)
}

func (h *Handler) form(c *gin.Context) {
	file, err := h.service.Form(c.Request.Context(), actorOf(c), FormRequest{PurchaseRequestID: c.Param("purchaseRequestId"),
		RoutingID: c.Param("routingId"), Format: c.Query("format")})
	if err != nil {
		web.RespondError(c, err)
		return
	}
	c.Header("Content-Disposition", contentDisposition(file.Name))
	c.Data(http.StatusOK, file.ContentType, file.Body)
}

func (h *Handler) lineItemSection(c *gin.Context) {
	section, err := h.service.LineItemSection(c.Request.Context(), actorOf(c), c.Param("lineItemId"))
	answer(c, http.StatusOK, section, err)
}

func (h *Handler) activitySection(c *gin.Context) {
	section, err := h.service.ActivitySection(c.Request.Context(), actorOf(c), c.Param("activityId"))
	answer(c, http.StatusOK, section, err)
}

func (h *Handler) configuration(c *gin.Context) {
	view, err := h.service.Configuration(c.Request.Context(), actorOf(c))
	answer(c, http.StatusOK, view, err)
}

func (h *Handler) saveConfiguration(c *gin.Context) {
	var in ConfigurationInput
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	view, err := h.service.SaveConfiguration(c.Request.Context(), actorOf(c), in)
	answer(c, http.StatusOK, view, err)
}
