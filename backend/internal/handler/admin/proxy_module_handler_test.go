package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	proxymodule "github.com/WilliamWang1721/LightBridge/internal/modules/proxy"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestProxyModuleHandlerListNodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectQuery("SELECT id, name, node_type, source_type, config_json, status").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "node_type", "source_type", "config_json", "status"}).
			AddRow(int64(1), "Proxy", "http", "manual", []byte(`{"server":"proxy.example.com","port":8080}`), "active"))

	h := NewProxyModuleHandler(proxymodule.NewService(db))
	r := gin.New()
	r.GET("/api/v1/admin/proxy/nodes", h.ListNodes)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxy/nodes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"nodes"`)
	require.NotContains(t, w.Body.String(), "password")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProxyModuleHandlerGetProfileRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectQuery("SELECT id, profile_id, runtime_type, COALESCE").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "profile_id", "runtime_type", "pid", "mixed_port", "controller_port", "controller_secret_ref", "config_path", "work_dir", "status", "last_error"}))

	h := NewProxyModuleHandler(proxymodule.NewService(db))
	r := gin.New()
	r.GET("/api/v1/admin/proxy/profiles/:id/runtime", h.GetProfileRuntime)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxy/profiles/10/runtime", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"status":"stopped"`)
	require.NoError(t, mock.ExpectationsWereMet())
}
