package proxy

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

type serviceRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn serviceRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestServiceListNodesDoesNotReturnSecrets(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT id, name, node_type, source_type, config_json, status").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "node_type", "source_type", "config_json", "status"}).
			AddRow(int64(1), "Proxy", "http", "manual", []byte(`{"server":"proxy.example.com","port":8080}`), "active"))

	nodes, err := NewService(db).ListNodes(context.Background())
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "proxy.example.com", nodes[0].Config["server"])
	require.NotContains(t, nodes[0].Config, "password")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServiceCreateManualNodeStoresSecretSeparately(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("INSERT INTO proxy_nodes").
		WithArgs("Proxy", "http", "manual", `{"port":8080,"server":"proxy.example.com"}`, `{"password":"pass","username":"user"}`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))
	mock.ExpectQuery("SELECT id, name, node_type, source_type, config_json, status").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "node_type", "source_type", "config_json", "status"}).
			AddRow(int64(10), "Proxy", "http", "manual", []byte(`{"server":"proxy.example.com","port":8080}`), "active"))

	node, err := NewService(db).CreateManualNode(context.Background(), CreateNodeInput{
		Name: "Proxy",
		URL:  "http://user:pass@proxy.example.com:8080",
	})
	require.NoError(t, err)
	require.Equal(t, int64(10), node.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServiceCreateBinding(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("INSERT INTO proxy_bindings").
		WithArgs("global", "default", int64(10), 0, false).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(20)))
	mock.ExpectQuery("SELECT id, entity_type, entity_id, profile_id, priority, enabled, fallback_to_direct").
		WithArgs(int64(20)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "entity_type", "entity_id", "profile_id", "priority", "enabled", "fallback_to_direct"}).
			AddRow(int64(20), "global", "default", int64(10), 0, true, false))

	binding, err := NewService(db).CreateBinding(context.Background(), CreateBindingInput{
		EntityType: "global",
		EntityID:   "default",
		ProfileID:  10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(20), binding.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLegacyProxyURLNormalizesSocks5H(t *testing.T) {
	require.Equal(t, "socks5://user:pass@127.0.0.1:1080", legacyProxyURL(legacyProxyRow{
		Protocol: "socks5h",
		Host:     "127.0.0.1",
		Port:     1080,
		Username: "user",
		Password: "pass",
	}))
}

func TestServiceMigrateLegacyProxies(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, name, protocol, host, port").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "protocol", "host", "port", "username", "password"}).
			AddRow(int64(7), "Legacy", "http", "proxy.example.com", 8080, "user", "pass"))
	mock.ExpectQuery("SELECT id FROM proxy_nodes").
		WithArgs(int64(7)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("INSERT INTO proxy_nodes").
		WithArgs("Legacy", "http", int64(7), `{"port":8080,"server":"proxy.example.com"}`, `{"password":"pass","username":"user"}`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(70)))
	mock.ExpectQuery("SELECT id FROM proxy_profiles").
		WithArgs("7").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("INSERT INTO proxy_profiles").
		WithArgs("Migrated Proxy 7", `{"legacy_proxy_id":7}`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(700)))
	mock.ExpectExec("INSERT INTO proxy_profile_nodes").
		WithArgs(int64(700), int64(70)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT id, proxy_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "proxy_id"}).
			AddRow(int64(42), int64(7)))
	mock.ExpectQuery("SELECT id FROM proxy_bindings").
		WithArgs("42").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("INSERT INTO proxy_bindings").
		WithArgs("42", int64(700)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(900)))
	mock.ExpectCommit()

	report, err := NewService(db).MigrateLegacyProxies(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, report.ProxiesScanned)
	require.Equal(t, 1, report.ProxiesMigrated)
	require.Equal(t, 1, report.AccountsScanned)
	require.Equal(t, 1, report.BindingsMigrated)
	require.Empty(t, report.Warnings)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServiceMigrateLegacyProxiesIsIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, name, protocol, host, port").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "protocol", "host", "port", "username", "password"}).
			AddRow(int64(7), "Legacy", "http", "proxy.example.com", 8080, "", ""))
	mock.ExpectQuery("SELECT id FROM proxy_nodes").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(70)))
	mock.ExpectQuery("SELECT id FROM proxy_profiles").
		WithArgs("7").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(700)))
	mock.ExpectExec("INSERT INTO proxy_profile_nodes").
		WithArgs(int64(700), int64(70)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT id, proxy_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "proxy_id"}).
			AddRow(int64(42), int64(7)))
	mock.ExpectQuery("SELECT id FROM proxy_bindings").
		WithArgs("42").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(900)))
	mock.ExpectCommit()

	report, err := NewService(db).MigrateLegacyProxies(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, report.ProxiesScanned)
	require.Equal(t, 0, report.ProxiesMigrated)
	require.Equal(t, 1, report.AccountsScanned)
	require.Equal(t, 0, report.BindingsMigrated)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServiceUpdateProfileReplacesNodes(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE proxy_profiles").
		WithArgs(int64(10), "Updated", "url_test", "https://example.com/health", 60).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM proxy_profile_nodes").
		WithArgs(int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("INSERT INTO proxy_profile_nodes").
		WithArgs(int64(10), int64(100), 0, 2).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO proxy_profile_nodes").
		WithArgs(int64(10), int64(101), 1, 1).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT id, name, strategy, test_url, interval_seconds, status, config_json, runtime_json").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "strategy", "test_url", "interval_seconds", "status", "config_json", "runtime_json"}).
			AddRow(int64(10), "Updated", "url_test", "https://example.com/health", 60, "active", []byte(`{}`), []byte(`{}`)))

	profile, err := NewService(db).UpdateProfile(context.Background(), 10, UpdateProfileInput{
		Name:            "Updated",
		Strategy:        "url_test",
		TestURL:         "https://example.com/health",
		IntervalSeconds: 60,
		NodeIDs:         []int64{100, 101},
		Weights:         []int{2},
	})
	require.NoError(t, err)
	require.Equal(t, "Updated", profile.Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServiceGetRuntimeReturnsStoppedWhenMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT id, profile_id, runtime_type, COALESCE").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "profile_id", "runtime_type", "pid", "mixed_port", "controller_port", "controller_secret_ref", "config_path", "work_dir", "status", "last_error"}))

	runtime, err := NewService(db).GetRuntime(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, "stopped", runtime.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServiceGetRuntimeStatusAggregatesInstances(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT status, COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).
			AddRow("running", 2).
			AddRow("failed", 1).
			AddRow("stopped", 3))

	status, err := NewService(db).GetRuntimeStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, 6, status.Total)
	require.Equal(t, 2, status.Running)
	require.Equal(t, 1, status.Failed)
	require.Equal(t, 3, status.Stopped)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServiceTestProfileReportsStoppedRuntime(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT id, profile_id, runtime_type, COALESCE").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "profile_id", "runtime_type", "pid", "mixed_port", "controller_port", "controller_secret_ref", "config_path", "work_dir", "status", "last_error"}))

	result, err := NewService(db).TestProfile(context.Background(), 10)
	require.NoError(t, err)
	require.False(t, result.Healthy)
	require.Equal(t, "stopped", result.Status)
	require.Equal(t, "proxy runtime is not running", result.Error)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServiceTestProfileChecksControllerWithStoredSecret(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	rows := sqlmock.NewRows([]string{"id", "profile_id", "runtime_type", "pid", "mixed_port", "controller_port", "controller_secret_ref", "config_path", "work_dir", "status", "last_error"}).
		AddRow(int64(1), int64(10), "mihomo", 1234, 17001, 18001, "stored-secret", "/tmp/config.yaml", "/tmp", "running", "")
	mock.ExpectQuery("SELECT id, profile_id, runtime_type, COALESCE").
		WithArgs(int64(10)).
		WillReturnRows(rows)
	rows2 := sqlmock.NewRows([]string{"id", "profile_id", "runtime_type", "pid", "mixed_port", "controller_port", "controller_secret_ref", "config_path", "work_dir", "status", "last_error"}).
		AddRow(int64(1), int64(10), "mihomo", 1234, 17001, 18001, "stored-secret", "/tmp/config.yaml", "/tmp", "running", "")
	mock.ExpectQuery("SELECT id, profile_id, runtime_type, COALESCE").
		WithArgs(int64(10)).
		WillReturnRows(rows2)

	service := NewService(db)
	service.SetControllerHTTPClient(&http.Client{Transport: serviceRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "127.0.0.1:18001", req.URL.Host)
		require.Equal(t, "Bearer stored-secret", req.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(`{"version":"v1.2.3"}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})})

	result, err := service.TestProfile(context.Background(), 10)
	require.NoError(t, err)
	require.True(t, result.Healthy)
	require.Equal(t, "v1.2.3", result.Version)
	require.Equal(t, "http://127.0.0.1:17001", result.ProxyURL)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServiceStartProfileRequiresConfiguredMihomoBinary(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT id, name, strategy, test_url, interval_seconds, status, config_json, runtime_json").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "strategy", "test_url", "interval_seconds", "status", "config_json", "runtime_json"}).
			AddRow(int64(10), "Default", "select", "https://www.gstatic.com/generate_204", 300, "active", []byte(`{}`), []byte(`{}`)))
	mock.ExpectQuery("SELECT id, profile_id, runtime_type, COALESCE").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "profile_id", "runtime_type", "pid", "mixed_port", "controller_port", "controller_secret_ref", "config_path", "work_dir", "status", "last_error"}))

	_, err = NewServiceWithRuntime(db, "", "").StartProfile(context.Background(), 10)
	require.ErrorContains(t, err, "mihomo binary path is required")
	require.NoError(t, mock.ExpectationsWereMet())
}
