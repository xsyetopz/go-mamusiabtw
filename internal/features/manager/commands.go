package manager

import (
	"context"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/omit"
	"github.com/disgoorg/snowflake/v2"

	"github.com/xsyetopz/go-mamusiabtw/internal/features/commandapi"
	"github.com/xsyetopz/go-mamusiabtw/internal/platform/discord/discordutil"
	"github.com/xsyetopz/go-mamusiabtw/internal/platform/discord/interactions"
	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
)

const (
	managerSuccessColor = 0x57F287
	managerErrorColor   = 0xED4245
)

func Commands() []commandapi.SlashCommand {
	return []commandapi.SlashCommand{
		slowmode(),
		nick(),
		purge(),
	}
}

func slowmode() commandapi.SlashCommand {
	return commandapi.SlashCommand{
		Name:          "slowmode",
		NameID:        "cmd.slowmode.name",
		DescID:        "cmd.slowmode.desc",
		CreateCommand: slowmodeCreateCommand,
		Handle:        slowmodeHandle,
	}
}

func slowmodeCreateCommand(locales []string, t commandapi.Translator) discord.ApplicationCommandCreate {
	minSeconds, maxSeconds := 1, 21600
	perm := discord.PermissionManageChannels
	return discord.SlashCommandCreate{
		Name: "slowmode",
		NameLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{
				Locale:    locale,
				MessageID: "cmd.slowmode.name",
			})
		}),
		Description: t.S("cmd.slowmode.desc", nil),
		DescriptionLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{
				Locale:    locale,
				MessageID: "cmd.slowmode.desc",
			})
		}),
		DefaultMemberPermissions: omit.NewPtr(perm),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionChannel{
				Name: "channel",
				NameLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{
						Locale:    locale,
						MessageID: "cmd.slowmode.opt.channel.name",
					})
				}),
				Description: t.S("cmd.slowmode.opt.channel.desc", nil),
				DescriptionLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{
						Locale:    locale,
						MessageID: "cmd.slowmode.opt.channel.desc",
					})
				}),
				ChannelTypes: []discord.ChannelType{
					discord.ChannelTypeGuildText,
					discord.ChannelTypeGuildVoice,
					discord.ChannelTypeGuildStageVoice,
					discord.ChannelTypeGuildForum,
				},
			},
			discord.ApplicationCommandOptionInt{
				Name: "seconds",
				NameLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{
						Locale:    locale,
						MessageID: "cmd.slowmode.opt.seconds.name",
					})
				}),
				Description: t.S("cmd.slowmode.opt.seconds.desc", nil),
				DescriptionLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{
						Locale:    locale,
						MessageID: "cmd.slowmode.opt.seconds.desc",
					})
				}),
				MinValue: &minSeconds,
				MaxValue: &maxSeconds,
			},
		},
	}
}

func slowmodeHandle(
	_ context.Context,
	_ *events.ApplicationCommandInteractionCreate,
	t commandapi.Translator,
	_ commandapi.Services,
) (interactions.SlashAction, error) {
	return interactions.SlashFunc(func(e *events.ApplicationCommandInteractionCreate) error {
		guildID := e.GuildID()
		if guildID == nil {
			return (interactions.SlashMessage{Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.not_in_guild", nil))}).Execute(
				e,
			)
		}

		data := e.SlashCommandInteractionData()
		seconds, hasSeconds := data.OptInt("seconds")
		if !hasSeconds {
			seconds = 0
		}
		if seconds < 0 {
			seconds = 0
		}

		channelID := e.Channel().ID()
		if ch, hasChannel := data.OptChannel("channel"); hasChannel {
			channelID = ch.ID
		}

		if err := (interactions.SlashDefer{Ephemeral: true}).Execute(e); err != nil {
			return err
		}

		updSeconds := seconds
		_, err := e.Client().Rest.UpdateChannel(channelID, discord.GuildTextChannelUpdate{
			RateLimitPerUser: &updSeconds,
		})
		if err != nil {
			return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
				Embeds: &[]discord.Embed{{
					Description: t.S("mgr.slowmode.error", nil),
					Color:       managerErrorColor,
				}},
				AllowedMentions: &discord.AllowedMentions{},
			}}).Execute(e)
		}

		var desc string
		if seconds == 0 {
			desc = t.S("mgr.slowmode.removed", map[string]any{"Channel": discord.ChannelMention(channelID)})
		} else {
			desc = t.S(
				"mgr.slowmode.set",
				map[string]any{"Channel": discord.ChannelMention(channelID), "Seconds": seconds},
			)
		}

		return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
			Embeds: &[]discord.Embed{{
				Description: desc,
				Color:       managerSuccessColor,
			}},
			AllowedMentions: &discord.AllowedMentions{},
		}}).Execute(e)
	}), nil
}

