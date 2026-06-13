package outbound

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoopResolverReturnsDirect(t *testing.T) {
	resolved, err := NoopResolver{}.Resolve(context.Background(), Scope{Type: ScopeGlobal})
	require.NoError(t, err)
	require.Equal(t, "direct", resolved.Mode)
	require.Empty(t, resolved.ProxyURL)
}

func TestRegistryRegisterResolveAndUnregister(t *testing.T) {
	registry := NewRegistry()
	resolver := NoopResolver{}

	require.NoError(t, registry.Register(" lightbridge.proxy ", resolver))
	require.ElementsMatch(t, []string{"lightbridge.proxy"}, registry.IDs())

	got, err := registry.ResolveAdapter("lightbridge.proxy")
	require.NoError(t, err)
	require.NotNil(t, got)

	registry.Unregister("lightbridge.proxy")
	_, err = registry.ResolveAdapter("lightbridge.proxy")
	require.Error(t, err)
}

func TestProxyURLFromResolved(t *testing.T) {
	require.Equal(t, "http://legacy:8080", ProxyURLFromResolved(" http://legacy:8080 ", nil))
	require.Empty(t, ProxyURLFromResolved("http://legacy:8080", &ResolvedOutbound{Mode: "direct"}))
	require.Equal(t, "http://127.0.0.1:7890", ProxyURLFromResolved("", &ResolvedOutbound{Mode: "proxy", ProxyURL: " http://127.0.0.1:7890 "}))
}
