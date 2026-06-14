package handler

import (
	"github.com/Wei-Shaw/LightBridge/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// applyPrivacyFilter 在请求体转发上游前做隐私脱敏；未启用/未命中作用域时原样返回。
func (h *GatewayHandler) applyPrivacyFilter(c *gin.Context, reqLog *zap.Logger, apiKey *service.APIKey, protocol string, model string, body []byte) []byte {
	if h == nil || h.privacyFilterService == nil {
		return body
	}
	return runPrivacyRequestFilter(c, reqLog, h.privacyFilterService, apiKey, protocol, model, body)
}

// applyPrivacyFilter 同上，OpenAI 网关 handler 版本。
func (h *OpenAIGatewayHandler) applyPrivacyFilter(c *gin.Context, reqLog *zap.Logger, apiKey *service.APIKey, protocol string, model string, body []byte) []byte {
	if h == nil || h.privacyFilterService == nil {
		return body
	}
	return runPrivacyRequestFilter(c, reqLog, h.privacyFilterService, apiKey, protocol, model, body)
}

func runPrivacyRequestFilter(c *gin.Context, reqLog *zap.Logger, svc *service.PrivacyFilterService, apiKey *service.APIKey, protocol string, model string, body []byte) []byte {
	if svc == nil || len(body) == 0 || c == nil || c.Request == nil {
		return body
	}
	var groupID *int64
	if apiKey != nil && apiKey.GroupID != nil {
		id := *apiKey.GroupID
		groupID = &id
	}
	redactor := svc.RequestRedactor(c.Request.Context(), groupID, model)
	if redactor == nil {
		return body
	}
	filtered := svc.RedactRequestBody(protocol, body, redactor)
	if reqLog != nil && len(filtered) != len(body) {
		reqLog.Info("privacy_filter.request_redacted",
			zap.String("protocol", protocol),
			zap.String("model", model),
			zap.Int("orig_bytes", len(body)),
			zap.Int("filtered_bytes", len(filtered)),
		)
	}
	return filtered
}
