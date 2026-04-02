package discordplatform

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/snowflake/v2"

	"github.com/xsyetopz/go-mamusiabtw/internal/buildinfo"
	"github.com/xsyetopz/go-mamusiabtw/internal/config"
	"github.com/xsyetopz/go-mamusiabtw/internal/features/commandapi"
	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
	"github.com/xsyetopz/go-mamusiabtw/internal/platform/discord/interactions"
	"github.com/xsyetopz/go-mamusiabtw/internal/pluginhost"
	"github.com/xsyetopz/go-mamusiabtw/internal/present"
	"github.com/xsyetopz/go-mamusiabtw/internal/store"
)

type Dependencies struct {
	Logger *slog.Logger
	Token  string
	Kawaii KawaiiConfig

	Owners                   []uint64
	DevGuildID               *uint64
	CommandRegistrationMode  string
	CommandGuildIDs          []uint64
	CommandRegisterAllGuilds bool
	PluginsDir               string
	PermissionsFile          string
	ModulesFile              string

	ProdMode             bool
	AllowUnsignedPlugins bool
	TrustedKeysFile      string

	I18n  i18n.Registry
	Store commandapi.Store

	SlashCooldown          time.Duration
	ComponentCooldown      time.Duration
	ModalCooldown          time.Duration
	SlashCooldownBypass    []string
	SlashCooldownOverrides map[string]time.Duration
}

type Bot struct {
	logger *slog.Logger
	i18n   i18n.Registry
	store  commandapi.Store

	prodMode bool
	kawaii   commandapi.Kawaii

	cooldowns *cooldownTracker

	slashCooldown          time.Duration
	componentCooldownDur   time.Duration
	modalCooldownDur       time.Duration
	slashBypass            map[string]struct{}
	slashCooldownOverrides map[string]time.Duration

	devGuildID *uint64
	owners     map[uint64]struct{}

	commandRegistrationMode  string
	commandGuildIDs          []uint64
	commandRegisterAllGuilds bool

	client   *bot.Client
	commands map[string]commandapi.SlashCommand
	order    []commandapi.SlashCommand

	moduleSeed config.ModulesFile
	modules    map[string]commandapi.ModuleInfo

	pluginHost     *pluginhost.Host
	pluginCommands map[string]pluginCommandRoute
	pluginRoutes   map[string]pluginRoute
	pluginAuto     *pluginAutomation
}

type pluginRoute struct {
	host *pluginhost.Host
}

type pluginCommandRoute struct {
	host     *pluginhost.Host
	pluginID string
}

func New(deps Dependencies) (*Bot, error) {
	if err := validateNewDeps(deps); err != nil {
		return nil, err
	}

	kc, err := newKawaiiClient(deps.Kawaii)
	if err != nil {
		return nil, err
	}

	commandRegistrationMode, err := normalizeCommandRegistrationMode(deps.CommandRegistrationMode)
	if err != nil {
		return nil, err
	}

	moduleSeed, err := config.LoadModulesFile(deps.ModulesFile)
	if err != nil {
		return nil, err
	}

	b := &Bot{
		logger:     deps.Logger.With(slog.String("component", "discord")),
		i18n:       deps.I18n,
		store:      deps.Store,
		prodMode:   deps.ProdMode,
		kawaii:     kc,
		devGuildID: deps.DevGuildID,
		owners:     toSet(deps.Owners),
		cooldowns:  newCooldownTracker(),

		commandRegistrationMode:  commandRegistrationMode,
		commandGuildIDs:          append([]uint64(nil), deps.CommandGuildIDs...),
		commandRegisterAllGuilds: deps.CommandRegisterAllGuilds,
		moduleSeed:               moduleSeed,
		modules:                  map[string]commandapi.ModuleInfo{},
		pluginCommands:           map[string]pluginCommandRoute{},
		pluginRoutes:             map[string]pluginRoute{},
	}
	b.slashCooldown = deps.SlashCooldown
	b.componentCooldownDur = deps.ComponentCooldown
	b.modalCooldownDur = deps.ModalCooldown
	b.slashBypass = buildSlashBypass(deps.SlashCooldownBypass)
	b.slashCooldownOverrides = cloneCooldownOverrides(deps.SlashCooldownOverrides)

	if initErr := b.initPlugins(deps); initErr != nil {
		return nil, initErr
	}

	if refreshErr := b.refreshRuntimeCatalog(context.Background()); refreshErr != nil {
		return nil, refreshErr
	}

	client, err := b.newClient(deps.Token)
	if err != nil {
		return nil, err
	}
	b.client = client

	return b, nil
}

