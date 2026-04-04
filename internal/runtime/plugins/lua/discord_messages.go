package luaplugin

import (
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type MessageGetSpec struct {
	ChannelID uint64
	MessageID uint64
}

type ReactionSpec struct {
	ChannelID uint64
	MessageID uint64
	Emoji     string
}

type ReactionUserSpec struct {
	ChannelID uint64
	MessageID uint64
	UserID    uint64
	Emoji     string
}

type ReactionListSpec struct {
	ChannelID uint64
	MessageID uint64
	Emoji     string
	AfterID   uint64
	Limit     int
}

func (v *VM) luaDiscordGetMessage(l *lua.LState) int {
	if !v.perms.Discord.Messages {
		l.RaiseError("permission denied: discord.get_message")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := v.luaMessageGetSpec(l.CheckTable(1))
	if spec.ChannelID == 0 || spec.MessageID == 0 {
		l.RaiseError("invalid get_message spec")
		return 0
	}

	message, err := v.discord.GetMessage(v.ctx(), spec)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, messageMap(message), "")
}

func (v *VM) luaDiscordCrosspostMessage(l *lua.LState) int {
	if !v.perms.Discord.Messages {
		l.RaiseError("permission denied: discord.crosspost_message")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := v.luaMessageGetSpec(l.CheckTable(1))
	if spec.ChannelID == 0 || spec.MessageID == 0 {
		l.RaiseError("invalid crosspost_message spec")
		return 0
	}

	message, err := v.discord.CrosspostMessage(v.ctx(), spec)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	return pushDiscordValueResult(l, messageMap(message), "")
}

func (v *VM) luaDiscordPinMessage(l *lua.LState) int {
	return v.luaDiscordPinMutation(l, true)
}

func (v *VM) luaDiscordUnpinMessage(l *lua.LState) int {
	return v.luaDiscordPinMutation(l, false)
}

func (v *VM) luaDiscordPinMutation(l *lua.LState, shouldPin bool) int {
	permName := "discord.pin_message"
	allowed := v.perms.Discord.Messages
	if !shouldPin {
		permName = "discord.unpin_message"
		allowed = v.perms.Discord.Messages
	}
	if !allowed {
		l.RaiseError("permission denied: %s", permName)
		return 0
	}
	if v.discord == nil {
		return pushDiscordBoolResult(l, false, "discord unavailable")
	}

	spec := v.luaMessageGetSpec(l.CheckTable(1))
	if spec.ChannelID == 0 || spec.MessageID == 0 {
		l.RaiseError("invalid pin_message spec")
		return 0
	}

	var err error
	if shouldPin {
		err = v.discord.PinMessage(v.ctx(), spec)
	} else {
		err = v.discord.UnpinMessage(v.ctx(), spec)
	}
	if err != nil {
		return pushDiscordBoolResult(l, false, err.Error())
	}
	return pushDiscordBoolResult(l, true, "")
}

func (v *VM) luaMessageGetSpec(spec *lua.LTable) MessageGetSpec {
	return MessageGetSpec{
		ChannelID: luaSnowflake(spec.RawGetString("channel_id"), v.channel),
		MessageID: luaSnowflake(spec.RawGetString("message_id"), 0),
	}
}

func (v *VM) luaDiscordGetReactions(l *lua.LState) int {
	if !v.perms.Discord.Reactions {
		l.RaiseError("permission denied: discord.get_reactions")
		return 0
	}
	if v.discord == nil {
		return pushDiscordValueResult(l, nil, "discord unavailable")
	}

	spec := l.CheckTable(1)
	input := ReactionListSpec{
		ChannelID: luaSnowflake(spec.RawGetString("channel_id"), v.channel),
		MessageID: luaSnowflake(spec.RawGetString("message_id"), 0),
		Emoji:     strings.TrimSpace(luaStringDefault(spec.RawGetString("emoji"), "")),
		AfterID:   luaSnowflake(spec.RawGetString("after_user_id"), 0),
		Limit:     int(luaIntDefault(spec.RawGetString("limit"), 25)),
	}
	if input.ChannelID == 0 || input.MessageID == 0 || input.Emoji == "" || input.Limit <= 0 {
		l.RaiseError("invalid get_reactions spec")
		return 0
	}

	users, err := v.discord.GetReactions(v.ctx(), input)
	if err != nil {
		return pushDiscordValueResult(l, nil, err.Error())
	}
	out := make([]any, 0, len(users))
	for _, user := range users {
		out = append(out, userMap(user))
	}
	return pushDiscordValueResult(l, out, "")
}

func (v *VM) luaDiscordAddReaction(l *lua.LState) int {
	return v.luaDiscordReactionMutation(l, "discord.add_reaction", v.perms.Discord.Reactions, func(spec ReactionSpec) error {
		return v.discord.AddReaction(v.ctx(), spec)
	})
}

func (v *VM) luaDiscordRemoveOwnReaction(l *lua.LState) int {
	return v.luaDiscordReactionMutation(l, "discord.remove_own_reaction", v.perms.Discord.Reactions, func(spec ReactionSpec) error {
		return v.discord.RemoveOwnReaction(v.ctx(), spec)
	})
}

func (v *VM) luaDiscordClearReactionsForEmoji(l *lua.LState) int {
	return v.luaDiscordReactionMutation(
		l,
		"discord.clear_reactions_for_emoji",
		v.perms.Discord.Reactions,
		func(spec ReactionSpec) error {
			return v.discord.ClearReactionsForEmoji(v.ctx(), spec)
		},
	)
}

func (v *VM) luaDiscordReactionMutation(
	l *lua.LState,
	permName string,
	allowed bool,
	run func(ReactionSpec) error,
) int {
	if !allowed {
		l.RaiseError("permission denied: %s", permName)
		return 0
	}
	if v.discord == nil {
		return pushDiscordBoolResult(l, false, "discord unavailable")
	}

	input := luaReactionSpec(l.CheckTable(1), v.channel)
	if input.ChannelID == 0 || input.MessageID == 0 || input.Emoji == "" {
		l.RaiseError("invalid reaction spec")
		return 0
	}
	if err := run(input); err != nil {
		return pushDiscordBoolResult(l, false, err.Error())
	}
	return pushDiscordBoolResult(l, true, "")
}

func (v *VM) luaDiscordRemoveUserReaction(l *lua.LState) int {
	if !v.perms.Discord.Reactions {
		l.RaiseError("permission denied: discord.remove_user_reaction")
		return 0
	}
	if v.discord == nil {
		return pushDiscordBoolResult(l, false, "discord unavailable")
	}

	spec := l.CheckTable(1)
	input := ReactionUserSpec{
		ChannelID: luaSnowflake(spec.RawGetString("channel_id"), v.channel),
		MessageID: luaSnowflake(spec.RawGetString("message_id"), 0),
		UserID:    luaSnowflake(spec.RawGetString("user_id"), v.userID),
		Emoji:     strings.TrimSpace(luaStringDefault(spec.RawGetString("emoji"), "")),
	}
	if input.ChannelID == 0 || input.MessageID == 0 || input.UserID == 0 || input.Emoji == "" {
		l.RaiseError("invalid reaction spec")
		return 0
	}
	if err := v.discord.RemoveUserReaction(v.ctx(), input); err != nil {
		return pushDiscordBoolResult(l, false, err.Error())
	}
	return pushDiscordBoolResult(l, true, "")
}

func (v *VM) luaDiscordClearReactions(l *lua.LState) int {
	if !v.perms.Discord.Reactions {
		l.RaiseError("permission denied: discord.clear_reactions")
		return 0
	}
	if v.discord == nil {
		return pushDiscordBoolResult(l, false, "discord unavailable")
	}

	spec := v.luaMessageGetSpec(l.CheckTable(1))
	if spec.ChannelID == 0 || spec.MessageID == 0 {
		l.RaiseError("invalid clear_reactions spec")
		return 0
	}
	if err := v.discord.ClearReactions(v.ctx(), spec); err != nil {
		return pushDiscordBoolResult(l, false, err.Error())
	}
	return pushDiscordBoolResult(l, true, "")
}

func luaReactionSpec(spec *lua.LTable, fallbackChannelID uint64) ReactionSpec {
	return ReactionSpec{
		ChannelID: luaSnowflake(spec.RawGetString("channel_id"), fallbackChannelID),
		MessageID: luaSnowflake(spec.RawGetString("message_id"), 0),
		Emoji:     strings.TrimSpace(luaStringDefault(spec.RawGetString("emoji"), "")),
	}
}
