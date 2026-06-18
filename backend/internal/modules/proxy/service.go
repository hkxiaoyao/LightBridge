package proxy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	proxynode "github.com/WilliamWang1721/LightBridge/internal/modules/proxy/internal/node"
	"github.com/WilliamWang1721/LightBridge/internal/modules/proxy/internal/profile"
	proxyruntime "github.com/WilliamWang1721/LightBridge/internal/modules/proxy/internal/runtime"
)

type Service struct {
	db               *sql.DB
	runtimeManager   *proxyruntime.ProcessManager
	runtimeStore     *proxyruntime.SQLStore
	controllerClient proxyruntime.ControllerClient
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db, runtimeStore: proxyruntime.NewSQLStore(db)}
}

func NewServiceWithRuntime(db *sql.DB, mihomoBinaryPath string, runtimeRoot string) *Service {
	runtimeStore := proxyruntime.NewSQLStore(db)
	return &Service{
		db:           db,
		runtimeStore: runtimeStore,
		runtimeManager: &proxyruntime.ProcessManager{
			BinaryPath:    strings.TrimSpace(mihomoBinaryPath),
			RuntimeRoot:   strings.TrimSpace(runtimeRoot),
			Nodes:         proxynode.NewSQLStore(db),
			Instances:     runtimeStore,
			PortAllocator: proxyruntime.NewPortAllocator(),
		},
	}
}

func (s *Service) SetControllerHTTPClient(client *http.Client) {
	if s != nil {
		s.controllerClient.HTTPClient = client
	}
}

type NodeView struct {
	ID         int64          `json:"id"`
	Name       string         `json:"name"`
	NodeType   string         `json:"node_type"`
	SourceType string         `json:"source_type"`
	Config     map[string]any `json:"config"`
	Status     string         `json:"status"`
}

type ProfileView struct {
	ID              int64          `json:"id"`
	Name            string         `json:"name"`
	Strategy        string         `json:"strategy"`
	TestURL         string         `json:"test_url"`
	IntervalSeconds int            `json:"interval_seconds"`
	Status          string         `json:"status"`
	Config          map[string]any `json:"config"`
	Runtime         map[string]any `json:"runtime"`
}

type BindingView struct {
	ID               int64  `json:"id"`
	EntityType       string `json:"entity_type"`
	EntityID         string `json:"entity_id"`
	ProfileID        int64  `json:"profile_id"`
	Priority         int    `json:"priority"`
	Enabled          bool   `json:"enabled"`
	FallbackToDirect bool   `json:"fallback_to_direct"`
}

type RuntimeView struct {
	ProfileID      int64  `json:"profile_id"`
	RuntimeType    string `json:"runtime_type"`
	PID            int    `json:"pid,omitempty"`
	MixedPort      int    `json:"mixed_port,omitempty"`
	ControllerPort int    `json:"controller_port,omitempty"`
	ConfigPath     string `json:"config_path,omitempty"`
	WorkDir        string `json:"work_dir,omitempty"`
	Status         string `json:"status"`
	LastError      string `json:"last_error,omitempty"`
	ProxyURL       string `json:"proxy_url,omitempty"`
}

type RuntimeStatusView struct {
	Total    int `json:"total"`
	Starting int `json:"starting"`
	Running  int `json:"running"`
	Failed   int `json:"failed"`
	Stopped  int `json:"stopped"`
}

type ProfileTestView struct {
	ProfileID int64  `json:"profile_id"`
	Healthy   bool   `json:"healthy"`
	Status    string `json:"status"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`
	ProxyURL  string `json:"proxy_url,omitempty"`
}

type LegacyMigrationReport struct {
	ProxiesScanned   int      `json:"proxies_scanned"`
	ProxiesMigrated  int      `json:"proxies_migrated"`
	AccountsScanned  int      `json:"accounts_scanned"`
	BindingsMigrated int      `json:"bindings_migrated"`
	Warnings         []string `json:"warnings,omitempty"`
}

