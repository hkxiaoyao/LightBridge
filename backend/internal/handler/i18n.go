package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/WilliamWang1721/LightBridge/internal/i18n"
)

// langFromContext resolves the language for client-facing error messages from
// the request's Accept-Language header, falling back to the server default.
func langFromContext(c *gin.Context) i18n.Lang {
	if c == nil || c.Request == nil {
		return i18n.Default()
	}
	return i18n.ResolveAcceptLanguage(c.GetHeader("Accept-Language"))
}

// localizeMessage translates a canonical English error message for the request
// language. Unknown messages pass through unchanged, so it is safe to call on
// any message (including upstream-passthrough and already-localized strings).
func localizeMessage(c *gin.Context, msg string) string {
	return i18n.Translate(langFromContext(c), msg)
}

// localizef localizes a format string for the request language and then applies
// the arguments. Use it for messages that embed dynamic, non-translatable
// detail such as an underlying error or a size limit.
func localizef(c *gin.Context, format string, args ...any) string {
	return i18n.Translatef(langFromContext(c), format, args...)
}
