package permissions

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Permissions are host-enforced capabilities.
//
// Security model: a plugin can only do what it both (1) requests in plugin.json
// and (2) is granted in the host permissions policy. Everything is denied by
// default.
type Permissions struct {
	Storage StoragePermissions `json:"storage"`
}

type StoragePermissions struct {
	KV bool `json:"kv"`
}

// Policy is a Claude-style central permissions file: defaults + per-plugin overrides.
type Policy struct {
	Defaults Permissions            `json:"defaults"`
	Plugins  map[string]Permissions `json:"plugins"`
	Version  string                 `json:"version,omitempty"`
}

func LoadPolicyFile(path string) (Policy, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Policy{}, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Policy{}, nil
		}
		return Policy{}, fmt.Errorf("read permissions policy %q: %w", path, err)
	}

	var p Policy
	if unmarshalErr := json.Unmarshal(b, &p); unmarshalErr != nil {
		return Policy{}, fmt.Errorf("parse permissions policy %q: %w", path, unmarshalErr)
	}
	if p.Plugins == nil {
		p.Plugins = map[string]Permissions{}
	}
	return p, nil
}

func (p Policy) Granted(pluginID string) Permissions {
	g := p.Defaults
	if strings.TrimSpace(pluginID) == "" {
		return g
	}
	if override, ok := p.Plugins[pluginID]; ok {
		g = merge(g, override)
	}
	return g
}

// Effective returns the permissions a plugin actually gets: requested ∩ granted.
func Effective(requested, granted Permissions) Permissions {
	return Permissions{
		Storage: StoragePermissions{
			KV: requested.Storage.KV && granted.Storage.KV,
		},
	}
}

func merge(base, override Permissions) Permissions {
	// For v1, simple boolean OR for "grants" is sufficient.
	out := base
	out.Storage.KV = out.Storage.KV || override.Storage.KV
	return out
}

func (p Permissions) Validate() error {
	return nil
}
