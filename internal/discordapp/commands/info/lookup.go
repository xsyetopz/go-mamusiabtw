package cmdinfo

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	"github.com/xsuetopz/go-mamusiabtw/internal/discordapp/core"
	"github.com/xsuetopz/go-mamusiabtw/internal/discordapp/interactions"
	"github.com/xsuetopz/go-mamusiabtw/internal/i18n"
)

func lookup() core.SlashCommand {
	return core.SlashCommand{
		Name:          "lookup",
		NameID:        "cmd.lookup.name",
		DescID:        "cmd.lookup.desc",
		CreateCommand: lookupCreateCommand,
		Handle:        lookupHandle,
	}
}

func lookupCreateCommand(locales []string, t core.Translator) discord.ApplicationCommandCreate {
	loc := func(id string) map[discord.Locale]string {
		return localize(locales, t, id)
	}

	return discord.SlashCommandCreate{
		Name:                     "lookup",
		NameLocalizations:        loc("cmd.lookup.name"),
		Description:              t.S("cmd.lookup.desc", nil),
		DescriptionLocalizations: loc("cmd.lookup.desc"),
		Options: []discord.ApplicationCommandOption{
			lookupUserSubCommand(loc, t),
			lookupGuildSubCommand(loc, t),
			lookupRoleSubCommand(loc, t),
			lookupChannelSubCommand(loc, t),
		},
	}
}

func lookupUserSubCommand(
	loc func(id string) map[discord.Locale]string,
	t core.Translator,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "user",
		NameLocalizations:        loc("cmd.lookup.sub.user.name"),
		Description:              t.S("cmd.lookup.sub.user.desc", nil),
		DescriptionLocalizations: loc("cmd.lookup.sub.user.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{
				Name:                     "user",
				NameLocalizations:        loc("cmd.lookup.sub.user.opt.user.name"),
				Description:              t.S("cmd.lookup.sub.user.opt.user.desc", nil),
				DescriptionLocalizations: loc("cmd.lookup.sub.user.opt.user.desc"),
				Required:                 false,
			},
		},
	}
}

func lookupGuildSubCommand(
	loc func(id string) map[discord.Locale]string,
	t core.Translator,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "guild",
		NameLocalizations:        loc("cmd.lookup.sub.guild.name"),
		Description:              t.S("cmd.lookup.sub.guild.desc", nil),
		DescriptionLocalizations: loc("cmd.lookup.sub.guild.desc"),
	}
}

func lookupRoleSubCommand(
	loc func(id string) map[discord.Locale]string,
	t core.Translator,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "role",
		NameLocalizations:        loc("cmd.lookup.sub.role.name"),
		Description:              t.S("cmd.lookup.sub.role.desc", nil),
		DescriptionLocalizations: loc("cmd.lookup.sub.role.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionRole{
				Name:                     "role",
				NameLocalizations:        loc("cmd.lookup.sub.role.opt.role.name"),
				Description:              t.S("cmd.lookup.sub.role.opt.role.desc", nil),
				DescriptionLocalizations: loc("cmd.lookup.sub.role.opt.role.desc"),
				Required:                 true,
			},
		},
	}
}

func lookupChannelSubCommand(
	loc func(id string) map[discord.Locale]string,
	t core.Translator,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "channel",
		NameLocalizations:        loc("cmd.lookup.sub.channel.name"),
		Description:              t.S("cmd.lookup.sub.channel.desc", nil),
		DescriptionLocalizations: loc("cmd.lookup.sub.channel.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionChannel{
				Name:                     "channel",
				NameLocalizations:        loc("cmd.lookup.sub.channel.opt.channel.name"),
				Description:              t.S("cmd.lookup.sub.channel.opt.channel.desc", nil),
				DescriptionLocalizations: loc("cmd.lookup.sub.channel.opt.channel.desc"),
				Required:                 true,
			},
		},
	}
}

func lookupHandle(
	ctx context.Context,
	_ *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	_ core.Services,
) (interactions.SlashAction, error) {
	return interactions.SlashFunc(func(e *events.ApplicationCommandInteractionCreate) error {
		data := e.SlashCommandInteractionData()
		sub := data.SubCommandName
		if sub == nil {
			return (interactions.SlashMessage{
				Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil)),
			}).Execute(e)
		}

		switch strings.ToLower(strings.TrimSpace(*sub)) {
		case "user":
			return lookupUser(ctx, e, t)
		case "guild":
			return lookupGuild(ctx, e, t)
		case "role":
			return lookupRole(e, t)
		case "channel":
			return lookupChannel(e, t)
		default:
			return (interactions.SlashMessage{
				Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil)),
			}).Execute(e)
		}
	}), nil
}

