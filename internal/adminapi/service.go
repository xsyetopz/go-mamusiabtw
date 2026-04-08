package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/xsyetopz/go-mamusiabtw/internal/buildinfo"
	commandapi "github.com/xsyetopz/go-mamusiabtw/internal/commands/api"
	"github.com/xsyetopz/go-mamusiabtw/internal/config"
	"github.com/xsyetopz/go-mamusiabtw/internal/guildconfig"
	migrate "github.com/xsyetopz/go-mamusiabtw/internal/migration"
	"github.com/xsyetopz/go-mamusiabtw/internal/ops"
	"github.com/xsyetopz/go-mamusiabtw/internal/permissions"
	pluginhost "github.com/xsyetopz/go-mamusiabtw/internal/runtime/plugins"
	pluginhostlua "github.com/xsyetopz/go-mamusiabtw/internal/runtime/plugins/lua"
)

var pluginIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}$`)

type Service struct {
	Logger *slog.Logger
	Config config.Config

	Snapshot      func() ops.Snapshot
	ModuleAdmin   commandapi.ModuleAdmin
	PluginAdmin   commandapi.PluginAdmin
	Store         commandapi.Store
	BuildInfo     func() buildinfo.Info
	OAuth         OAuthClient
	OwnerStatus   func() OwnerStatus
	KnownGuildIDs func() []uint64
	BotHasGuild   func(ctx context.Context, guildID uint64) (bool, error)

	ListGuildChannels  func(ctx context.Context, guildID uint64) ([]GuildChannelInfo, error)
	ListGuildRoles     func(ctx context.Context, guildID uint64) ([]GuildRoleInfo, error)
	SearchGuildMembers func(ctx context.Context, guildID uint64, query string, limit int) ([]GuildMemberInfo, error)
	ListGuildEmojis    func(ctx context.Context, guildID uint64) ([]GuildEmojiInfo, error)
	ListGuildStickers  func(ctx context.Context, guildID uint64) ([]GuildStickerInfo, error)

	SetSlowmode         func(ctx context.Context, channelID uint64, seconds int) error
	SetNickname         func(ctx context.Context, guildID, userID uint64, nickname *string) error
	TimeoutMember       func(ctx context.Context, guildID, userID uint64, untilUnix int64) error
	CreateRole          func(ctx context.Context, spec pluginhostlua.RoleCreateSpec) (pluginhostlua.RoleResult, error)
	EditRole            func(ctx context.Context, spec pluginhostlua.RoleEditSpec) (pluginhostlua.RoleResult, error)
	DeleteRole          func(ctx context.Context, guildID, roleID uint64) error
	AddRole             func(ctx context.Context, spec pluginhostlua.RoleMemberSpec) error
	RemoveRole          func(ctx context.Context, spec pluginhostlua.RoleMemberSpec) error
	PurgeMessages       func(ctx context.Context, spec pluginhostlua.PurgeSpec) (int, error)
	CreateEmojiUpload   func(ctx context.Context, guildID uint64, name, filename string, body []byte, width, height int) (pluginhostlua.EmojiResult, error)
	EditEmoji           func(ctx context.Context, spec pluginhostlua.EmojiEditSpec) (pluginhostlua.EmojiResult, error)
	DeleteEmoji         func(ctx context.Context, spec pluginhostlua.EmojiDeleteSpec) error
	CreateStickerUpload func(ctx context.Context, guildID uint64, name, description, emojiTag, filename string, body []byte, width, height int) (pluginhostlua.StickerResult, error)
	EditSticker         func(ctx context.Context, spec pluginhostlua.StickerEditSpec) (pluginhostlua.StickerResult, error)
	DeleteSticker       func(ctx context.Context, spec pluginhostlua.StickerDeleteSpec) error
}

type OwnerStatus struct {
	Configured      bool
	Resolved        bool
	Source          string
	EffectiveUserID *uint64
}

func cloneOptionalUint64(value *uint64) *uint64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

type StatusResponse struct {
	Snapshot SnapshotResponse `json:"snapshot"`
	Build    BuildResponse    `json:"build"`
	Config   StatusConfig     `json:"config"`
	Setup    SetupResponse    `json:"setup"`
}

type SnapshotResponse struct {
	Ready               bool   `json:"ready"`
	StartedAt           string `json:"started_at"`
	MigrationVersion    int    `json:"migration_version"`
	ProdMode            bool   `json:"prod_mode"`
	ModuleCount         int    `json:"module_count"`
	EnabledModuleCount  int    `json:"enabled_module_count"`
	PluginCount         int    `json:"plugin_count"`
	EnabledPluginCount  int    `json:"enabled_plugin_count"`
	BuiltinCommandCount int    `json:"builtin_command_count"`
	SlashCommandCount   int    `json:"slash_command_count"`
	UserCommandCount    int    `json:"user_command_count"`
	MessageCommandCount int    `json:"message_command_count"`
	InteractionsTotal   uint64 `json:"interactions_total"`
	InteractionFailures uint64 `json:"interaction_failures"`
	PluginFailures      uint64 `json:"plugin_failures"`
	AutomationFailures  uint64 `json:"automation_failures"`
	ReminderFailures    uint64 `json:"reminder_failures"`
}

type BuildResponse struct {
	Version          string `json:"version"`
	Repository       string `json:"repository,omitempty"`
	Description      string `json:"description,omitempty"`
	DeveloperURL     string `json:"developer_url,omitempty"`
	SupportServerURL string `json:"support_server_url,omitempty"`
	MascotImageURL   string `json:"mascot_image_url,omitempty"`
}

type StatusConfig struct {
	SQLitePath              string  `json:"sqlite_path"`
	MigrationsDir           string  `json:"migrations_dir"`
	MigrationBackupsDir     string  `json:"migration_backups_dir"`
	LocalesDir              string  `json:"locales_dir"`
	PluginsDir              string  `json:"plugins_dir"`
	PermissionsFile         string  `json:"permissions_file"`
	ModulesFile             string  `json:"modules_file"`
	TrustedKeysFile         string  `json:"trusted_keys_file"`
	OpsAddr                 string  `json:"ops_addr"`
	AdminAddr               string  `json:"admin_addr"`
	DevGuildID              *uint64 `json:"dev_guild_id,omitempty"`
	CommandRegistrationMode string  `json:"command_registration_mode"`
	ProdMode                bool    `json:"prod_mode"`
	AllowUnsignedPlugins    bool    `json:"allow_unsigned_plugins"`
}

type SetupResponse struct {
	AdminEnabled          bool     `json:"admin_enabled"`
	AuthConfigured        bool     `json:"auth_configured"`
	LoginReady            bool     `json:"login_ready"`
	OwnerConfigured       bool     `json:"owner_configured"`
	OwnerResolved         bool     `json:"owner_resolved"`
	OwnerSource           string   `json:"owner_source"`
	EffectiveOwnerUserID  *uint64  `json:"effective_owner_user_id,omitempty"`
	SigningConfigured     bool     `json:"signing_configured"`
	TrustedKeysConfigured bool     `json:"trusted_keys_configured"`
	AdminAddr             string   `json:"admin_addr"`
	AppOrigin             string   `json:"app_origin"`
	RedirectURL           string   `json:"redirect_url"`
	InstallRedirectURL    string   `json:"install_redirect_url"`
	HasClientID           bool     `json:"has_client_id"`
	HasClientSecret       bool     `json:"has_client_secret"`
	HasSessionSecret      bool     `json:"has_session_secret"`
	Hints                 []string `json:"hints"`
}

type ModuleResponse struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Kind           string   `json:"kind"`
	Runtime        string   `json:"runtime"`
	Enabled        bool     `json:"enabled"`
	DefaultEnabled bool     `json:"default_enabled"`
	Toggleable     bool     `json:"toggleable"`
	Signed         bool     `json:"signed"`
	Source         string   `json:"source"`
	Commands       []string `json:"commands"`
}

type PluginSummary struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	Commands         []string `json:"commands"`
	Loaded           bool     `json:"loaded"`
	Signed           bool     `json:"signed"`
	HasSignatureFile bool     `json:"has_signature_file"`
}

type TrustedKeysResponse struct {
	FileKeys []TrustedKeyResponse    `json:"file_keys"`
	DBKeys   []TrustedSignerResponse `json:"db_keys"`
}

type TrustedKeyResponse struct {
	KeyID        string `json:"key_id"`
	PublicKeyB64 string `json:"public_key_b64"`
}

type TrustedSignerResponse struct {
	KeyID        string `json:"key_id"`
	PublicKeyB64 string `json:"public_key_b64"`
	AddedAt      string `json:"added_at"`
}

type MigrationStatusResponse struct {
	CurrentVersion int                `json:"current_version"`
	Applied        []MigrationItemDTO `json:"applied"`
	Pending        []MigrationItemDTO `json:"pending"`
}

type MigrationItemDTO struct {
	Version int    `json:"version"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
}

