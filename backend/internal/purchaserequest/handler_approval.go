package purchaserequest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/approval"
	"mapp/internal/platform/web"
)

// ApprovalHandler serves the approval operations of purchase requests (032 operations 12–18).
type ApprovalHandler struct {
	service *Service
}

func NewApprovalHandler(s *Service) *ApprovalHandler {
	return &ApprovalHandler{service: s}
}

func (h *ApprovalHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET(prPath+"/approval", h.approval)
	r.GET(prPath+"/approval/submission", h.submission)
	r.POST(prPath+"/approval/submit", h.submit)
	r.POST(prPath+"/approval/approve", h.decide(approval.ActionApproved))
	r.POST(prPath+"/approval/decline", h.decide(approval.ActionDeclined))
	r.POST(prPath+"/approval/recall", h.recall)
	r.GET("/approvals/purchase-requests/:purchaseRequestId", h.preview)
}

func (h *ApprovalHandler) approval(c *gin.Context) {
	view, err := h.service.Approval(c.Request.Context(), actorOf(c), c.Param("purchaseRequestId"))
	answer(c, http.StatusOK, view, err)
}

func (h *ApprovalHandler) submission(c *gin.Context) {
	view, err := h.service.Submission(c.Request.Context(), actorOf(c), c.Param("purchaseRequestId"))
	answer(c, http.StatusOK, view, err)
}

func (h *ApprovalHandler) submit(c *gin.Context) {
	var in SubmitInput
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	view, err := h.service.Submit(c.Request.Context(), actorOf(c), c.Param("purchaseRequestId"), in)
	answer(c, http.StatusOK, view, err)
}

func (h *ApprovalHandler) decide(action approval.ActionKind) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in approval.DecisionInput
		if err := web.BindJSON(c, &in); err != nil {
			web.RespondError(c, err)
			return
		}
		view, err := h.service.Decide(c.Request.Context(), actorOf(c), c.Param("purchaseRequestId"),
			approval.SubjectDecision{Action: action, DecisionInput: in})
		answer(c, http.StatusOK, view, err)
	}
}

func (h *ApprovalHandler) recall(c *gin.Context) {
	var in approval.RecallInput
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	view, err := h.service.Recall(c.Request.Context(), actorOf(c), c.Param("purchaseRequestId"))
	answer(c, http.StatusOK, view, err)
}

func (h *ApprovalHandler) preview(c *gin.Context) {
	view, err := h.service.Preview(c.Request.Context(), actorOf(c), c.Param("purchaseRequestId"))
	answer(c, http.StatusOK, view, err)
}
