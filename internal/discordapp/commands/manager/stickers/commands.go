package cmdstickers

import (
	"bytes"
	"context"
	"image"
	_ "image/gif" // Register GIF decoder.
	_ "image/png" // Register PNG decoder.
	"path/filepath"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/omit"
	"github.com/disgoorg/snowflake/v2"

	"github.com/xsyetopz/jagpda/internal/discordapp/commands/shared"
	"github.com/xsyetopz/jagpda/internal/discordapp/core"
	"github.com/xsyetopz/jagpda/internal/discordapp/discordutil"
	"github.com/xsyetopz/jagpda/internal/discordapp/interactions"
	"github.com/xsyetopz/jagpda/internal/i18n"
)

const (
	maxStickerFileBytes = 512 * 1024
	maxStickerDimension = 320

	maxGuildStickersTierNone = 5
	maxGuildStickersTier1    = 15
	maxGuildStickersTier2    = 30
	maxGuildStickersTier3    = 60
)

func Commands() []core.SlashCommand { return []core.SlashCommand{stickers()} }

func stickers() core.SlashCommand {
	return core.SlashCommand{
		Name:          "stickers",
		NameID:        "cmd.stickers.name",
		DescID:        "cmd.stickers.desc",
		CreateCommand: stickersCreateCommand,
		Handle:        stickersHandle,
	}
}

func stickersCreateCommand(locales []string, t core.Translator) discord.ApplicationCommandCreate {
	perm := discord.PermissionManageGuildExpressions.Add(discord.PermissionCreateGuildExpressions)
	loc := func(id string) map[discord.Locale]string {
		return core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{
				Locale:    locale,
				MessageID: id,
			})
		})
	}

	minName, maxName := 2, 30
	minDesc, maxDesc := 2, 100
	minTag, maxTag := 1, 64
	minID, maxID := 1, 255
	options := stickersOptions(t, loc, &minName, &maxName, &minDesc, &maxDesc, &minTag, &maxTag, &minID, &maxID)

	return discord.SlashCommandCreate{
		Name: "stickers",
		NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{
				Locale:    locale,
				MessageID: "cmd.stickers.name",
			})
		}),
		Description: t.S("cmd.stickers.desc", nil),
		DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{
				Locale:    locale,
				MessageID: "cmd.stickers.desc",
			})
		}),
		DefaultMemberPermissions: omit.NewPtr(perm),
		Options:                  options,
	}
}

type stickersLocalizer func(id string) map[discord.Locale]string

func stickersOptions(
	t core.Translator,
	loc stickersLocalizer,
	minName, maxName, minDesc, maxDesc, minTag, maxTag, minID, maxID *int,
) []discord.ApplicationCommandOption {
	return []discord.ApplicationCommandOption{
		stickersSubCreate(t, loc, minName, maxName, minDesc, maxDesc, minTag, maxTag),
		stickersSubEdit(t, loc, minName, maxName, minDesc, maxDesc, minID, maxID),
		stickersSubDelete(t, loc, minID, maxID),
	}
}

func stickersSubCreate(
	t core.Translator,
	loc stickersLocalizer,
	minName, maxName, minDesc, maxDesc, minTag, maxTag *int,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "create",
		NameLocalizations:        loc("cmd.stickers.sub.create.name"),
		Description:              t.S("cmd.stickers.sub.create.desc", nil),
		DescriptionLocalizations: loc("cmd.stickers.sub.create.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:                     "name",
				NameLocalizations:        loc("cmd.stickers.opt.name.name"),
				Description:              t.S("cmd.stickers.opt.name.desc", nil),
				DescriptionLocalizations: loc("cmd.stickers.opt.name.desc"),
				Required:                 true,
				MinLength:                minName,
				MaxLength:                maxName,
			},
			discord.ApplicationCommandOptionString{
				Name:                     "description",
				NameLocalizations:        loc("cmd.stickers.opt.description.name"),
				Description:              t.S("cmd.stickers.opt.description.desc", nil),
				DescriptionLocalizations: loc("cmd.stickers.opt.description.desc"),
				MinLength:                minDesc,
				MaxLength:                maxDesc,
			},
			discord.ApplicationCommandOptionString{
				Name:                     "emoji_tag",
				NameLocalizations:        loc("cmd.stickers.opt.emoji_tag.name"),
				Description:              t.S("cmd.stickers.opt.emoji_tag.desc", nil),
				DescriptionLocalizations: loc("cmd.stickers.opt.emoji_tag.desc"),
				Required:                 true,
				MinLength:                minTag,
				MaxLength:                maxTag,
			},
			discord.ApplicationCommandOptionAttachment{
				Name:                     "file",
				NameLocalizations:        loc("cmd.stickers.opt.file.name"),
				Description:              t.S("cmd.stickers.opt.file.desc", nil),
				DescriptionLocalizations: loc("cmd.stickers.opt.file.desc"),
				Required:                 true,
			},
		},
	}
}

