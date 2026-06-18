package binding

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/WilliamWang1721/LightBridge/internal/modules/proxy/internal/profile"
	proxyruntime "github.com/WilliamWang1721/LightBridge/internal/modules/proxy/internal/runtime"
	"github.com/WilliamWang1721/LightBridge/internal/outbound"
)

const AdapterID = "lightbridge.proxy"

type BindingStore interface {
	FindEnabledBinding(context.Context, Query) (*Binding, error)
}

type ProfileStore interface {
	GetProfile(context.Context, int64) (*profile.Profile, error)
}

type RuntimeManager interface {
	EnsureRunning(context.Context, profile.Profile) (*proxyruntime.Instance, error)
}

type Resolver struct {
	Bindings BindingStore
	Profiles ProfileStore
	Runtime  RuntimeManager
}

func (r *Resolver) Resolve(ctx context.Context, scope outbound.Scope) (*outbound.ResolvedOutbound, error) {
	if r == nil || r.Bindings == nil || r.Profiles == nil || r.Runtime == nil {
		return nil, errors.New("proxy resolver is not configured")
	}
	for _, query := range QueriesForScope(scope) {
		binding, err := r.Bindings.FindEnabledBinding(ctx, query)
		if err != nil {
			return nil, err
		}
		if binding == nil {
			continue
		}
		resolved, err := r.resolveBinding(ctx, *binding)
		if err == nil {
			return resolved, nil
		}
		if binding.FallbackToDirect {
			return &outbound.ResolvedOutbound{
				Mode:      "direct",
				AdapterID: AdapterID,
				ProfileID: binding.ProfileID,
				Metadata:  map[string]any{"fallback_to_direct": true, "proxy_error": err.Error()},
			}, nil
		}
		return nil, err
	}
	return &outbound.ResolvedOutbound{Mode: "direct"}, nil
}

func (r *Resolver) resolveBinding(ctx context.Context, binding Binding) (*outbound.ResolvedOutbound, error) {
	if binding.ProfileID <= 0 {
		return nil, errors.New("proxy binding profile id is required")
	}
	prof, err := r.Profiles.GetProfile(ctx, binding.ProfileID)
	if err != nil {
		return nil, err
	}
	if prof == nil {
		return nil, fmt.Errorf("proxy profile %d was not found", binding.ProfileID)
	}
	if err := prof.ValidateActive(); err != nil {
		return nil, err
	}
	instance, err := r.Runtime.EnsureRunning(ctx, *prof)
	if err != nil {
		return nil, err
	}
	proxyURL, err := instance.ProxyURL()
	if err != nil {
		return nil, err
	}
	return &outbound.ResolvedOutbound{
		Mode:      "proxy",
		ProxyURL:  strings.TrimSpace(proxyURL),
		AdapterID: AdapterID,
		ProfileID: prof.ID,
		Metadata: map[string]any{
			"binding_id":  binding.ID,
			"entity_type": string(binding.EntityType),
			"entity_id":   binding.EntityID,
		},
	}, nil
}
