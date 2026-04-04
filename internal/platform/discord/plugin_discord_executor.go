package discordplatform

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"

	"github.com/xsyetopz/go-mamusiabtw/internal/platform/discord/discordutil"
	pluginhostlua "github.com/xsyetopz/go-mamusiabtw/internal/pluginhost/lua"
)

const (
	pluginEmojiMaxFileBytes   = 256 * 1024
	pluginEmojiMaxDimension   = 320
	pluginStickerMaxFileBytes = 512 * 1024
	pluginStickerMaxDimension = 320
)

func (e pluginDiscordExecutor) SetSlowmode(ctx context.Context, channelID uint64, seconds int) error {
	if e.bot == nil || e.bot.client == nil {
		return errors.New("discord client unavailable")
	}
	if channelID == 0 || seconds < 0 {
		return errors.New("invalid slowmode spec")
	}

	updSeconds := seconds
	_, err := e.bot.client.Rest.UpdateChannel(snowflake.ID(channelID), discord.GuildTextChannelUpdate{
		RateLimitPerUser: &updSeconds,
	}, rest.WithCtx(ctx))
	return err
}

func (e pluginDiscordExecutor) SetNickname(ctx context.Context, guildID, userID uint64, nickname *string) error {
	if e.bot == nil || e.bot.client == nil {
		return errors.New("discord client unavailable")
	}
	if guildID == 0 || userID == 0 {
		return errors.New("invalid nickname spec")
	}

	nick := ""
	if nickname != nil {
		nick = strings.TrimSpace(*nickname)
	}
	_, err := e.bot.client.Rest.UpdateMember(snowflake.ID(guildID), snowflake.ID(userID), discord.MemberUpdate{
		Nick: &nick,
	}, rest.WithCtx(ctx))
	return err
}

func (e pluginDiscordExecutor) CreateRole(
	ctx context.Context,
	spec pluginhostlua.RoleCreateSpec,
) (pluginhostlua.RoleResult, error) {
	if e.bot == nil || e.bot.client == nil {
		return pluginhostlua.RoleResult{}, errors.New("discord client unavailable")
	}
	if spec.GuildID == 0 || strings.TrimSpace(spec.Name) == "" {
		return pluginhostlua.RoleResult{}, errors.New("invalid role spec")
	}

	input := discord.RoleCreate{Name: strings.TrimSpace(spec.Name)}
	if spec.Color != nil {
		input.Color = *spec.Color
	}
	if spec.Hoist != nil {
		input.Hoist = *spec.Hoist
	}
	if spec.Mentionable != nil {
		input.Mentionable = *spec.Mentionable
	}

	role, err := e.bot.client.Rest.CreateRole(snowflake.ID(spec.GuildID), input, rest.WithCtx(ctx))
	if err != nil {
		return pluginhostlua.RoleResult{}, err
	}
	return roleResult(*role), nil
}

func (e pluginDiscordExecutor) EditRole(
	ctx context.Context,
	spec pluginhostlua.RoleEditSpec,
) (pluginhostlua.RoleResult, error) {
	if e.bot == nil || e.bot.client == nil {
		return pluginhostlua.RoleResult{}, errors.New("discord client unavailable")
	}
	if spec.GuildID == 0 || spec.RoleID == 0 {
		return pluginhostlua.RoleResult{}, errors.New("invalid role spec")
	}
	if spec.RoleID == spec.GuildID {
		return pluginhostlua.RoleResult{}, errors.New("cannot_edit_everyone")
	}

	input := discord.RoleUpdate{}
	if spec.Name != nil && strings.TrimSpace(*spec.Name) != "" {
		name := strings.TrimSpace(*spec.Name)
		input.Name = &name
	}
	if spec.Color != nil {
		input.Color = spec.Color
	}
	if spec.Hoist != nil {
		input.Hoist = spec.Hoist
	}
	if spec.Mentionable != nil {
		input.Mentionable = spec.Mentionable
	}

	role, err := e.bot.client.Rest.UpdateRole(
		snowflake.ID(spec.GuildID),
		snowflake.ID(spec.RoleID),
		input,
		rest.WithCtx(ctx),
	)
	if err != nil {
		return pluginhostlua.RoleResult{}, err
	}
	return roleResult(*role), nil
}

