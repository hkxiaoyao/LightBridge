package proxy

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/Wei-Shaw/LightBridge/internal/modules"
	proxybinding "github.com/Wei-Shaw/LightBridge/internal/modules/proxy/internal/binding"
	proxyruntime "github.com/Wei-Shaw/LightBridge/internal/modules/proxy/internal/runtime"
	"github.com/Wei-Shaw/LightBridge/internal/outbound"
)

type OutboundRuntime struct {
	registry *outbound.Registry
	db       *sql.DB
}

func NewOutboundRuntime(registry *outbound.Registry, db *sql.DB) *OutboundRuntime {
	return &OutboundRuntime{registry: registry, db: db}
}

func (r *OutboundRuntime) StartOutbound(ctx context.Context, module modules.InstalledModule) error {
	if r == nil || r.registry == nil {
		return errors.New("outbound registry is not configured")
	}
	if module.ID != proxybinding.AdapterID {
		return nil
	}
	if r.db == nil {
		return errors.New("proxy outbound runtime database is not configured")
	}
	bindingStore := proxybinding.NewSQLStore(r.db)
	runtimeStore := proxyruntime.NewSQLStore(r.db)
	resolver := &proxybinding.Resolver{
		Bindings: bindingStore,
		Profiles: bindingStore,
		Runtime:  proxyruntime.ExistingRuntimeManager{Store: runtimeStore},
	}
	return r.registry.Register(module.ID, resolver)
}

func (r *OutboundRuntime) StopOutbound(_ context.Context, id string) error {
	if r == nil || r.registry == nil {
		return nil
	}
	if strings.TrimSpace(id) == proxybinding.AdapterID {
		r.registry.Unregister(id)
		if r.db != nil {
			return proxyruntime.NewSQLStore(r.db).MarkAllRunningStopped(context.Background())
		}
	}
	return nil
}
