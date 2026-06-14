package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/LightBridge/internal/service"
	"github.com/gin-gonic/gin"
)

// fakePrivacySettingRepo 实现 service.SettingRepository，仅用于启用响应脱敏。
type fakePrivacySettingRepo struct {
	values map[string]string
}

func (f *fakePrivacySettingRepo) Get(ctx context.Context, key string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}
func (f *fakePrivacySettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	if v, ok := f.values[key]; ok {
		return v, nil
	}
	return "", service.ErrSettingNotFound
}
func (f *fakePrivacySettingRepo) Set(ctx context.Context, key, value string) error { return nil }
func (f *fakePrivacySettingRepo) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (f *fakePrivacySettingRepo) SetMultiple(ctx context.Context, settings map[string]string) error {
	return nil
}
func (f *fakePrivacySettingRepo) GetAll(ctx context.Context) (map[string]string, error) {
	return f.values, nil
}
func (f *fakePrivacySettingRepo) Delete(ctx context.Context, key string) error { return nil }

func newEnabledPrivacyService() *service.PrivacyFilterService {
	repo := &fakePrivacySettingRepo{values: map[string]string{
		service.SettingKeyPrivacyFilterEnabled: "true",
		service.SettingKeyPrivacyFilterConfig:  `{"enabled":true,"filter_request":true,"filter_response":true,"all_groups":true}`,
	}}
	return service.NewPrivacyFilterService(repo, nil)
}

func TestPrivacyFilterResponseWriter_RedactsSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newEnabledPrivacyService()

	rec := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(rec)
	engine.Use(PrivacyFilterResponseWriter(svc))
	engine.GET("/x", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Writer.WriteHeader(http.StatusOK)
		// 把手机号人为切在两个写入之间，验证跨 chunk 缓冲。
		_, _ = c.Writer.WriteString("data: {\"text\":\"call 138001")
		_, _ = c.Writer.WriteString("38000 now\"}\n\n")
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Request = req
	engine.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "13800138000") {
		t.Fatalf("phone number leaked across chunks: %q", body)
	}
	if !strings.Contains(body, "[PHONE]") {
		t.Fatalf("expected redacted phone placeholder, got: %q", body)
	}
}

func TestPrivacyFilterResponseWriter_PassthroughBinary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newEnabledPrivacyService()

	rec := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(rec)
	engine.Use(PrivacyFilterResponseWriter(svc))
	engine.GET("/bin", func(c *gin.Context) {
		c.Header("Content-Type", "application/octet-stream")
		c.Writer.WriteHeader(http.StatusOK)
		_, _ = c.Writer.WriteString("raw 13800138000 bytes")
	})

	req := httptest.NewRequest(http.MethodGet, "/bin", nil)
	engine.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "13800138000") {
		t.Fatalf("binary response should pass through unchanged: %q", rec.Body.String())
	}
}
