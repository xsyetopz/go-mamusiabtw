package luaplugin

import (
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type RoleResult struct {
	ID          uint64
	Name        string
	Mention     string
	Color       int
	Hoist       bool
	Mentionable bool
	Position    int
	Managed     bool
	Permissions int64
	CreatedAt   int64
}

type MessageInfo struct {
	ID        uint64
	ChannelID uint64
	AuthorID  uint64
	Content   string
	CreatedAt int64
}

type EmojiResult struct {
	ID   uint64
	Name string
}

type StickerResult struct {
	ID   uint64
	Name string
}

type AttachmentInput struct {
	ID          uint64
	Filename    string
	URL         string
	ContentType string
	Size        int64
	Width       int
	Height      int
}

type RoleCreateSpec struct {
	GuildID     uint64
	Name        string
	Color       *int
	Hoist       *bool
	Mentionable *bool
}

type RoleEditSpec struct {
	GuildID     uint64
	RoleID      uint64
	Name        *string
	Color       *int
	Hoist       *bool
	Mentionable *bool
}

type RoleMemberSpec struct {
	GuildID uint64
	UserID  uint64
	RoleID  uint64
}

type MessageListSpec struct {
	ChannelID uint64
	AroundID  uint64
	BeforeID  uint64
	AfterID   uint64
	Limit     int
}

type MessageDeleteSpec struct {
	ChannelID uint64
	MessageID uint64
}

type PurgeSpec struct {
	ChannelID uint64
	Mode      string
	AnchorRaw string
	Count     int
}

type EmojiCreateSpec struct {
	GuildID uint64
	Name    string
	File    AttachmentInput
}

type EmojiEditSpec struct {
	GuildID  uint64
	RawEmoji string
	Name     string
}

type EmojiDeleteSpec struct {
	GuildID  uint64
	RawEmoji string
}

type StickerCreateSpec struct {
	GuildID     uint64
	Name        string
	Description string
	EmojiTag    string
	File        AttachmentInput
}

type StickerEditSpec struct {
	GuildID     uint64
	RawID       string
	Name        string
	Description *string
}

type StickerDeleteSpec struct {
	GuildID uint64
	RawID   string
}

func (v *VM) luaDiscordSetSlowmode(l *lua.LState) int {
	if !v.perms.Discord.SetSlowmode {
		l.RaiseError("permission denied: discord.set_slowmode")
		return 0
	}
	if v.discord == nil {
		return pushDiscordBoolResult(l, false, "discord unavailable")
	}

	spec := l.CheckTable(1)
	channelID := luaSnowflake(spec.RawGetString("channel_id"), v.channel)
	seconds := 0
	if value := spec.RawGetString("seconds"); value != lua.LNil {
		seconds = int(luaIntDefault(value, 0))
	}
	if channelID == 0 || seconds < 0 {
		l.RaiseError("invalid slowmode spec")
		return 0
	}

	if err := v.discord.SetSlowmode(v.ctx(), channelID, seconds); err != nil {
		return pushDiscordBoolResult(l, false, err.Error())
	}
	return pushDiscordBoolResult(l, true, "")
}

func (v *VM) luaDiscordSetNickname(l *lua.LState) int {
	if !v.perms.Discord.SetNickname {
		l.RaiseError("permission denied: discord.set_nickname")
		return 0
	}
	if v.discord == nil {
		return pushDiscordBoolResult(l, false, "discord unavailable")
	}

	spec := l.CheckTable(1)
	guildID := v.tableGuildID(spec, "guild_id")
	userID := luaSnowflake(spec.RawGetString("user_id"), v.userID)
	var nickname *string
	if raw := spec.RawGetString("nickname"); raw != lua.LNil {
		value := strings.TrimSpace(luaStringDefault(raw, ""))
		nickname = &value
	}
	if guildID == 0 || userID == 0 {
		l.RaiseError("invalid nickname spec")
		return 0
	}

	if err := v.discord.SetNickname(v.ctx(), guildID, userID, nickname); err != nil {
		return pushDiscordBoolResult(l, false, err.Error())
	}
	return pushDiscordBoolResult(l, true, "")
}

func (v *VM) luaDiscordCreateRole(l *lua.LState) int {
	if !v.perms.Discord.CreateRole {
		l.RaiseError("permission denied: discord.create_role")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.CheckTable(1)
	input := RoleCreateSpec{
		GuildID:     v.tableGuildID(spec, "guild_id"),
		Name:        strings.TrimSpace(luaStringDefault(spec.RawGetString("name"), "")),
		Color:       luaOptionalInt(spec, "color"),
		Hoist:       luaOptionalBool(spec, "hoist"),
		Mentionable: luaOptionalBool(spec, "mentionable"),
	}
	if input.GuildID == 0 || input.Name == "" {
		l.RaiseError("invalid role spec")
		return 0
	}

	role, err := v.discord.CreateRole(v.ctx(), input)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, roleMap(role), "")
}

func (v *VM) luaDiscordEditRole(l *lua.LState) int {
	if !v.perms.Discord.EditRole {
		l.RaiseError("permission denied: discord.edit_role")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.CheckTable(1)
	input := RoleEditSpec{
		GuildID:     v.tableGuildID(spec, "guild_id"),
		RoleID:      luaSnowflake(spec.RawGetString("role_id"), 0),
		Name:        luaOptionalTrimmedString(spec, "name"),
		Color:       luaOptionalInt(spec, "color"),
		Hoist:       luaOptionalBool(spec, "hoist"),
		Mentionable: luaOptionalBool(spec, "mentionable"),
	}
	if input.GuildID == 0 || input.RoleID == 0 {
		l.RaiseError("invalid role spec")
		return 0
	}

	role, err := v.discord.EditRole(v.ctx(), input)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, roleMap(role), "")
}

func (v *VM) luaDiscordDeleteRole(l *lua.LState) int {
	if !v.perms.Discord.DeleteRole {
		l.RaiseError("permission denied: discord.delete_role")
		return 0
	}
	if v.discord == nil {
		return pushDiscordBoolResult(l, false, "discord unavailable")
	}

	spec := l.CheckTable(1)
	guildID := v.tableGuildID(spec, "guild_id")
	roleID := luaSnowflake(spec.RawGetString("role_id"), 0)
	if guildID == 0 || roleID == 0 {
		l.RaiseError("invalid role spec")
		return 0
	}

	if err := v.discord.DeleteRole(v.ctx(), guildID, roleID); err != nil {
		return pushDiscordBoolResult(l, false, err.Error())
	}
	return pushDiscordBoolResult(l, true, "")
}

func (v *VM) luaDiscordAddRole(l *lua.LState) int {
	return v.luaDiscordRoleMemberMutation(l, true)
}

func (v *VM) luaDiscordRemoveRole(l *lua.LState) int {
	return v.luaDiscordRoleMemberMutation(l, false)
}

func (v *VM) luaDiscordRoleMemberMutation(l *lua.LState, add bool) int {
	permName := "discord.add_role"
	if !add {
		permName = "discord.remove_role"
	}
	allowed := v.perms.Discord.AddRole
	if !add {
		allowed = v.perms.Discord.RemoveRole
	}
	if !allowed {
		l.RaiseError("permission denied: %s", permName)
		return 0
	}
	if v.discord == nil {
		return pushDiscordBoolResult(l, false, "discord unavailable")
	}

	spec := l.CheckTable(1)
	input := RoleMemberSpec{
		GuildID: v.tableGuildID(spec, "guild_id"),
		UserID:  luaSnowflake(spec.RawGetString("user_id"), v.userID),
		RoleID:  luaSnowflake(spec.RawGetString("role_id"), 0),
	}
	if input.GuildID == 0 || input.UserID == 0 || input.RoleID == 0 {
		l.RaiseError("invalid role member spec")
		return 0
	}

	var err error
	if add {
		err = v.discord.AddRole(v.ctx(), input)
	} else {
		err = v.discord.RemoveRole(v.ctx(), input)
	}
	if err != nil {
		return pushDiscordBoolResult(l, false, err.Error())
	}
	return pushDiscordBoolResult(l, true, "")
}

func (v *VM) luaDiscordListMessages(l *lua.LState) int {
	if !v.perms.Discord.ListMessages {
		l.RaiseError("permission denied: discord.list_messages")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.CheckTable(1)
	input := MessageListSpec{
		ChannelID: luaSnowflake(spec.RawGetString("channel_id"), v.channel),
		AroundID:  luaSnowflake(spec.RawGetString("around_message_id"), 0),
		BeforeID:  luaSnowflake(spec.RawGetString("before_message_id"), 0),
		AfterID:   luaSnowflake(spec.RawGetString("after_message_id"), 0),
		Limit:     int(luaIntDefault(spec.RawGetString("limit"), 0)),
	}
	if input.ChannelID == 0 || input.Limit <= 0 {
		l.RaiseError("invalid list_messages spec")
		return 0
	}

	items, err := v.discord.ListMessages(v.ctx(), input)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, messageMap(item))
	}
	return pushDiscordValueResult(l, out, "")
}

