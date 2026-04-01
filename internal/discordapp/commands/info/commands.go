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

	"github.com/xsyetopz/jagpda/internal/buildinfo"
	"github.com/xsyetopz/jagpda/internal/discordapp/core"
	"github.com/xsyetopz/jagpda/internal/discordapp/interactions"
	"github.com/xsyetopz/jagpda/internal/i18n"
)

const (
	infoEmbedColor      = 0x5865F2
	infoErrorEmbedColor = 0xED4245
)

func Commands() []core.SlashCommand {
	return []core.SlashCommand{
		about(),
		lookup(),
	}
}

func about() core.SlashCommand {
	return core.SlashCommand{
		Name:          "about",
		NameID:        "cmd.about.name",
		DescID:        "cmd.about.desc",
		CreateCommand: aboutCreateCommand,
		Handle:        aboutHandle,
	}
}

func aboutCreateCommand(locales []string, t core.Translator) discord.ApplicationCommandCreate {
	loc := func(id string) map[discord.Locale]string {
		return core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: id})
		})
	}

	return discord.SlashCommandCreate{
		Name:                     "about",
		NameLocalizations:        loc("cmd.about.name"),
		Description:              t.S("cmd.about.desc", nil),
		DescriptionLocalizations: loc("cmd.about.desc"),
	}
}

func aboutHandle(
	_ context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	_ core.Services,
) (interactions.SlashAction, error) {
	embed := buildAboutEmbed(e, t)
	return interactions.SlashMessage{Create: discord.MessageCreate{
		Flags:           discord.MessageFlagEphemeral,
		Embeds:          []discord.Embed{embed},
		AllowedMentions: &discord.AllowedMentions{},
	}}, nil
}

func buildAboutEmbed(e *events.ApplicationCommandInteractionCreate, t core.Translator) discord.Embed {
	title := t.S("info.about.title", map[string]any{"Version": strings.TrimSpace(buildinfo.Version)})
	embed := discord.Embed{
		Title:       title,
		Description: strings.TrimSpace(buildinfo.Description),
		Color:       infoEmbedColor,
	}

	if repo, ok := httpsURL(strings.TrimSpace(buildinfo.Repository)); ok {
		embed.URL = repo
	}
	if mascot, ok := httpsURL(strings.TrimSpace(buildinfo.MascotImageURL)); ok {
		embed.Image = &discord.EmbedResource{URL: mascot}
	}

	applySelfUserAuthor(e, &embed)
	applyUserFooter(e.User(), &embed)
	return embed
}

func httpsURL(s string) (string, bool) {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(s)), "https://") {
		return strings.TrimSpace(s), true
	}
	return "", false
}

func applySelfUserAuthor(e *events.ApplicationCommandInteractionCreate, embed *discord.Embed) {
	if e == nil || embed == nil {
		return
	}

	self, ok := e.Client().Caches.SelfUser()
	if !ok {
		return
	}

	authorName := strings.TrimSpace(self.Username)
	if version := strings.TrimSpace(buildinfo.Version); authorName != "" && version != "" {
		authorName = authorName + " " + version
	}

	author := discord.EmbedAuthor{Name: authorName}
	if u := self.AvatarURL(); u != nil {
		author.IconURL = *u
	}
	if strings.TrimSpace(author.Name) == "" && strings.TrimSpace(author.IconURL) == "" {
		return
	}

	embed.Author = &author
	if strings.TrimSpace(embed.URL) == "" {
		repo := strings.TrimSpace(buildinfo.Repository)
		if repo != "" {
			embed.URL = repo
		}
	}
}

func applyUserFooter(user discord.User, embed *discord.Embed) {
	if embed == nil {
		return
	}

	username := strings.TrimSpace(user.Username)
	if u := user.AvatarURL(); u != nil {
		embed.Footer = &discord.EmbedFooter{Text: username, IconURL: *u}
		return
	}
	embed.Footer = &discord.EmbedFooter{Text: username}
}

func lookup() core.SlashCommand {
	return core.SlashCommand{
		Name:   "lookup",
		NameID: "cmd.lookup.name",
		DescID: "cmd.lookup.desc",
		CreateCommand: func(locales []string, t core.Translator) discord.ApplicationCommandCreate {
			loc := func(id string) map[discord.Locale]string {
				return core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: id})
				})
			}

			return discord.SlashCommandCreate{
				Name: "lookup",
				NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.lookup.name"})
				}),
				Description: t.S("cmd.lookup.desc", nil),
				DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.lookup.desc"})
				}),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommand{
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
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "guild",
						NameLocalizations:        loc("cmd.lookup.sub.guild.name"),
						Description:              t.S("cmd.lookup.sub.guild.desc", nil),
						DescriptionLocalizations: loc("cmd.lookup.sub.guild.desc"),
					},
				},
			}
		},
		Handle: func(ctx context.Context, _ *events.ApplicationCommandInteractionCreate, t core.Translator, _ core.Services) (interactions.SlashAction, error) {
			return interactions.SlashFunc(func(e *events.ApplicationCommandInteractionCreate) error {
				data := e.SlashCommandInteractionData()
				sub := data.SubCommandName
				if sub == nil {
					return (interactions.SlashMessage{Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil))}).Execute(
						e,
					)
				}

				switch *sub {
				case "user":
					return lookupUser(ctx, e, t)
				case "guild":
					return lookupGuild(ctx, e, t)
				default:
					return (interactions.SlashMessage{Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil))}).Execute(
						e,
					)
				}
			}), nil
		},
	}
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
		return (interactions.SlashMessage{Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.not_in_guild", nil))}).Execute(
			e,
		)
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

// Timestamp pointers are set from local vars to avoid taking the address of a non-addressable value.

func discordTimestamp(t time.Time) string {
	return fmt.Sprintf("<t:%d:F>", t.Unix())
}