func (e pluginDiscordExecutor) DeleteRole(ctx context.Context, guildID, roleID uint64) error {
	if e.bot == nil || e.bot.client == nil {
		return errors.New("discord client unavailable")
	}
	if guildID == 0 || roleID == 0 {
		return errors.New("invalid role spec")
	}
	if guildID == roleID {
		return errors.New("cannot_delete_everyone")
	}
	return e.bot.client.Rest.DeleteRole(snowflake.ID(guildID), snowflake.ID(roleID), rest.WithCtx(ctx))
}

func (e pluginDiscordExecutor) AddRole(ctx context.Context, spec pluginhostlua.RoleMemberSpec) error {
	if e.bot == nil || e.bot.client == nil {
		return errors.New("discord client unavailable")
	}
	if spec.GuildID == 0 || spec.UserID == 0 || spec.RoleID == 0 {
		return errors.New("invalid role member spec")
	}
	return e.bot.client.Rest.AddMemberRole(
		snowflake.ID(spec.GuildID),
		snowflake.ID(spec.UserID),
		snowflake.ID(spec.RoleID),
		rest.WithCtx(ctx),
	)
}

func (e pluginDiscordExecutor) RemoveRole(ctx context.Context, spec pluginhostlua.RoleMemberSpec) error {
	if e.bot == nil || e.bot.client == nil {
		return errors.New("discord client unavailable")
	}
	if spec.GuildID == 0 || spec.UserID == 0 || spec.RoleID == 0 {
		return errors.New("invalid role member spec")
	}
	return e.bot.client.Rest.RemoveMemberRole(
		snowflake.ID(spec.GuildID),
		snowflake.ID(spec.UserID),
		snowflake.ID(spec.RoleID),
		rest.WithCtx(ctx),
	)
}

func (e pluginDiscordExecutor) ListMessages(
	ctx context.Context,
	spec pluginhostlua.MessageListSpec,
) ([]pluginhostlua.MessageInfo, error) {
	if e.bot == nil || e.bot.client == nil {
		return nil, errors.New("discord client unavailable")
	}
	if spec.ChannelID == 0 || spec.Limit <= 0 {
		return nil, errors.New("invalid list_messages spec")
	}

	messages, err := e.bot.client.Rest.GetMessages(
		snowflake.ID(spec.ChannelID),
		snowflake.ID(spec.AroundID),
		snowflake.ID(spec.BeforeID),
		snowflake.ID(spec.AfterID),
		spec.Limit,
		rest.WithCtx(ctx),
	)
	if err != nil {
		return nil, err
	}

	out := make([]pluginhostlua.MessageInfo, 0, len(messages))
	for _, message := range messages {
		out = append(out, pluginhostlua.MessageInfo{
			ID:        uint64(message.ID),
			ChannelID: uint64(message.ChannelID),
			AuthorID:  uint64(message.Author.ID),
			Content:   message.Content,
			CreatedAt: message.CreatedAt.UTC().Unix(),
		})
	}
	return out, nil
}

func (e pluginDiscordExecutor) DeleteMessage(ctx context.Context, spec pluginhostlua.MessageDeleteSpec) error {
	if e.bot == nil || e.bot.client == nil {
		return errors.New("discord client unavailable")
	}
	if spec.ChannelID == 0 || spec.MessageID == 0 {
		return errors.New("invalid delete_message spec")
	}
	return e.bot.client.Rest.DeleteMessage(
		snowflake.ID(spec.ChannelID),
		snowflake.ID(spec.MessageID),
		rest.WithCtx(ctx),
	)
}

