package simulatedgsap

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// OrderRequest is the body of the run: the purchase request's version the person saw (contract SimulatedGsapOrderRequest).
type OrderRequest struct {
	Version int `json:"version"`
}

type PurchaseOrderHandler struct {
	prs PurchaseRequests
}

func NewPurchaseOrderHandler(prs PurchaseRequests) *PurchaseOrderHandler {
	return &PurchaseOrderHandler{prs: prs}
}

func (h *PurchaseOrderHandler) RegisterRoutes(r gin.IRoutes) {
	r.POST("/simulated-gsap/purchase-requests/:purchaseRequestId/purchase-order", h.create)
}

func (h *PurchaseOrderHandler) create(c *gin.Context) {
	var in OrderRequest
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.prs.CreateGSAPOrder(c.Request.Context(), a, c.Param("purchaseRequestId"), in.Version)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}
