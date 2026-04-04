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
	Warnings     bool `json:"warnings"`
	Audit        bool `json:"audit"`
}

type DiscordPermissions struct {
	SendChannel        bool `json:"send_channel"`
	SendDM             bool `json:"send_dm"`
	TimeoutMember      bool `json:"timeout_member"`
	SetSlowmode        bool `json:"set_slowmode"`
	SetNickname        bool `json:"set_nickname"`
	CreateRole         bool `json:"create_role"`
	EditRole           bool `json:"edit_role"`
	DeleteRole         bool `json:"delete_role"`
	AddRole            bool `json:"add_role"`
	RemoveRole         bool `json:"remove_role"`
	ListMessages       bool `json:"list_messages"`
	DeleteMessage      bool `json:"delete_message"`
	BulkDeleteMessages bool `json:"bulk_delete_messages"`
	PurgeMessages      bool `json:"purge_messages"`
	CreateEmoji        bool `json:"create_emoji"`
	EditEmoji          bool `json:"edit_emoji"`
	DeleteEmoji        bool `json:"delete_emoji"`
	CreateSticker      bool `json:"create_sticker"`
	EditSticker        bool `json:"edit_sticker"`
	DeleteSticker      bool `json:"delete_sticker"`
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
			Warnings:     requested.Storage.Warnings && granted.Storage.Warnings,
			Audit:        requested.Storage.Audit && granted.Storage.Audit,
		},
		Discord: DiscordPermissions{
			SendChannel:        requested.Discord.SendChannel && granted.Discord.SendChannel,
			SendDM:             requested.Discord.SendDM && granted.Discord.SendDM,
			TimeoutMember:      requested.Discord.TimeoutMember && granted.Discord.TimeoutMember,
			SetSlowmode:        requested.Discord.SetSlowmode && granted.Discord.SetSlowmode,
			SetNickname:        requested.Discord.SetNickname && granted.Discord.SetNickname,
			CreateRole:         requested.Discord.CreateRole && granted.Discord.CreateRole,
			EditRole:           requested.Discord.EditRole && granted.Discord.EditRole,
			DeleteRole:         requested.Discord.DeleteRole && granted.Discord.DeleteRole,
			AddRole:            requested.Discord.AddRole && granted.Discord.AddRole,
			RemoveRole:         requested.Discord.RemoveRole && granted.Discord.RemoveRole,
			ListMessages:       requested.Discord.ListMessages && granted.Discord.ListMessages,
			DeleteMessage:      requested.Discord.DeleteMessage && granted.Discord.DeleteMessage,
			BulkDeleteMessages: requested.Discord.BulkDeleteMessages && granted.Discord.BulkDeleteMessages,
			PurgeMessages:      requested.Discord.PurgeMessages && granted.Discord.PurgeMessages,
			CreateEmoji:        requested.Discord.CreateEmoji && granted.Discord.CreateEmoji,
			EditEmoji:          requested.Discord.EditEmoji && granted.Discord.EditEmoji,
			DeleteEmoji:        requested.Discord.DeleteEmoji && granted.Discord.DeleteEmoji,
			CreateSticker:      requested.Discord.CreateSticker && granted.Discord.CreateSticker,
			EditSticker:        requested.Discord.EditSticker && granted.Discord.EditSticker,
			DeleteSticker:      requested.Discord.DeleteSticker && granted.Discord.DeleteSticker,
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
	out.Storage.Warnings = out.Storage.Warnings || override.Storage.Warnings
	out.Storage.Audit = out.Storage.Audit || override.Storage.Audit
	out.Discord.SendChannel = out.Discord.SendChannel || override.Discord.SendChannel
	out.Discord.SendDM = out.Discord.SendDM || override.Discord.SendDM
	out.Discord.TimeoutMember = out.Discord.TimeoutMember || override.Discord.TimeoutMember
	out.Discord.SetSlowmode = out.Discord.SetSlowmode || override.Discord.SetSlowmode
	out.Discord.SetNickname = out.Discord.SetNickname || override.Discord.SetNickname
	out.Discord.CreateRole = out.Discord.CreateRole || override.Discord.CreateRole
	out.Discord.EditRole = out.Discord.EditRole || override.Discord.EditRole
	out.Discord.DeleteRole = out.Discord.DeleteRole || override.Discord.DeleteRole
	out.Discord.AddRole = out.Discord.AddRole || override.Discord.AddRole
	out.Discord.RemoveRole = out.Discord.RemoveRole || override.Discord.RemoveRole
	out.Discord.ListMessages = out.Discord.ListMessages || override.Discord.ListMessages
	out.Discord.DeleteMessage = out.Discord.DeleteMessage || override.Discord.DeleteMessage
	out.Discord.BulkDeleteMessages = out.Discord.BulkDeleteMessages || override.Discord.BulkDeleteMessages
	out.Discord.PurgeMessages = out.Discord.PurgeMessages || override.Discord.PurgeMessages
	out.Discord.CreateEmoji = out.Discord.CreateEmoji || override.Discord.CreateEmoji
	out.Discord.EditEmoji = out.Discord.EditEmoji || override.Discord.EditEmoji
	out.Discord.DeleteEmoji = out.Discord.DeleteEmoji || override.Discord.DeleteEmoji
	out.Discord.CreateSticker = out.Discord.CreateSticker || override.Discord.CreateSticker
	out.Discord.EditSticker = out.Discord.EditSticker || override.Discord.EditSticker
	out.Discord.DeleteSticker = out.Discord.DeleteSticker || override.Discord.DeleteSticker
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