func (v *VM) luaDiscordDeleteMessage(l *lua.LState) int {
	if !v.perms.Discord.DeleteMessage {
		l.RaiseError("permission denied: discord.delete_message")
		return 0
	}
	if v.discord == nil {
		return pushDiscordBoolResult(l, false, "discord unavailable")
	}

	spec := l.CheckTable(1)
	input := MessageDeleteSpec{
		ChannelID: luaSnowflake(spec.RawGetString("channel_id"), v.channel),
		MessageID: luaSnowflake(spec.RawGetString("message_id"), 0),
	}
	if input.ChannelID == 0 || input.MessageID == 0 {
		l.RaiseError("invalid delete_message spec")
		return 0
	}
	if err := v.discord.DeleteMessage(v.ctx(), input); err != nil {
		return pushDiscordBoolResult(l, false, err.Error())
	}
	return pushDiscordBoolResult(l, true, "")
}

func (v *VM) luaDiscordBulkDeleteMessages(l *lua.LState) int {
	if !v.perms.Discord.BulkDeleteMessages {
		l.RaiseError("permission denied: discord.bulk_delete_messages")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.CheckTable(1)
	channelID := luaSnowflake(spec.RawGetString("channel_id"), v.channel)
	messageIDs := luaSnowflakeSlice(spec.RawGetString("message_ids"))
	if channelID == 0 || len(messageIDs) == 0 {
		l.RaiseError("invalid bulk_delete_messages spec")
		return 0
	}

	deleted, err := v.discord.BulkDeleteMessages(v.ctx(), channelID, messageIDs)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, map[string]any{"deleted_count": deleted}, "")
}

