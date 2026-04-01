package botengine

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/snowflake/v2"

	"github.com/xsyetopz/imotherbtw/internal/buildinfo"
	"github.com/xsyetopz/imotherbtw/internal/discordapp/commands"
	"github.com/xsyetopz/imotherbtw/internal/discordapp/core"
	"github.com/xsyetopz/imotherbtw/internal/discordapp/interactions"
	"github.com/xsyetopz/imotherbtw/internal/i18n"
	"github.com/xsyetopz/imotherbtw/internal/plugins"
	"github.com/xsyetopz/imotherbtw/internal/present"
	"github.com/xsyetopz/imotherbtw/internal/store"
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

	ProdMode             bool
	AllowUnsignedPlugins bool
	TrustedKeysFile      string

	I18n  i18n.Registry
	Store core.Store

	SlashCooldown       time.Duration
	ComponentCooldown   time.Duration
	ModalCooldown       time.Duration
	SlashCooldownBypass []string
}

type Bot struct {
	logger *slog.Logger
	i18n   i18n.Registry
	store  core.Store

	prodMode bool
	kawaii   core.Kawaii

	cooldowns *cooldownTracker

	slashCooldown        time.Duration
	componentCooldownDur time.Duration
	modalCooldownDur     time.Duration
	slashBypass          map[string]struct{}

	devGuildID *uint64
	owners     map[uint64]struct{}

	commandRegistrationMode  string
	commandGuildIDs          []uint64
	commandRegisterAllGuilds bool

	client   *bot.Client
	commands map[string]core.SlashCommand
	order    []core.SlashCommand

	plugins *plugins.Manager
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
	}
	b.slashCooldown = deps.SlashCooldown
	b.componentCooldownDur = deps.ComponentCooldown
	b.modalCooldownDur = deps.ModalCooldown
	b.slashBypass = buildSlashBypass(deps.SlashCooldownBypass)
	b.order, b.commands = buildCommands()

	if initErr := b.initPlugins(deps); initErr != nil {
		return nil, initErr
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

func buildCommands() ([]core.SlashCommand, map[string]core.SlashCommand) {
	order := commands.All()
	m := map[string]core.SlashCommand{}
	for _, c := range order {
		if strings.TrimSpace(c.Name) == "" {
			continue
		}
		m[c.Name] = c
	}
	return order, m
}

func (b *Bot) initPlugins(deps Dependencies) error {
	if strings.TrimSpace(deps.PluginsDir) == "" {
		return nil
	}

	pm, err := plugins.NewManager(plugins.Options{
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
	b.plugins = pm
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
	if b.plugins != nil {
		if err := b.plugins.LoadAll(ctx); err != nil {
			return err
		}
	}

	if err := b.registerCommands(ctx); err != nil {
		return err
	}

	if err := b.client.OpenGateway(ctx); err != nil {
		return err
	}
	return nil
}

func (b *Bot) Close(ctx context.Context) {
	if b.client != nil {
		b.client.Close(ctx)
	}
}

func (b *Bot) services(_ discord.Locale) core.Services {
	s := core.Services{
		Logger:   b.logger,
		Store:    b.store,
		ProdMode: b.prodMode,
		IsOwner:  b.isOwner,
		Kawaii:   b.kawaii,
		HelpNames: func(locale discord.Locale) []string {
			t := core.Translator{Registry: b.i18n, Locale: locale}
			out := make([]string, 0, len(b.order))
			for _, c := range b.order {
				if strings.TrimSpace(c.NameID) == "" {
					continue
				}
				out = append(out, t.S(c.NameID, nil))
			}
			return out
		},
	}

	if b.plugins != nil {
		s.Plugins = pluginAdmin{b: b}
	}
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
	t core.Translator,
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

func (p pluginAdmin) Configured() bool { return p.b != nil && p.b.plugins != nil }

func (p pluginAdmin) Infos() []plugins.PluginInfo {
	if p.b == nil || p.b.plugins == nil {
		return nil
	}
	return p.b.plugins.Infos()
}

func (p pluginAdmin) Reload(ctx context.Context) error {
	if p.b == nil || p.b.plugins == nil {
		return errors.New("plugins not configured")
	}
	if err := p.b.plugins.LoadAll(ctx); err != nil {
		return err
	}
	if err := p.b.registerCommands(ctx); err != nil {
		return err
	}
	if p.b.commandRegisterAllGuilds && p.b.devGuildID == nil {
		if err := p.b.registerCommandsInCachedGuilds(ctx); err != nil {
			return err
		}
	}
	return nil
}

var _ core.PluginAdmin = pluginAdmin{}