type CreateNodeInput struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ImportNodesInput struct {
	Format  string `json:"format"`
	Content string `json:"content"`
	URL     string `json:"url"`
}

type CreateProfileInput struct {
	Name            string  `json:"name"`
	Strategy        string  `json:"strategy"`
	TestURL         string  `json:"test_url"`
	IntervalSeconds int     `json:"interval_seconds"`
	NodeIDs         []int64 `json:"node_ids"`
	Weights         []int   `json:"weights,omitempty"`
}

type UpdateProfileInput = CreateProfileInput

type CreateBindingInput struct {
	EntityType       string `json:"entity_type"`
	EntityID         string `json:"entity_id"`
	ProfileID        int64  `json:"profile_id"`
	Priority         int    `json:"priority"`
	FallbackToDirect bool   `json:"fallback_to_direct"`
}

func (s *Service) ListNodes(ctx context.Context) ([]NodeView, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("proxy module service is not configured")
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, node_type, source_type, config_json, status
FROM proxy_nodes
WHERE deleted_at IS NULL
ORDER BY id DESC
`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []NodeView
	for rows.Next() {
		var item NodeView
		var configRaw []byte
		if err := rows.Scan(&item.ID, &item.Name, &item.NodeType, &item.SourceType, &configRaw, &item.Status); err != nil {
			return nil, err
		}
		if err := decodeJSONMap(configRaw, &item.Config); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) CreateManualNode(ctx context.Context, input CreateNodeInput) (*NodeView, error) {
	node, err := proxynode.ManualURL(input.Name, input.URL)
	if err != nil {
		return nil, err
	}
	id, err := s.insertNode(ctx, *node)
	if err != nil {
		return nil, err
	}
	return s.getNode(ctx, id)
}

func (s *Service) DeleteNode(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("node id is required")
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE proxy_nodes
SET status = 'disabled', deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
`, id)
	return err
}

func (s *Service) ImportNodes(ctx context.Context, input ImportNodesInput) ([]NodeView, error) {
	format := strings.ToLower(strings.TrimSpace(input.Format))
	content := input.Content
	if strings.TrimSpace(input.URL) != "" {
		downloaded, err := proxynode.DownloadSubscription(ctx, input.URL, proxynode.SubscriptionDownloadOptions{
			AllowInsecureHTTP: true,
		})
		if err != nil {
			return nil, err
		}
		content = string(downloaded)
		if format == "" {
			format = "clash_yaml"
		}
	}
	var nodes []proxynode.Node
	switch format {
	case "clash_yaml", "clash", "yaml":
		parsed, err := proxynode.ImportClashYAML([]byte(content))
		if err != nil {
			return nil, err
		}
		nodes = parsed
	case "uri":
		node, err := proxynode.ImportURI(content)
		if err != nil {
			return nil, err
		}
		nodes = []proxynode.Node{*node}
	default:
		return nil, fmt.Errorf("unsupported proxy import format %q", input.Format)
	}
	views := make([]NodeView, 0, len(nodes))
	for _, node := range nodes {
		id, err := s.insertNode(ctx, node)
		if err != nil {
			return nil, err
		}
		view, err := s.getNode(ctx, id)
		if err != nil {
			return nil, err
		}
		views = append(views, *view)
	}
	return views, nil
}