type PluginScaffoldRequest struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	Version            string                  `json:"version"`
	Locale             string                  `json:"locale"`
	CommandName        string                  `json:"command_name"`
	CommandDescription string                  `json:"command_description"`
	ResponseMessage    string                  `json:"response_message"`
	Permissions        permissions.Permissions `json:"permissions"`
	Sign               bool                    `json:"sign"`
}

type PluginScaffoldResponse struct {
	ID        string   `json:"id"`
	Dir       string   `json:"dir"`
	Files     []string `json:"files"`
	Signed    bool     `json:"signed"`
	Signature string   `json:"signature,omitempty"`
}

type SessionResponse struct {
	Authenticated bool `json:"authenticated"`
	User          struct {
		ID        uint64 `json:"id"`
		Username  string `json:"username"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url,omitempty"`
	} `json:"user"`
	IsOwner   bool   `json:"is_owner"`
	CSRFToken string `json:"csrf_token"`
}

type UserGuildSummary struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	IconURL      string `json:"icon_url,omitempty"`
	Owner        bool   `json:"owner"`
	CanManage    bool   `json:"can_manage"`
	BotInstalled bool   `json:"bot_installed"`
}

type GuildDashboardResponse struct {
	Guild       UserGuildSummary  `json:"guild"`
	InstallURL  string            `json:"install_url"`
	SetupChecks []SetupCheck      `json:"setup_checks"`
	Manager     ManagerSection    `json:"manager"`
	Moderation  ModerationSection `json:"moderation"`
	Fun         PluginSection     `json:"fun"`
	Info        PluginSection     `json:"info"`
	Wellness    WellnessSection   `json:"wellness"`
}

