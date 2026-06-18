package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	proxynode "github.com/WilliamWang1721/LightBridge/internal/modules/proxy/internal/node"
	"github.com/WilliamWang1721/LightBridge/internal/modules/proxy/internal/profile"
	"github.com/stretchr/testify/require"
)

type fakeNodeStore struct {
	nodes []proxynode.Node
}

func (s fakeNodeStore) ListProfileNodes(context.Context, int64) ([]proxynode.Node, error) {
	return s.nodes, nil
}

type fakeInstanceStore struct {
	existing *Instance
	started  *Instance
	running  bool
	failed   string
}

func (s *fakeInstanceStore) GetRunningRuntime(context.Context, int64) (*Instance, error) {
	return s.existing, nil
}

func (s *fakeInstanceStore) GetRuntime(context.Context, int64) (*Instance, error) {
	return s.existing, nil
}

func (s *fakeInstanceStore) SaveRuntimeStarting(_ context.Context, instance Instance) error {
	copied := instance
	s.started = &copied
	return nil
}

func (s *fakeInstanceStore) MarkRuntimeRunning(context.Context, int64, int) error {
	s.running = true
	return nil
}

func (s *fakeInstanceStore) MarkRuntimeFailed(_ context.Context, _ int64, errMsg string) error {
	s.failed = errMsg
	return nil
}

func (s *fakeInstanceStore) MarkRuntimeStopped(context.Context, int64) error {
	return nil
}

func TestProcessManagerRejectsMissingBinary(t *testing.T) {
	manager := &ProcessManager{
		BinaryPath:  filepath.Join(t.TempDir(), "missing-mihomo"),
		RuntimeRoot: t.TempDir(),
		Nodes: fakeNodeStore{nodes: []proxynode.Node{{
			ID:     1,
			Type:   proxynode.TypeHTTP,
			Config: map[string]any{"server": "proxy.example.com", "port": 8080},
		}}},
		Instances: &fakeInstanceStore{},
	}
	_, err := manager.EnsureRunning(context.Background(), profile.Profile{ID: 1, Strategy: profile.StrategySelect, Status: profile.StatusActive})
	require.Error(t, err)
}

func TestProcessManagerReturnsExistingRunningRuntime(t *testing.T) {
	manager := &ProcessManager{
		Nodes:     fakeNodeStore{},
		Instances: &fakeInstanceStore{existing: &Instance{ProfileID: 1, MixedPort: 17001, Status: StatusRunning}},
	}
	instance, err := manager.EnsureRunning(context.Background(), profile.Profile{ID: 1, Strategy: profile.StrategySelect, Status: profile.StatusActive})
	require.NoError(t, err)
	require.Equal(t, 17001, instance.MixedPort)
}

func TestProcessManagerWritesConfigBeforeStartAttempt(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "mihomo")
	require.NoError(t, os.WriteFile(binary, []byte("#!/bin/sh\nexit 1\n"), 0o700))
	store := &fakeInstanceStore{}
	manager := &ProcessManager{
		BinaryPath:  binary,
		RuntimeRoot: dir,
		Nodes: fakeNodeStore{nodes: []proxynode.Node{{
			ID:     1,
			Type:   proxynode.TypeHTTP,
			Config: map[string]any{"server": "proxy.example.com", "port": 8080},
		}}},
		Instances: store,
		PortAllocator: &PortAllocator{
			mixedStart:      17000,
			mixedEnd:        17020,
			controllerStart: 18000,
			controllerEnd:   18020,
			isFree:          func(int) bool { return true },
		},
	}
	instance, err := manager.EnsureRunning(context.Background(), profile.Profile{ID: 1, Strategy: profile.StrategySelect, Status: profile.StatusActive})
	require.NoError(t, err)
	require.NotNil(t, instance)
	require.True(t, store.running)
	require.NotNil(t, store.started)
	content, err := os.ReadFile(store.started.ConfigPath)
	require.NoError(t, err)
	require.Contains(t, string(content), "allow-lan: false")
	require.Contains(t, string(content), "bind-address: 127.0.0.1")
}