func (s *Service) ListProfiles(ctx context.Context) ([]ProfileView, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, strategy, test_url, interval_seconds, status, config_json, runtime_json
FROM proxy_profiles
WHERE deleted_at IS NULL
ORDER BY id DESC
`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []ProfileView
	for rows.Next() {
		item, err := scanProfileView(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) CreateProfile(ctx context.Context, input CreateProfileInput) (*ProfileView, error) {
	strategy := profile.Strategy(strings.TrimSpace(input.Strategy))
	if !profile.IsAllowedStrategy(strategy) {
		return nil, errors.New("proxy profile strategy is not supported")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("profile name is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var id int64
	if err := tx.QueryRowContext(ctx, `
INSERT INTO proxy_profiles (name, strategy, test_url, interval_seconds, status)
VALUES ($1, $2, COALESCE(NULLIF($3, ''), 'https://www.gstatic.com/generate_204'), CASE WHEN $4 > 0 THEN $4 ELSE 300 END, 'active')
RETURNING id
`, name, string(strategy), strings.TrimSpace(input.TestURL), input.IntervalSeconds).Scan(&id); err != nil {
		return nil, err
	}
	for idx, nodeID := range input.NodeIDs {
		weight := 1
		if idx < len(input.Weights) && input.Weights[idx] > 0 {
			weight = input.Weights[idx]
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO proxy_profile_nodes (profile_id, node_id, sort_order, weight, enabled)
VALUES ($1, $2, $3, $4, TRUE)
ON CONFLICT (profile_id, node_id) DO UPDATE SET sort_order = EXCLUDED.sort_order, weight = EXCLUDED.weight, enabled = TRUE
`, id, nodeID, idx, weight); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.getProfile(ctx, id)
}

