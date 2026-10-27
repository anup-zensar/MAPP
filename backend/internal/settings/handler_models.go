package settings

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// listModels answers GET /api/v1/models.
func (h *Handler) listModels(c *gin.Context) {
	models, err := h.service.ListModels(c.Request.Context(), actorOf(c))
	respond(c, http.StatusOK, ModelPage{Items: models, Total: len(models)}, err)
}

// createModel answers POST /api/v1/models with 201 and the new model.
func (h *Handler) createModel(c *gin.Context) {
	var input ModelInput
	if !readBody(c, &input) {
		return
	}
	model, err := h.service.CreateModel(c.Request.Context(), actorOf(c), input)
	respond(c, http.StatusCreated, model, err)
}

// getModel answers GET /api/v1/models/{modelId}.
func (h *Handler) getModel(c *gin.Context) {
	model, err := h.service.GetModel(c.Request.Context(), actorOf(c), c.Param("modelId"))
	respond(c, http.StatusOK, model, err)
}

// updateModel answers PATCH /api/v1/models/{modelId} with the model after the change.
func (h *Handler) updateModel(c *gin.Context) {
	var update ModelUpdate
	if !readBody(c, &update) {
		return
	}
	model, err := h.service.UpdateModel(c.Request.Context(), actorOf(c), c.Param("modelId"), update)
	respond(c, http.StatusOK, model, err)
}

// deleteModel answers DELETE /api/v1/models/{modelId} with the recycle-bin entry, so the web app can offer
// Undo.
func (h *Handler) deleteModel(c *gin.Context) {
	entry, err := h.service.DeleteModel(c.Request.Context(), actorOf(c), c.Param("modelId"))
	respondDeleted(c, entry, err)
}

// modelImpact answers POST /api/v1/models/{modelId}/impact with the warnings to confirm before saving.
func (h *Handler) modelImpact(c *gin.Context) {
	var update ModelUpdate
	if !readBody(c, &update) {
		return
	}
	impact, err := h.service.ModelImpact(c.Request.Context(), actorOf(c), c.Param("modelId"), update)
	respond(c, http.StatusOK, impact, err)
}

// setModelStatus answers POST /api/v1/models/{modelId}/status with the model in its new status.
func (h *Handler) setModelStatus(c *gin.Context) {
	var update StatusUpdate
	if !readBody(c, &update) {
		return
	}
	model, err := h.service.SetModelActive(c.Request.Context(), actorOf(c), c.Param("modelId"), update)
	respond(c, http.StatusOK, model, err)
}
