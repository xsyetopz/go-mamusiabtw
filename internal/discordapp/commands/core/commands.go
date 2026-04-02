package cmdcore

import (
	"context"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"github.com/xsyetopz/go-mamusiabtw/internal/discordapp/core"
	"github.com/xsyetopz/go-mamusiabtw/internal/discordapp/interactions"
	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
)

func Commands() []core.SlashCommand {
	return []core.SlashCommand{
		ping(),
		help(),
	}
}

func ping() core.SlashCommand {
	return core.SlashCommand{
		Name:   "ping",
		NameID: "cmd.ping.name",
		DescID: "cmd.ping.desc",
		CreateCommand: func(locales []string, t core.Translator) discord.ApplicationCommandCreate {
			return discord.SlashCommandCreate{
				Name: "ping",
				NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.ping.name"})
				}),
				Description: t.S("cmd.ping.desc", nil),
				DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.ping.desc"})
				}),
			}
		},
		Handle: func(ctx context.Context, e *events.ApplicationCommandInteractionCreate, t core.Translator, s core.Services) (interactions.SlashAction, error) {
			_ = ctx
			_ = e
			_ = s
			return interactions.SlashMessage{
				Create: discord.NewMessageCreate().
					WithEphemeral(true).
					WithContent(t.S("ok.pong", nil)),
			}, nil
		},
	}
}

func help() core.SlashCommand {
	return core.SlashCommand{
		Name:   "help",
		NameID: "cmd.help.name",
		DescID: "cmd.help.desc",
		CreateCommand: func(locales []string, t core.Translator) discord.ApplicationCommandCreate {
			return discord.SlashCommandCreate{
				Name: "help",
				NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.help.name"})
				}),
				Description: t.S("cmd.help.desc", nil),
				DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.help.desc"})
				}),
			}
		},
		Handle: func(ctx context.Context, e *events.ApplicationCommandInteractionCreate, t core.Translator, s core.Services) (interactions.SlashAction, error) {
			_ = ctx
			_ = e
			names := []string{}
			if s.HelpNames != nil {
				names = s.HelpNames(t.Locale)
			}
			content := t.S("cmd.help.content", map[string]any{
				"Commands": strings.Join(names, ", "),
			})
			return interactions.SlashMessage{
				Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(content),
			}, nil
		},
	}
}
