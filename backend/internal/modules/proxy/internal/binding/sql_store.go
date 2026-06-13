package binding

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/Wei-Shaw/LightBridge/internal/modules/proxy/internal/profile"
)

type SQLStore struct {
	DB *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{DB: db}
}

func (s *SQLStore) FindEnabledBinding(ctx context.Context, query Query) (*Binding, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("proxy binding sql store is not configured")
	}
	row := s.DB.QueryRowContext(ctx, `
SELECT id, entity_type, entity_id, profile_id, priority, enabled, fallback_to_direct
FROM proxy_bindings
WHERE entity_type = $1 AND entity_id = $2 AND enabled = TRUE
ORDER BY priority ASC, id ASC
LIMIT 1
`, string(query.EntityType), query.EntityID)
	var binding Binding
	var entityType string
	if err := row.Scan(&binding.ID, &entityType, &binding.EntityID, &binding.ProfileID, &binding.Priority, &binding.Enabled, &binding.FallbackToDirect); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return s.findLegacyAccountBinding(ctx, query)
		}
		return nil, err
	}
	binding.EntityType = EntityType(entityType)
	return &binding, nil
}

func (s *SQLStore) findLegacyAccountBinding(ctx context.Context, query Query) (*Binding, error) {
	if query.EntityType != EntityAccount {
		return nil, nil
	}
	accountID, err := strconv.ParseInt(query.EntityID, 10, 64)
	if err != nil || accountID <= 0 {
		return nil, nil
	}
	row := s.DB.QueryRowContext(ctx, `
SELECT pp.id
FROM accounts a
JOIN proxy_profiles pp ON pp.config_json ->> 'legacy_proxy_id' = a.proxy_id::text
WHERE a.id = $1
  AND a.proxy_id IS NOT NULL
  AND a.deleted_at IS NULL
  AND pp.deleted_at IS NULL
LIMIT 1
`, accountID)
	var profileID int64
	if err := row.Scan(&profileID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &Binding{
		EntityType:       EntityAccount,
		EntityID:         query.EntityID,
		ProfileID:        profileID,
		Priority:         0,
		Enabled:          true,
		FallbackToDirect: false,
	}, nil
}

func (s *SQLStore) GetProfile(ctx context.Context, id int64) (*profile.Profile, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("proxy profile sql store is not configured")
	}
	row := s.DB.QueryRowContext(ctx, `
SELECT id, name, strategy, test_url, interval_seconds, status, config_json, runtime_json
FROM proxy_profiles
WHERE id = $1 AND deleted_at IS NULL
`, id)
	var prof profile.Profile
	var strategy, status string
	var configRaw, runtimeRaw []byte
	if err := row.Scan(&prof.ID, &prof.Name, &strategy, &prof.TestURL, &prof.IntervalSeconds, &status, &configRaw, &runtimeRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	prof.Strategy = profile.Strategy(strategy)
	prof.Status = profile.Status(status)
	if err := decodeJSONMap(configRaw, &prof.Config); err != nil {
		return nil, fmt.Errorf("decode proxy profile config: %w", err)
	}
	if err := decodeJSONMap(runtimeRaw, &prof.Runtime); err != nil {
		return nil, fmt.Errorf("decode proxy profile runtime: %w", err)
	}
	return &prof, nil
}

func decodeJSONMap(raw []byte, target *map[string]any) error {
	if len(raw) == 0 {
		*target = map[string]any{}
		return nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return err
	}
	if *target == nil {
		*target = map[string]any{}
	}
	return nil
}