func (s *Service) UpdateProfile(ctx context.Context, id int64, input UpdateProfileInput) (*ProfileView, error) {
	if id <= 0 {
		return nil, errors.New("profile id is required")
	}
	strategy := profile.Strategy(strings.TrimSpace(input.Strategy))
	if !profile.IsAllowedStrategy(strategy) {
		return nil, errors.New("proxy profile strategy is not supported")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("profile name is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `
UPDATE proxy_profiles
SET name = $2, strategy = $3, test_url = COALESCE(NULLIF($4, ''), 'https://www.gstatic.com/generate_204'), interval_seconds = CASE WHEN $5 > 0 THEN $5 ELSE 300 END, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
`, id, name, string(strategy), strings.TrimSpace(input.TestURL), input.IntervalSeconds)
	if err != nil {
		return nil, err
	}
	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return nil, sql.ErrNoRows
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM proxy_profile_nodes WHERE profile_id = $1`, id); err != nil {
		return nil, err
	}
	for idx, nodeID := range input.NodeIDs {
		weight := 1
		if idx < len(input.Weights) && input.Weights[idx] > 0 {
			weight = input.Weights[idx]
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO proxy_profile_nodes (profile_id, node_id, sort_order, weight, enabled)
VALUES ($1, $2, $3, $4, TRUE)
ON CONFLICT (profile_id, node_id) DO UPDATE SET sort_order = EXCLUDED.sort_order, weight = EXCLUDED.weight, enabled = TRUE
`, id, nodeID, idx, weight); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.getProfile(ctx, id)
}

func (s *Service) StartProfile(ctx context.Context, id int64) (*RuntimeView, error) {
	if s == nil || s.runtimeManager == nil {
		return nil, errors.New("proxy runtime manager is not configured")
	}
	prof, err := s.getProfileModel(ctx, id)
	if err != nil {
		return nil, err
	}
	instance, err := s.runtimeManager.EnsureRunning(ctx, *prof)
	if err != nil {
		return nil, err
	}
	return runtimeView(instance), nil
}

func (s *Service) StopProfile(ctx context.Context, id int64) (*RuntimeView, error) {
	if id <= 0 {
		return nil, errors.New("profile id is required")
	}
	if s != nil && s.runtimeManager != nil {
		if err := s.runtimeManager.Stop(id); err != nil {
			return nil, err
		}
	} else if s != nil && s.runtimeStore != nil {
		if err := s.runtimeStore.MarkRuntimeStopped(ctx, id); err != nil {
			return nil, err
		}
	}
	return s.GetRuntime(ctx, id)
}

func (s *Service) RestartProfile(ctx context.Context, id int64) (*RuntimeView, error) {
	if _, err := s.StopProfile(ctx, id); err != nil {
		return nil, err
	}
	return s.StartProfile(ctx, id)
}

func (s *Service) GetRuntime(ctx context.Context, id int64) (*RuntimeView, error) {
	if s == nil || s.runtimeStore == nil {
		return nil, errors.New("proxy runtime store is not configured")
	}
	instance, err := s.runtimeStore.GetRuntime(ctx, id)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return &RuntimeView{ProfileID: id, Status: string(proxyruntime.StatusStopped)}, nil
	}
	return runtimeView(instance), nil
}

func (s *Service) TestProfile(ctx context.Context, id int64) (*ProfileTestView, error) {
	runtimeView, err := s.GetRuntime(ctx, id)
	if err != nil {
		return nil, err
	}
	result := &ProfileTestView{
		ProfileID: id,
		Status:    runtimeView.Status,
		ProxyURL:  runtimeView.ProxyURL,
	}
	if runtimeView.Status != string(proxyruntime.StatusRunning) {
		result.Error = "proxy runtime is not running"
		return result, nil
	}
	instance, err := s.runtimeStore.GetRuntime(ctx, id)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		result.Status = string(proxyruntime.StatusStopped)
		result.Error = "proxy runtime is not running"
		return result, nil
	}
	info, err := s.controllerClient.Version(ctx, *instance, instance.ControllerSecretRef)
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	result.Healthy = true
	result.Version = info.Version
	return result, nil
}

func (s *Service) GetRuntimeStatus(ctx context.Context) (*RuntimeStatusView, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("proxy module service is not configured")
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT status, COUNT(*)
FROM proxy_runtime_instances
GROUP BY status
`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	status := &RuntimeStatusView{}
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, err
		}
		status.Total += count
		switch proxyruntime.Status(name) {
		case proxyruntime.StatusStarting:
			status.Starting += count
		case proxyruntime.StatusRunning:
			status.Running += count
		case proxyruntime.StatusFailed:
			status.Failed += count
		case proxyruntime.StatusStopped:
			status.Stopped += count
		}
	}
	return status, rows.Err()
}

func (s *Service) ListBindings(ctx context.Context) ([]BindingView, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, entity_type, entity_id, profile_id, priority, enabled, fallback_to_direct
FROM proxy_bindings
ORDER BY entity_type ASC, entity_id ASC, priority ASC
`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []BindingView
	for rows.Next() {
		var item BindingView
		if err := rows.Scan(&item.ID, &item.EntityType, &item.EntityID, &item.ProfileID, &item.Priority, &item.Enabled, &item.FallbackToDirect); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) CreateBinding(ctx context.Context, input CreateBindingInput) (*BindingView, error) {
	entityType := strings.TrimSpace(input.EntityType)
	entityID := strings.TrimSpace(input.EntityID)
	if entityType == "" || entityID == "" {
		return nil, errors.New("binding entity_type and entity_id are required")
	}
	var id int64
	if err := s.db.QueryRowContext(ctx, `
INSERT INTO proxy_bindings (entity_type, entity_id, profile_id, priority, enabled, fallback_to_direct)
VALUES ($1, $2, $3, $4, TRUE, $5)
ON CONFLICT (entity_type, entity_id, priority) DO UPDATE SET profile_id = EXCLUDED.profile_id, enabled = TRUE, fallback_to_direct = EXCLUDED.fallback_to_direct, updated_at = NOW()
RETURNING id
`, entityType, entityID, input.ProfileID, input.Priority, input.FallbackToDirect).Scan(&id); err != nil {
		return nil, err
	}
	return s.getBinding(ctx, id)
}

func (s *Service) DeleteBinding(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("binding id is required")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM proxy_bindings WHERE id = $1`, id)
	return err
}

