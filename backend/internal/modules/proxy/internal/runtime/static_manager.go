package runtime

import (
	"context"
	"errors"

	"github.com/Wei-Shaw/LightBridge/internal/modules/proxy/internal/profile"
)

type Store interface {
	GetRunningRuntime(context.Context, int64) (*Instance, error)
}

type ExistingRuntimeManager struct {
	Store Store
}

func (m ExistingRuntimeManager) EnsureRunning(ctx context.Context, profile profile.Profile) (*Instance, error) {
	if m.Store == nil {
		return nil, errors.New("proxy runtime store is not configured")
	}
	instance, err := m.Store.GetRunningRuntime(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.New("proxy runtime is not running")
	}
	if _, err := instance.ProxyURL(); err != nil {
		return nil, err
	}
	return instance, nil
}
