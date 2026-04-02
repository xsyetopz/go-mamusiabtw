package cmdfun

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"github.com/xsyetopz/go-mamusiabtw/internal/discordapp/core"
	"github.com/xsyetopz/go-mamusiabtw/internal/discordapp/interactions"
	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
	"github.com/xsyetopz/go-mamusiabtw/internal/integrations/kawaii"
)

const (
	funEmbedColor      = 0x5865F2
	funWarnEmbedColor  = 0xFEE75C
	funErrorEmbedColor = 0xED4245
)

func Commands() []core.SlashCommand {
	return []core.SlashCommand{
		flip(),
		roll(),
		eightBall(),
		hug(),
		pat(),
		poke(),
		shrug(),
	}
}

func flip() core.SlashCommand {
	return core.SlashCommand{
		Name:   "flip",
		NameID: "cmd.flip.name",
		DescID: "cmd.flip.desc",
		CreateCommand: func(locales []string, t core.Translator) discord.ApplicationCommandCreate {
			return discord.SlashCommandCreate{
				Name: "flip",
				NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.flip.name"})
				}),
				Description: t.S("cmd.flip.desc", nil),
				DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.flip.desc"})
				}),
			}
		},
		Handle: func(ctx context.Context, e *events.ApplicationCommandInteractionCreate, t core.Translator, s core.Services) (interactions.SlashAction, error) {
			_ = ctx
			_ = s

			const coinSides = 2
			n, err := cryptoRandIntn(coinSides)
			if err != nil {
				return nil, err
			}
			flip := "heads"
			if n == 0 {
				flip = "tails"
			}

			desc := t.S("fun.flip.result", map[string]any{
				"User":   e.User().Mention(),
				"Result": flip,
			})

			return interactions.SlashMessage{Create: discord.MessageCreate{
				Embeds: []discord.Embed{{
					Description: desc,
					Color:       funEmbedColor,
				}},
				AllowedMentions: &discord.AllowedMentions{},
			}}, nil
		},
	}
}

func roll() core.SlashCommand {
	return core.SlashCommand{
		Name:          "roll",
		NameID:        "cmd.roll.name",
		DescID:        "cmd.roll.desc",
		CreateCommand: rollCreateCommand,
		Handle:        rollHandle,
	}
}

func rollCreateCommand(locales []string, t core.Translator) discord.ApplicationCommandCreate {
	const (
		rollMinNumber   = 1
		rollMaxNumber   = 99
		rollMinSides    = 4
		rollMaxSides    = 20
		rollMinModifier = -99
		rollMaxModifier = 99
	)

	minNumber, maxNumber := rollMinNumber, rollMaxNumber
	minSides, maxSides := rollMinSides, rollMaxSides
	minMod, maxMod := rollMinModifier, rollMaxModifier

	return discord.SlashCommandCreate{
		Name: "roll",
		NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.roll.name"})
		}),
		Description: t.S("cmd.roll.desc", nil),
		DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.roll.desc"})
		}),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{
				Name: "number",
				NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.roll.opt.number.name"})
				}),
				Description: t.S("cmd.roll.opt.number.desc", nil),
				DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.roll.opt.number.desc"})
				}),
				Required: true,
				MinValue: &minNumber,
				MaxValue: &maxNumber,
			},
			discord.ApplicationCommandOptionInt{
				Name: "sides",
				NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.roll.opt.sides.name"})
				}),
				Description: t.S("cmd.roll.opt.sides.desc", nil),
				DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.roll.opt.sides.desc"})
				}),
				Required: true,
				MinValue: &minSides,
				MaxValue: &maxSides,
			},
			discord.ApplicationCommandOptionInt{
				Name: "modifier",
				NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.roll.opt.modifier.name"})
				}),
				Description: t.S("cmd.roll.opt.modifier.desc", nil),
				DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.roll.opt.modifier.desc"})
				}),
				MinValue: &minMod,
				MaxValue: &maxMod,
			},
		},
	}
}