func (s *Service) MigrateLegacyProxies(ctx context.Context) (*LegacyMigrationReport, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("proxy module service is not configured")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	report := &LegacyMigrationReport{}
	proxyRows, err := tx.QueryContext(ctx, `
SELECT id, name, protocol, host, port, COALESCE(username, ''), COALESCE(password, '')
FROM proxies
WHERE deleted_at IS NULL
ORDER BY id ASC
`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = proxyRows.Close() }()

	profileByProxyID := map[int64]int64{}
	for proxyRows.Next() {
		var legacyID int64
		var name, protocol, host, username, password string
		var port int
		if err := proxyRows.Scan(&legacyID, &name, &protocol, &host, &port, &username, &password); err != nil {
			return nil, err
		}
		report.ProxiesScanned++
		profileID, migrated, err := migrateLegacyProxy(ctx, tx, legacyProxyRow{
			ID:       legacyID,
			Name:     name,
			Protocol: protocol,
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
		})
		if err != nil {
			report.Warnings = append(report.Warnings, err.Error())
			continue
		}
		profileByProxyID[legacyID] = profileID
		if migrated {
			report.ProxiesMigrated++
		}
	}
	if err := proxyRows.Err(); err != nil {
		return nil, err
	}

	accountRows, err := tx.QueryContext(ctx, `
SELECT id, proxy_id
FROM accounts
WHERE proxy_id IS NOT NULL AND deleted_at IS NULL
ORDER BY id ASC
`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = accountRows.Close() }()
	for accountRows.Next() {
		var accountID, legacyProxyID int64
		if err := accountRows.Scan(&accountID, &legacyProxyID); err != nil {
			return nil, err
		}
		report.AccountsScanned++
		profileID := profileByProxyID[legacyProxyID]
		if profileID <= 0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("legacy proxy %d for account %d was not migrated", legacyProxyID, accountID))
			continue
		}
		inserted, err := migrateLegacyAccountBinding(ctx, tx, accountID, profileID)
		if err != nil {
			return nil, err
		}
		if inserted {
			report.BindingsMigrated++
		}
	}
	if err := accountRows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return report, nil
}

func (s *Service) insertNode(ctx context.Context, node proxynode.Node) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("proxy module service is not configured")
	}
	configRaw, err := json.Marshal(node.Config)
	if err != nil {
		return 0, err
	}
	secretRaw, err := json.Marshal(node.Secret)
	if err != nil {
		return 0, err
	}
	name := strings.TrimSpace(node.Name)
	if name == "" {
		name = "Imported Proxy"
	}
	var id int64
	err = s.db.QueryRowContext(ctx, `
INSERT INTO proxy_nodes (name, node_type, source_type, config_json, secret_json, status)
VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, 'active')
RETURNING id
`, name, string(node.Type), string(node.SourceType), string(configRaw), string(secretRaw)).Scan(&id)
	return id, err
}

func (s *Service) getNode(ctx context.Context, id int64) (*NodeView, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, node_type, source_type, config_json, status
FROM proxy_nodes
WHERE id = $1 AND deleted_at IS NULL
`, id)
	var item NodeView
	var configRaw []byte
	if err := row.Scan(&item.ID, &item.Name, &item.NodeType, &item.SourceType, &configRaw, &item.Status); err != nil {
		return nil, err
	}
	if err := decodeJSONMap(configRaw, &item.Config); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) getProfile(ctx context.Context, id int64) (*ProfileView, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, strategy, test_url, interval_seconds, status, config_json, runtime_json
FROM proxy_profiles
WHERE id = $1 AND deleted_at IS NULL
`, id)
	item, err := scanProfileView(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) getProfileModel(ctx context.Context, id int64) (*profile.Profile, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, strategy, test_url, interval_seconds, status, config_json, runtime_json
FROM proxy_profiles
WHERE id = $1 AND deleted_at IS NULL
`, id)
	var prof profile.Profile
	var strategy, status string
	var configRaw, runtimeRaw []byte
	if err := row.Scan(&prof.ID, &prof.Name, &strategy, &prof.TestURL, &prof.IntervalSeconds, &status, &configRaw, &runtimeRaw); err != nil {
		return nil, err
	}
	prof.Strategy = profile.Strategy(strategy)
	prof.Status = profile.Status(status)
	if err := decodeJSONMap(configRaw, &prof.Config); err != nil {
		return nil, err
	}
	if err := decodeJSONMap(runtimeRaw, &prof.Runtime); err != nil {
		return nil, err
	}
	return &prof, nil
}

func (s *Service) getBinding(ctx context.Context, id int64) (*BindingView, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, entity_type, entity_id, profile_id, priority, enabled, fallback_to_direct
FROM proxy_bindings
WHERE id = $1
`, id)
	var item BindingView
	if err := row.Scan(&item.ID, &item.EntityType, &item.EntityID, &item.ProfileID, &item.Priority, &item.Enabled, &item.FallbackToDirect); err != nil {
		return nil, err
	}
	return &item, nil
}

