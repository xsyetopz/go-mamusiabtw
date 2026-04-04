package catalog

import (
	"github.com/disgoorg/disgo/discord"

	"github.com/xsyetopz/go-mamusiabtw/internal/features/commandapi"
	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
	"github.com/xsyetopz/go-mamusiabtw/internal/pluginhost"
)

type CommandCreateOptions struct {
	Builtins         []commandapi.SlashCommand
	PluginHost       *pluginhost.Host
	EnabledPluginIDs map[string]struct{}
	I18n             i18n.Registry
	Locales          []string
}

func CommandCreates(opts CommandCreateOptions) []discord.ApplicationCommandCreate {
	const extraCreatesCapacity = 8
	creates := make([]discord.ApplicationCommandCreate, 0, len(opts.Builtins)+extraCreatesCapacity)
	for _, cmd := range opts.Builtins {
		creates = append(
			creates,
			cmd.CreateCommand(opts.Locales, commandapi.Translator{Registry: opts.I18n, Locale: discord.LocaleEnglishUS}),
		)
	}
	if opts.PluginHost == nil {
		return creates
	}

	return append(
		creates,
		opts.PluginHost.CommandCreatesFiltered(opts.EnabledPluginIDs, opts.Locales, func(pluginID, locale, messageID string) (string, bool) {
			return opts.I18n.TryLocalize(i18n.Config{
				Locale:    locale,
				PluginID:  pluginID,
				MessageID: messageID,
			})
		})...,
	)
}