func rollHandle(
	_ context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	_ core.Services,
) (interactions.SlashAction, error) {
	data := e.SlashCommandInteractionData()
	number := data.Int("number")
	sides := data.Int("sides")
	modifier, _ := data.OptInt("modifier")

	if !isAllowedDiceSides(sides) {
		return interactions.SlashMessage{Create: discord.MessageCreate{
			Flags: discord.MessageFlagEphemeral,
			Embeds: []discord.Embed{{
				Description: t.S("fun.roll.invalid_sides", map[string]any{"Sides": sides}),
				Color:       funWarnEmbedColor,
			}},
			AllowedMentions: &discord.AllowedMentions{},
		}}, nil
	}

	rolls := make([]int, 0, number)
	sum := 0
	for range number {
		r, err := cryptoRandIntn(sides)
		if err != nil {
			return nil, err
		}
		r++ // 1..sides
		rolls = append(rolls, r)
		sum += r
	}

	total := sum + modifier
	notation := fmtRollNotation(number, sides, modifier)

	desc := t.S("fun.roll.result", map[string]any{
		"User":     e.User().Mention(),
		"Notation": notation,
		"Total":    total,
	})

	verbose := fmt.Sprintf("%v", rolls)
	if modifier > 0 {
		verbose = fmt.Sprintf("%v + %d", rolls, modifier)
	} else if modifier < 0 {
		verbose = fmt.Sprintf("%v - %d", rolls, -modifier)
	}

	return interactions.SlashMessage{Create: discord.MessageCreate{
		Embeds: []discord.Embed{{
			Description: desc,
			Color:       funEmbedColor,
			Footer: &discord.EmbedFooter{
				Text: verbose,
			},
		}},
		AllowedMentions: &discord.AllowedMentions{},
	}}, nil
}

func eightBall() core.SlashCommand {
	return core.SlashCommand{
		Name:   "8ball",
		NameID: "cmd.8ball.name",
		DescID: "cmd.8ball.desc",
		CreateCommand: func(locales []string, t core.Translator) discord.ApplicationCommandCreate {
			minLen, maxLen := 3, 255
			return discord.SlashCommandCreate{
				Name: "8ball",
				NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.8ball.name"})
				}),
				Description: t.S("cmd.8ball.desc", nil),
				DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.8ball.desc"})
				}),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name: "question",
						NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
							return t.Registry.MustLocalize(
								i18n.Config{Locale: locale, MessageID: "cmd.8ball.opt.question.name"},
							)
						}),
						Description: t.S("cmd.8ball.opt.question.desc", nil),
						DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
							return t.Registry.MustLocalize(
								i18n.Config{Locale: locale, MessageID: "cmd.8ball.opt.question.desc"},
							)
						}),
						Required:  true,
						MinLength: &minLen,
						MaxLength: &maxLen,
					},
				},
			}
		},
		Handle: func(ctx context.Context, e *events.ApplicationCommandInteractionCreate, t core.Translator, s core.Services) (interactions.SlashAction, error) {
			_ = ctx
			_ = s

			data := e.SlashCommandInteractionData()
			question := strings.TrimSpace(data.String("question"))
			if !isOpenEndedQuestion(question) {
				return interactions.SlashMessage{Create: discord.MessageCreate{
					Flags: discord.MessageFlagEphemeral,
					Embeds: []discord.Embed{{
						Description: t.S("fun.8ball.question_error", map[string]any{"Question": question}),
						Color:       funErrorEmbedColor,
					}},
					AllowedMentions: &discord.AllowedMentions{},
				}}, nil
			}

			answers := eightBallAnswers()
			idx, err := cryptoRandIntn(len(answers))
			if err != nil {
				return nil, err
			}

			desc := t.S("fun.8ball.result", map[string]any{
				"Answer": answers[idx],
				"User":   e.User().Mention(),
			})

			return interactions.SlashMessage{Create: discord.MessageCreate{
				Embeds: []discord.Embed{{
					Description: desc,
					Color:       funEmbedColor,
				}},
				AllowedMentions: &discord.AllowedMentions{},
			}}, nil
		},
	}
}