type SetupCheck struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type PluginCommandState struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
}

type PluginSection struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	Enabled       bool                 `json:"enabled"`
	GlobalEnabled bool                 `json:"global_enabled"`
	Commands      []PluginCommandState `json:"commands"`
}

type ManagerSection struct {
	PluginSection
	ChannelCount int `json:"channel_count"`
	RoleCount    int `json:"role_count"`
	EmojiCount   int `json:"emoji_count"`
	StickerCount int `json:"sticker_count"`
}

type ModerationSection struct {
	PluginSection
	WarningLimit     int `json:"warning_limit"`
	TimeoutThreshold int `json:"timeout_threshold"`
	TimeoutMinutes   int `json:"timeout_minutes"`
}

type WellnessSection struct {
	PluginSection
	AllowChannelReminders    bool   `json:"allow_channel_reminders"`
	DefaultReminderChannelID uint64 `json:"default_reminder_channel_id,omitempty"`
}

type GuildChannelInfo struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	ParentID uint64 `json:"parent_id,omitempty"`
}

type GuildRoleInfo struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Position    int    `json:"position"`
	Managed     bool   `json:"managed"`
	Mentionable bool   `json:"mentionable"`
}

type GuildMemberInfo struct {
	UserID      uint64   `json:"user_id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	AvatarURL   string   `json:"avatar_url,omitempty"`
	Bot         bool     `json:"bot"`
	JoinedAt    int64    `json:"joined_at,omitempty"`
	RoleIDs     []uint64 `json:"role_ids"`
}

type GuildEmojiInfo struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	Animated bool   `json:"animated"`
}

type GuildStickerInfo struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Tags        string `json:"tags,omitempty"`
}

type WarningInfo struct {
	ID          string `json:"id"`
	UserID      uint64 `json:"user_id"`
	ModeratorID uint64 `json:"moderator_id"`
	Reason      string `json:"reason"`
	CreatedAt   string `json:"created_at"`
}

func (s Service) Status(ctx context.Context) (StatusResponse, error) {
	resp := StatusResponse{
		Config: StatusConfig{
			SQLitePath:              s.Config.SQLitePath,
			MigrationsDir:           s.Config.Migrations,
			MigrationBackupsDir:     s.Config.MigrationBackups,
			LocalesDir:              s.Config.LocalesDir,
			PluginsDir:              s.Config.PluginsDir,
			PermissionsFile:         s.Config.PermissionsFile,
			ModulesFile:             s.Config.ModulesFile,
			TrustedKeysFile:         s.Config.TrustedKeysFile,
			OpsAddr:                 s.Config.OpsAddr,
			AdminAddr:               s.Config.AdminAddr,
			DevGuildID:              s.Config.DevGuildID,
			CommandRegistrationMode: s.Config.CommandRegistrationMode,
			ProdMode:                s.Config.ProdMode,
			AllowUnsignedPlugins:    s.Config.AllowUnsignedPlugins,
		},
		Setup: s.setupResponse(false),
	}
	if s.BuildInfo != nil {
		resp.Build = buildResponse(s.BuildInfo())
	}
	if s.Snapshot != nil {
		resp.Snapshot = snapshotResponse(s.Snapshot())
	}
	keys, err := s.TrustedKeys(ctx)
	if err != nil {
		return StatusResponse{}, err
	}
	resp.Setup.TrustedKeysConfigured = len(keys.FileKeys) > 0 || len(keys.DBKeys) > 0
	return resp, nil
}

func (s Service) Setup(ctx context.Context) (SetupResponse, error) {
	resp := s.setupResponse(true)
	keys, err := s.TrustedKeys(ctx)
	if err != nil {
		return SetupResponse{}, err
	}
	resp.TrustedKeysConfigured = len(keys.FileKeys) > 0 || len(keys.DBKeys) > 0
	return resp, nil
}

func (s Service) UserGuilds(ctx context.Context, accessToken string) ([]UserGuildSummary, error) {
	if s.OAuth == nil {
		return nil, errors.New("oauth client is not configured")
	}
	guilds, err := s.OAuth.FetchGuilds(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	// Prefer an explicit "bot has guild" check (REST) so install-state updates
	// even when the gateway cache isn't available yet.
	knownInstalled := toUint64Set(s.KnownGuildIDs)
	installedCache := map[uint64]bool{}

	out := make([]UserGuildSummary, 0, len(guilds))
	for _, guild := range guilds {
		id, err := parseDiscordID(guild.ID)
		if err != nil {
			continue
		}
		canManage := guild.Owner || hasManageGuildPermissions(string(guild.Permissions))
		if !canManage {
			continue
		}

		botInstalled := knownInstalled[id]
		if s.BotHasGuild != nil {
			if cached, ok := installedCache[id]; ok {
				botInstalled = cached
			} else {
				installed, installErr := s.BotHasGuild(ctx, id)
				if installErr == nil {
					botInstalled = installed
				}
				installedCache[id] = botInstalled
			}
		}

		out = append(out, UserGuildSummary{
			ID:           id,
			Name:         strings.TrimSpace(guild.Name),
			IconURL:      guildIconURL(guild),
			Owner:        guild.Owner,
			CanManage:    canManage,
			BotInstalled: botInstalled,
		})
	}
	slices.SortFunc(out, func(a, b UserGuildSummary) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out, nil
}

func (s Service) GuildDashboard(ctx context.Context, accessToken string, guildID uint64) (GuildDashboardResponse, error) {
	guilds, err := s.UserGuilds(ctx, accessToken)
	if err != nil {
		return GuildDashboardResponse{}, err
	}
	var guild UserGuildSummary
	found := false
	for _, item := range guilds {
		if item.ID == guildID {
			guild = item
			found = true
			break
		}
	}
	if !found {
		return GuildDashboardResponse{}, errors.New("guild is not accessible to this user")
	}
	installURL := fmt.Sprintf("/api/install/start?guild_id=%d", guildID)

	managerCfg, err := guildconfig.Load(ctx, s.Store, guildID, "manager")
	if err != nil {
		return GuildDashboardResponse{}, err
	}
	moderationCfg, err := guildconfig.Load(ctx, s.Store, guildID, "moderation")
	if err != nil {
		return GuildDashboardResponse{}, err
	}
	funCfg, err := guildconfig.Load(ctx, s.Store, guildID, "fun")
	if err != nil {
		return GuildDashboardResponse{}, err
	}
	infoCfg, err := guildconfig.Load(ctx, s.Store, guildID, "info")
	if err != nil {
		return GuildDashboardResponse{}, err
	}
	wellnessCfg, err := guildconfig.Load(ctx, s.Store, guildID, "wellness")
	if err != nil {
		return GuildDashboardResponse{}, err
	}

	channels, _ := s.guildChannels(ctx, guildID)
	roles, _ := s.guildRoles(ctx, guildID)
	emojis, _ := s.guildEmojis(ctx, guildID)
	stickers, _ := s.guildStickers(ctx, guildID)
	return GuildDashboardResponse{
		Guild:      guild,
		InstallURL: installURL,
		SetupChecks: []SetupCheck{
			{
				ID:      "user_access",
				Label:   "You can manage this server",
				OK:      guild.CanManage,
				Message: boolMessage(guild.CanManage, "You have permission to manage this server.", "You do not have permission to manage this server."),
			},
			{
				ID:      "bot_installed",
				Label:   "Bot installed",
				OK:      guild.BotInstalled,
				Message: boolMessage(guild.BotInstalled, "The bot is already in this server.", "Add the bot to this server to continue."),
			},
		},
		Manager: ManagerSection{
			PluginSection: s.pluginSection("manager", "Manager", managerCfg),
			ChannelCount:  len(channels),
			RoleCount:     len(roles),
			EmojiCount:    len(emojis),
			StickerCount:  len(stickers),
		},
		Moderation: ModerationSection{
			PluginSection:    s.pluginSection("moderation", "Moderation", moderationCfg),
			WarningLimit:     moderationCfg.WarningLimit,
			TimeoutThreshold: moderationCfg.TimeoutThreshold,
			TimeoutMinutes:   moderationCfg.TimeoutMinutes,
		},
		Fun:  s.pluginSection("fun", "Fun", funCfg),
		Info: s.pluginSection("info", "Info", infoCfg),
		Wellness: WellnessSection{
			PluginSection:            s.pluginSection("wellness", "Wellness", wellnessCfg),
			AllowChannelReminders:    wellnessCfg.AllowChannelReminders,
			DefaultReminderChannelID: wellnessCfg.DefaultReminderChannelID,
		},
	}, nil
}

func (s Service) InstallURL(guildID uint64, baseURL string) (string, error) {
	_ = baseURL
	clientID := strings.TrimSpace(s.Config.DashboardClientID)
	if clientID == "" {
		return "", errors.New("dashboard client id is not configured")
	}

	values := url.Values{}
	values.Set("client_id", clientID)
	values.Set("scope", "bot applications.commands")
	values.Set("permissions", "8")
	values.Set("guild_id", fmt.Sprintf("%d", guildID))
	values.Set("disable_guild_select", "true")
	return "https://discord.com/oauth2/authorize?" + values.Encode(), nil
}

func (s Service) InstallURLAnyGuild(baseURL string) (string, error) {
	_ = baseURL
	clientID := strings.TrimSpace(s.Config.DashboardClientID)
	if clientID == "" {
		return "", errors.New("dashboard client id is not configured")
	}

	values := url.Values{}
	values.Set("client_id", clientID)
	values.Set("scope", "bot applications.commands")
	values.Set("permissions", "8")
	return "https://discord.com/oauth2/authorize?" + values.Encode(), nil
}

func (s Service) Modules() []ModuleResponse {
	if s.ModuleAdmin == nil {
		return nil
	}
	infos := s.ModuleAdmin.Infos()
	out := make([]ModuleResponse, 0, len(infos))
	for _, info := range infos {
		out = append(out, ModuleResponse{
			ID:             info.ID,
			Name:           info.Name,
			Kind:           string(info.Kind),
			Runtime:        string(info.Runtime),
			Enabled:        info.Enabled,
			DefaultEnabled: info.DefaultEnabled,
			Toggleable:     info.Toggleable,
			Signed:         info.Signed,
			Source:         info.Source,
			Commands:       append([]string(nil), info.Commands...),
		})
	}
	return out
}

func (s Service) SetModuleEnabled(ctx context.Context, moduleID string, enabled bool, actorID uint64) error {
	if s.ModuleAdmin == nil {
		return errors.New("modules not configured")
	}
	return s.ModuleAdmin.SetEnabled(ctx, moduleID, enabled, actorID)
}

func (s Service) ResetModule(ctx context.Context, moduleID string) error {
	if s.ModuleAdmin == nil {
		return errors.New("modules not configured")
	}
	return s.ModuleAdmin.Reset(ctx, moduleID)
}

func (s Service) ReloadModules(ctx context.Context) error {
	if s.ModuleAdmin == nil {
		return errors.New("modules not configured")
	}
	return s.ModuleAdmin.Reload(ctx)
}

func (s Service) Plugins() ([]PluginSummary, error) {
	infosByID := map[string]pluginhost.PluginInfo{}
	if s.PluginAdmin != nil {
		for _, info := range s.PluginAdmin.Infos() {
			infosByID[info.ID] = info
		}
	}

	entries, err := os.ReadDir(s.Config.PluginsDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	out := make([]PluginSummary, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		dir := filepath.Join(s.Config.PluginsDir, id)
		manifestPath := filepath.Join(dir, "plugin.json")
		if _, err := os.Stat(manifestPath); err != nil {
			continue
		}

		var summary PluginSummary
		summary.ID = id
		summary.HasSignatureFile = fileExists(filepath.Join(dir, "signature.json"))

		if manifest, err := pluginhost.ReadManifest(manifestPath); err == nil {
			summary.ID = manifest.ID
			summary.Name = manifest.Name
			summary.Version = manifest.Version
		}
		if info, ok := infosByID[summary.ID]; ok {
			summary.Name = fallbackString(summary.Name, info.Name)
			summary.Version = fallbackString(summary.Version, info.Version)
			summary.Commands = make([]string, 0, len(info.Commands))
			for _, cmd := range info.Commands {
				if strings.TrimSpace(cmd.Name) != "" {
					summary.Commands = append(summary.Commands, cmd.Name)
				}
			}
			summary.Loaded = true
			summary.Signed = info.Signed
		}
		out = append(out, summary)
	}

	slices.SortFunc(out, func(a, b PluginSummary) int {
		return strings.Compare(a.ID, b.ID)
	})
	return out, nil
}

func (s Service) ReloadPlugins(ctx context.Context) error {
	if s.PluginAdmin == nil {
		return errors.New("plugins not configured")
	}
	return s.PluginAdmin.Reload(ctx)
}

func (s Service) LoadModulesConfig() (config.ModulesFile, error) {
	return config.LoadModulesFile(s.Config.ModulesFile)
}

func (s Service) SaveModulesConfig(file config.ModulesFile) error {
	return config.WriteModulesFile(s.Config.ModulesFile, file)
}

func (s Service) LoadPermissionsConfig() (permissions.Policy, error) {
	return permissions.LoadPolicyFile(s.Config.PermissionsFile)
}

func (s Service) SavePermissionsConfig(policy permissions.Policy) error {
	return permissions.WritePolicyFile(s.Config.PermissionsFile, policy)
}

func (s Service) TrustedKeys(ctx context.Context) (TrustedKeysResponse, error) {
	resp := TrustedKeysResponse{}
	path := strings.TrimSpace(s.Config.TrustedKeysFile)
	if path != "" && fileExists(path) {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return TrustedKeysResponse{}, err
		}
		var file pluginhost.TrustedKeys
		if err := json.Unmarshal(bytes, &file); err != nil {
			return TrustedKeysResponse{}, err
		}
		resp.FileKeys = make([]TrustedKeyResponse, 0, len(file.Keys))
		for _, key := range file.Keys {
			resp.FileKeys = append(resp.FileKeys, TrustedKeyResponse{
				KeyID:        key.KeyID,
				PublicKeyB64: key.PublicKeyB64,
			})
		}
	}
	if s.Store != nil {
		keys, err := s.Store.TrustedSigners().ListTrustedSigners(ctx)
		if err != nil {
			return TrustedKeysResponse{}, err
		}
		resp.DBKeys = make([]TrustedSignerResponse, 0, len(keys))
		for _, key := range keys {
			resp.DBKeys = append(resp.DBKeys, TrustedSignerResponse{
				KeyID:        key.KeyID,
				PublicKeyB64: key.PublicKeyB64,
				AddedAt:      formatTime(key.AddedAt),
			})
		}
	}
	return resp, nil
}

func (s Service) MigrationStatus(ctx context.Context) (MigrationStatusResponse, error) {
	runner, err := s.migrationRunner()
	if err != nil {
		return MigrationStatusResponse{}, err
	}
	status, err := runner.StatusPath(ctx, s.Config.SQLitePath)
	if err != nil {
		return MigrationStatusResponse{}, err
	}
	return migrationStatusResponse(status), nil
}

func (s Service) MigrateUp(ctx context.Context) (MigrationStatusResponse, error) {
	runner, err := s.migrationRunner()
	if err != nil {
		return MigrationStatusResponse{}, err
	}
	status, err := runner.UpPath(ctx, s.Config.SQLitePath)
	if err != nil {
		return MigrationStatusResponse{}, err
	}
	return migrationStatusResponse(status), nil
}

func (s Service) BackupMigrations(ctx context.Context) (string, error) {
	runner, err := s.migrationRunner()
	if err != nil {
		return "", err
	}
	return runner.BackupPath(ctx, s.Config.SQLitePath)
}

func (s Service) ScaffoldPlugin(req PluginScaffoldRequest) (PluginScaffoldResponse, error) {
	id := strings.TrimSpace(req.ID)
	name := strings.TrimSpace(req.Name)
	version := strings.TrimSpace(req.Version)
	locale := strings.TrimSpace(req.Locale)
	commandName := strings.TrimSpace(req.CommandName)
	commandDescription := strings.TrimSpace(req.CommandDescription)
	responseMessage := strings.TrimSpace(req.ResponseMessage)

	switch {
	case !pluginIDPattern.MatchString(id):
		return PluginScaffoldResponse{}, errors.New("plugin id must match ^[a-z][a-z0-9_]{1,31}$")
	case name == "":
		return PluginScaffoldResponse{}, errors.New("plugin name is required")
	case version == "":
		version = "0.1.0"
	case locale == "":
		locale = "en-US"
	case !pluginIDPattern.MatchString(commandName):
		if commandName == "" {
			commandName = id
		} else {
			return PluginScaffoldResponse{}, errors.New("command name must match ^[a-z][a-z0-9_]{1,31}$")
		}
	}
	if commandDescription == "" {
		commandDescription = "Run the " + name + " plugin command"
	}
	if responseMessage == "" {
		responseMessage = "Hello from " + name + "."
	}

	dir := filepath.Join(s.Config.PluginsDir, id)
	if fileExists(dir) {
		return PluginScaffoldResponse{}, fmt.Errorf("plugin %q already exists", id)
	}
	if err := os.MkdirAll(filepath.Join(dir, "commands"), 0o755); err != nil {
		return PluginScaffoldResponse{}, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "locales", locale), 0o755); err != nil {
		return PluginScaffoldResponse{}, err
	}

	descID := "cmd." + commandName + ".desc"
	messageID := id + ".hello"

	manifest := pluginhost.Manifest{
		ID:          id,
		Name:        name,
		Version:     version,
		Permissions: req.Permissions,
	}
	manifestBytes, err := json.MarshalIndent(map[string]any{
		"$schema":     "https://raw.githubusercontent.com/xsyetopz/go-mamusiabtw/refs/heads/main/schemas/plugin.schema.v1.json",
		"id":          manifest.ID,
		"name":        manifest.Name,
		"version":     manifest.Version,
		"permissions": manifest.Permissions,
	}, "", "  ")
	if err != nil {
		return PluginScaffoldResponse{}, err
	}

	pluginLua := fmt.Sprintf(`local hello = bot.require("commands/hello.lua")

return bot.plugin({
  commands = {
    bot.command("%s", {
      description_id = "%s",
      ephemeral = true,
      run = hello
    })
  }
})
`, commandName, descID)

	commandLua := fmt.Sprintf(`local i18n = bot.i18n
local ui = bot.ui

return function(_ctx)
  return ui.reply({
    content = i18n.t("%s", nil, nil),
    ephemeral = true
  })
end
`, messageID)

	localeBytes, err := json.MarshalIndent([]map[string]string{
		{"id": descID, "translation": commandDescription},
		{"id": messageID, "translation": responseMessage},
	}, "", "  ")
	if err != nil {
		return PluginScaffoldResponse{}, err
	}

	files := []struct {
		rel  string
		data []byte
	}{
		{rel: "plugin.json", data: append(manifestBytes, '\n')},
		{rel: "plugin.lua", data: []byte(pluginLua)},
		{rel: filepath.Join("commands", "hello.lua"), data: []byte(commandLua)},
		{rel: filepath.Join("locales", locale, "messages.json"), data: append(localeBytes, '\n')},
	}

	created := make([]string, 0, len(files)+1)
	for _, file := range files {
		fullPath := filepath.Join(dir, file.rel)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return PluginScaffoldResponse{}, err
		}
		if err := os.WriteFile(fullPath, file.data, 0o644); err != nil {
			return PluginScaffoldResponse{}, err
		}
		created = append(created, file.rel)
	}

	resp := PluginScaffoldResponse{
		ID:    id,
		Dir:   dir,
		Files: created,
	}
	if req.Sign {
		signaturePath, err := s.SignPlugin(id)
		if err != nil {
			return PluginScaffoldResponse{}, err
		}
		resp.Signed = true
		resp.Signature = signaturePath
		resp.Files = append(resp.Files, filepath.Base(signaturePath))
	}
	return resp, nil
}

func (s Service) SignPlugin(pluginID string) (string, error) {
	if !signingReady(s.Config) {
		return "", errors.New("dashboard signing is not configured")
	}
	dir := filepath.Join(s.Config.PluginsDir, strings.TrimSpace(pluginID))
	if !fileExists(filepath.Join(dir, "plugin.json")) {
		return "", fmt.Errorf("plugin %q not found", pluginID)
	}

	privateKey, err := pluginhost.ReadEd25519PrivateKeyFile(s.Config.DashboardSigningKeyFile)
	if err != nil {
		return "", err
	}
	sig, _, err := pluginhost.SignDir(dir, s.Config.DashboardSigningKeyID, privateKey)
	if err != nil {
		return "", err
	}
	payload := map[string]any{
		"$schema":       pluginhost.SignatureSchemaURL,
		"key_id":        sig.KeyID,
		"hash_b64":      sig.HashB64,
		"signature_b64": sig.SignatureB64,
		"algorithm":     sig.Algorithm,
	}
	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	target := filepath.Join(dir, "signature.json")
	if err := os.WriteFile(target, append(bytes, '\n'), 0o644); err != nil {
		return "", err
	}
	return target, nil
}

func (s Service) migrationRunner() (migrate.Runner, error) {
	return migrate.New(migrate.Options{
		Dir:       s.Config.Migrations,
		BackupDir: s.Config.MigrationBackups,
	})
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fallbackString(primary, secondary string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return strings.TrimSpace(secondary)
}

func (s Service) setupResponse(includeHints bool) SetupResponse {
	ownerStatus := OwnerStatus{
		Configured: s.Config.OwnerUserID != nil,
		Resolved:   s.Config.OwnerUserID != nil,
		Source:     "unresolved",
	}
	if s.Config.OwnerUserID != nil {
		ownerStatus.Source = "config_fallback"
		ownerStatus.EffectiveUserID = s.Config.OwnerUserID
	}
	if s.OwnerStatus != nil {
		ownerStatus = s.OwnerStatus()
	}

	resp := SetupResponse{
		AdminEnabled:         strings.TrimSpace(s.Config.AdminAddr) != "",
		AuthConfigured:       dashboardAuthReady(s.Config),
		LoginReady:           dashboardAuthReady(s.Config),
		OwnerConfigured:      ownerStatus.Configured,
		OwnerResolved:        ownerStatus.Resolved,
		OwnerSource:          strings.TrimSpace(ownerStatus.Source),
		EffectiveOwnerUserID: cloneOptionalUint64(ownerStatus.EffectiveUserID),
		SigningConfigured:    signingReady(s.Config),
		AdminAddr:            strings.TrimSpace(s.Config.AdminAddr),
		// Filled by the HTTP layer based on configured public origins.
		AppOrigin:        "",
		RedirectURL:      "",
		HasClientID:      strings.TrimSpace(s.Config.DashboardClientID) != "",
		HasClientSecret:  strings.TrimSpace(s.Config.DashboardClientSecret) != "",
		HasSessionSecret: len(strings.TrimSpace(s.Config.DashboardSessionSecret)) >= 32,
	}
	if includeHints {
		resp.Hints = setupHints(resp)
	}
	return resp
}

func setupHints(resp SetupResponse) []string {
	hints := make([]string, 0, 6)
	if !resp.AdminEnabled {
		hints = append(hints, "Set MAMUSIABTW_ADMIN_ADDR to start the admin API.")
	}
	if !resp.HasClientID {
		hints = append(hints, "Set MAMUSIABTW_DASHBOARD_CLIENT_ID.")
	}
	if !resp.HasClientSecret {
		hints = append(hints, "Set MAMUSIABTW_DASHBOARD_CLIENT_SECRET.")
	}
	if !resp.HasSessionSecret {
		hints = append(hints, "Set MAMUSIABTW_DASHBOARD_SESSION_SECRET to at least 32 characters.")
	}
	if !resp.OwnerResolved {
		hints = append(hints, "Owner access is unavailable. Discord owner lookup did not resolve an owner, and no OWNER_USER_ID fallback is configured.")
	}
	return hints
}

func snapshotResponse(snap ops.Snapshot) SnapshotResponse {
	return SnapshotResponse{
		Ready:               snap.Ready,
		StartedAt:           formatTime(snap.StartedAt),
		MigrationVersion:    snap.MigrationVersion,
		ProdMode:            snap.ProdMode,
		ModuleCount:         snap.ModuleCount,
		EnabledModuleCount:  snap.EnabledModuleCount,
		PluginCount:         snap.PluginCount,
		EnabledPluginCount:  snap.EnabledPluginCount,
		BuiltinCommandCount: snap.BuiltinCommandCount,
		SlashCommandCount:   snap.SlashCommandCount,
		UserCommandCount:    snap.UserCommandCount,
		MessageCommandCount: snap.MessageCommandCount,
		InteractionsTotal:   snap.InteractionsTotal,
		InteractionFailures: snap.InteractionFailures,
		PluginFailures:      snap.PluginFailures,
		AutomationFailures:  snap.AutomationFailures,
		ReminderFailures:    snap.ReminderFailures,
	}
}

func buildResponse(info buildinfo.Info) BuildResponse {
	return BuildResponse{
		Version:          info.Version,
		Repository:       info.Repository,
		Description:      info.Description,
		DeveloperURL:     info.DeveloperURL,
		SupportServerURL: info.SupportServerURL,
		MascotImageURL:   info.MascotImageURL,
	}
}

func migrationStatusResponse(status migrate.Status) MigrationStatusResponse {
	return MigrationStatusResponse{
		CurrentVersion: status.CurrentVersion,
		Applied:        migrationItems(status.Applied),
		Pending:        migrationItems(status.Pending),
	}
}

func migrationItems(items []migrate.Item) []MigrationItemDTO {
	out := make([]MigrationItemDTO, 0, len(items))
	for _, item := range items {
		out = append(out, MigrationItemDTO{
			Version: item.Version,
			Name:    item.Name,
			Kind:    string(item.Kind),
		})
	}
	return out
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func toUint64Set(fn func() []uint64) map[uint64]bool {
	out := map[uint64]bool{}
	if fn == nil {
		return out
	}
	for _, id := range fn() {
		out[id] = true
	}
	return out
}

func parseDiscordID(raw string) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
}

func hasManageGuildPermissions(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false
	}
	perm, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return false
	}
	administrator := big.NewInt(0x8)
	manageGuild := big.NewInt(0x20)
	return new(big.Int).And(perm, administrator).Cmp(big.NewInt(0)) != 0 ||
		new(big.Int).And(perm, manageGuild).Cmp(big.NewInt(0)) != 0
}

func guildIconURL(guild OAuthGuild) string {
	id := strings.TrimSpace(guild.ID)
	icon := strings.TrimSpace(guild.Icon)
	if id == "" || icon == "" {
		return ""
	}
	return "https://cdn.discordapp.com/icons/" + id + "/" + icon + ".png"
}

func boolMessage(value bool, okMessage, noMessage string) string {
	if value {
		return okMessage
	}
	return noMessage
}

func dashboardAuthReady(cfg config.Config) bool {
	return cfg.AdminAddr != "" &&
		cfg.DashboardClientID != "" &&
		cfg.DashboardClientSecret != "" &&
		len(cfg.DashboardSessionSecret) >= 32
}

func signingReady(cfg config.Config) bool {
	return cfg.DashboardSigningKeyID != "" && cfg.DashboardSigningKeyFile != ""
}
