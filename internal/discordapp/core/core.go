package core

import (
	"context"
	"log/slog"
	"maps"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"github.com/xsyetopz/go-mamusiabtw/internal/discordapp/interactions"
	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
	"github.com/xsyetopz/go-mamusiabtw/internal/integrations/kawaii"
	"github.com/xsyetopz/go-mamusiabtw/internal/persona"
	"github.com/xsyetopz/go-mamusiabtw/internal/plugins"
	"github.com/xsyetopz/go-mamusiabtw/internal/store"
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
	UserSettings() store.UserSettingsStore
	Reminders() store.ReminderStore
	CheckIns() store.CheckInStore
}

type Translator struct {
	Registry i18n.Registry
	Locale   discord.Locale
	UserID   uint64
}

func (t Translator) S(messageID string, data map[string]any) string {
	if t.UserID != 0 {
		data = withPersonaTemplateData(data, t.Locale, t.UserID, messageID)
	}
	return t.Registry.MustLocalize(i18n.Config{
		Locale:       t.Locale.Code(),
		MessageID:    messageID,
		TemplateData: data,
	})
}

func withPersonaTemplateData(
	data map[string]any,
	locale discord.Locale,
	userID uint64,
	messageID string,
) map[string]any {
	if data == nil {
		data = map[string]any{}
	} else {
		clone := make(map[string]any, len(data)+1)
		maps.Copy(clone, data)
		data = clone
	}

	if _, ok := data["Pet"]; !ok {
		data["Pet"] = personaPet(locale, userID, messageID)
	}
	if _, ok := data["Mommy"]; !ok {
		data["Mommy"] = personaMommy(locale)
	}
	return data
}

func personaPet(locale discord.Locale, userID uint64, messageID string) string {
	return persona.PetName(locale, userID, messageID)
}

func personaMommy(locale discord.Locale) string {
	return persona.Mommy(locale)
}

func LocalizeMap(locales []string, fn func(locale string) string) map[discord.Locale]string {
	const baseLocale = "en-US"
	base := strings.TrimSpace(fn(baseLocale))

	out := map[discord.Locale]string{}
	for _, locale := range locales {
		locale = strings.TrimSpace(locale)
		if locale == "" || strings.EqualFold(locale, baseLocale) {
			continue
		}

		translated := fn(locale)
		translated = strings.TrimSpace(translated)
		if translated == "" {
			continue
		}
		if base != "" && translated == base {
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
