package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func extractMaxBytesError(err error) (*http.MaxBytesError, bool) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return maxErr, true
	}
	return nil, false
}

func formatBodyLimit(limit int64) string {
	const mb = 1024 * 1024
	if limit >= mb {
		return fmt.Sprintf("%dMB", limit/mb)
	}
	return fmt.Sprintf("%dB", limit)
}

func buildBodyTooLargeMessage(c *gin.Context, limit int64) string {
	return localizef(c, "Request body too large, limit is %s", formatBodyLimit(limit))
}
