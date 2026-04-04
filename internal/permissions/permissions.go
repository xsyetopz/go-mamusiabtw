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
	Discord DiscordPermissions `json:"discord"`
	Network NetworkPermissions `json:"network"`

	// Automation covers non-interaction triggers: scheduled jobs and gateway events.
	Automation AutomationPermissions `json:"automation"`
}

type StoragePermissions struct {
	KV           bool `json:"kv"`
	UserSettings bool `json:"user_settings"`
	CheckIns     bool `json:"checkins"`
	Reminders    bool `json:"reminders"`
}

type DiscordPermissions struct {
	SendChannel bool `json:"send_channel"`
	SendDM      bool `json:"send_dm"`
}

type NetworkPermissions struct {
	HTTP bool `json:"http"`
}

type AutomationPermissions struct {
	// Jobs allows scheduled triggers declared by a plugin manifest.
	Jobs bool `json:"jobs"`

	Events AutomationEventPermissions `json:"events"`
}

type AutomationEventPermissions struct {
	// MemberJoinLeave covers member join/leave hooks.
	MemberJoinLeave bool `json:"member_join_leave"`

	// Moderation covers ban/unban hooks.
	Moderation bool `json:"moderation"`
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
			KV:           requested.Storage.KV && granted.Storage.KV,
			UserSettings: requested.Storage.UserSettings && granted.Storage.UserSettings,
			CheckIns:     requested.Storage.CheckIns && granted.Storage.CheckIns,
			Reminders:    requested.Storage.Reminders && granted.Storage.Reminders,
		},
		Discord: DiscordPermissions{
			SendChannel: requested.Discord.SendChannel && granted.Discord.SendChannel,
			SendDM:      requested.Discord.SendDM && granted.Discord.SendDM,
		},
		Network: NetworkPermissions{
			HTTP: requested.Network.HTTP && granted.Network.HTTP,
		},
		Automation: AutomationPermissions{
			Jobs: requested.Automation.Jobs && granted.Automation.Jobs,
			Events: AutomationEventPermissions{
				MemberJoinLeave: requested.Automation.Events.MemberJoinLeave &&
					granted.Automation.Events.MemberJoinLeave,
				Moderation: requested.Automation.Events.Moderation && granted.Automation.Events.Moderation,
			},
		},
	}
}

func merge(base, override Permissions) Permissions {
	// For v1, simple boolean OR for "grants" is sufficient.
	out := base
	out.Storage.KV = out.Storage.KV || override.Storage.KV
	out.Storage.UserSettings = out.Storage.UserSettings || override.Storage.UserSettings
	out.Storage.CheckIns = out.Storage.CheckIns || override.Storage.CheckIns
	out.Storage.Reminders = out.Storage.Reminders || override.Storage.Reminders
	out.Discord.SendChannel = out.Discord.SendChannel || override.Discord.SendChannel
	out.Discord.SendDM = out.Discord.SendDM || override.Discord.SendDM
	out.Network.HTTP = out.Network.HTTP || override.Network.HTTP
	out.Automation.Jobs = out.Automation.Jobs || override.Automation.Jobs
	out.Automation.Events.MemberJoinLeave = out.Automation.Events.MemberJoinLeave ||
		override.Automation.Events.MemberJoinLeave
	out.Automation.Events.Moderation = out.Automation.Events.Moderation || override.Automation.Events.Moderation
	return out
}

func (p Permissions) Validate() error {
	return nil
}
