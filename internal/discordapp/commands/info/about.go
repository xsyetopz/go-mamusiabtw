package cmdinfo

import (
	"context"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"github.com/xsyetopz/go-mamusiabtw/internal/buildinfo"
	"github.com/xsyetopz/go-mamusiabtw/internal/discordapp/core"
	"github.com/xsyetopz/go-mamusiabtw/internal/discordapp/interactions"
)

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
		return localize(locales, t, id)
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
