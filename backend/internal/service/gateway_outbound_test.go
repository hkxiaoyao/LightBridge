//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/outbound"
	"github.com/stretchr/testify/require"
)

type gatewayOutboundResolverStub struct {
	scope outbound.Scope
}

func (s *gatewayOutboundResolverStub) Resolve(_ context.Context, scope outbound.Scope) (*outbound.ResolvedOutbound, error) {
	s.scope = scope
	return &outbound.ResolvedOutbound{Mode: "proxy", ProxyURL: "http://127.0.0.1:17001"}, nil
}

func TestGatewayResolveAccountProxyURL_UsesOutboundRegistryWithChannelScope(t *testing.T) {
	resolver := &gatewayOutboundResolverStub{}
	registry := outbound.NewRegistry()
	require.NoError(t, registry.Register(lightbridgeProxyAdapterID, resolver))
	channelSvc := NewChannelService(&mockChannelRepository{
		listAllFn: func(context.Context) ([]Channel, error) {
			return []Channel{{ID: 77, Status: StatusActive, GroupIDs: []int64{42}}}, nil
		},
	}, nil, nil, nil)

	svc := &GatewayService{outboundRegistry: registry, channelService: channelSvc}
	groupID := int64(42)
	proxyURL, err := svc.resolveAccountProxyURL(context.Background(), &Account{ID: 12}, "anthropic", &groupID)
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:17001", proxyURL)
	require.Equal(t, outbound.ScopeAccount, resolver.scope.Type)
	require.Equal(t, "anthropic", resolver.scope.ProviderID)
	require.Equal(t, int64(77), resolver.scope.ChannelID)
	require.Equal(t, int64(12), resolver.scope.AccountID)
}

func TestGatewayResolveAccountProxyURL_FallsBackToLegacyProxy(t *testing.T) {
	proxy := &Proxy{Protocol: "http", Host: "proxy.local", Port: 8080}
	proxyID := int64(1)
	svc := &GatewayService{outboundRegistry: outbound.NewRegistry()}

	proxyURL, err := svc.resolveAccountProxyURL(context.Background(), &Account{ID: 12, ProxyID: &proxyID, Proxy: proxy}, "anthropic", nil)
	require.NoError(t, err)
	require.Equal(t, "http://proxy.local:8080", proxyURL)
}