func (e pluginDiscordExecutor) BulkDeleteMessages(
	ctx context.Context,
	channelID uint64,
	messageIDs []uint64,
) (int, error) {
	if e.bot == nil || e.bot.client == nil {
		return 0, errors.New("discord client unavailable")
	}
	if channelID == 0 || len(messageIDs) == 0 {
		return 0, errors.New("invalid bulk_delete_messages spec")
	}

	ids := make([]snowflake.ID, 0, len(messageIDs))
	for _, id := range messageIDs {
		if id != 0 {
			ids = append(ids, snowflake.ID(id))
		}
	}
	if len(ids) == 0 {
		return 0, errors.New("invalid bulk_delete_messages spec")
	}

	if err := e.bot.client.Rest.BulkDeleteMessages(snowflake.ID(channelID), ids, rest.WithCtx(ctx)); err != nil {
		return 0, err
	}
	return len(ids), nil
}

func (e pluginDiscordExecutor) PurgeMessages(ctx context.Context, spec pluginhostlua.PurgeSpec) (int, error) {
	if e.bot == nil || e.bot.client == nil {
		return 0, errors.New("discord client unavailable")
	}
	if spec.ChannelID == 0 || spec.Count <= 0 {
		return 0, errors.New("invalid purge_messages spec")
	}

	around, before, after, ok := purgeAnchorIDs(spec.Mode, spec.AnchorRaw)
	if !ok {
		return 0, errors.New("invalid_message")
	}

	messages, err := e.bot.client.Rest.GetMessages(
		snowflake.ID(spec.ChannelID),
		around,
		before,
		after,
		spec.Count,
		rest.WithCtx(ctx),
	)
	if err != nil {
		return 0, err
	}

	ids := make([]snowflake.ID, 0, len(messages))
	for _, message := range messages {
		ids = append(ids, message.ID)
	}
	return deleteMessagesBestEffort(e.bot.client.Rest, snowflake.ID(spec.ChannelID), ids, time.Now()), nil
}

func (e pluginDiscordExecutor) CreateEmoji(
	ctx context.Context,
	spec pluginhostlua.EmojiCreateSpec,
) (pluginhostlua.EmojiResult, error) {
	if e.bot == nil || e.bot.client == nil {
		return pluginhostlua.EmojiResult{}, errors.New("discord client unavailable")
	}
	if spec.GuildID == 0 || strings.TrimSpace(spec.Name) == "" {
		return pluginhostlua.EmojiResult{}, errors.New("invalid emoji spec")
	}

	if err := validateEmojiAttachment(spec.File); err != nil {
		return pluginhostlua.EmojiResult{}, err
	}

	guild, err := e.bot.client.Rest.GetGuild(snowflake.ID(spec.GuildID), false, rest.WithCtx(ctx))
	if err != nil || guild == nil {
		return pluginhostlua.EmojiResult{}, errors.New("create_error")
	}
	maxAllowed := maxGuildEmojis(guild.PremiumTier)
	if len(guild.Emojis) >= maxAllowed {
		return pluginhostlua.EmojiResult{}, errors.New("too_many:" + strconv.Itoa(maxAllowed))
	}

	body, err := fetchDiscordAttachment(ctx, spec.File, pluginEmojiMaxFileBytes)
	if err != nil {
		return pluginhostlua.EmojiResult{}, errors.New("download_error")
	}
	if !allowedEmojiExtension(spec.File.Filename) {
		return pluginhostlua.EmojiResult{}, errors.New("bad_extension")
	}
	width, height, ok := imageDims(spec.File.Width, spec.File.Height, body)
	if !ok {
		return pluginhostlua.EmojiResult{}, errors.New("dimensions_error")
	}
	if width > pluginEmojiMaxDimension || height > pluginEmojiMaxDimension {
		return pluginhostlua.EmojiResult{}, errors.New("too_large_dims:" + strconv.Itoa(width) + ":" + strconv.Itoa(height))
	}

	icon, err := discord.ParseIconRaw(body)
	if err != nil || icon == nil {
		return pluginhostlua.EmojiResult{}, errors.New("bad_image")
	}

	emoji, err := e.bot.client.Rest.CreateEmoji(snowflake.ID(spec.GuildID), discord.EmojiCreate{
		Name:  strings.TrimSpace(spec.Name),
		Image: *icon,
	}, rest.WithCtx(ctx))
	if err != nil {
		return pluginhostlua.EmojiResult{}, errors.New("create_error")
	}
	return pluginhostlua.EmojiResult{ID: uint64(emoji.ID), Name: emoji.Name}, nil
}