func nick() commandapi.SlashCommand {
	return commandapi.SlashCommand{
		Name:          "nick",
		NameID:        "cmd.nick.name",
		DescID:        "cmd.nick.desc",
		CreateCommand: nickCreateCommand,
		Handle:        nickHandle,
	}
}

func nickCreateCommand(locales []string, t commandapi.Translator) discord.ApplicationCommandCreate {
	minLen, maxLen := 1, 32
	perm := discord.PermissionManageNicknames

	return discord.SlashCommandCreate{
		Name: "nick",
		NameLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.nick.name"})
		}),
		Description: t.S("cmd.nick.desc", nil),
		DescriptionLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.nick.desc"})
		}),
		DefaultMemberPermissions: omit.NewPtr(perm),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{
				Name: "user",
				NameLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(
						i18n.Config{Locale: locale, MessageID: "cmd.nick.opt.user.name"},
					)
				}),
				Description: t.S("cmd.nick.opt.user.desc", nil),
				DescriptionLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(
						i18n.Config{Locale: locale, MessageID: "cmd.nick.opt.user.desc"},
					)
				}),
				Required: true,
			},
			discord.ApplicationCommandOptionString{
				Name: "nickname",
				NameLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(
						i18n.Config{Locale: locale, MessageID: "cmd.nick.opt.nickname.name"},
					)
				}),
				Description: t.S("cmd.nick.opt.nickname.desc", nil),
				DescriptionLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(
						i18n.Config{Locale: locale, MessageID: "cmd.nick.opt.nickname.desc"},
					)
				}),
				MinLength: &minLen,
				MaxLength: &maxLen,
			},
		},
	}
}

func nickHandle(
	_ context.Context,
	_ *events.ApplicationCommandInteractionCreate,
	t commandapi.Translator,
	_ commandapi.Services,
) (interactions.SlashAction, error) {
	return interactions.SlashFunc(func(e *events.ApplicationCommandInteractionCreate) error {
		guildID := e.GuildID()
		if guildID == nil {
			return (interactions.SlashMessage{
				Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.not_in_guild", nil)),
			}).Execute(e)
		}

		data := e.SlashCommandInteractionData()
		user := data.User("user")
		nickname, _ := data.OptString("nickname")
		nickname = strings.TrimSpace(nickname)

		actorID := uint64(e.User().ID)
		targetID := uint64(user.ID)
		if actorID == targetID {
			return (interactions.SlashMessage{
				Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("mgr.nick.self_error", nil)),
			}).Execute(e)
		}
		if user.Bot || user.System {
			return (interactions.SlashMessage{
				Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("mod.warn.bot", nil)),
			}).Execute(e)
		}

		if err := (interactions.SlashDefer{Ephemeral: true}).Execute(e); err != nil {
			return err
		}

		nick := nickname
		if _, err := e.Client().Rest.UpdateMember(*guildID, user.ID, discord.MemberUpdate{Nick: &nick}); err != nil {
			return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
				Embeds: &[]discord.Embed{{
					Description: t.S("mgr.nick.error", map[string]any{"User": user.Mention(), "Nickname": nickname}),
					Color:       managerErrorColor,
				}},
				AllowedMentions: &discord.AllowedMentions{},
			}}).Execute(e)
		}

		var desc string
		if nickname == "" {
			desc = t.S("mgr.nick.reset", map[string]any{"User": user.Mention()})
		} else {
			desc = t.S("mgr.nick.set", map[string]any{"User": user.Mention(), "Nickname": nickname})
		}

		return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
			Embeds: &[]discord.Embed{{
				Description: desc,
				Color:       managerSuccessColor,
			}},
			AllowedMentions: &discord.AllowedMentions{},
		}}).Execute(e)
	}), nil
}

func purge() commandapi.SlashCommand {
	return commandapi.SlashCommand{
		Name:          "purge",
		NameID:        "cmd.purge.name",
		DescID:        "cmd.purge.desc",
		CreateCommand: purgeCreateCommand,
		Handle:        purgeHandle,
	}
}

type purgeLocalizer func(id string) map[discord.Locale]string