func (v *VM) luaDiscordPurgeMessages(l *lua.LState) int {
	if !v.perms.Discord.PurgeMessages {
		l.RaiseError("permission denied: discord.purge_messages")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.CheckTable(1)
	input := PurgeSpec{
		ChannelID: luaSnowflake(spec.RawGetString("channel_id"), v.channel),
		Mode:      strings.ToLower(strings.TrimSpace(luaStringDefault(spec.RawGetString("mode"), ""))),
		AnchorRaw: strings.TrimSpace(luaStringDefault(spec.RawGetString("anchor_message_id"), "")),
		Count:     int(luaIntDefault(spec.RawGetString("count"), 0)),
	}
	if input.ChannelID == 0 || input.Mode == "" || input.Count <= 0 {
		l.RaiseError("invalid purge_messages spec")
		return 0
	}

	deleted, err := v.discord.PurgeMessages(v.ctx(), input)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, map[string]any{"deleted_count": deleted}, "")
}

func (v *VM) luaDiscordCreateEmoji(l *lua.LState) int {
	if !v.perms.Discord.CreateEmoji {
		l.RaiseError("permission denied: discord.create_emoji")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.CheckTable(1)
	file, ok := luaAttachmentInput(spec.RawGetString("file"))
	if !ok {
		l.RaiseError("invalid emoji spec")
		return 0
	}
	input := EmojiCreateSpec{
		GuildID: v.tableGuildID(spec, "guild_id"),
		Name:    strings.TrimSpace(luaStringDefault(spec.RawGetString("name"), "")),
		File:    file,
	}
	if input.GuildID == 0 || input.Name == "" {
		l.RaiseError("invalid emoji spec")
		return 0
	}

	emoji, err := v.discord.CreateEmoji(v.ctx(), input)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, map[string]any{
		"id":   emoji.ID,
		"name": emoji.Name,
	}, "")
}