func (e pluginDiscordExecutor) EditEmoji(
	ctx context.Context,
	spec pluginhostlua.EmojiEditSpec,
) (pluginhostlua.EmojiResult, error) {
	if e.bot == nil || e.bot.client == nil {
		return pluginhostlua.EmojiResult{}, errors.New("discord client unavailable")
	}
	if spec.GuildID == 0 || strings.TrimSpace(spec.RawEmoji) == "" || strings.TrimSpace(spec.Name) == "" {
		return pluginhostlua.EmojiResult{}, errors.New("invalid emoji spec")
	}

	emojiID, ok := discordutil.ParseEmojiID(spec.RawEmoji)
	if !ok {
		return pluginhostlua.EmojiResult{}, errors.New("invalid_emoji")
	}

	emoji, err := e.bot.client.Rest.UpdateEmoji(
		snowflake.ID(spec.GuildID),
		emojiID,
		discord.EmojiUpdate{Name: ptr(strings.TrimSpace(spec.Name))},
		rest.WithCtx(ctx),
	)
	if err != nil {
		return pluginhostlua.EmojiResult{}, errors.New("edit_error")
	}
	return pluginhostlua.EmojiResult{ID: uint64(emoji.ID), Name: emoji.Name}, nil
}

func (e pluginDiscordExecutor) DeleteEmoji(ctx context.Context, spec pluginhostlua.EmojiDeleteSpec) error {
	if e.bot == nil || e.bot.client == nil {
		return errors.New("discord client unavailable")
	}
	if spec.GuildID == 0 || strings.TrimSpace(spec.RawEmoji) == "" {
		return errors.New("invalid emoji spec")
	}
	emojiID, ok := discordutil.ParseEmojiID(spec.RawEmoji)
	if !ok {
		return errors.New("invalid_emoji")
	}
	if err := e.bot.client.Rest.DeleteEmoji(snowflake.ID(spec.GuildID), emojiID, rest.WithCtx(ctx)); err != nil {
		return errors.New("delete_error")
	}
	return nil
}

func (e pluginDiscordExecutor) CreateSticker(
	ctx context.Context,
	spec pluginhostlua.StickerCreateSpec,
) (pluginhostlua.StickerResult, error) {
	if e.bot == nil || e.bot.client == nil {
		return pluginhostlua.StickerResult{}, errors.New("discord client unavailable")
	}
	if spec.GuildID == 0 || strings.TrimSpace(spec.Name) == "" || strings.TrimSpace(spec.EmojiTag) == "" {
		return pluginhostlua.StickerResult{}, errors.New("invalid sticker spec")
	}

	if err := validateStickerAttachment(spec.File); err != nil {
		return pluginhostlua.StickerResult{}, err
	}

	guild, err := e.bot.client.Rest.GetGuild(snowflake.ID(spec.GuildID), false, rest.WithCtx(ctx))
	if err != nil || guild == nil {
		return pluginhostlua.StickerResult{}, errors.New("create_error")
	}
	maxAllowed := maxGuildStickers(guild.PremiumTier)
	if len(guild.Stickers) >= maxAllowed {
		return pluginhostlua.StickerResult{}, errors.New("too_many:" + strconv.Itoa(maxAllowed))
	}

	body, err := fetchDiscordAttachment(ctx, spec.File, pluginStickerMaxFileBytes)
	if err != nil {
		return pluginhostlua.StickerResult{}, errors.New("download_error")
	}
	if !allowedStickerExtension(spec.File.Filename) {
		return pluginhostlua.StickerResult{}, errors.New("bad_extension")
	}
	width, height, ok := imageDims(spec.File.Width, spec.File.Height, body)
	if !ok {
		return pluginhostlua.StickerResult{}, errors.New("dimensions_error")
	}
	if width > pluginStickerMaxDimension || height > pluginStickerMaxDimension {
		return pluginhostlua.StickerResult{}, errors.New("too_large_dims:" + strconv.Itoa(width) + ":" + strconv.Itoa(height))
	}

	sticker, err := e.bot.client.Rest.CreateSticker(snowflake.ID(spec.GuildID), discord.StickerCreate{
		Name:        strings.TrimSpace(spec.Name),
		Description: strings.TrimSpace(spec.Description),
		Tags:        strings.TrimSpace(spec.EmojiTag),
		File:        discord.NewFile(spec.File.Filename, "", bytes.NewReader(body)),
	}, rest.WithCtx(ctx))
	if err != nil {
		return pluginhostlua.StickerResult{}, errors.New("create_error")
	}
	return pluginhostlua.StickerResult{ID: uint64(sticker.ID), Name: sticker.Name}, nil
}

