package service

import (
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/WilliamWang1721/LightBridge/internal/config"
	"github.com/WilliamWang1721/LightBridge/internal/modules"
	proxymodule "github.com/WilliamWang1721/LightBridge/internal/modules/proxy"
	"github.com/WilliamWang1721/LightBridge/internal/outbound"
)

type CoreBridge struct{}

func ProvideCoreBridge(_ modules.Store, _ modules.Store, _ UserRepository, _ AccountRepository) *CoreBridge {
	return &CoreBridge{}
}
func ProvideModuleInstaller(cfg *config.Config, store modules.Store, _ BuildInfo) modules.Installer {
	dataDir := "data"
	if cfg != nil && cfg.Modules.DataDir != "" {
		dataDir = cfg.Modules.DataDir
	}
	var verifier modules.SignatureVerifier
	if cfg != nil && strings.TrimSpace(cfg.Modules.SignaturePublicKeyPath) != "" {
		if v, err := modules.NewEd25519SignatureVerifierFromFile(strings.TrimSpace(cfg.Modules.SignaturePublicKeyPath)); err == nil {
			verifier = v
		}
	}
	return modules.NewPackageInstallerWithVerifier(dataDir, store, verifier)
}
func ProvideProviderRuntime(_ *config.Config, registry *modules.ProviderRegistry, _ modules.Store, _ *CoreBridge) modules.ProviderRuntime {
	return modules.NewProcessProviderRuntime(registry)
}

func ProvideOutboundRegistry() *outbound.Registry {
	return outbound.NewRegistry()
}

func ProvideOutboundRuntime(_ *config.Config, registry *outbound.Registry, db *sql.DB) modules.OutboundRuntime {
	return proxymodule.NewOutboundRuntime(registry, db)
}

func ProvideProxyModuleService(cfg *config.Config, db *sql.DB) *proxymodule.Service {
	dataDir := "data"
	binaryPath := ""
	runtimeDir := ""
	if cfg != nil {
		if strings.TrimSpace(cfg.Modules.DataDir) != "" {
			dataDir = strings.TrimSpace(cfg.Modules.DataDir)
		}
		binaryPath = strings.TrimSpace(cfg.Modules.Proxy.MihomoBinaryPath)
		runtimeDir = strings.TrimSpace(cfg.Modules.Proxy.RuntimeDir)
	}
	if runtimeDir == "" {
		runtimeDir = filepath.Join(dataDir, "modules", "lightbridge.proxy", "runtime")
	}
	return proxymodule.NewServiceWithRuntime(db, binaryPath, runtimeDir)
}