func hug() core.SlashCommand {
	return kawaiiUserCommand(
		"hug",
		"cmd.hug.name",
		"cmd.hug.desc",
		"cmd.hug.opt.user.name",
		"cmd.hug.opt.user.desc",
		kawaii.EndpointHug,
	)
}
func pat() core.SlashCommand {
	return kawaiiUserCommand(
		"pat",
		"cmd.pat.name",
		"cmd.pat.desc",
		"cmd.pat.opt.user.name",
		"cmd.pat.opt.user.desc",
		kawaii.EndpointPat,
	)
}
func poke() core.SlashCommand {
	return kawaiiUserCommand(
		"poke",
		"cmd.poke.name",
		"cmd.poke.desc",
		"cmd.poke.opt.user.name",
		"cmd.poke.opt.user.desc",
		kawaii.EndpointPoke,
	)
}

func kawaiiUserCommand(name, nameID, descID, optNameID, optDescID string, endpoint kawaii.Endpoint) core.SlashCommand {
	return core.SlashCommand{
		Name:   name,
		NameID: nameID,
		DescID: descID,
		CreateCommand: func(locales []string, t core.Translator) discord.ApplicationCommandCreate {
			return discord.SlashCommandCreate{
				Name: name,
				NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: nameID})
				}),
				Description: t.S(descID, nil),
				DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: descID})
				}),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionUser{
						Name: "user",
						NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
							return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: optNameID})
						}),
						Description: t.S(optDescID, nil),
						DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
							return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: optDescID})
						}),
						Required: true,
					},
				},
			}
		},
		Handle: func(ctx context.Context, _ *events.ApplicationCommandInteractionCreate, t core.Translator, s core.Services) (interactions.SlashAction, error) {
			return interactions.SlashFunc(func(e *events.ApplicationCommandInteractionCreate) error {
				guildID := e.GuildID()
				if guildID == nil {
					return (interactions.SlashMessage{Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.not_in_guild", nil))}).Execute(
						e,
					)
				}

				data := e.SlashCommandInteractionData()
				user := data.User("user")

				actorID := uint64(e.User().ID)
				targetID := uint64(user.ID)
				if actorID == targetID {
					return (interactions.SlashMessage{Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("fun.kawaii.self_error", nil))}).Execute(
						e,
					)
				}

				if err := e.DeferCreateMessage(false); err != nil {
					return err
				}

				if s.Kawaii == nil {
					return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
						Embeds: &[]discord.Embed{{
							Description: t.S("fun.kawaii.error", nil),
							Color:       funErrorEmbedColor,
						}},
						AllowedMentions: &discord.AllowedMentions{},
					}}).Execute(e)
				}

				gifURL, err := s.Kawaii.FetchGIF(ctx, endpoint)
				if err != nil {
					return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
						Embeds: &[]discord.Embed{{
							Description: t.S("fun.kawaii.error", nil),
							Color:       funErrorEmbedColor,
						}},
						AllowedMentions: &discord.AllowedMentions{},
					}}).Execute(e)
				}

				desc := t.S("fun.kawaii.user_mention_only", map[string]any{
					"Emoji": endpointEmoji(endpoint),
					"User":  user.Mention(),
				})

				embed := discord.Embed{
					Description: desc,
					Color:       funEmbedColor,
					Image:       &discord.EmbedResource{URL: gifURL},
					Footer: &discord.EmbedFooter{
						Text: t.S("fun.kawaii.footer", nil),
					},
				}

				return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
					Embeds:          &[]discord.Embed{embed},
					AllowedMentions: &discord.AllowedMentions{},
				}}).Execute(e)
			}), nil
		},
	}
}

