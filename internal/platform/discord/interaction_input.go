package discordplatform

import (
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	"github.com/xsyetopz/go-mamusiabtw/internal/pluginhost"
)

func snowflakePtrToString(id *snowflake.ID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func pluginOptions(data discord.SlashCommandInteractionData) map[string]any {
	opts := map[string]any{}

	if data.SubCommandGroupName != nil {
		name := strings.TrimSpace(*data.SubCommandGroupName)
		if name != "" {
			opts["__group"] = name
		}
	}
	if data.SubCommandName != nil {
		name := strings.TrimSpace(*data.SubCommandName)
		if name != "" {
			opts["__subcommand"] = name
		}
	}

	for _, opt := range data.All() {
		name := strings.TrimSpace(opt.Name)
		if name == "" {
			continue
		}

		if opt.Type == discord.ApplicationCommandOptionTypeString {
			opts[name] = opt.String()
			continue
		}
		if opt.Type == discord.ApplicationCommandOptionTypeInt {
			opts[name] = opt.Int()
			continue
		}
		if opt.Type == discord.ApplicationCommandOptionTypeBool {
			opts[name] = opt.Bool()
			continue
		}
		if opt.Type == discord.ApplicationCommandOptionTypeFloat {
			opts[name] = opt.Float()
			continue
		}
		if opt.Type == discord.ApplicationCommandOptionTypeUser ||
			opt.Type == discord.ApplicationCommandOptionTypeMentionable {
			opts[name] = opt.Snowflake().String()
			if opt.Type == discord.ApplicationCommandOptionTypeUser {
				user := data.User(name)
				opts["__resolved:"+name] = map[string]any{
					"id":      user.ID.String(),
					"bot":     user.Bot,
					"system":  user.System,
					"mention": user.Mention(),
				}
			}
			continue
		}
		if opt.Type == discord.ApplicationCommandOptionTypeChannel ||
			opt.Type == discord.ApplicationCommandOptionTypeRole ||
			opt.Type == discord.ApplicationCommandOptionTypeAttachment {
			opts[name] = opt.Snowflake().String()
			continue
		}
	}

	return opts
}

func componentOptions(e *events.ComponentInteractionCreate) map[string]any {
	opts := map[string]any{}

	if e.Data.Type() == discord.ComponentTypeButton {
		opts["type"] = "button"
		return opts
	}
	if e.Data.Type() == discord.ComponentTypeStringSelectMenu {
		opts["type"] = "string_select"
		data := e.StringSelectMenuInteractionData()
		vals := make([]any, 0, len(data.Values))
		for _, v := range data.Values {
			vals = append(vals, v)
		}
		opts["values"] = vals
		return opts
	}
	if e.Data.Type() == discord.ComponentTypeUserSelectMenu {
		opts["type"] = "user_select"
		data := e.UserSelectMenuInteractionData()
		vals := make([]any, 0, len(data.Values))
		for _, v := range data.Values {
			vals = append(vals, v.String())
		}
		opts["values"] = vals
		return opts
	}
	if e.Data.Type() == discord.ComponentTypeRoleSelectMenu {
		opts["type"] = "role_select"
		data := e.RoleSelectMenuInteractionData()
		vals := make([]any, 0, len(data.Values))
		for _, v := range data.Values {
			vals = append(vals, v.String())
		}
		opts["values"] = vals
		return opts
	}
	if e.Data.Type() == discord.ComponentTypeMentionableSelectMenu {
		opts["type"] = "mentionable_select"
		data := e.MentionableSelectMenuInteractionData()
		vals := make([]any, 0, len(data.Values))
		for _, v := range data.Values {
			vals = append(vals, v.String())
		}
		opts["values"] = vals
		return opts
	}
	if e.Data.Type() == discord.ComponentTypeChannelSelectMenu {
		opts["type"] = "channel_select"
		data := e.ChannelSelectMenuInteractionData()
		vals := make([]any, 0, len(data.Values))
		for _, v := range data.Values {
			vals = append(vals, v.String())
		}
		opts["values"] = vals
		return opts
	}

	return opts
}

func modalOptions(e *events.ModalSubmitInteractionCreate, pluginID string) map[string]any {
	opts := map[string]any{}

	fields := map[string]any{}
	for component := range e.Data.AllComponents() {
		var customID, value string
		switch ti := component.(type) {
		case discord.TextInputComponent:
			customID = ti.CustomID
			value = ti.Value
		case *discord.TextInputComponent:
			if ti == nil {
				continue
			}
			customID = ti.CustomID
			value = ti.Value
		default:
			continue
		}

		cid := strings.TrimSpace(customID)
		if cid == "" {
			continue
		}
		pid, localID, ok := pluginhost.ParseCustomID(cid)
		if !ok || pid != pluginID {
			continue
		}
		fields[localID] = value
	}
	opts["fields"] = fields
	return opts
}