func (v *VM) luaDiscordEditEmoji(l *lua.LState) int {
	if !v.perms.Discord.EditEmoji {
		l.RaiseError("permission denied: discord.edit_emoji")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.CheckTable(1)
	input := EmojiEditSpec{
		GuildID:  v.tableGuildID(spec, "guild_id"),
		RawEmoji: strings.TrimSpace(luaStringDefault(spec.RawGetString("emoji"), "")),
		Name:     strings.TrimSpace(luaStringDefault(spec.RawGetString("name"), "")),
	}
	if input.GuildID == 0 || input.RawEmoji == "" || input.Name == "" {
		l.RaiseError("invalid emoji spec")
		return 0
	}

	emoji, err := v.discord.EditEmoji(v.ctx(), input)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, map[string]any{
		"id":   emoji.ID,
		"name": emoji.Name,
	}, "")
}

func (v *VM) luaDiscordDeleteEmoji(l *lua.LState) int {
	if !v.perms.Discord.DeleteEmoji {
		l.RaiseError("permission denied: discord.delete_emoji")
		return 0
	}
	if v.discord == nil {
		return pushDiscordBoolResult(l, false, "discord unavailable")
	}

	spec := l.CheckTable(1)
	input := EmojiDeleteSpec{
		GuildID:  v.tableGuildID(spec, "guild_id"),
		RawEmoji: strings.TrimSpace(luaStringDefault(spec.RawGetString("emoji"), "")),
	}
	if input.GuildID == 0 || input.RawEmoji == "" {
		l.RaiseError("invalid emoji spec")
		return 0
	}
	if err := v.discord.DeleteEmoji(v.ctx(), input); err != nil {
		return pushDiscordBoolResult(l, false, err.Error())
	}
	return pushDiscordBoolResult(l, true, "")
}

func (v *VM) luaDiscordCreateSticker(l *lua.LState) int {
	if !v.perms.Discord.CreateSticker {
		l.RaiseError("permission denied: discord.create_sticker")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.CheckTable(1)
	file, ok := luaAttachmentInput(spec.RawGetString("file"))
	if !ok {
		l.RaiseError("invalid sticker spec")
		return 0
	}
	input := StickerCreateSpec{
		GuildID:     v.tableGuildID(spec, "guild_id"),
		Name:        strings.TrimSpace(luaStringDefault(spec.RawGetString("name"), "")),
		Description: strings.TrimSpace(luaStringDefault(spec.RawGetString("description"), "")),
		EmojiTag:    strings.TrimSpace(luaStringDefault(spec.RawGetString("emoji_tag"), "")),
		File:        file,
	}
	if input.GuildID == 0 || input.Name == "" || input.EmojiTag == "" {
		l.RaiseError("invalid sticker spec")
		return 0
	}

	sticker, err := v.discord.CreateSticker(v.ctx(), input)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, map[string]any{
		"id":   sticker.ID,
		"name": sticker.Name,
	}, "")
}

func (v *VM) luaDiscordEditSticker(l *lua.LState) int {
	if !v.perms.Discord.EditSticker {
		l.RaiseError("permission denied: discord.edit_sticker")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.CheckTable(1)
	input := StickerEditSpec{
		GuildID: v.tableGuildID(spec, "guild_id"),
		RawID:   strings.TrimSpace(luaStringDefault(spec.RawGetString("id"), "")),
		Name:    strings.TrimSpace(luaStringDefault(spec.RawGetString("name"), "")),
	}
	if description := luaOptionalTrimmedString(spec, "description"); description != nil {
		input.Description = description
	}
	if input.GuildID == 0 || input.RawID == "" || input.Name == "" {
		l.RaiseError("invalid sticker spec")
		return 0
	}

	sticker, err := v.discord.EditSticker(v.ctx(), input)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, map[string]any{
		"id":   sticker.ID,
		"name": sticker.Name,
	}, "")
}

func (v *VM) luaDiscordDeleteSticker(l *lua.LState) int {
	if !v.perms.Discord.DeleteSticker {
		l.RaiseError("permission denied: discord.delete_sticker")
		return 0
	}
	if v.discord == nil {
		return pushDiscordBoolResult(l, false, "discord unavailable")
	}

	spec := l.CheckTable(1)
	input := StickerDeleteSpec{
		GuildID: v.tableGuildID(spec, "guild_id"),
		RawID:   strings.TrimSpace(luaStringDefault(spec.RawGetString("id"), "")),
	}
	if input.GuildID == 0 || input.RawID == "" {
		l.RaiseError("invalid sticker spec")
		return 0
	}
	if err := v.discord.DeleteSticker(v.ctx(), input); err != nil {
		return pushDiscordBoolResult(l, false, err.Error())
	}
	return pushDiscordBoolResult(l, true, "")
}

