package cmdemojis

import (
	"bytes"
	"context"
	"image"
	"path/filepath"
	"strings"

	_ "image/gif"  // Register GIF decoder.
	_ "image/jpeg" // Register JPEG decoder.
	_ "image/png"  // Register PNG decoder.

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/omit"
	"github.com/disgoorg/snowflake/v2"

	"github.com/xsyetopz/imotherbtw/internal/discordapp/commands/shared"
	"github.com/xsyetopz/imotherbtw/internal/discordapp/core"
	"github.com/xsyetopz/imotherbtw/internal/discordapp/discordutil"
	"github.com/xsyetopz/imotherbtw/internal/discordapp/interactions"
	"github.com/xsyetopz/imotherbtw/internal/i18n"
)

const (
	maxEmojiFileBytes = 256 * 1024
	maxEmojiDimension = 320

	emojiLimitTier0 = 50
	emojiLimitTier1 = 100
	emojiLimitTier2 = 150
	emojiLimitTier3 = 250
)

func Commands() []core.SlashCommand { return []core.SlashCommand{emojis()} }

func emojis() core.SlashCommand {
	return core.SlashCommand{
		Name:          "emojis",
		NameID:        "cmd.emojis.name",
		DescID:        "cmd.emojis.desc",
		CreateCommand: emojisCreateCommand,
		Handle:        emojisHandle,
	}
}

func emojisCreateCommand(locales []string, t core.Translator) discord.ApplicationCommandCreate {
	perm := discord.PermissionManageGuildExpressions.Add(discord.PermissionCreateGuildExpressions)
	loc := func(id string) map[discord.Locale]string {
		return core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{
				Locale:    locale,
				MessageID: id,
			})
		})
	}

	minName, maxName := 2, 32
	minEmojiRaw, maxEmojiRaw := 1, 128

	return discord.SlashCommandCreate{
		Name: "emojis",
		NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{
				Locale:    locale,
				MessageID: "cmd.emojis.name",
			})
		}),
		Description: t.S("cmd.emojis.desc", nil),
		DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{
				Locale:    locale,
				MessageID: "cmd.emojis.desc",
			})
		}),
		DefaultMemberPermissions: omit.NewPtr(perm),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:                     "create",
				NameLocalizations:        loc("cmd.emojis.sub.create.name"),
				Description:              t.S("cmd.emojis.sub.create.desc", nil),
				DescriptionLocalizations: loc("cmd.emojis.sub.create.desc"),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:                     "name",
						NameLocalizations:        loc("cmd.emojis.opt.name.name"),
						Description:              t.S("cmd.emojis.opt.name.desc", nil),
						DescriptionLocalizations: loc("cmd.emojis.opt.name.desc"),
						Required:                 true,
						MinLength:                &minName,
						MaxLength:                &maxName,
					},
					discord.ApplicationCommandOptionAttachment{
						Name:                     "file",
						NameLocalizations:        loc("cmd.emojis.opt.file.name"),
						Description:              t.S("cmd.emojis.opt.file.desc", nil),
						DescriptionLocalizations: loc("cmd.emojis.opt.file.desc"),
						Required:                 true,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:                     "edit",
				NameLocalizations:        loc("cmd.emojis.sub.edit.name"),
				Description:              t.S("cmd.emojis.sub.edit.desc", nil),
				DescriptionLocalizations: loc("cmd.emojis.sub.edit.desc"),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:                     "emoji",
						NameLocalizations:        loc("cmd.emojis.opt.emoji.name"),
						Description:              t.S("cmd.emojis.opt.emoji.desc", nil),
						DescriptionLocalizations: loc("cmd.emojis.opt.emoji.desc"),
						Required:                 true,
						MinLength:                &minEmojiRaw,
						MaxLength:                &maxEmojiRaw,
					},
					discord.ApplicationCommandOptionString{
						Name:                     "name",
						NameLocalizations:        loc("cmd.emojis.opt.name.name"),
						Description:              t.S("cmd.emojis.opt.name.desc", nil),
						DescriptionLocalizations: loc("cmd.emojis.opt.name.desc"),
						Required:                 true,
						MinLength:                &minName,
						MaxLength:                &maxName,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:                     "delete",
				NameLocalizations:        loc("cmd.emojis.sub.delete.name"),
				Description:              t.S("cmd.emojis.sub.delete.desc", nil),
				DescriptionLocalizations: loc("cmd.emojis.sub.delete.desc"),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:                     "emoji",
						NameLocalizations:        loc("cmd.emojis.opt.emoji.name"),
						Description:              t.S("cmd.emojis.opt.emoji.desc", nil),
						DescriptionLocalizations: loc("cmd.emojis.opt.emoji.desc"),
						Required:                 true,
						MinLength:                &minEmojiRaw,
						MaxLength:                &maxEmojiRaw,
					},
				},
			},
		},
	}
}

func emojisHandle(
	ctx context.Context,
	_ *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	_ core.Services,
) (interactions.SlashAction, error) {
	return interactions.SlashFunc(func(e *events.ApplicationCommandInteractionCreate) error {
		return emojisExecute(ctx, e, t)
	}), nil
}