func lookupUser(ctx context.Context, e *events.ApplicationCommandInteractionCreate, t core.Translator) error {
	_ = ctx

	data := e.SlashCommandInteractionData()
	targetID := e.User().ID
	if u, ok := data.OptUser("user"); ok && u.ID != 0 {
		targetID = u.ID
	}

	if err := (interactions.SlashDefer{Ephemeral: true}).Execute(e); err != nil {
		return err
	}

	full, err := e.Client().Rest.GetUser(targetID)
	if err != nil || full == nil {
		return updateLookupErr(e, t.S("info.lookup.user.error", nil))
	}

	member := lookupUserMember(e, full.ID)
	embed := buildUserLookupEmbed(full, member, t)

	return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
		Embeds:          &[]discord.Embed{embed},
		AllowedMentions: &discord.AllowedMentions{},
	}}).Execute(e)
}

func lookupUserMember(e *events.ApplicationCommandInteractionCreate, userID snowflake.ID) *discord.Member {
	if e == nil {
		return nil
	}
	gid := e.GuildID()
	if gid == nil {
		return nil
	}
	m, err := e.Client().Rest.GetMember(*gid, userID)
	if err != nil || m == nil {
		return nil
	}
	return m
}

func buildUserLookupEmbed(full *discord.User, member *discord.Member, t core.Translator) discord.Embed {
	color := infoEmbedColor
	if full != nil && full.AccentColor != nil && *full.AccentColor != 0 {
		color = *full.AccentColor
	}

	embed := discord.Embed{
		Title:  full.EffectiveName(),
		Color:  color,
		Fields: userLookupFields(full, member, t),
		Footer: &discord.EmbedFooter{Text: "🆔" + full.ID.String()},
	}
	ts := full.ID.Time()
	embed.Timestamp = &ts

	applyUserLookupMedia(&embed, full, member)
	return embed
}

func userLookupFields(full *discord.User, member *discord.Member, t core.Translator) []discord.EmbedField {
	inlineTrue := true
	fields := []discord.EmbedField{
		{Name: t.S("info.lookup.user.field.bot", nil), Value: boolString(full.Bot), Inline: &inlineTrue},
		{Name: t.S("info.lookup.user.field.system", nil), Value: boolString(full.System), Inline: &inlineTrue},
		{
			Name:   t.S("info.lookup.user.field.locale", nil),
			Value:  strings.TrimSpace(t.Locale.Code()),
			Inline: &inlineTrue,
		},
		{
			Name:   t.S("info.lookup.user.field.created", nil),
			Value:  discordTimestamp(full.ID.Time()),
			Inline: &inlineTrue,
		},
	}

	if member == nil {
		return fields
	}
	if member.JoinedAt != nil && !member.JoinedAt.IsZero() {
		fields = append(fields, discord.EmbedField{
			Name:   t.S("info.lookup.user.field.joined", nil),
			Value:  discordTimestamp(*member.JoinedAt),
			Inline: &inlineTrue,
		})
	}
	if len(member.RoleIDs) > 0 {
		fields = append(fields, discord.EmbedField{
			Name:   t.S("info.lookup.user.field.roles", nil),
			Value:  strconv.Itoa(len(member.RoleIDs)),
			Inline: &inlineTrue,
		})
	}

	return fields
}

func applyUserLookupMedia(embed *discord.Embed, full *discord.User, member *discord.Member) {
	if embed == nil || full == nil {
		return
	}

	if member != nil {
		if u := strings.TrimSpace(member.EffectiveAvatarURL()); u != "" {
			embed.Thumbnail = &discord.EmbedResource{URL: u}
		}
		if banner := member.BannerURL(); banner != nil && strings.TrimSpace(*banner) != "" {
			embed.Image = &discord.EmbedResource{URL: *banner}
		}
	}

	if embed.Thumbnail == nil {
		if u := strings.TrimSpace(full.EffectiveAvatarURL()); u != "" {
			embed.Thumbnail = &discord.EmbedResource{URL: u}
		}
	}
	if embed.Image == nil {
		if banner := full.BannerURL(); banner != nil && strings.TrimSpace(*banner) != "" {
			embed.Image = &discord.EmbedResource{URL: *banner}
		}
	}
}

