package binding

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/LightBridge/internal/modules/proxy/internal/profile"
	proxyruntime "github.com/Wei-Shaw/LightBridge/internal/modules/proxy/internal/runtime"
	"github.com/Wei-Shaw/LightBridge/internal/outbound"
	"github.com/stretchr/testify/require"
)

type fakeBindingStore struct {
	bindings map[Query]*Binding
	queries  []Query
}

func (s *fakeBindingStore) FindEnabledBinding(_ context.Context, query Query) (*Binding, error) {
	s.queries = append(s.queries, query)
	return s.bindings[query], nil
}

type fakeProfileStore struct {
	profiles map[int64]*profile.Profile
}

func (s fakeProfileStore) GetProfile(_ context.Context, id int64) (*profile.Profile, error) {
	return s.profiles[id], nil
}

type fakeRuntimeManager struct {
	instance *proxyruntime.Instance
	err      error
}

func (m fakeRuntimeManager) EnsureRunning(context.Context, profile.Profile) (*proxyruntime.Instance, error) {
	return m.instance, m.err
}

func TestQueriesForScopePriority(t *testing.T) {
	queries := QueriesForScope(outbound.Scope{
		ProviderID: "openai",
		ChannelID:  20,
		AccountID:  10,
		APIKeyID:   30,
		UserID:     40,
		Metadata:   map[string]any{"proxy_profile_id": int64(99)},
	})
	require.Equal(t, []Query{
		{EntityType: EntityRequest, EntityID: "profile:99|api_key:30|user:40"},
		{EntityType: EntityAccount, EntityID: "10"},
		{EntityType: EntityChannel, EntityID: "20"},
		{EntityType: EntityProvider, EntityID: "openai"},
		{EntityType: EntityGlobal, EntityID: "default"},
	}, queries)
}

func TestResolverUsesHighestPriorityBinding(t *testing.T) {
	store := &fakeBindingStore{bindings: map[Query]*Binding{
		{EntityType: EntityAccount, EntityID: "10"}: {
			ID:         1,
			EntityType: EntityAccount,
			EntityID:   "10",
			ProfileID:  100,
			Enabled:    true,
		},
		{EntityType: EntityGlobal, EntityID: "default"}: {
			ID:         2,
			EntityType: EntityGlobal,
			EntityID:   "default",
			ProfileID:  200,
			Enabled:    true,
		},
	}}
	resolver := Resolver{
		Bindings: store,
		Profiles: fakeProfileStore{profiles: map[int64]*profile.Profile{
			100: {ID: 100, Strategy: profile.StrategySelect, Status: profile.StatusActive},
		}},
		Runtime: fakeRuntimeManager{instance: &proxyruntime.Instance{ProfileID: 100, MixedPort: 17010, Status: proxyruntime.StatusRunning}},
	}

	resolved, err := resolver.Resolve(context.Background(), outbound.Scope{AccountID: 10})
	require.NoError(t, err)
	require.Equal(t, "proxy", resolved.Mode)
	require.Equal(t, "http://127.0.0.1:17010", resolved.ProxyURL)
	require.Equal(t, int64(100), resolved.ProfileID)
	require.Equal(t, []Query{{EntityType: EntityAccount, EntityID: "10"}}, store.queries)
}

func TestResolverNoBindingReturnsDirect(t *testing.T) {
	store := &fakeBindingStore{bindings: map[Query]*Binding{}}
	resolver := Resolver{
		Bindings: store,
		Profiles: fakeProfileStore{},
		Runtime:  fakeRuntimeManager{},
	}

	resolved, err := resolver.Resolve(context.Background(), outbound.Scope{ProviderID: "openai"})
	require.NoError(t, err)
	require.Equal(t, "direct", resolved.Mode)
	require.Empty(t, resolved.ProxyURL)
	require.Equal(t, []Query{
		{EntityType: EntityProvider, EntityID: "openai"},
		{EntityType: EntityGlobal, EntityID: "default"},
	}, store.queries)
}

func TestResolverBoundProxyFailureDoesNotFallbackByDefault(t *testing.T) {
	resolver := Resolver{
		Bindings: &fakeBindingStore{bindings: map[Query]*Binding{
			{EntityType: EntityGlobal, EntityID: "default"}: {ID: 1, EntityType: EntityGlobal, EntityID: "default", ProfileID: 100, Enabled: true},
		}},
		Profiles: fakeProfileStore{profiles: map[int64]*profile.Profile{
			100: {ID: 100, Strategy: profile.StrategySelect, Status: profile.StatusActive},
		}},
		Runtime: fakeRuntimeManager{err: errors.New("mihomo failed")},
	}

	_, err := resolver.Resolve(context.Background(), outbound.Scope{})
	require.ErrorContains(t, err, "mihomo failed")
}

func TestResolverFallbackToDirectWhenExplicit(t *testing.T) {
	resolver := Resolver{
		Bindings: &fakeBindingStore{bindings: map[Query]*Binding{
			{EntityType: EntityGlobal, EntityID: "default"}: {
				ID:               1,
				EntityType:       EntityGlobal,
				EntityID:         "default",
				ProfileID:        100,
				Enabled:          true,
				FallbackToDirect: true,
			},
		}},
		Profiles: fakeProfileStore{profiles: map[int64]*profile.Profile{
			100: {ID: 100, Strategy: profile.StrategySelect, Status: profile.StatusActive},
		}},
		Runtime: fakeRuntimeManager{err: errors.New("mihomo failed")},
	}

	resolved, err := resolver.Resolve(context.Background(), outbound.Scope{})
	require.NoError(t, err)
	require.Equal(t, "direct", resolved.Mode)
	require.Equal(t, true, resolved.Metadata["fallback_to_direct"])
}
