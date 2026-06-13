package binding

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/LightBridge/internal/modules/proxy/internal/profile"
	"github.com/stretchr/testify/require"
)

func newMockStore(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *SQLStore) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock, NewSQLStore(db)
}

func TestSQLStoreFindEnabledBinding(t *testing.T) {
	_, mock, store := newMockStore(t)
	mock.ExpectQuery("SELECT id, entity_type, entity_id, profile_id, priority, enabled, fallback_to_direct").
		WithArgs("account", "10").
		WillReturnRows(sqlmock.NewRows([]string{"id", "entity_type", "entity_id", "profile_id", "priority", "enabled", "fallback_to_direct"}).
			AddRow(int64(1), "account", "10", int64(100), 0, true, false))

	binding, err := store.FindEnabledBinding(context.Background(), Query{EntityType: EntityAccount, EntityID: "10"})
	require.NoError(t, err)
	require.Equal(t, int64(100), binding.ProfileID)
	require.Equal(t, EntityAccount, binding.EntityType)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLStoreFindEnabledBindingFallsBackToMigratedLegacyAccountProxy(t *testing.T) {
	_, mock, store := newMockStore(t)
	mock.ExpectQuery("SELECT id, entity_type, entity_id, profile_id, priority, enabled, fallback_to_direct").
		WithArgs("account", "10").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT pp.id").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(100)))

	binding, err := store.FindEnabledBinding(context.Background(), Query{EntityType: EntityAccount, EntityID: "10"})
	require.NoError(t, err)
	require.Equal(t, int64(100), binding.ProfileID)
	require.Equal(t, EntityAccount, binding.EntityType)
	require.Equal(t, "10", binding.EntityID)
	require.False(t, binding.FallbackToDirect)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLStoreFindEnabledBindingDoesNotFallbackForProvider(t *testing.T) {
	_, mock, store := newMockStore(t)
	mock.ExpectQuery("SELECT id, entity_type, entity_id, profile_id, priority, enabled, fallback_to_direct").
		WithArgs("provider", "openai").
		WillReturnError(sql.ErrNoRows)

	binding, err := store.FindEnabledBinding(context.Background(), Query{EntityType: EntityProvider, EntityID: "openai"})
	require.NoError(t, err)
	require.Nil(t, binding)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLStoreGetProfile(t *testing.T) {
	_, mock, store := newMockStore(t)
	mock.ExpectQuery("SELECT id, name, strategy, test_url, interval_seconds, status, config_json, runtime_json").
		WithArgs(int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "strategy", "test_url", "interval_seconds", "status", "config_json", "runtime_json"}).
			AddRow(int64(100), "Default", "url_test", "https://example.com/204", 60, "active", []byte(`{"a":1}`), []byte(`{}`)))

	prof, err := store.GetProfile(context.Background(), 100)
	require.NoError(t, err)
	require.Equal(t, profile.StrategyURLTest, prof.Strategy)
	require.Equal(t, profile.StatusActive, prof.Status)
	require.Equal(t, float64(1), prof.Config["a"])
	require.NoError(t, mock.ExpectationsWereMet())
}