func emojisExecute(ctx context.Context, e *events.ApplicationCommandInteractionCreate, t core.Translator) error {
	guildID := e.GuildID()
	if guildID == nil {
		return (interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.not_in_guild", nil)),
		}).Execute(e)
	}

	data := e.SlashCommandInteractionData()
	sub := data.SubCommandName
	if sub == nil {
		return (interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil)),
		}).Execute(e)
	}

	if err := (interactions.SlashDefer{Ephemeral: true}).Execute(e); err != nil {
		return err
	}

	switch *sub {
	case "create":
		return emojisCreate(ctx, e, *guildID, data, t)
	case "edit":
		return emojisEdit(e, *guildID, data, t)
	case "delete":
		return emojisDelete(e, *guildID, data, t)
	default:
		return shared.UpdateInteractionError(e, t.S("err.generic", nil))
	}
}

func emojisCreate(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	guildID snowflake.ID,
	data discord.SlashCommandInteractionData,
	t core.Translator,
) error {
	name := strings.TrimSpace(data.String("name"))
	file := data.Attachment("file")
	if file.ID == 0 || strings.TrimSpace(file.URL) == "" {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.file_missing", nil))
	}

	if file.Size > maxEmojiFileBytes {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.file_too_large", map[string]any{
			"Max":  maxEmojiFileBytes,
			"Size": file.Size,
		}))
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Filename), "."))
	if !isAllowedEmojiExtension(ext) {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.bad_extension", map[string]any{"Ext": ext}))
	}

	g, err := e.Client().Rest.GetGuild(guildID, false)
	if err != nil || g == nil {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.create_error", map[string]any{"Name": name}))
	}

	maxAllowed := maxGuildEmojis(g.PremiumTier)
	if len(g.Emojis) >= maxAllowed {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.too_many", map[string]any{"Max": maxAllowed}))
	}

	fetcher := discordutil.NewDiscordCDNFetcher()
	body, err := fetcher.Fetch(ctx, file.URL, maxEmojiFileBytes)
	if err != nil {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.download_error", nil))
	}

	width, height, ok := imageDims(file.Width, file.Height, body)
	if !ok {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.dimensions_error", nil))
	}
	if width > maxEmojiDimension || height > maxEmojiDimension {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.too_large_dims", map[string]any{
			"Width":  width,
			"Height": height,
		}))
	}

	icon, err := discord.ParseIconRaw(body)
	if err != nil || icon == nil {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.bad_image", nil))
	}

	emoji, err := e.Client().Rest.CreateEmoji(guildID, discord.EmojiCreate{
		Name:  name,
		Image: *icon,
	})
	if err != nil || emoji == nil {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.create_error", map[string]any{"Name": name}))
	}

	return shared.UpdateInteractionSuccess(e, t.S("mgr.emojis.create_success", map[string]any{"Name": name}))
}

func emojisEdit(
	e *events.ApplicationCommandInteractionCreate,
	guildID snowflake.ID,
	data discord.SlashCommandInteractionData,
	t core.Translator,
) error {
	raw := strings.TrimSpace(data.String("emoji"))
	emojiID, ok := discordutil.ParseEmojiID(raw)
	if !ok {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.invalid_emoji", map[string]any{"Emoji": raw}))
	}
	name := strings.TrimSpace(data.String("name"))

	if _, err := e.Client().Rest.UpdateEmoji(guildID, emojiID, discord.EmojiUpdate{
		Name: &name,
	}); err != nil {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.edit_error", nil))
	}

	return shared.UpdateInteractionSuccess(e, t.S("mgr.emojis.edit_success", map[string]any{"Name": name}))
}

func emojisDelete(
	e *events.ApplicationCommandInteractionCreate,
	guildID snowflake.ID,
	data discord.SlashCommandInteractionData,
	t core.Translator,
) error {
	raw := strings.TrimSpace(data.String("emoji"))
	emojiID, ok := discordutil.ParseEmojiID(raw)
	if !ok {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.invalid_emoji", map[string]any{"Emoji": raw}))
	}

	if err := e.Client().Rest.DeleteEmoji(guildID, emojiID); err != nil {
		return shared.UpdateInteractionError(e, t.S("mgr.emojis.delete_error", nil))
	}

	return shared.UpdateInteractionSuccess(e, t.S("mgr.emojis.delete_success", nil))
}

func isAllowedEmojiExtension(ext string) bool {
	switch ext {
	case "gif", "jpeg", "jpg", "png":
		return true
	default:
		return false
	}
}

func maxGuildEmojis(tier discord.PremiumTier) int {
	switch tier {
	case discord.PremiumTierNone:
		return emojiLimitTier0
	case discord.PremiumTier1:
		return emojiLimitTier1
	case discord.PremiumTier2:
		return emojiLimitTier2
	case discord.PremiumTier3:
		return emojiLimitTier3
	default:
		return emojiLimitTier0
	}
}

func imageDims(wPtr, hPtr *int, raw []byte) (int, int, bool) {
	if wPtr != nil && hPtr != nil && *wPtr > 0 && *hPtr > 0 {
		return *wPtr, *hPtr, true
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return 0, 0, false
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}