func lookupGuild(_ context.Context, e *events.ApplicationCommandInteractionCreate, t core.Translator) error {
	guildID := e.GuildID()
	if guildID == nil {
		return (interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.not_in_guild", nil)),
		}).Execute(e)
	}

	if err := (interactions.SlashDefer{Ephemeral: true}).Execute(e); err != nil {
		return err
	}

	g, err := e.Client().Rest.GetGuild(*guildID, true)
	if err != nil || g == nil {
		return updateLookupErr(e, t.S("info.lookup.guild.error", nil))
	}

	owner, _ := e.Client().Rest.GetUser(g.OwnerID)

	channels, err := e.Client().Rest.GetGuildChannels(*guildID)
	if err != nil {
		return updateLookupErr(e, t.S("info.lookup.guild.error", nil))
	}

	memberCount := g.ApproximateMemberCount
	if memberCount <= 0 && g.MemberCount > 0 {
		memberCount = g.MemberCount
	}

	desc := ""
	if g.Description != nil {
		desc = strings.TrimSpace(*g.Description)
	}

	inlineTrue := true
	embed := discord.Embed{
		Title:       strings.TrimSpace(g.Name),
		Description: desc,
		Color:       infoEmbedColor,
		Fields: []discord.EmbedField{
			{Name: t.S("info.lookup.guild.field.roles", nil), Value: strconv.Itoa(len(g.Roles)), Inline: &inlineTrue},
			{Name: t.S("info.lookup.guild.field.emojis", nil), Value: strconv.Itoa(len(g.Emojis)), Inline: &inlineTrue},
			{
				Name:   t.S("info.lookup.guild.field.stickers", nil),
				Value:  strconv.Itoa(len(g.Stickers)),
				Inline: &inlineTrue,
			},
			{Name: t.S("info.lookup.guild.field.members", nil), Value: strconv.Itoa(memberCount), Inline: &inlineTrue},
			{
				Name:   t.S("info.lookup.guild.field.channels", nil),
				Value:  strconv.Itoa(len(channels)),
				Inline: &inlineTrue,
			},
			{
				Name:   t.S("info.lookup.guild.field.created", nil),
				Value:  discordTimestamp(g.ID.Time()),
				Inline: &inlineTrue,
			},
		},
		Footer: &discord.EmbedFooter{
			Text: "🆔" + g.ID.String(),
		},
	}
	ts := g.ID.Time()
	embed.Timestamp = &ts

	if owner != nil {
		embed.Author = &discord.EmbedAuthor{Name: owner.Username}
		if u := owner.EffectiveAvatarURL(); strings.TrimSpace(u) != "" {
			embed.Author.IconURL = u
		}
	}

	if icon := g.IconURL(); icon != nil && strings.TrimSpace(*icon) != "" {
		embed.Thumbnail = &discord.EmbedResource{URL: *icon}
	}
	if banner := g.BannerURL(); banner != nil && strings.TrimSpace(*banner) != "" {
		embed.Image = &discord.EmbedResource{URL: *banner}
	}

	return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
		Embeds:          &[]discord.Embed{embed},
		AllowedMentions: &discord.AllowedMentions{},
	}}).Execute(e)
}

func lookupRole(e *events.ApplicationCommandInteractionCreate, t core.Translator) error {
	guildID := e.GuildID()
	if guildID == nil {
		return (interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.not_in_guild", nil)),
		}).Execute(e)
	}

	data := e.SlashCommandInteractionData()
	role := data.Role("role")
	if role.ID == 0 {
		return (interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil)),
		}).Execute(e)
	}

	inlineTrue := true
	color := infoEmbedColor
	if role.Color != 0 {
		color = role.Color
	}

	embed := discord.Embed{
		Title: strings.TrimSpace(role.Name),
		Color: color,
		Fields: []discord.EmbedField{
			{
				Name:   t.S("info.lookup.role.field.mention", nil),
				Value:  discord.RoleMention(role.ID),
				Inline: &inlineTrue,
			},
			{
				Name:   t.S("info.lookup.role.field.position", nil),
				Value:  strconv.Itoa(role.Position),
				Inline: &inlineTrue,
			},
			{
				Name:   t.S("info.lookup.role.field.hoist", nil),
				Value:  boolString(role.Hoist),
				Inline: &inlineTrue,
			},
			{
				Name:   t.S("info.lookup.role.field.mentionable", nil),
				Value:  boolString(role.Mentionable),
				Inline: &inlineTrue,
			},
			{
				Name:   t.S("info.lookup.role.field.managed", nil),
				Value:  boolString(role.Managed),
				Inline: &inlineTrue,
			},
			{
				Name:   t.S("info.lookup.role.field.permissions", nil),
				Value:  fmt.Sprintf("`%d`", int64(role.Permissions)),
				Inline: &inlineTrue,
			},
			{
				Name:   t.S("info.lookup.role.field.created", nil),
				Value:  discordTimestamp(role.CreatedAt()),
				Inline: &inlineTrue,
			},
		},
		Footer: &discord.EmbedFooter{Text: "🆔" + role.ID.String()},
	}

	ts := role.CreatedAt()
	embed.Timestamp = &ts

	return (interactions.SlashMessage{Create: discord.MessageCreate{
		Flags:           discord.MessageFlagEphemeral,
		Embeds:          []discord.Embed{embed},
		AllowedMentions: &discord.AllowedMentions{},
	}}).Execute(e)
}

