package settings

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

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