func (e pluginDiscordExecutor) EditSticker(
	ctx context.Context,
	spec pluginhostlua.StickerEditSpec,
) (pluginhostlua.StickerResult, error) {
	if e.bot == nil || e.bot.client == nil {
		return pluginhostlua.StickerResult{}, errors.New("discord client unavailable")
	}
	if spec.GuildID == 0 || strings.TrimSpace(spec.RawID) == "" || strings.TrimSpace(spec.Name) == "" {
		return pluginhostlua.StickerResult{}, errors.New("invalid sticker spec")
	}
	stickerID, ok := discordutil.ParseStickerID(spec.RawID)
	if !ok {
		return pluginhostlua.StickerResult{}, errors.New("invalid_id")
	}

	update := discord.StickerUpdate{Name: ptr(strings.TrimSpace(spec.Name))}
	if spec.Description != nil {
		value := strings.TrimSpace(*spec.Description)
		if value != "" {
			update.Description = &value
		}
	}

	sticker, err := e.bot.client.Rest.UpdateSticker(
		snowflake.ID(spec.GuildID),
		stickerID,
		update,
		rest.WithCtx(ctx),
	)
	if err != nil {
		return pluginhostlua.StickerResult{}, errors.New("edit_error")
	}
	return pluginhostlua.StickerResult{ID: uint64(sticker.ID), Name: sticker.Name}, nil
}

func (e pluginDiscordExecutor) DeleteSticker(ctx context.Context, spec pluginhostlua.StickerDeleteSpec) error {
	if e.bot == nil || e.bot.client == nil {
		return errors.New("discord client unavailable")
	}
	if spec.GuildID == 0 || strings.TrimSpace(spec.RawID) == "" {
		return errors.New("invalid sticker spec")
	}
	stickerID, ok := discordutil.ParseStickerID(spec.RawID)
	if !ok {
		return errors.New("invalid_id")
	}
	if err := e.bot.client.Rest.DeleteSticker(snowflake.ID(spec.GuildID), stickerID, rest.WithCtx(ctx)); err != nil {
		return errors.New("delete_error")
	}
	return nil
}

func roleResult(role discord.Role) pluginhostlua.RoleResult {
	return pluginhostlua.RoleResult{
		ID:          uint64(role.ID),
		Name:        role.Name,
		Mention:     discord.RoleMention(role.ID),
		Color:       role.Color,
		Hoist:       role.Hoist,
		Mentionable: role.Mentionable,
		Position:    role.Position,
		Managed:     role.Managed,
		Permissions: int64(role.Permissions),
		CreatedAt:   role.CreatedAt().UTC().Unix(),
	}
}

