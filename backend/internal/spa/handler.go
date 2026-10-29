package spa

import (
	"net/http"
	"net/url"
	"regexp"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

const (
	maxDimensionLength = 80
	msgChooseCampaign  = "Choose a campaign of this plan."
)

var campaignIDForm = regexp.MustCompile(`^C-[0-9]+$`)

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/plans/:planId/spa", h.view)
}

// view answers GET /api/v1/plans/{planId}/spa (contract getSpaView).
func (h *Handler) view(c *gin.Context) {
	q, err := parseQuery(c.Request.URL.Query())
	if err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.View(c.Request.Context(), a, c.Param("planId"), q)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}

func parseQuery(values url.Values) (Query, error) {
	q := Query{CampaignID: values.Get("campaignId"), Dimension: values.Get("dimension")}
	var problems errs.Problems
	if q.CampaignID != "" && !campaignIDForm.MatchString(q.CampaignID) {
		problems.Add("campaignId", msgChooseCampaign)
	}
	if len(q.Dimension) > maxDimensionLength {
		problems.Add(fieldDimension, msgChooseListed)
	}
	return q, problems.Err()
}