func purgeCreateCommand(locales []string, t commandapi.Translator) discord.ApplicationCommandCreate {
	const (
		purgeMinAll   = 2
		purgeMinCount = 1
		purgeMaxCount = 100
	)

	perm := discord.PermissionManageMessages
	loc := func(id string) map[discord.Locale]string {
		return commandapi.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: id})
		})
	}

	return discord.SlashCommandCreate{
		Name:                     "purge",
		NameLocalizations:        loc("cmd.purge.name"),
		Description:              t.S("cmd.purge.desc", nil),
		DescriptionLocalizations: loc("cmd.purge.desc"),
		DefaultMemberPermissions: omit.NewPtr(perm),
		Options: []discord.ApplicationCommandOption{
			purgeSubAll(t, loc, purgeMinAll, purgeMaxCount),
			purgeSubBefore(t, loc, purgeMinCount, purgeMaxCount),
			purgeSubAfter(t, loc, purgeMinCount, purgeMaxCount),
			purgeSubAround(t, loc, purgeMinAll, purgeMaxCount),
		},
	}
}

func purgeSubAll(t commandapi.Translator, loc purgeLocalizer, minAll, maxAll int) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "all",
		NameLocalizations:        loc("cmd.purge.sub.all.name"),
		Description:              t.S("cmd.purge.sub.all.desc", nil),
		DescriptionLocalizations: loc("cmd.purge.sub.all.desc"),
		Options: []discord.ApplicationCommandOption{
			purgeOptionCount(t, loc, minAll, maxAll),
		},
	}
}

func purgeSubBefore(
	t commandapi.Translator,
	loc purgeLocalizer,
	minCount, maxCount int,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "before",
		NameLocalizations:        loc("cmd.purge.sub.before.name"),
		Description:              t.S("cmd.purge.sub.before.desc", nil),
		DescriptionLocalizations: loc("cmd.purge.sub.before.desc"),
		Options: []discord.ApplicationCommandOption{
			purgeOptionMessage(t, loc),
			purgeOptionCount(t, loc, minCount, maxCount),
		},
	}
}

func purgeSubAfter(
	t commandapi.Translator,
	loc purgeLocalizer,
	minCount, maxCount int,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "after",
		NameLocalizations:        loc("cmd.purge.sub.after.name"),
		Description:              t.S("cmd.purge.sub.after.desc", nil),
		DescriptionLocalizations: loc("cmd.purge.sub.after.desc"),
		Options: []discord.ApplicationCommandOption{
			purgeOptionMessage(t, loc),
			purgeOptionCount(t, loc, minCount, maxCount),
		},
	}
}

func purgeSubAround(
	t commandapi.Translator,
	loc purgeLocalizer,
	minAll, maxAll int,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "around",
		NameLocalizations:        loc("cmd.purge.sub.around.name"),
		Description:              t.S("cmd.purge.sub.around.desc", nil),
		DescriptionLocalizations: loc("cmd.purge.sub.around.desc"),
		Options: []discord.ApplicationCommandOption{
			purgeOptionMessage(t, loc),
			purgeOptionCount(t, loc, minAll, maxAll),
		},
	}
}

func purgeOptionMessage(t commandapi.Translator, loc purgeLocalizer) discord.ApplicationCommandOptionString {
	maxLen := 255
	return discord.ApplicationCommandOptionString{
		Name:                     "message",
		NameLocalizations:        loc("cmd.purge.opt.message.name"),
		Description:              t.S("cmd.purge.opt.message.desc", nil),
		DescriptionLocalizations: loc("cmd.purge.opt.message.desc"),
		Required:                 true,
		MaxLength:                &maxLen,
	}
}

func purgeOptionCount(
	t commandapi.Translator,
	loc purgeLocalizer,
	minValue, maxValue int,
) discord.ApplicationCommandOptionInt {
	return discord.ApplicationCommandOptionInt{
		Name:                     "count",
		NameLocalizations:        loc("cmd.purge.opt.count.name"),
		Description:              t.S("cmd.purge.opt.count.desc", nil),
		DescriptionLocalizations: loc("cmd.purge.opt.count.desc"),
		MinValue:                 &minValue,
		MaxValue:                 &maxValue,
	}
}

func purgeHandle(
	ctx context.Context,
	_ *events.ApplicationCommandInteractionCreate,
	t commandapi.Translator,
	_ commandapi.Services,
) (interactions.SlashAction, error) {
	return interactions.SlashFunc(func(e *events.ApplicationCommandInteractionCreate) error {
		return purgeExecute(ctx, e, t)
	}), nil
}

