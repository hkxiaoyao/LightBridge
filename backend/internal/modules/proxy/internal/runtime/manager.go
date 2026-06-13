package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Wei-Shaw/LightBridge/internal/modules/proxy/internal/mihomo"
	proxynode "github.com/Wei-Shaw/LightBridge/internal/modules/proxy/internal/node"
	"github.com/Wei-Shaw/LightBridge/internal/modules/proxy/internal/profile"
)

type NodeStore interface {
	ListProfileNodes(context.Context, int64) ([]proxynode.Node, error)
}

type InstanceStore interface {
	GetRunningRuntime(context.Context, int64) (*Instance, error)
	GetRuntime(context.Context, int64) (*Instance, error)
	SaveRuntimeStarting(context.Context, Instance) error
	MarkRuntimeRunning(context.Context, int64, int) error
	MarkRuntimeFailed(context.Context, int64, string) error
	MarkRuntimeStopped(context.Context, int64) error
}

type ProcessManager struct {
	BinaryPath    string
	RuntimeRoot   string
	Nodes         NodeStore
	Instances     InstanceStore
	PortAllocator *PortAllocator

	mu        sync.Mutex
	processes map[int64]*exec.Cmd
}

func (m *ProcessManager) EnsureRunning(ctx context.Context, prof profile.Profile) (*Instance, error) {
	if m == nil || m.Nodes == nil || m.Instances == nil {
		return nil, errors.New("proxy process manager is not configured")
	}
	if existing, err := m.Instances.GetRunningRuntime(ctx, prof.ID); err != nil {
		return nil, err
	} else if existing != nil && existing.Status == StatusRunning {
		return existing, nil
	}
	if err := prof.ValidateActive(); err != nil {
		return nil, err
	}
	binary := strings.TrimSpace(m.BinaryPath)
	if binary == "" {
		return nil, errors.New("mihomo binary path is required")
	}
	if info, err := os.Stat(binary); err != nil {
		return nil, err
	} else if info.IsDir() {
		return nil, errors.New("mihomo binary path is a directory")
	}
	nodes, err := m.Nodes.ListProfileNodes(ctx, prof.ID)
	if err != nil {
		return nil, err
	}
	mixedPort, controllerPort, err := m.portAllocator().AllocatePair()
	if err != nil {
		return nil, err
	}
	secret, err := randomSecret()
	if err != nil {
		return nil, err
	}
	workDir := filepath.Join(m.runtimeRoot(), fmt.Sprintf("profile-%d", prof.ID))
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return nil, err
	}
	configPath := filepath.Join(workDir, "config.yaml")
	config, err := mihomo.Compile(mihomo.Profile{
		ID:              prof.ID,
		Strategy:        prof.Strategy,
		TestURL:         prof.TestURL,
		IntervalSeconds: prof.IntervalSeconds,
	}, nodes, mihomo.RuntimeConfig{MixedPort: mixedPort, ControllerPort: controllerPort, ControllerSecret: secret})
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		return nil, err
	}
	instance := Instance{
		ProfileID:           prof.ID,
		RuntimeType:         "mihomo",
		MixedPort:           mixedPort,
		ControllerPort:      controllerPort,
		ControllerSecretRef: secret,
		ConfigPath:          configPath,
		WorkDir:             workDir,
		Status:              StatusStarting,
	}
	if err := m.Instances.SaveRuntimeStarting(ctx, instance); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, binary, "-d", workDir, "-f", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		_ = m.Instances.MarkRuntimeFailed(ctx, prof.ID, err.Error())
		return nil, err
	}
	m.trackProcess(prof.ID, cmd)
	if err := m.Instances.MarkRuntimeRunning(ctx, prof.ID, cmd.Process.Pid); err != nil {
		return nil, err
	}
	go func() {
		err := cmd.Wait()
		m.untrackProcess(prof.ID)
		if err != nil {
			_ = m.Instances.MarkRuntimeFailed(context.Background(), prof.ID, err.Error())
		}
	}()
	instance.PID = cmd.Process.Pid
	instance.Status = StatusRunning
	return &instance, nil
}

func (m *ProcessManager) Stop(profileID int64) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	cmd := m.processes[profileID]
	delete(m.processes, profileID)
	m.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		if err := cmd.Process.Kill(); err != nil {
			return err
		}
	}
	if m.Instances != nil {
		return m.Instances.MarkRuntimeStopped(context.Background(), profileID)
	}
	return nil
}

func (m *ProcessManager) trackProcess(profileID int64, cmd *exec.Cmd) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.processes == nil {
		m.processes = map[int64]*exec.Cmd{}
	}
	m.processes[profileID] = cmd
}

func (m *ProcessManager) untrackProcess(profileID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.processes, profileID)
}

func (m *ProcessManager) portAllocator() *PortAllocator {
	if m.PortAllocator != nil {
		return m.PortAllocator
	}
	m.PortAllocator = NewPortAllocator()
	return m.PortAllocator
}

func (m *ProcessManager) runtimeRoot() string {
	if root := strings.TrimSpace(m.RuntimeRoot); root != "" {
		return root
	}
	return filepath.Join("data", "modules", "lightbridge.proxy", "runtime")
}

func randomSecret() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
