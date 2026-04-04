package discordplatform

import (
	"strings"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/gateway"

	discordplugin "github.com/xsyetopz/go-mamusiabtw/internal/platform/discord/plugin"
	"github.com/xsyetopz/go-mamusiabtw/internal/pluginhost"
)

const (
	commandRegistrationModeGlobal = "global"
	commandRegistrationModeGuilds = "guilds"
	commandRegistrationModeHybrid = "hybrid"
)

func (b *Bot) initPlugins(deps Dependencies) error {
	if strings.TrimSpace(deps.PluginsDir) != "" {
		host, err := pluginhost.NewHost(pluginhost.Options{
			Dir:                 deps.PluginsDir,
			ProdMode:            deps.ProdMode,
			AllowUnsignedPlugin: deps.AllowUnsignedPlugins,
			TrustedKeysFile:     deps.TrustedKeysFile,
			PermissionsFile:     deps.PermissionsFile,
			Store:               deps.Store,
			Discord: discordplugin.Executor{
				ClientProvider:      func() *bot.Client { return b.client },
				EnsureDMChannelFunc: b.ensureDMChannel,
			},
			Logger: b.logger,
			I18n:   &b.i18n,
		})
		if err != nil {
			return err
		}
		b.pluginHost = host
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
		bot.WithEventListenerFunc(b.onAutocomplete),
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