func validateNewDeps(deps Dependencies) error {
	if deps.Logger == nil {
		return errors.New("logger is required")
	}
	if strings.TrimSpace(deps.Token) == "" {
		return errors.New("discord token is required")
	}
	if deps.Store == nil {
		return errors.New("store is required")
	}
	return nil
}

func normalizeCommandRegistrationMode(mode string) (string, error) {
	m := strings.ToLower(strings.TrimSpace(mode))
	if m == "" {
		return commandRegistrationModeGlobal, nil
	}
	switch m {
	case commandRegistrationModeGlobal, commandRegistrationModeGuilds, commandRegistrationModeHybrid:
		return m, nil
	default:
		return "", errors.New("invalid command registration mode")
	}
}

func buildSlashBypass(names []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, name := range names {
		n := strings.ToLower(strings.TrimSpace(name))
		if n == "" {
			continue
		}
		out[n] = struct{}{}
	}
	return out
}

func cloneCooldownOverrides(in map[string]time.Duration) map[string]time.Duration {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]time.Duration, len(in))
	for k, v := range in {
		key := strings.ToLower(strings.TrimSpace(k))
		if key == "" {
			continue
		}
		out[key] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (b *Bot) initPlugins(deps Dependencies) error {
	if strings.TrimSpace(deps.PluginsDir) != "" {
		host, err := pluginhost.NewHost(pluginhost.Options{
			Dir:                 deps.PluginsDir,
			ProdMode:            deps.ProdMode,
			AllowUnsignedPlugin: deps.AllowUnsignedPlugins,
			TrustedKeysFile:     deps.TrustedKeysFile,
			PermissionsFile:     deps.PermissionsFile,
			Store:               deps.Store,
			Logger:              b.logger,
			I18n:                &b.i18n,
		})
		if err != nil {
			return err
		}
		b.pluginHost = host
	}

	if b.pluginHost != nil {
		b.pluginAuto = newPluginAutomation(b)
	}
	return nil
}

func (b *Bot) newClient(token string) (*bot.Client, error) {
	return disgo.New(token,
		bot.WithLogger(b.logger),
		bot.WithGatewayConfigOpts(gateway.WithIntents(
			gateway.IntentGuilds,
			gateway.IntentGuildMembers,
			gateway.IntentGuildModeration,
			gateway.IntentGuildInvites,
			gateway.IntentDirectMessages,
		)),
		bot.WithEventListenerFunc(b.onCommand),
		bot.WithEventListenerFunc(b.onComponent),
		bot.WithEventListenerFunc(b.onModal),
		bot.WithEventListenerFunc(b.onGuildJoin),
		bot.WithEventListenerFunc(b.onGuildLeave),
		bot.WithEventListenerFunc(b.onGuildUpdate),
		bot.WithEventListenerFunc(b.onGuildMemberJoin),
		bot.WithEventListenerFunc(b.onGuildMemberLeave),
		bot.WithEventListenerFunc(b.onGuildBan),
		bot.WithEventListenerFunc(b.onGuildUnban),
		bot.WithEventListenerFunc(b.onGuildChannelCreate),
		bot.WithEventListenerFunc(b.onGuildChannelDelete),
		bot.WithEventListenerFunc(b.onRoleCreate),
		bot.WithEventListenerFunc(b.onRoleDelete),
		bot.WithEventListenerFunc(b.onInviteCreate),
		bot.WithEventListenerFunc(b.onInviteDelete),
		bot.WithEventListenerFunc(b.onGuildsReady),
	)
}

const (
	commandRegistrationModeGlobal = "global"
	commandRegistrationModeGuilds = "guilds"
	commandRegistrationModeHybrid = "hybrid"
)

func (b *Bot) Start(ctx context.Context) error {
	if err := b.reloadModules(ctx); err != nil {
		return err
	}

	if err := b.client.OpenGateway(ctx); err != nil {
		return err
	}

	if b.pluginAuto != nil {
		b.pluginAuto.Start(ctx)
	}

	b.startReminderScheduler(ctx)
	return nil
}

func (b *Bot) Close(ctx context.Context) {
	if b.client != nil {
		b.client.Close(ctx)
	}
	if b.pluginAuto != nil {
		b.pluginAuto.Stop()
	}
}

func (b *Bot) services(_ discord.Locale) commandapi.Services {
	s := commandapi.Services{
		Logger:   b.logger,
		Store:    b.store,
		ProdMode: b.prodMode,
		IsOwner:  b.isOwner,
		Kawaii:   b.kawaii,
		HelpNames: func(locale discord.Locale) []string {
			t := commandapi.Translator{Registry: b.i18n, Locale: locale}
			out := make([]string, 0, len(b.order)+len(b.pluginCommands))
			for _, c := range b.order {
				name := strings.TrimSpace(c.Name)
				if strings.TrimSpace(c.NameID) != "" {
					name = t.S(c.NameID, nil)
				}
				if name != "" {
					out = append(out, name)
				}
			}
			for name := range b.pluginCommands {
				out = append(out, name)
			}
			sort.Strings(out)
			return out
		},
	}

	if b.pluginHost != nil {
		s.Plugins = pluginAdmin{b: b}
	}
	s.Modules = moduleAdmin{b: b}
	return s
}

func (b *Bot) registerCommands(ctx context.Context) error {
	locales := b.i18n.SupportedLocales()
	creates := b.commandCreates(locales)

	if b.devGuildID != nil {
		_, err := b.client.Rest.SetGuildCommands(b.client.ApplicationID, snowflake.ID(*b.devGuildID), creates)
		return err
	}

	switch b.commandRegistrationMode {
	case "global":
		_, err := b.client.Rest.SetGlobalCommands(b.client.ApplicationID, creates)
		return err
	case "guilds":
		return b.setCommandsInGuilds(ctx, creates, b.commandGuildIDs)
	case "hybrid":
		if _, err := b.client.Rest.SetGlobalCommands(b.client.ApplicationID, creates); err != nil {
			return err
		}
		return b.setCommandsInGuilds(ctx, creates, b.commandGuildIDs)
	default:
		return errors.New("invalid command registration mode")
	}
}

func (b *Bot) checkRestrictions(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t commandapi.Translator,
) (bool, error) {
	restrictions := b.store.Restrictions()

	msgID := "err.restricted"
	var msgData map[string]any
	dev := strings.TrimSpace(buildinfo.DeveloperURL)
	support := strings.TrimSpace(buildinfo.SupportServerURL)
	if dev != "" && dev != "UNKNOWN" && support != "" && support != "UNKNOWN" {
		msgID = "err.restricted_links"
		msgData = map[string]any{
			"DeveloperURL":     dev,
			"SupportServerURL": support,
		}
	}
	msgText := t.S(msgID, msgData)

	userID := uint64(e.User().ID)
	if _, ok, err := restrictions.GetRestriction(ctx, store.TargetTypeUser, userID); err != nil {
		return false, err
	} else if ok {
		return true, e.CreateMessage(interactions.NoticeMessage(present.KindError, "", msgText, true))
	}

	guildID := e.GuildID()
	if guildID == nil {
		return false, nil
	}

	if _, ok, err := restrictions.GetRestriction(ctx, store.TargetTypeGuild, uint64(*guildID)); err != nil {
		return false, err
	} else if ok {
		return true, e.CreateMessage(interactions.NoticeMessage(present.KindError, "", msgText, true))
	}

	return false, nil
}

type pluginAdmin struct{ b *Bot }

func (p pluginAdmin) Configured() bool { return p.b != nil && p.b.pluginHost != nil }

func (p pluginAdmin) Infos() []pluginhost.PluginInfo {
	if p.b == nil {
		return nil
	}
	out := []pluginhost.PluginInfo{}
	if p.b.pluginHost != nil {
		out = append(out, p.b.pluginHost.Infos()...)
	}
	return out
}

func (p pluginAdmin) Reload(ctx context.Context) error {
	if p.b == nil || p.b.pluginHost == nil {
		return errors.New("plugins not configured")
	}
	return p.b.reloadModules(ctx)
}

var _ commandapi.PluginAdmin = pluginAdmin{}

type moduleAdmin struct{ b *Bot }

func (m moduleAdmin) Configured() bool { return m.b != nil }

func (m moduleAdmin) Infos() []commandapi.ModuleInfo {
	if m.b == nil {
		return nil
	}
	return m.b.moduleInfos()
}

func (m moduleAdmin) Reload(ctx context.Context) error {
	if m.b == nil {
		return errors.New("modules not configured")
	}
	return m.b.reloadModules(ctx)
}

func (m moduleAdmin) SetEnabled(ctx context.Context, moduleID string, enabled bool, actorID uint64) error {
	if m.b == nil {
		return errors.New("modules not configured")
	}
	return m.b.setModuleEnabled(ctx, moduleID, enabled, actorID)
}

func (m moduleAdmin) Reset(ctx context.Context, moduleID string) error {
	if m.b == nil {
		return errors.New("modules not configured")
	}
	return m.b.resetModule(ctx, moduleID)
}

var _ commandapi.ModuleAdmin = moduleAdmin{}