type profileScanner interface {
	Scan(dest ...any) error
}

func scanProfileView(row profileScanner) (ProfileView, error) {
	var item ProfileView
	var configRaw, runtimeRaw []byte
	if err := row.Scan(&item.ID, &item.Name, &item.Strategy, &item.TestURL, &item.IntervalSeconds, &item.Status, &configRaw, &runtimeRaw); err != nil {
		return item, err
	}
	if err := decodeJSONMap(configRaw, &item.Config); err != nil {
		return item, err
	}
	if err := decodeJSONMap(runtimeRaw, &item.Runtime); err != nil {
		return item, err
	}
	return item, nil
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

type legacyProxyRow struct {
	ID       int64
	Name     string
	Protocol string
	Host     string
	Port     int
	Username string
	Password string
}

type sqlTx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func migrateLegacyProxy(ctx context.Context, tx sqlTx, legacy legacyProxyRow) (int64, bool, error) {
	if legacy.ID <= 0 {
		return 0, false, errors.New("legacy proxy id is required")
	}
	node, err := proxynode.ManualURL(legacy.Name, legacyProxyURL(legacy))
	if err != nil {
		return 0, false, fmt.Errorf("legacy proxy %d: %w", legacy.ID, err)
	}
	node.SourceType = proxynode.SourceMigrated
	configRaw, err := json.Marshal(node.Config)
	if err != nil {
		return 0, false, err
	}
	secretRaw, err := json.Marshal(node.Secret)
	if err != nil {
		return 0, false, err
	}

	nodeID, insertedNode, err := findOrInsertLegacyNode(ctx, tx, legacy, node, string(configRaw), string(secretRaw))
	if err != nil {
		return 0, false, err
	}
	var profileID int64
	var insertedProfile bool
	config := fmt.Sprintf(`{"legacy_proxy_id":%d}`, legacy.ID)
	profileID, insertedProfile, err = findOrInsertLegacyProfile(ctx, tx, legacy.ID, config)
	if err != nil {
		return 0, false, err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO proxy_profile_nodes (profile_id, node_id, sort_order, weight, enabled)
VALUES ($1, $2, 0, 1, TRUE)
ON CONFLICT (profile_id, node_id) DO UPDATE SET enabled = TRUE, sort_order = 0, weight = 1
`, profileID, nodeID); err != nil {
		return 0, false, err
	}
	return profileID, insertedNode || insertedProfile, nil
}

func findOrInsertLegacyNode(ctx context.Context, tx sqlTx, legacy legacyProxyRow, node *proxynode.Node, configRaw string, secretRaw string) (int64, bool, error) {
	var existingID int64
	err := tx.QueryRowContext(ctx, `
SELECT id FROM proxy_nodes
WHERE source_type = 'migrated' AND source_id = $1 AND deleted_at IS NULL
LIMIT 1
`, legacy.ID).Scan(&existingID)
	if err == nil {
		return existingID, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, false, err
	}
	var insertedID int64
	if err := tx.QueryRowContext(ctx, `
INSERT INTO proxy_nodes (name, node_type, source_type, source_id, config_json, secret_json, status)
VALUES ($1, $2, 'migrated', $3, $4::jsonb, $5::jsonb, 'active')
RETURNING id
`, legacyProxyName(legacy), string(node.Type), legacy.ID, configRaw, secretRaw).Scan(&insertedID); err != nil {
		return 0, false, err
	}
	return insertedID, true, nil
}

func findOrInsertLegacyProfile(ctx context.Context, tx sqlTx, legacyProxyID int64, configRaw string) (int64, bool, error) {
	var existingID int64
	if err := tx.QueryRowContext(ctx, `
SELECT id FROM proxy_profiles
WHERE config_json ->> 'legacy_proxy_id' = $1 AND deleted_at IS NULL
LIMIT 1
`, fmt.Sprintf("%d", legacyProxyID)).Scan(&existingID); err == nil {
		return existingID, false, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, false, err
	}
	var insertedID int64
	if err := tx.QueryRowContext(ctx, `
INSERT INTO proxy_profiles (name, strategy, status, config_json)
VALUES ($1, 'select', 'active', $2::jsonb)
RETURNING id
`, fmt.Sprintf("Migrated Proxy %d", legacyProxyID), configRaw).Scan(&insertedID); err != nil {
		return 0, false, err
	}
	return insertedID, true, nil
}

func migrateLegacyAccountBinding(ctx context.Context, tx sqlTx, accountID int64, profileID int64) (bool, error) {
	entityID := fmt.Sprintf("%d", accountID)
	var existingID int64
	if err := tx.QueryRowContext(ctx, `
SELECT id FROM proxy_bindings
WHERE entity_type = 'account' AND entity_id = $1 AND priority = 0
LIMIT 1
`, entityID).Scan(&existingID); err == nil {
		return false, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	var id int64
	if err := tx.QueryRowContext(ctx, `
INSERT INTO proxy_bindings (entity_type, entity_id, profile_id, priority, enabled, fallback_to_direct)
VALUES ('account', $1, $2, 0, TRUE, FALSE)
RETURNING id
`, entityID, profileID).Scan(&id); err != nil {
		return false, err
	}
	return true, nil
}

func legacyProxyName(legacy legacyProxyRow) string {
	name := strings.TrimSpace(legacy.Name)
	if name != "" {
		return name
	}
	return fmt.Sprintf("Migrated Proxy %d", legacy.ID)
}

func legacyProxyURL(legacy legacyProxyRow) string {
	scheme := strings.ToLower(strings.TrimSpace(legacy.Protocol))
	if scheme == "socks5h" {
		scheme = "socks5"
	}
	host := strings.TrimSpace(legacy.Host)
	u := fmt.Sprintf("%s://%s:%d", scheme, host, legacy.Port)
	username := strings.TrimSpace(legacy.Username)
	password := strings.TrimSpace(legacy.Password)
	if username == "" && password == "" {
		return u
	}
	escapedUser := url.QueryEscape(username)
	escapedPassword := url.QueryEscape(password)
	return fmt.Sprintf("%s://%s:%s@%s:%d", scheme, escapedUser, escapedPassword, host, legacy.Port)
}

func runtimeView(instance *proxyruntime.Instance) *RuntimeView {
	if instance == nil {
		return nil
	}
	view := &RuntimeView{
		ProfileID:      instance.ProfileID,
		RuntimeType:    instance.RuntimeType,
		PID:            instance.PID,
		MixedPort:      instance.MixedPort,
		ControllerPort: instance.ControllerPort,
		ConfigPath:     instance.ConfigPath,
		WorkDir:        instance.WorkDir,
		Status:         string(instance.Status),
		LastError:      instance.LastError,
	}
	if proxyURL, err := instance.ProxyURL(); err == nil {
		view.ProxyURL = proxyURL
	}
	return view
}