func fetchDiscordAttachment(ctx context.Context, file pluginhostlua.AttachmentInput, maxBytes int64) ([]byte, error) {
	if strings.TrimSpace(file.URL) == "" || strings.TrimSpace(file.Filename) == "" {
		return nil, errors.New("file_missing")
	}
	fetcher := discordutil.NewDiscordCDNFetcher()
	return fetcher.Fetch(ctx, file.URL, maxBytes)
}

func validateEmojiAttachment(file pluginhostlua.AttachmentInput) error {
	if file.ID == 0 || strings.TrimSpace(file.URL) == "" {
		return errors.New("file_missing")
	}
	if file.Size > pluginEmojiMaxFileBytes {
		return errors.New("file_too_large")
	}
	if !allowedEmojiExtension(file.Filename) {
		return errors.New("bad_extension")
	}
	return nil
}

func validateStickerAttachment(file pluginhostlua.AttachmentInput) error {
	if file.ID == 0 || strings.TrimSpace(file.URL) == "" {
		return errors.New("file_missing")
	}
	if file.Size > pluginStickerMaxFileBytes {
		return errors.New("file_too_large")
	}
	if !allowedStickerExtension(file.Filename) {
		return errors.New("bad_extension")
	}
	return nil
}

func allowedEmojiExtension(filename string) bool {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), ".")) {
	case "gif", "jpeg", "jpg", "png":
		return true
	default:
		return false
	}
}

func allowedStickerExtension(filename string) bool {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), ".")) {
	case "png", "gif", "apng":
		return true
	default:
		return false
	}
}

func imageDims(width, height int, raw []byte) (int, int, bool) {
	if width > 0 && height > 0 {
		return width, height, true
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

func maxGuildEmojis(tier discord.PremiumTier) int {
	switch tier {
	case discord.PremiumTierNone:
		return 50
	case discord.PremiumTier1:
		return 100
	case discord.PremiumTier2:
		return 150
	case discord.PremiumTier3:
		return 250
	default:
		return 50
	}
}

func maxGuildStickers(tier discord.PremiumTier) int {
	switch tier {
	case discord.PremiumTierNone:
		return 5
	case discord.PremiumTier1:
		return 15
	case discord.PremiumTier2:
		return 30
	case discord.PremiumTier3:
		return 60
	default:
		return 5
	}
}

func purgeAnchorIDs(mode, raw string) (snowflake.ID, snowflake.ID, snowflake.ID, bool) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "all":
		return 0, 0, 0, true
	case "before":
		id, ok := discordutil.ParseMessageID(raw)
		return 0, id, 0, ok
	case "after":
		id, ok := discordutil.ParseMessageID(raw)
		return 0, 0, id, ok
	case "around":
		id, ok := discordutil.ParseMessageID(raw)
		return id, 0, 0, ok
	default:
		return 0, 0, 0, false
	}
}

func deleteMessagesBestEffort(r rest.Rest, channelID snowflake.ID, messageIDs []snowflake.ID, now time.Time) int {
	if len(messageIDs) == 0 {
		return 0
	}

	const (
		discordBulkDeleteMaxAge  = 14 * 24 * time.Hour
		discordBulkDeletePadding = time.Hour
		bulkDeleteChunkMax       = 100
	)
	cutoff := now.Add(-discordBulkDeleteMaxAge).Add(discordBulkDeletePadding)

	var bulkIDs []snowflake.ID
	var singleIDs []snowflake.ID
	for _, id := range messageIDs {
		if id == 0 {
			continue
		}
		if id.Time().Before(cutoff) {
			singleIDs = append(singleIDs, id)
			continue
		}
		bulkIDs = append(bulkIDs, id)
	}

	deleted := 0
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
			singleIDs = append(singleIDs, chunk...)
			continue
		}
		deleted += len(chunk)
	}

	for _, id := range singleIDs {
		if err := r.DeleteMessage(channelID, id); err != nil {
			continue
		}
		deleted++
	}
	return deleted
}

func ptr[T any](value T) *T {
	return &value
}
