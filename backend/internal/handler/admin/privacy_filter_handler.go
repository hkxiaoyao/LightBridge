package admin

import (
	"github.com/Wei-Shaw/LightBridge/internal/pkg/response"
	"github.com/Wei-Shaw/LightBridge/internal/service"
	"github.com/gin-gonic/gin"
)

// PrivacyFilterHandler 管理隐私过滤配置。
type PrivacyFilterHandler struct {
	service *service.PrivacyFilterService
}

func NewPrivacyFilterHandler(svc *service.PrivacyFilterService) *PrivacyFilterHandler {
	return &PrivacyFilterHandler{service: svc}
}

type privacyFilterConfigRequest struct {
	Enabled        *bool                             `json:"enabled"`
	FilterRequest  *bool                             `json:"filter_request"`
	FilterResponse *bool                             `json:"filter_response"`
	BuiltinRules   *map[string]bool                  `json:"builtin_rules"`
	CustomRules    *[]service.PrivacyFilterRule      `json:"custom_rules"`
	AllGroups      *bool                             `json:"all_groups"`
	GroupIDs       *[]int64                          `json:"group_ids"`
	ModelFilter    *service.PrivacyFilterModelFilter `json:"model_filter"`
}

func (h *PrivacyFilterHandler) GetConfig(c *gin.Context) {
	cfg, err := h.service.GetConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

func (h *PrivacyFilterHandler) UpdateConfig(c *gin.Context) {
	var req privacyFilterConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	cfg, err := h.service.UpdateConfig(c.Request.Context(), service.UpdatePrivacyFilterConfigInput{
		Enabled:        req.Enabled,
		FilterRequest:  req.FilterRequest,
		FilterResponse: req.FilterResponse,
		BuiltinRules:   req.BuiltinRules,
		CustomRules:    req.CustomRules,
		AllGroups:      req.AllGroups,
		GroupIDs:       req.GroupIDs,
		ModelFilter:    req.ModelFilter,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}
