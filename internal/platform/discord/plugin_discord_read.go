package discordplatform

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"

	pluginhostlua "github.com/xsyetopz/go-mamusiabtw/internal/pluginhost/lua"
)

func (e pluginDiscordExecutor) SelfUser(ctx context.Context) (pluginhostlua.UserResult, error) {
	if e.bot == nil || e.bot.client == nil {
		return pluginhostlua.UserResult{}, errors.New("discord client unavailable")
	}
	self, ok := e.bot.client.Caches.SelfUser()
	if !ok {
		return pluginhostlua.UserResult{}, errors.New("self user unavailable")
	}

	result := pluginhostlua.UserResult{
		ID:          uint64(self.ID),
		Username:    strings.TrimSpace(self.Username),
		DisplayName: strings.TrimSpace(self.Username),
		Mention:     "<@" + self.ID.String() + ">",
		Bot:         true,
		CreatedAt:   self.ID.Time().UTC().Unix(),
	}
	if avatar := self.AvatarURL(); avatar != nil {
		result.AvatarURL = strings.TrimSpace(*avatar)
	}
	return result, nil
}

func (e pluginDiscordExecutor) GetUser(ctx context.Context, userID uint64) (pluginhostlua.UserResult, error) {
	if e.bot == nil || e.bot.client == nil {
		return pluginhostlua.UserResult{}, errors.New("discord client unavailable")
	}
	if userID == 0 {
		return pluginhostlua.UserResult{}, errors.New("invalid user")
	}

	user, err := e.bot.client.Rest.GetUser(snowflake.ID(userID), rest.WithCtx(ctx))
	if err != nil || user == nil {
		return pluginhostlua.UserResult{}, errors.New("get_user_error")
	}
	return userResult(*user), nil
}

func (e pluginDiscordExecutor) GetMember(
	ctx context.Context,
	guildID, userID uint64,
) (pluginhostlua.MemberResult, error) {
	if e.bot == nil || e.bot.client == nil {
		return pluginhostlua.MemberResult{}, errors.New("discord client unavailable")
	}
	if guildID == 0 || userID == 0 {
		return pluginhostlua.MemberResult{}, errors.New("invalid member")
	}

	member, err := e.bot.client.Rest.GetMember(snowflake.ID(guildID), snowflake.ID(userID), rest.WithCtx(ctx))
	if err != nil || member == nil {
		return pluginhostlua.MemberResult{}, errors.New("get_member_error")
	}
	return memberResult(uint64(guildID), *member), nil
}

func (e pluginDiscordExecutor) GetGuild(ctx context.Context, guildID uint64) (pluginhostlua.GuildResult, error) {
	if e.bot == nil || e.bot.client == nil {
		return pluginhostlua.GuildResult{}, errors.New("discord client unavailable")
	}
	if guildID == 0 {
		return pluginhostlua.GuildResult{}, errors.New("invalid guild")
	}

	guild, err := e.bot.client.Rest.GetGuild(snowflake.ID(guildID), true, rest.WithCtx(ctx))
	if err != nil || guild == nil {
		return pluginhostlua.GuildResult{}, errors.New("get_guild_error")
	}

	channels, err := e.bot.client.Rest.GetGuildChannels(snowflake.ID(guildID), rest.WithCtx(ctx))
	if err != nil {
		return pluginhostlua.GuildResult{}, errors.New("get_guild_error")
	}

	memberCount := guild.ApproximateMemberCount
	if memberCount <= 0 && guild.MemberCount > 0 {
		memberCount = guild.MemberCount
	}

	result := pluginhostlua.GuildResult{
		ID:            uint64(guild.ID),
		Name:          strings.TrimSpace(guild.Name),
		OwnerID:       uint64(guild.OwnerID),
		RolesCount:    len(guild.Roles),
		EmojisCount:   len(guild.Emojis),
		StickersCount: len(guild.Stickers),
		MemberCount:   memberCount,
		ChannelsCount: len(channels),
		CreatedAt:     guild.CreatedAt().UTC().Unix(),
	}
	if guild.Description != nil {
		result.Description = strings.TrimSpace(*guild.Description)
	}
	if icon := guild.IconURL(); icon != nil {
		result.IconURL = strings.TrimSpace(*icon)
	}
	if banner := guild.BannerURL(); banner != nil {
		result.BannerURL = strings.TrimSpace(*banner)
	}
	return result, nil
}