func shrug() core.SlashCommand {
	return core.SlashCommand{
		Name:   "shrug",
		NameID: "cmd.shrug.name",
		DescID: "cmd.shrug.desc",
		CreateCommand: func(locales []string, t core.Translator) discord.ApplicationCommandCreate {
			maxLen := 2000
			return discord.SlashCommandCreate{
				Name: "shrug",
				NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.shrug.name"})
				}),
				Description: t.S("cmd.shrug.desc", nil),
				DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.shrug.desc"})
				}),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name: "message",
						NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
							return t.Registry.MustLocalize(
								i18n.Config{Locale: locale, MessageID: "cmd.shrug.opt.message.name"},
							)
						}),
						Description: t.S("cmd.shrug.opt.message.desc", nil),
						DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
							return t.Registry.MustLocalize(
								i18n.Config{Locale: locale, MessageID: "cmd.shrug.opt.message.desc"},
							)
						}),
						MaxLength: &maxLen,
					},
				},
			}
		},
		Handle: func(ctx context.Context, _ *events.ApplicationCommandInteractionCreate, t core.Translator, s core.Services) (interactions.SlashAction, error) {
			return interactions.SlashFunc(func(e *events.ApplicationCommandInteractionCreate) error {
				guildID := e.GuildID()
				if guildID == nil {
					return (interactions.SlashMessage{Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.not_in_guild", nil))}).Execute(
						e,
					)
				}

				data := e.SlashCommandInteractionData()
				message, _ := data.OptString("message")
				message = strings.TrimSpace(message)

				if err := e.DeferCreateMessage(false); err != nil {
					return err
				}

				if s.Kawaii == nil {
					return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
						Embeds: &[]discord.Embed{{
							Description: t.S("fun.kawaii.error", nil),
							Color:       funErrorEmbedColor,
						}},
						AllowedMentions: &discord.AllowedMentions{},
					}}).Execute(e)
				}

				gifURL, err := s.Kawaii.FetchGIF(ctx, kawaii.EndpointShrug)
				if err != nil {
					return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
						Embeds: &[]discord.Embed{{
							Description: t.S("fun.kawaii.error", nil),
							Color:       funErrorEmbedColor,
						}},
						AllowedMentions: &discord.AllowedMentions{},
					}}).Execute(e)
				}

				embed := discord.Embed{
					Color: funEmbedColor,
					Image: &discord.EmbedResource{URL: gifURL},
					Footer: &discord.EmbedFooter{
						Text: t.S("fun.kawaii.footer", nil),
					},
				}

				var content *string
				if message != "" {
					content = &message
				}

				return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
					Content:         content,
					Embeds:          &[]discord.Embed{embed},
					AllowedMentions: &discord.AllowedMentions{},
				}}).Execute(e)
			}), nil
		},
	}
}

func cryptoRandIntn(upperBound int) (int, error) {
	if upperBound <= 0 {
		return 0, fmt.Errorf("invalid upperBound %d", upperBound)
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(upperBound)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

func isAllowedDiceSides(sides int) bool {
	const (
		diceSidesD4  = 4
		diceSidesD6  = 6
		diceSidesD8  = 8
		diceSidesD10 = 10
		diceSidesD12 = 12
		diceSidesD20 = 20
	)

	switch sides {
	case diceSidesD4, diceSidesD6, diceSidesD8, diceSidesD10, diceSidesD12, diceSidesD20:
		return true
	default:
		return false
	}
}

func fmtRollNotation(number, sides, modifier int) string {
	base := fmt.Sprintf("%dd%d", number, sides)
	if modifier > 0 {
		return fmt.Sprintf("%s+%d", base, modifier)
	}
	if modifier < 0 {
		return fmt.Sprintf("%s-%d", base, -modifier)
	}
	return base
}

func isOpenEndedQuestion(q string) bool {
	// Keep this simple and predictable.
	const minQuestionLen = 3
	if len(q) < minQuestionLen {
		return false
	}
	if !strings.HasSuffix(q, "?") && !strings.HasSuffix(q, ".") && !strings.HasSuffix(q, "!") {
		return false
	}
	return true
}

func eightBallAnswers() []string {
	return []string{
		"It is certain.",
		"It is decidedly so.",
		"Without a doubt.",
		"Yes - definitely.",
		"You may rely on it.",
		"As I see it, yes.",
		"Most likely.",
		"Outlook good.",
		"Yes.",
		"Signs point to yes.",
		"Reply hazy, try again.",
		"Ask again later.",
		"Better not tell you now.",
		"Cannot predict now.",
		"Concentrate and ask again.",
		"Don't count on it.",
		"My reply is no.",
		"My sources say no.",
		"Outlook not so good.",
		"Very doubtful.",
	}
}

func endpointEmoji(endpoint kawaii.Endpoint) string {
	switch endpoint {
	case kawaii.EndpointHug:
		return "🤗"
	case kawaii.EndpointPat:
		return "🫳"
	case kawaii.EndpointPoke:
		return "👉"
	case kawaii.EndpointShrug:
		return "🤷"
	default:
		return "✨"
	}
}
