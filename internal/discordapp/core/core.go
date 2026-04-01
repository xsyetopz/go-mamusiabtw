package core

import (
	"context"
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"github.com/xsyetopz/jagpda/internal/discordapp/interactions"
	"github.com/xsyetopz/jagpda/internal/i18n"
	"github.com/xsyetopz/jagpda/internal/integrations/kawaii"
	"github.com/xsyetopz/jagpda/internal/plugins"
	"github.com/xsyetopz/jagpda/internal/store"
)

type Store interface {
	Restrictions() store.RestrictionStore
	Warnings() store.WarningStore
	Audit() store.AuditStore
	TrustedSigners() store.TrustedSignerStore
	PluginKV() store.PluginKVStore
	Users() store.UserStore
	Guilds() store.GuildStore
	GuildMembers() store.GuildMemberStore
}

type Translator struct {
	Registry i18n.Registry
	Locale   discord.Locale
}

func (t Translator) S(messageID string, data map[string]any) string {
	return t.Registry.MustLocalize(i18n.Config{
		Locale:       t.Locale.Code(),
		MessageID:    messageID,
		TemplateData: data,
	})
}

func LocalizeMap(locales []string, fn func(locale string) string) map[discord.Locale]string {
	out := map[discord.Locale]string{}
	for _, locale := range locales {
		translated := fn(locale)
		if translated == "" {
			continue
		}
		out[discord.Locale(locale)] = translated
	}
	return out
}

type Kawaii interface {
	FetchGIF(ctx context.Context, endpoint kawaii.Endpoint) (string, error)
}

type PluginAdmin interface {
	Configured() bool
	Infos() []plugins.PluginInfo
	Reload(ctx context.Context) error
}

type Services struct {
	Logger   *slog.Logger
	Store    Store
	ProdMode bool

	IsOwner func(userID uint64) bool

	Kawaii Kawaii

	Plugins PluginAdmin

	// HelpNames returns the localized slash command names for help output.
	HelpNames func(locale discord.Locale) []string
}

type SlashCommand struct {
	Name          string
	NameID        string
	DescID        string
	CreateCommand func(locales []string, t Translator) discord.ApplicationCommandCreate
	Handle        func(ctx context.Context, e *events.ApplicationCommandInteractionCreate, t Translator, s Services) (interactions.SlashAction, error)
}
