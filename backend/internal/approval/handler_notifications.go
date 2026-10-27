package approval

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// listNotifications answers GET /api/v1/notifications: the signed-in person's 50 most recent notifications
// and the unread count.
func (h *Handler) listNotifications(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	page, err := h.service.ListNotifications(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, page)
}

// markRead answers POST /api/v1/notifications/{notificationId}/read with the notification, now read.
func (h *Handler) markRead(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	notification, err := h.service.MarkRead(c.Request.Context(), a, c.Param("notificationId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, notification)
}

// markAllRead answers POST /api/v1/notifications/read-all with how many notifications it marked read.
func (h *Handler) markAllRead(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	result, err := h.service.MarkAllRead(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, result)
}
