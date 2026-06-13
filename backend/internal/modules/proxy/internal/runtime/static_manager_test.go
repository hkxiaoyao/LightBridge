package runtime

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/LightBridge/internal/modules/proxy/internal/profile"
	"github.com/stretchr/testify/require"
)

type fakeRuntimeStore struct {
	instance *Instance
}

func (s fakeRuntimeStore) GetRunningRuntime(context.Context, int64) (*Instance, error) {
	return s.instance, nil
}

func (s fakeRuntimeStore) GetRuntime(context.Context, int64) (*Instance, error) {
	return s.instance, nil
}

func TestExistingRuntimeManagerEnsureRunning(t *testing.T) {
	manager := ExistingRuntimeManager{Store: fakeRuntimeStore{instance: &Instance{ProfileID: 1, MixedPort: 17001, Status: StatusRunning}}}
	instance, err := manager.EnsureRunning(context.Background(), profile.Profile{ID: 1})
	require.NoError(t, err)
	require.Equal(t, 17001, instance.MixedPort)
}

func TestExistingRuntimeManagerRejectsMissingRuntime(t *testing.T) {
	manager := ExistingRuntimeManager{Store: fakeRuntimeStore{}}
	_, err := manager.EnsureRunning(context.Background(), profile.Profile{ID: 1})
	require.ErrorContains(t, err, "not running")
}