func purgeExecute(_ context.Context, e *events.ApplicationCommandInteractionCreate, t commandapi.Translator) error {
	const (
		purgeDefaultCount = 2
		purgeMinCount     = 1
		purgeMaxCount     = 100
	)

	if e.GuildID() == nil {
		return (interactions.SlashMessage{Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.not_in_guild", nil))}).Execute(
			e,
		)
	}

	data := e.SlashCommandInteractionData()
	sub := data.SubCommandName
	if sub == nil {
		return (interactions.SlashMessage{Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil))}).Execute(
			e,
		)
	}

	channelID := e.Channel().ID()

	limit := purgeDefaultCount
	if v, ok := data.OptInt("count"); ok {
		limit = v
	}
	if limit < purgeMinCount {
		limit = purgeMinCount
	}
	if limit > purgeMaxCount {
		limit = purgeMaxCount
	}

	around, before, after, ok := purgeAnchorIDs(*sub, data)
	if !ok {
		return (interactions.SlashMessage{Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("mgr.purge.invalid_message", nil))}).Execute(
			e,
		)
	}

	if err := (interactions.SlashDefer{Ephemeral: true}).Execute(e); err != nil {
		return err
	}

	msgs, err := e.Client().Rest.GetMessages(channelID, around, before, after, limit)
	if err != nil {
		return purgeUpdateError(e, t)
	}

	ids := make([]snowflake.ID, 0, len(msgs))
	for _, m := range msgs {
		ids = append(ids, m.ID)
	}

	deleted := deleteMessagesBestEffort(e.Client().Rest, channelID, ids, time.Now())

	return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
		Embeds: &[]discord.Embed{{
			Description: t.S("mgr.purge.success", map[string]any{"Count": deleted}),
			Color:       managerSuccessColor,
		}},
		AllowedMentions: &discord.AllowedMentions{},
	}}).Execute(e)
}

func purgeUpdateError(e *events.ApplicationCommandInteractionCreate, t commandapi.Translator) error {
	return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
		Embeds: &[]discord.Embed{{
			Description: t.S("mgr.purge.error", nil),
			Color:       managerErrorColor,
		}},
		AllowedMentions: &discord.AllowedMentions{},
	}}).Execute(e)
}

func purgeAnchorIDs(
	sub string,
	data discord.SlashCommandInteractionData,
) (snowflake.ID, snowflake.ID, snowflake.ID, bool) {
	switch sub {
	case "all":
		return 0, 0, 0, true
	case "before":
		id, parsed := discordutil.ParseMessageID(data.String("message"))
		return 0, id, 0, parsed
	case "after":
		id, parsed := discordutil.ParseMessageID(data.String("message"))
		return 0, 0, id, parsed
	case "around":
		id, parsed := discordutil.ParseMessageID(data.String("message"))
		return id, 0, 0, parsed
	default:
		return 0, 0, 0, false
	}
}

func deleteMessagesBestEffort(
	r rest.Rest,
	channelID snowflake.ID,
	messageIDs []snowflake.ID,
	now time.Time,
) int {
	if len(messageIDs) == 0 {
		return 0
	}

	// Discord bulk-delete limitation: messages older than 14 days cannot be bulk deleted.
	const discordBulkDeleteMaxAge = 14 * 24 * time.Hour
	const discordBulkDeleteSafetyBuffer = time.Hour // avoid edge cases
	cutoff := now.Add(-discordBulkDeleteMaxAge).Add(discordBulkDeleteSafetyBuffer)

	var bulkIDs []snowflake.ID
	var singleIDs []snowflake.ID

	for _, id := range messageIDs {
		if id == 0 {
			continue
		}
		if id.Time().Before(cutoff) {
			singleIDs = append(singleIDs, id)
		} else {
			bulkIDs = append(bulkIDs, id)
		}
	}

	deleted := 0

	// Bulk delete in chunks of <= 100.
	const bulkDeleteChunkMax = 100
	for len(bulkIDs) > 0 {
		chunk := bulkIDs
		if len(chunk) > bulkDeleteChunkMax {
			chunk = bulkIDs[:bulkDeleteChunkMax]
		}
		bulkIDs = bulkIDs[len(chunk):]

		if len(chunk) == 1 {
			singleIDs = append(singleIDs, chunk[0])
			continue
		}

		if err := r.BulkDeleteMessages(channelID, chunk); err != nil {
			// Fall back to single deletes for this chunk.
			singleIDs = append(singleIDs, chunk...)
			continue
		}
		deleted += len(chunk)
	}

	for _, id := range singleIDs {
		if err := r.DeleteMessage(channelID, id); err != nil {
			// best effort
			continue
		}
		deleted++
	}

	return deleted
}