func pushDiscordBoolResult(l *lua.LState, ok bool, errText string) int {
	if ok {
		l.Push(lua.LTrue)
		l.Push(lua.LNil)
		return 2
	}
	l.Push(lua.LFalse)
	l.Push(lua.LString(errText))
	return 2
}

func pushDiscordValueResult(l *lua.LState, value any, errText string) int {
	if strings.TrimSpace(errText) != "" {
		l.Push(lua.LNil)
		l.Push(lua.LString(errText))
		return 2
	}
	lv, err := anyToLuaValue(l, value, 0)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString("discord decode error"))
		return 2
	}
	l.Push(lv)
	l.Push(lua.LNil)
	return 2
}

func roleMap(role RoleResult) map[string]any {
	mention := role.Mention
	if strings.TrimSpace(mention) == "" {
		mention = "<@&" + luaUintString(role.ID) + ">"
	}
	return map[string]any{
		"id":          luaUintString(role.ID),
		"name":        role.Name,
		"mention":     mention,
		"color":       role.Color,
		"hoist":       role.Hoist,
		"mentionable": role.Mentionable,
		"position":    role.Position,
		"managed":     role.Managed,
		"permissions": strconv.FormatInt(role.Permissions, 10),
		"created_at":  role.CreatedAt,
	}
}

func messageMap(message MessageInfo) map[string]any {
	return map[string]any{
		"id":         message.ID,
		"channel_id": message.ChannelID,
		"author_id":  message.AuthorID,
		"content":    message.Content,
		"created_at": message.CreatedAt,
	}
}

func luaOptionalInt(spec *lua.LTable, key string) *int {
	if spec == nil {
		return nil
	}
	value := spec.RawGetString(key)
	if value == lua.LNil {
		return nil
	}
	out := int(luaIntDefault(value, 0))
	return &out
}

func luaOptionalBool(spec *lua.LTable, key string) *bool {
	if spec == nil {
		return nil
	}
	value := spec.RawGetString(key)
	if value == lua.LNil {
		return nil
	}
	boolean, ok := value.(lua.LBool)
	if !ok {
		return nil
	}
	out := bool(boolean)
	return &out
}

func luaOptionalTrimmedString(spec *lua.LTable, key string) *string {
	if spec == nil {
		return nil
	}
	value := spec.RawGetString(key)
	if value == lua.LNil {
		return nil
	}
	out := strings.TrimSpace(luaStringDefault(value, ""))
	return &out
}

func luaSnowflakeSlice(value lua.LValue) []uint64 {
	table, ok := value.(*lua.LTable)
	if !ok {
		return nil
	}
	out := make([]uint64, 0, table.Len())
	for idx := 1; idx <= table.Len(); idx++ {
		if parsed := luaSnowflake(table.RawGetInt(idx), 0); parsed != 0 {
			out = append(out, parsed)
		}
	}
	return out
}

func luaAttachmentInput(value lua.LValue) (AttachmentInput, bool) {
	table, ok := value.(*lua.LTable)
	if !ok {
		return AttachmentInput{}, false
	}

	input := AttachmentInput{
		ID:          luaSnowflake(table.RawGetString("id"), 0),
		Filename:    strings.TrimSpace(luaStringDefault(table.RawGetString("filename"), "")),
		URL:         strings.TrimSpace(luaStringDefault(table.RawGetString("url"), "")),
		ContentType: strings.TrimSpace(luaStringDefault(table.RawGetString("content_type"), "")),
		Size:        luaIntDefault(table.RawGetString("size"), 0),
		Width:       int(luaIntDefault(table.RawGetString("width"), 0)),
		Height:      int(luaIntDefault(table.RawGetString("height"), 0)),
	}
	if input.ID == 0 || input.Filename == "" || input.URL == "" {
		return AttachmentInput{}, false
	}
	return input, true
}

func luaUintString(value uint64) string {
	return strconv.FormatUint(value, 10)
}