func stickersSubEdit(
	t core.Translator,
	loc stickersLocalizer,
	minName, maxName, minDesc, maxDesc, minID, maxID *int,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "edit",
		NameLocalizations:        loc("cmd.stickers.sub.edit.name"),
		Description:              t.S("cmd.stickers.sub.edit.desc", nil),
		DescriptionLocalizations: loc("cmd.stickers.sub.edit.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:                     "id",
				NameLocalizations:        loc("cmd.stickers.opt.id.name"),
				Description:              t.S("cmd.stickers.opt.id.desc", nil),
				DescriptionLocalizations: loc("cmd.stickers.opt.id.desc"),
				Required:                 true,
				MinLength:                minID,
				MaxLength:                maxID,
			},
			discord.ApplicationCommandOptionString{
				Name:                     "name",
				NameLocalizations:        loc("cmd.stickers.opt.name.name"),
				Description:              t.S("cmd.stickers.opt.name.desc", nil),
				DescriptionLocalizations: loc("cmd.stickers.opt.name.desc"),
				Required:                 true,
				MinLength:                minName,
				MaxLength:                maxName,
			},
			discord.ApplicationCommandOptionString{
				Name:                     "description",
				NameLocalizations:        loc("cmd.stickers.opt.description.name"),
				Description:              t.S("cmd.stickers.opt.description.desc", nil),
				DescriptionLocalizations: loc("cmd.stickers.opt.description.desc"),
				MinLength:                minDesc,
				MaxLength:                maxDesc,
			},
		},
	}
}

func stickersSubDelete(
	t core.Translator,
	loc stickersLocalizer,
	minID, maxID *int,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "delete",
		NameLocalizations:        loc("cmd.stickers.sub.delete.name"),
		Description:              t.S("cmd.stickers.sub.delete.desc", nil),
		DescriptionLocalizations: loc("cmd.stickers.sub.delete.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:                     "id",
				NameLocalizations:        loc("cmd.stickers.opt.id.name"),
				Description:              t.S("cmd.stickers.opt.id.desc", nil),
				DescriptionLocalizations: loc("cmd.stickers.opt.id.desc"),
				Required:                 true,
				MinLength:                minID,
				MaxLength:                maxID,
			},
		},
	}
}

func stickersHandle(
	ctx context.Context,
	_ *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	_ core.Services,
) (interactions.SlashAction, error) {
	return interactions.SlashFunc(func(e *events.ApplicationCommandInteractionCreate) error {
		return stickersExecute(ctx, e, t)
	}), nil
}

func stickersExecute(ctx context.Context, e *events.ApplicationCommandInteractionCreate, t core.Translator) error {
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
		return stickersCreate(ctx, e, *guildID, data, t)
	case "edit":
		return stickersEdit(e, *guildID, data, t)
	case "delete":
		return stickersDelete(e, *guildID, data, t)
	default:
		return shared.UpdateInteractionError(e, t.S("err.generic", nil))
	}
}

