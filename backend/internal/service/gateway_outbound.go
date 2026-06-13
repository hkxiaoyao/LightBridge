package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/LightBridge/internal/outbound"
	"github.com/gin-gonic/gin"
)

const lightbridgeProxyAdapterID = "lightbridge.proxy"

func (s *GatewayService) resolveAccountProxyURL(ctx context.Context, account *Account, providerID string, groupID *int64) (string, error) {
	if s == nil {
		return legacyAccountProxyURL(account), nil
	}
	return resolveAccountProxyURLWithOutbound(ctx, account, providerID, groupID, s.outboundRegistry, s.channelService)
}

func (s *OpenAIGatewayService) resolveAccountProxyURL(ctx context.Context, account *Account, providerID string, groupID *int64) (string, error) {
	if s == nil {
		return legacyAccountProxyURL(account), nil
	}
	return resolveAccountProxyURLWithOutbound(ctx, account, providerID, groupID, s.outboundRegistry, s.channelService)
}

func (s *GeminiMessagesCompatService) resolveAccountProxyURL(ctx context.Context, account *Account, providerID string, groupID *int64) (string, error) {
	if s == nil {
		return legacyAccountProxyURL(account), nil
	}
	return resolveAccountProxyURLWithOutbound(ctx, account, providerID, groupID, s.outboundRegistry, s.channelService)
}

func (s *AntigravityGatewayService) resolveAccountProxyURL(ctx context.Context, account *Account, providerID string, groupID *int64) (string, error) {
	if s == nil {
		return legacyAccountProxyURL(account), nil
	}
	return resolveAccountProxyURLWithOutbound(ctx, account, providerID, groupID, s.outboundRegistry, s.channelService)
}

func (s *AccountTestService) resolveAccountProxyURL(ctx context.Context, account *Account, providerID string, groupID *int64) (string, error) {
	if s == nil {
		return legacyAccountProxyURL(account), nil
	}
	return resolveAccountProxyURLWithOutbound(ctx, account, providerID, groupID, s.outboundRegistry, s.channelService)
}

func (s *AccountTestService) resolveAccountTestProxyURL(c *gin.Context, ctx context.Context, account *Account) (string, error) {
	return s.resolveAccountProxyURL(ctx, account, account.Platform, apiKeyGroupID(getAPIKeyFromContext(c)))
}

func resolveAccountProxyURLWithOutbound(ctx context.Context, account *Account, providerID string, groupID *int64, registry *outbound.Registry, channelService *ChannelService) (string, error) {
	if account == nil {
		return "", nil
	}
	if registry != nil {
		if resolver, err := registry.ResolveAdapter(lightbridgeProxyAdapterID); err == nil && resolver != nil {
			channelID, err := resolveOutboundChannelID(ctx, groupID, channelService)
			if err != nil {
				return "", err
			}
			resolved, err := resolver.Resolve(ctx, outbound.Scope{
				Type:       outbound.ScopeAccount,
				ProviderID: strings.TrimSpace(providerID),
				ChannelID:  channelID,
				AccountID:  account.ID,
			})
			if err != nil {
				return "", err
			}
			return outbound.ProxyURLFromResolved("", resolved), nil
		}
	}
	return legacyAccountProxyURL(account), nil
}

func legacyAccountProxyURL(account *Account) string {
	if account == nil {
		return ""
	}
	if account.ProxyID != nil && account.Proxy != nil {
		return account.Proxy.URL()
	}
	return ""
}

func resolveOutboundChannelID(ctx context.Context, groupID *int64, channelService *ChannelService) (int64, error) {
	if channelService == nil || groupID == nil || *groupID <= 0 {
		return 0, nil
	}
	channel, err := channelService.GetChannelForGroup(ctx, *groupID)
	if err != nil {
		return 0, err
	}
	if channel == nil {
		return 0, nil
	}
	return channel.ID, nil
}

func apiKeyGroupID(apiKey *APIKey) *int64 {
	if apiKey == nil {
		return nil
	}
	return apiKey.GroupID
}