func (e pluginDiscordExecutor) GetRole(
	ctx context.Context,
	guildID, roleID uint64,
) (pluginhostlua.RoleResult, error) {
	if e.bot == nil || e.bot.client == nil {
		return pluginhostlua.RoleResult{}, errors.New("discord client unavailable")
	}
	if guildID == 0 || roleID == 0 {
		return pluginhostlua.RoleResult{}, errors.New("invalid role")
	}

	guild, err := e.bot.client.Rest.GetGuild(snowflake.ID(guildID), false, rest.WithCtx(ctx))
	if err != nil || guild == nil {
		return pluginhostlua.RoleResult{}, errors.New("get_role_error")
	}
	for _, role := range guild.Roles {
		if uint64(role.ID) == roleID {
			return roleResult(role), nil
		}
	}
	return pluginhostlua.RoleResult{}, errors.New("get_role_error")
}

func (e pluginDiscordExecutor) GetChannel(ctx context.Context, channelID uint64) (pluginhostlua.ChannelResult, error) {
	if e.bot == nil || e.bot.client == nil {
		return pluginhostlua.ChannelResult{}, errors.New("discord client unavailable")
	}
	if channelID == 0 {
		return pluginhostlua.ChannelResult{}, errors.New("invalid channel")
	}

	channel, err := e.bot.client.Rest.GetChannel(snowflake.ID(channelID), rest.WithCtx(ctx))
	if err != nil || channel == nil {
		return pluginhostlua.ChannelResult{}, errors.New("get_channel_error")
	}

	result := pluginhostlua.ChannelResult{
		ID:        uint64(channel.ID()),
		Name:      strings.TrimSpace(channel.Name()),
		Mention:   discord.ChannelMention(channel.ID()),
		Type:      pluginChannelTypeName(channel.Type()),
		CreatedAt: channel.CreatedAt().UTC().Unix(),
	}
	if guildChannel, ok := channel.(discord.GuildChannel); ok {
		if parentID := guildChannel.ParentID(); parentID != nil {
			result.ParentID = uint64(*parentID)
		}
	}
	return result, nil
}

func userResult(user discord.User) pluginhostlua.UserResult {
	result := pluginhostlua.UserResult{
		ID:          uint64(user.ID),
		Username:    strings.TrimSpace(user.Username),
		DisplayName: strings.TrimSpace(user.EffectiveName()),
		Mention:     user.Mention(),
		Bot:         user.Bot,
		System:      user.System,
		CreatedAt:   user.CreatedAt().UTC().Unix(),
	}
	if user.AccentColor != nil {
		result.AccentColor = *user.AccentColor
	}
	if avatar := user.AvatarURL(); avatar != nil {
		result.AvatarURL = strings.TrimSpace(*avatar)
	} else {
		result.AvatarURL = strings.TrimSpace(user.EffectiveAvatarURL())
	}
	if banner := user.BannerURL(); banner != nil {
		result.BannerURL = strings.TrimSpace(*banner)
	}
	return result
}

func memberResult(guildID uint64, member discord.Member) pluginhostlua.MemberResult {
	result := pluginhostlua.MemberResult{
		UserID:  uint64(member.User.ID),
		GuildID: guildID,
	}
	if member.JoinedAt != nil && !member.JoinedAt.IsZero() {
		result.JoinedAt = member.JoinedAt.UTC().Unix()
	}
	result.RoleIDs = make([]uint64, 0, len(member.RoleIDs))
	for _, roleID := range member.RoleIDs {
		result.RoleIDs = append(result.RoleIDs, uint64(roleID))
	}
	if avatar := strings.TrimSpace(member.EffectiveAvatarURL()); avatar != "" {
		result.AvatarURL = avatar
	}
	if banner := member.EffectiveBannerURL(); banner != "" {
		result.BannerURL = strings.TrimSpace(banner)
	}
	return result
}

func pluginChannelTypeName(t discord.ChannelType) string {
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
