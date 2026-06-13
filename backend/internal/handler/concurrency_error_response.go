package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

const statusClientClosedRequest = 499

func concurrencyErrorResponse(c *gin.Context, err error, slotType string) (int, string, string) {
	var concurrencyErr *ConcurrencyError
	if errors.As(err, &concurrencyErr) {
		if concurrencyErr.SlotType != "" {
			slotType = concurrencyErr.SlotType
		}
		return http.StatusTooManyRequests, "rate_limit_error",
			localizef(c, "Concurrency limit exceeded for %s, please retry later", slotType)
	}

	if errors.Is(err, context.Canceled) {
		return statusClientClosedRequest, "api_error", "context canceled"
	}

	return http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable, please retry later"
}
