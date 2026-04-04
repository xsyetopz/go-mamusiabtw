package luaplugin

import (
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type UserResult struct {
	ID          uint64
	Username    string
	DisplayName string
	Mention     string
	Bot         bool
	System      bool
	AccentColor int
	AvatarURL   string
	BannerURL   string
	CreatedAt   int64
}

type MemberResult struct {
	UserID    uint64
	GuildID   uint64
	JoinedAt  int64
	RoleIDs   []uint64
	AvatarURL string
	BannerURL string
}

type GuildResult struct {
	ID            uint64
	Name          string
	Description   string
	OwnerID       uint64
	RolesCount    int
	EmojisCount   int
	StickersCount int
	MemberCount   int
	ChannelsCount int
	IconURL       string
	BannerURL     string
	CreatedAt     int64
}

type ChannelResult struct {
	ID          uint64
	Name        string
	Mention     string
	Type        string
	ParentID    uint64
	Permissions int64
	CreatedAt   int64
}

func (v *VM) luaDiscordSelfUser(l *lua.LState) int {
	if !v.perms.Discord.Users {
		l.RaiseError("permission denied: discord.get_self_user")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	user, err := v.discord.SelfUser(v.ctx())
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, userMap(user), "")
}

func (v *VM) luaDiscordGetUser(l *lua.LState) int {
	if !v.perms.Discord.Users {
		l.RaiseError("permission denied: discord.get_user")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.OptTable(1, l.NewTable())
	userID := luaSnowflake(spec.RawGetString("user_id"), v.userID)
	if userID == 0 {
		l.RaiseError("invalid user spec")
		return 0
	}

	user, err := v.discord.GetUser(v.ctx(), userID)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, userMap(user), "")
}

func (v *VM) luaDiscordGetMember(l *lua.LState) int {
	if !v.perms.Discord.Members {
		l.RaiseError("permission denied: discord.get_member")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.OptTable(1, l.NewTable())
	guildID := v.tableGuildID(spec, "guild_id")
	userID := luaSnowflake(spec.RawGetString("user_id"), v.userID)
	if guildID == 0 || userID == 0 {
		l.RaiseError("invalid member spec")
		return 0
	}

	member, err := v.discord.GetMember(v.ctx(), guildID, userID)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, memberMap(member), "")
}

func (v *VM) luaDiscordGetGuild(l *lua.LState) int {
	if !v.perms.Discord.Guilds {
		l.RaiseError("permission denied: discord.get_guild")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.OptTable(1, l.NewTable())
	guildID := v.tableGuildID(spec, "guild_id")
	if guildID == 0 {
		l.RaiseError("invalid guild spec")
		return 0
	}

	guild, err := v.discord.GetGuild(v.ctx(), guildID)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, guildMap(guild), "")
}

func (v *VM) luaDiscordGetRole(l *lua.LState) int {
	if !v.perms.Discord.Roles {
		l.RaiseError("permission denied: discord.get_role")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.CheckTable(1)
	guildID := v.tableGuildID(spec, "guild_id")
	roleID := luaSnowflake(spec.RawGetString("role_id"), 0)
	if guildID == 0 || roleID == 0 {
		l.RaiseError("invalid role spec")
		return 0
	}

	role, err := v.discord.GetRole(v.ctx(), guildID, roleID)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, roleMap(role), "")
}

func (v *VM) luaDiscordGetChannel(l *lua.LState) int {
	if !v.perms.Discord.Channels {
		l.RaiseError("permission denied: discord.get_channel")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.OptTable(1, l.NewTable())
	channelID := luaSnowflake(spec.RawGetString("channel_id"), v.channel)
	if channelID == 0 {
		l.RaiseError("invalid channel spec")
		return 0
	}

	channel, err := v.discord.GetChannel(v.ctx(), channelID)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, channelMap(channel), "")
}

func userMap(user UserResult) map[string]any {
	return map[string]any{
		"id":           luaUintString(user.ID),
		"username":     user.Username,
		"display_name": user.DisplayName,
		"mention":      user.Mention,
		"bot":          user.Bot,
		"system":       user.System,
		"accent_color": user.AccentColor,
		"avatar_url":   user.AvatarURL,
		"banner_url":   user.BannerURL,
		"created_at":   user.CreatedAt,
	}
}

func memberMap(member MemberResult) map[string]any {
	roleIDs := make([]any, 0, len(member.RoleIDs))
	for _, roleID := range member.RoleIDs {
		roleIDs = append(roleIDs, luaUintString(roleID))
	}
	return map[string]any{
		"user_id":    luaUintString(member.UserID),
		"guild_id":   luaUintString(member.GuildID),
		"joined_at":  member.JoinedAt,
		"role_ids":   roleIDs,
		"avatar_url": member.AvatarURL,
		"banner_url": member.BannerURL,
	}
}

func guildMap(guild GuildResult) map[string]any {
	return map[string]any{
		"id":             luaUintString(guild.ID),
		"name":           guild.Name,
		"description":    guild.Description,
		"owner_id":       luaUintString(guild.OwnerID),
		"roles_count":    guild.RolesCount,
		"emojis_count":   guild.EmojisCount,
		"stickers_count": guild.StickersCount,
		"member_count":   guild.MemberCount,
		"channels_count": guild.ChannelsCount,
		"icon_url":       guild.IconURL,
		"banner_url":     guild.BannerURL,
		"created_at":     guild.CreatedAt,
	}
}

func channelMap(channel ChannelResult) map[string]any {
	out := map[string]any{
		"id":          luaUintString(channel.ID),
		"name":        channel.Name,
		"mention":     channel.Mention,
		"type":        channel.Type,
		"permissions": strconv.FormatInt(channel.Permissions, 10),
		"created_at":  channel.CreatedAt,
	}
	if channel.ParentID != 0 {
		out["parent_id"] = luaUintString(channel.ParentID)
	}
	return out
}

func luaStringPointer(table *lua.LTable, key string) *string {
	if table == nil {
		return nil
	}
	value := table.RawGetString(key)
	if value == lua.LNil {
		return nil
	}
	text := strings.TrimSpace(luaStringDefault(value, ""))
	return &text
}
