package modules

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateManifest_AllowsOutboundProxyModule(t *testing.T) {
	manifest := Manifest{
		APIVersion: ManifestAPIVersionV1Alpha1,
		ID:         "lightbridge.proxy",
		Name:       "LightBridge Proxy",
		Type:       ModuleTypeOutbound,
		Version:    "0.1.0",
		Capabilities: []Capability{
			CapabilityOutboundAdapter,
			CapabilityUIAdminRoute,
			CapabilityUIEntityPanel,
			CapabilityEntityBinding,
		},
		Backend: &BackendSpec{Entrypoints: map[string]string{"outbound": "./proxy-module"}},
	}

	require.NoError(t, ValidateManifest(manifest))
}

func TestValidateManifest_RejectsOutboundAdapterWithoutBackend(t *testing.T) {
	manifest := Manifest{
		APIVersion:   ManifestAPIVersionV1Alpha1,
		ID:           "lightbridge.proxy",
		Name:         "LightBridge Proxy",
		Type:         ModuleTypeOutbound,
		Version:      "0.1.0",
		Capabilities: []Capability{CapabilityOutboundAdapter},
	}

	require.ErrorContains(t, ValidateManifest(manifest), "outbound.adapter requires backend spec")
}

func TestValidateManifest_RejectsUnsupportedModuleType(t *testing.T) {
	manifest := Manifest{
		APIVersion: ManifestAPIVersionV1Alpha1,
		ID:         "lightbridge.unknown",
		Name:       "Unknown",
		Type:       ModuleType("unknown"),
		Version:    "0.1.0",
	}

	require.ErrorContains(t, ValidateManifest(manifest), "unsupported module type")
}