func lookupChannel(e *events.ApplicationCommandInteractionCreate, t core.Translator) error {
	guildID := e.GuildID()
	if guildID == nil {
		return (interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.not_in_guild", nil)),
		}).Execute(e)
	}

	data := e.SlashCommandInteractionData()
	ch := data.Channel("channel")
	if ch.ID == 0 {
		return (interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil)),
		}).Execute(e)
	}

	inlineTrue := true
	fields := []discord.EmbedField{
		{
			Name:   t.S("info.lookup.channel.field.mention", nil),
			Value:  discord.ChannelMention(ch.ID),
			Inline: &inlineTrue,
		},
		{
			Name:   t.S("info.lookup.channel.field.type", nil),
			Value:  channelTypeName(ch.Type),
			Inline: &inlineTrue,
		},
		{
			Name:   t.S("info.lookup.channel.field.permissions", nil),
			Value:  fmt.Sprintf("`%d`", int64(ch.Permissions)),
			Inline: &inlineTrue,
		},
		{
			Name:   t.S("info.lookup.channel.field.created", nil),
			Value:  discordTimestamp(ch.ID.Time()),
			Inline: &inlineTrue,
		},
	}

	if ch.ParentID != 0 {
		fields = append(fields, discord.EmbedField{
			Name:   t.S("info.lookup.channel.field.parent", nil),
			Value:  discord.ChannelMention(ch.ParentID),
			Inline: &inlineTrue,
		})
	}

	embed := discord.Embed{
		Title:  strings.TrimSpace(ch.Name),
		Color:  infoEmbedColor,
		Fields: fields,
		Footer: &discord.EmbedFooter{Text: "🆔" + ch.ID.String()},
	}
	ts := ch.ID.Time()
	embed.Timestamp = &ts

	return (interactions.SlashMessage{Create: discord.MessageCreate{
		Flags:           discord.MessageFlagEphemeral,
		Embeds:          []discord.Embed{embed},
		AllowedMentions: &discord.AllowedMentions{},
	}}).Execute(e)
}

func channelTypeName(t discord.ChannelType) string {
	switch t {
	case discord.ChannelTypeGuildText:
		return "guild_text"
	case discord.ChannelTypeGuildVoice:
		return "guild_voice"
	case discord.ChannelTypeGuildCategory:
		return "guild_category"
	case discord.ChannelTypeGuildNews:
		return "guild_news"
	case discord.ChannelTypeGuildNewsThread:
		return "guild_news_thread"
	case discord.ChannelTypeGuildPublicThread:
		return "guild_public_thread"
	case discord.ChannelTypeGuildPrivateThread:
		return "guild_private_thread"
	case discord.ChannelTypeGuildStageVoice:
		return "guild_stage_voice"
	case discord.ChannelTypeGuildDirectory:
		return "guild_directory"
	case discord.ChannelTypeGuildForum:
		return "guild_forum"
	case discord.ChannelTypeGuildMedia:
		return "guild_media"
	case discord.ChannelTypeDM:
		return "dm"
	case discord.ChannelTypeGroupDM:
		return "group_dm"
	default:
		return strconv.Itoa(int(t))
	}
}

func updateLookupErr(e *events.ApplicationCommandInteractionCreate, desc string) error {
	return (interactions.SlashUpdateInteractionResponse{Update: discord.MessageUpdate{
		Embeds: &[]discord.Embed{{
			Description: strings.TrimSpace(desc),
			Color:       infoErrorEmbedColor,
		}},
		AllowedMentions: &discord.AllowedMentions{},
	}}).Execute(e)
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func discordTimestamp(t time.Time) string {
	return fmt.Sprintf("<t:%d:F>", t.Unix())
}

func localize(locales []string, t core.Translator, id string) map[discord.Locale]string {
	return core.LocalizeMap(locales, func(locale string) string {
		return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: id})
	})
}
