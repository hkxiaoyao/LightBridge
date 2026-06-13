package outbound

import (
	"context"
	"errors"
	"strings"
	"sync"
)

type ScopeType string

const (
	ScopeGlobal   ScopeType = "global"
	ScopeProvider ScopeType = "provider"
	ScopeChannel  ScopeType = "channel"
	ScopeAccount  ScopeType = "account"
	ScopeRequest  ScopeType = "request"
)

type Scope struct {
	Type       ScopeType      `json:"type,omitempty"`
	ProviderID string         `json:"provider_id,omitempty"`
	ChannelID  int64          `json:"channel_id,omitempty"`
	AccountID  int64          `json:"account_id,omitempty"`
	APIKeyID   int64          `json:"api_key_id,omitempty"`
	UserID     int64          `json:"user_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type ResolvedOutbound struct {
	Mode      string         `json:"mode"`
	ProxyURL  string         `json:"proxy_url,omitempty"`
	AdapterID string         `json:"adapter_id,omitempty"`
	ProfileID int64          `json:"profile_id,omitempty"`
	NodeName  string         `json:"node_name,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type Resolver interface {
	Resolve(context.Context, Scope) (*ResolvedOutbound, error)
}

type NoopResolver struct{}

func (NoopResolver) Resolve(context.Context, Scope) (*ResolvedOutbound, error) {
	return &ResolvedOutbound{Mode: "direct"}, nil
}

type Registry struct {
	mu        sync.RWMutex
	resolvers map[string]Resolver
}

func NewRegistry() *Registry {
	return &Registry{resolvers: map[string]Resolver{}}
}

func (r *Registry) Register(adapterID string, resolver Resolver) error {
	if r == nil {
		return errors.New("outbound registry is nil")
	}
	id := strings.TrimSpace(adapterID)
	if id == "" {
		return errors.New("outbound adapter id is required")
	}
	if resolver == nil {
		return errors.New("outbound resolver is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resolvers[id] = resolver
	return nil
}

func (r *Registry) Unregister(adapterID string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.resolvers, strings.TrimSpace(adapterID))
}

func (r *Registry) ResolveAdapter(adapterID string) (Resolver, error) {
	if r == nil {
		return nil, errors.New("outbound registry is nil")
	}
	id := strings.TrimSpace(adapterID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	resolver := r.resolvers[id]
	if resolver == nil {
		return nil, errors.New("outbound adapter is not registered")
	}
	return resolver, nil
}

func (r *Registry) IDs() []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.resolvers))
	for id := range r.resolvers {
		ids = append(ids, id)
	}
	return ids
}

func ProxyURLFromResolved(legacyProxyURL string, resolved *ResolvedOutbound) string {
	if resolved == nil {
		return strings.TrimSpace(legacyProxyURL)
	}
	switch strings.TrimSpace(strings.ToLower(resolved.Mode)) {
	case "", "direct":
		return ""
	case "proxy":
		return strings.TrimSpace(resolved.ProxyURL)
	default:
		return strings.TrimSpace(resolved.ProxyURL)
	}
}