func stickersCreate(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	guildID snowflake.ID,
	data discord.SlashCommandInteractionData,
	t core.Translator,
) error {
	name := strings.TrimSpace(data.String("name"))
	desc, _ := data.OptString("description")
	desc = strings.TrimSpace(desc)
	tags := strings.TrimSpace(data.String("emoji_tag"))
	file := data.Attachment("file")
	if file.ID == 0 || strings.TrimSpace(file.URL) == "" {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.file_missing", nil))
	}
	if tags == "" {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.tags_missing", nil))
	}

	if file.Size > maxStickerFileBytes {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.file_too_large", map[string]any{
			"Max":  maxStickerFileBytes,
			"Size": file.Size,
		}))
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Filename), "."))
	if !isAllowedStickerExtension(ext) {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.bad_extension", map[string]any{"Ext": ext}))
	}

	g, err := e.Client().Rest.GetGuild(guildID, false)
	if err != nil || g == nil {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.create_error", map[string]any{"Name": name}))
	}

	maxAllowed := maxGuildStickers(g.PremiumTier)
	if len(g.Stickers) >= maxAllowed {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.too_many", map[string]any{"Max": maxAllowed}))
	}

	fetcher := discordutil.NewDiscordCDNFetcher()
	body, err := fetcher.Fetch(ctx, file.URL, maxStickerFileBytes)
	if err != nil {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.download_error", nil))
	}

	width, height, ok := imageDims(file.Width, file.Height, body)
	if !ok {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.dimensions_error", nil))
	}
	if width > maxStickerDimension || height > maxStickerDimension {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.too_large_dims", map[string]any{
			"Width":  width,
			"Height": height,
		}))
	}

	st, err := e.Client().Rest.CreateSticker(guildID, discord.StickerCreate{
		Name:        name,
		Description: desc,
		Tags:        tags,
		File:        discord.NewFile(file.Filename, "", bytes.NewReader(body)),
	})
	if err != nil || st == nil {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.create_error", map[string]any{"Name": name}))
	}

	return shared.UpdateInteractionSuccess(e, t.S("mgr.stickers.create_success", map[string]any{"Name": name}))
}

func stickersEdit(
	e *events.ApplicationCommandInteractionCreate,
	guildID snowflake.ID,
	data discord.SlashCommandInteractionData,
	t core.Translator,
) error {
	rawID := strings.TrimSpace(data.String("id"))
	stickerID, ok := discordutil.ParseStickerID(rawID)
	if !ok {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.invalid_id", map[string]any{"ID": rawID}))
	}

	name := strings.TrimSpace(data.String("name"))
	upd := discord.StickerUpdate{Name: &name}

	if desc, found := data.OptString("description"); found {
		s := strings.TrimSpace(desc)
		if s != "" {
			upd.Description = &s
		}
	}

	if _, err := e.Client().Rest.UpdateSticker(guildID, stickerID, upd); err != nil {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.edit_error", nil))
	}

	return shared.UpdateInteractionSuccess(e, t.S("mgr.stickers.edit_success", map[string]any{"Name": name}))
}

func stickersDelete(
	e *events.ApplicationCommandInteractionCreate,
	guildID snowflake.ID,
	data discord.SlashCommandInteractionData,
	t core.Translator,
) error {
	rawID := strings.TrimSpace(data.String("id"))
	stickerID, ok := discordutil.ParseStickerID(rawID)
	if !ok {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.invalid_id", map[string]any{"ID": rawID}))
	}

	if err := e.Client().Rest.DeleteSticker(guildID, stickerID); err != nil {
		return shared.UpdateInteractionError(e, t.S("mgr.stickers.delete_error", nil))
	}

	return shared.UpdateInteractionSuccess(e, t.S("mgr.stickers.delete_success", nil))
}

func isAllowedStickerExtension(ext string) bool {
	switch ext {
	case "png", "gif", "apng":
		return true
	default:
		return false
	}
}

func maxGuildStickers(tier discord.PremiumTier) int {
	switch tier {
	case discord.PremiumTierNone:
		return maxGuildStickersTierNone
	case discord.PremiumTier1:
		return maxGuildStickersTier1
	case discord.PremiumTier2:
		return maxGuildStickersTier2
	case discord.PremiumTier3:
		return maxGuildStickersTier3
	default:
		return maxGuildStickersTierNone
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
