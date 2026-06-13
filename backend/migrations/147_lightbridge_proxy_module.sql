CREATE TABLE IF NOT EXISTS proxy_nodes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    node_type VARCHAR(40) NOT NULL,
    source_type VARCHAR(40) NOT NULL,
    source_id BIGINT NULL,
    config_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    secret_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_proxy_nodes_status ON proxy_nodes(status);
CREATE INDEX IF NOT EXISTS idx_proxy_nodes_node_type ON proxy_nodes(node_type);
CREATE INDEX IF NOT EXISTS idx_proxy_nodes_source ON proxy_nodes(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_proxy_nodes_deleted_at ON proxy_nodes(deleted_at);

CREATE TABLE IF NOT EXISTS proxy_profiles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    strategy VARCHAR(40) NOT NULL,
    test_url TEXT NOT NULL DEFAULT 'https://www.gstatic.com/generate_204',
    interval_seconds INT NOT NULL DEFAULT 300,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    config_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    runtime_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_proxy_profiles_status ON proxy_profiles(status);
CREATE INDEX IF NOT EXISTS idx_proxy_profiles_deleted_at ON proxy_profiles(deleted_at);

CREATE TABLE IF NOT EXISTS proxy_profile_nodes (
    id BIGSERIAL PRIMARY KEY,
    profile_id BIGINT NOT NULL REFERENCES proxy_profiles(id) ON DELETE CASCADE,
    node_id BIGINT NOT NULL REFERENCES proxy_nodes(id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0,
    weight INT NOT NULL DEFAULT 1,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(profile_id, node_id)
);

CREATE INDEX IF NOT EXISTS idx_proxy_profile_nodes_profile ON proxy_profile_nodes(profile_id, enabled, sort_order);
CREATE INDEX IF NOT EXISTS idx_proxy_profile_nodes_node ON proxy_profile_nodes(node_id);

CREATE TABLE IF NOT EXISTS proxy_bindings (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(40) NOT NULL,
    entity_id VARCHAR(120) NOT NULL,
    profile_id BIGINT NOT NULL REFERENCES proxy_profiles(id) ON DELETE RESTRICT,
    priority INT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    fallback_to_direct BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(entity_type, entity_id, priority)
);

CREATE INDEX IF NOT EXISTS idx_proxy_bindings_lookup ON proxy_bindings(entity_type, entity_id, enabled, priority);
CREATE INDEX IF NOT EXISTS idx_proxy_bindings_profile ON proxy_bindings(profile_id);

CREATE TABLE IF NOT EXISTS proxy_runtime_instances (
    id BIGSERIAL PRIMARY KEY,
    profile_id BIGINT NOT NULL REFERENCES proxy_profiles(id) ON DELETE CASCADE,
    runtime_type VARCHAR(40) NOT NULL DEFAULT 'mihomo',
    pid INT NULL,
    mixed_port INT NOT NULL,
    controller_port INT NOT NULL,
    controller_secret_ref TEXT NOT NULL,
    config_path TEXT NOT NULL,
    work_dir TEXT NOT NULL,
    status VARCHAR(40) NOT NULL,
    last_error TEXT NULL,
    started_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_proxy_runtime_instances_profile ON proxy_runtime_instances(profile_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_proxy_runtime_instances_mixed_port ON proxy_runtime_instances(mixed_port);
CREATE UNIQUE INDEX IF NOT EXISTS idx_proxy_runtime_instances_controller_port ON proxy_runtime_instances(controller_port);
CREATE INDEX IF NOT EXISTS idx_proxy_runtime_instances_status ON proxy_runtime_instances(status);
