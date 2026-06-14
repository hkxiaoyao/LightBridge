package middleware

import (
	"bytes"
	"strings"

	"github.com/Wei-Shaw/LightBridge/internal/service"
	"github.com/gin-gonic/gin"
)

// PrivacyFilterResponseWriter 返回一个网关中间件：当响应侧脱敏开启且分组命中时，
// 用脱敏 ResponseWriter 包装 c.Writer，在字节流写回客户端前对文本做正则脱敏。
//
// 该方案不触碰任何上游转发函数，对所有协议、流式与非流式响应统一生效。脱敏作用
// 在序列化后的字节流上：采用按行缓冲（SSE 行均以 \n 结尾），仅在完整行上做替换，
// 从而天然处理被 TCP 分片切断的内容；剩余不完整片段在请求结束时统一刷出。
func PrivacyFilterResponseWriter(svc *service.PrivacyFilterService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil || c.Request == nil {
			c.Next()
			return
		}
		if !svc.ResponseFilterEnabled(c.Request.Context()) {
			c.Next()
			return
		}
		var groupID *int64
		if apiKey, ok := GetAPIKeyFromContext(c); ok && apiKey != nil && apiKey.GroupID != nil {
			id := *apiKey.GroupID
			groupID = &id
		}
		redactor := svc.ResponseRedactor(c.Request.Context(), groupID)
		if redactor == nil || !redactor.HasRules() {
			c.Next()
			return
		}
		w := &privacyRedactWriter{ResponseWriter: c.Writer, redactor: redactor}
		c.Writer = w
		c.Next()
		w.flushRemainder()
	}
}

// privacyRedactWriter 包装 gin.ResponseWriter，对输出字节流按行脱敏。
type privacyRedactWriter struct {
	gin.ResponseWriter
	redactor *service.PrivacyRedactor

	decided     bool
	passthrough bool
	pending     []byte
}

// shouldRedactContentType 仅对文本类响应脱敏，二进制（图片等）直通。
func shouldRedactContentType(ct string) bool {
	ct = strings.ToLower(ct)
	return strings.Contains(ct, "text/event-stream") ||
		strings.Contains(ct, "application/json") ||
		strings.HasPrefix(ct, "text/")
}

func (w *privacyRedactWriter) ensureDecided() {
	if w.decided {
		return
	}
	w.decided = true
	w.passthrough = !shouldRedactContentType(w.Header().Get("Content-Type"))
}

func (w *privacyRedactWriter) Write(p []byte) (int, error) {
	w.ensureDecided()
	if w.passthrough {
		return w.ResponseWriter.Write(p)
	}
	w.pending = append(w.pending, p...)
	if err := w.emitCompleteLines(); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *privacyRedactWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// emitCompleteLines 把 pending 中所有完整行（含末尾 \n）脱敏后写出，保留不完整尾部。
func (w *privacyRedactWriter) emitCompleteLines() error {
	idx := bytes.LastIndexByte(w.pending, '\n')
	if idx < 0 {
		return nil
	}
	complete := w.pending[:idx+1]
	redacted, _ := w.redactor.Redact(string(complete))
	if _, err := w.ResponseWriter.WriteString(redacted); err != nil {
		return err
	}
	// 保留尾部不完整片段。
	rest := w.pending[idx+1:]
	w.pending = append([]byte(nil), rest...)
	return nil
}

// flushRemainder 在请求结束时刷出剩余不完整片段（脱敏后）。
func (w *privacyRedactWriter) flushRemainder() {
	if w.passthrough || len(w.pending) == 0 {
		return
	}
	redacted, _ := w.redactor.Redact(string(w.pending))
	_, _ = w.ResponseWriter.WriteString(redacted)
	w.pending = nil
}

// Flush 透传底层 flush（SSE 需要），完整行已在 Write 中刷出。
func (w *privacyRedactWriter) Flush() {
	w.ResponseWriter.Flush()
}
