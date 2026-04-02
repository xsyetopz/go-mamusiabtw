package roles

import (
	"context"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/omit"
	"github.com/disgoorg/snowflake/v2"

	"github.com/xsyetopz/go-mamusiabtw/internal/features/shared"
	"github.com/xsyetopz/go-mamusiabtw/internal/features/commandapi"
	"github.com/xsyetopz/go-mamusiabtw/internal/platform/discord/discordutil"
	"github.com/xsyetopz/go-mamusiabtw/internal/platform/discord/interactions"
	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
)

func Commands() []commandapi.SlashCommand {
	return []commandapi.SlashCommand{roles()}
}

func roles() commandapi.SlashCommand {
	return commandapi.SlashCommand{
		Name:          "roles",
		NameID:        "cmd.roles.name",
		DescID:        "cmd.roles.desc",
		CreateCommand: rolesCreateCommand,
		Handle:        rolesHandle,
	}
}

func rolesCreateCommand(locales []string, t commandapi.Translator) discord.ApplicationCommandCreate {
	perm := discord.PermissionManageRoles

	loc := func(id string) map[discord.Locale]string {
		return commandapi.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{
				Locale:    locale,
				MessageID: id,
			})
		})
	}

	minName, maxName := 1, 100
	minColor, maxColor := 6, 7 // allow optional leading '#'
	options := rolesOptions(t, loc, &minName, &maxName, &minColor, &maxColor)

	return discord.SlashCommandCreate{
		Name: "roles",
		NameLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{
				Locale:    locale,
				MessageID: "cmd.roles.name",
			})
		}),
		Description: t.S("cmd.roles.desc", nil),
		DescriptionLocalizations: commandapi.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{
				Locale:    locale,
				MessageID: "cmd.roles.desc",
			})
		}),
		DefaultMemberPermissions: omit.NewPtr(perm),
		Options:                  options,
	}
}

type rolesLocalizer func(id string) map[discord.Locale]string

func rolesOptions(
	t commandapi.Translator,
	loc rolesLocalizer,
	minName, maxName, minColor, maxColor *int,
) []discord.ApplicationCommandOption {
	return []discord.ApplicationCommandOption{
		rolesSubCreate(t, loc, minName, maxName, minColor, maxColor),
		rolesSubEdit(t, loc, minName, maxName, minColor, maxColor),
		rolesSubDelete(t, loc),
		rolesSubAdd(t, loc),
		rolesSubRemove(t, loc),
	}
}

func rolesSubCreate(
	t commandapi.Translator,
	loc rolesLocalizer,
	minName, maxName, minColor, maxColor *int,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "create",
		NameLocalizations:        loc("cmd.roles.sub.create.name"),
		Description:              t.S("cmd.roles.sub.create.desc", nil),
		DescriptionLocalizations: loc("cmd.roles.sub.create.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:                     "name",
				NameLocalizations:        loc("cmd.roles.opt.name.name"),
				Description:              t.S("cmd.roles.opt.name.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.name.desc"),
				Required:                 true,
				MinLength:                minName,
				MaxLength:                maxName,
			},
			discord.ApplicationCommandOptionString{
				Name:                     "colour",
				NameLocalizations:        loc("cmd.roles.opt.colour.name"),
				Description:              t.S("cmd.roles.opt.colour.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.colour.desc"),
				MinLength:                minColor,
				MaxLength:                maxColor,
			},
			discord.ApplicationCommandOptionBool{
				Name:                     "hoist",
				NameLocalizations:        loc("cmd.roles.opt.hoist.name"),
				Description:              t.S("cmd.roles.opt.hoist.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.hoist.desc"),
			},
			discord.ApplicationCommandOptionBool{
				Name:                     "mentionable",
				NameLocalizations:        loc("cmd.roles.opt.mentionable.name"),
				Description:              t.S("cmd.roles.opt.mentionable.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.mentionable.desc"),
			},
		},
	}
}

func rolesSubEdit(
	t commandapi.Translator,
	loc rolesLocalizer,
	minName, maxName, minColor, maxColor *int,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "edit",
		NameLocalizations:        loc("cmd.roles.sub.edit.name"),
		Description:              t.S("cmd.roles.sub.edit.desc", nil),
		DescriptionLocalizations: loc("cmd.roles.sub.edit.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionRole{
				Name:                     "role",
				NameLocalizations:        loc("cmd.roles.opt.role.name"),
				Description:              t.S("cmd.roles.opt.role.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.role.desc"),
				Required:                 true,
			},
			discord.ApplicationCommandOptionString{
				Name:                     "name",
				NameLocalizations:        loc("cmd.roles.opt.name.name"),
				Description:              t.S("cmd.roles.opt.name.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.name.desc"),
				MinLength:                minName,
				MaxLength:                maxName,
			},
			discord.ApplicationCommandOptionString{
				Name:                     "colour",
				NameLocalizations:        loc("cmd.roles.opt.colour.name"),
				Description:              t.S("cmd.roles.opt.colour.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.colour.desc"),
				MinLength:                minColor,
				MaxLength:                maxColor,
			},
			discord.ApplicationCommandOptionBool{
				Name:                     "hoist",
				NameLocalizations:        loc("cmd.roles.opt.hoist.name"),
				Description:              t.S("cmd.roles.opt.hoist.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.hoist.desc"),
			},
			discord.ApplicationCommandOptionBool{
				Name:                     "mentionable",
				NameLocalizations:        loc("cmd.roles.opt.mentionable.name"),
				Description:              t.S("cmd.roles.opt.mentionable.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.mentionable.desc"),
			},
		},
	}
}

func rolesSubDelete(t commandapi.Translator, loc rolesLocalizer) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "delete",
		NameLocalizations:        loc("cmd.roles.sub.delete.name"),
		Description:              t.S("cmd.roles.sub.delete.desc", nil),
		DescriptionLocalizations: loc("cmd.roles.sub.delete.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionRole{
				Name:                     "role",
				NameLocalizations:        loc("cmd.roles.opt.role.name"),
				Description:              t.S("cmd.roles.opt.role.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.role.desc"),
				Required:                 true,
			},
		},
	}
}

func rolesSubAdd(t commandapi.Translator, loc rolesLocalizer) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "add",
		NameLocalizations:        loc("cmd.roles.sub.add.name"),
		Description:              t.S("cmd.roles.sub.add.desc", nil),
		DescriptionLocalizations: loc("cmd.roles.sub.add.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionRole{
				Name:                     "role",
				NameLocalizations:        loc("cmd.roles.opt.role.name"),
				Description:              t.S("cmd.roles.opt.role.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.role.desc"),
				Required:                 true,
			},
			discord.ApplicationCommandOptionUser{
				Name:                     "member",
				NameLocalizations:        loc("cmd.roles.opt.member.name"),
				Description:              t.S("cmd.roles.opt.member.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.member.desc"),
				Required:                 true,
			},
		},
	}
}

func rolesSubRemove(t commandapi.Translator, loc rolesLocalizer) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "remove",
		NameLocalizations:        loc("cmd.roles.sub.remove.name"),
		Description:              t.S("cmd.roles.sub.remove.desc", nil),
		DescriptionLocalizations: loc("cmd.roles.sub.remove.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionRole{
				Name:                     "role",
				NameLocalizations:        loc("cmd.roles.opt.role.name"),
				Description:              t.S("cmd.roles.opt.role.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.role.desc"),
				Required:                 true,
			},
			discord.ApplicationCommandOptionUser{
				Name:                     "member",
				NameLocalizations:        loc("cmd.roles.opt.member.name"),
				Description:              t.S("cmd.roles.opt.member.desc", nil),
				DescriptionLocalizations: loc("cmd.roles.opt.member.desc"),
				Required:                 true,
			},
		},
	}
}

func rolesHandle(
	_ context.Context,
	_ *events.ApplicationCommandInteractionCreate,
	t commandapi.Translator,
	_ commandapi.Services,
) (interactions.SlashAction, error) {
	return interactions.SlashFunc(func(e *events.ApplicationCommandInteractionCreate) error {
		return rolesExecute(e, t)
	}), nil
}

func rolesExecute(e *events.ApplicationCommandInteractionCreate, t commandapi.Translator) error {
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
		return rolesCreate(e, *guildID, data, t)
	case "edit":
		return rolesEdit(e, *guildID, data, t)
	case "delete":
		return rolesDelete(e, *guildID, data, t)
	case "add":
		return rolesAdd(e, *guildID, data, t)
	case "remove":
		return rolesRemove(e, *guildID, data, t)
	default:
		return shared.UpdateInteractionError(e, t.S("err.generic", nil))
	}
}

func rolesCreate(
	e *events.ApplicationCommandInteractionCreate,
	guildID snowflake.ID,
	data discord.SlashCommandInteractionData,
	t commandapi.Translator,
) error {
	name := strings.TrimSpace(data.String("name"))
	colourRaw, _ := data.OptString("colour")
	hoist, _ := data.OptBool("hoist")
	mentionable, _ := data.OptBool("mentionable")

	color, ok := parseOptionalHexColor(strings.TrimSpace(colourRaw))
	if !ok {
		return shared.UpdateInteractionError(e, t.S("mgr.roles.invalid_colour", map[string]any{"Colour": colourRaw}))
	}

	role, err := e.Client().Rest.CreateRole(guildID, discord.RoleCreate{
		Name:        name,
		Color:       color,
		Hoist:       hoist,
		Mentionable: mentionable,
	})
	if err != nil {
		return shared.UpdateInteractionError(e, t.S("mgr.roles.create_error", map[string]any{"Name": name}))
	}

	return shared.UpdateInteractionSuccess(e, t.S("mgr.roles.create_success", map[string]any{
		"Role": discord.RoleMention(role.ID),
		"Name": name,
	}))
}

func rolesRoleUpdate(
	data discord.SlashCommandInteractionData,
	t commandapi.Translator,
	e *events.ApplicationCommandInteractionCreate,
) (discord.RoleUpdate, error) {
	var upd discord.RoleUpdate
	if name, ok := data.OptString("name"); ok {
		s := strings.TrimSpace(name)
		if s != "" {
			upd.Name = &s
		}
	}
	if colourRaw, ok := data.OptString("colour"); ok {
		v, parsed := discordutil.ParseHexColor(colourRaw)
		if !parsed {
			return upd, shared.UpdateInteractionError(
				e,
				t.S("mgr.roles.invalid_colour", map[string]any{"Colour": colourRaw}),
			)
		}
		upd.Color = &v
	}
	if v, ok := data.OptBool("hoist"); ok {
		upd.Hoist = &v
	}
	if v, ok := data.OptBool("mentionable"); ok {
		upd.Mentionable = &v
	}
	return upd, nil
}

func rolesEdit(
	e *events.ApplicationCommandInteractionCreate,
	guildID snowflake.ID,
	data discord.SlashCommandInteractionData,
	t commandapi.Translator,
) error {
	role := data.Role("role")
	if uint64(role.ID) == uint64(guildID) {
		return shared.UpdateInteractionError(e, t.S("mgr.roles.cannot_edit_everyone", nil))
	}

	upd, err := rolesRoleUpdate(data, t, e)
	if err != nil {
		return err
	}

	if _, updateErr := e.Client().Rest.UpdateRole(guildID, role.ID, upd); updateErr != nil {
		return shared.UpdateInteractionError(
			e,
			t.S("mgr.roles.edit_error", map[string]any{"Role": discord.RoleMention(role.ID)}),
		)
	}
	return shared.UpdateInteractionSuccess(
		e,
		t.S("mgr.roles.edit_success", map[string]any{"Role": discord.RoleMention(role.ID)}),
	)
}

func rolesDelete(
	e *events.ApplicationCommandInteractionCreate,
	guildID snowflake.ID,
	data discord.SlashCommandInteractionData,
	t commandapi.Translator,
) error {
	role := data.Role("role")
	if uint64(role.ID) == uint64(guildID) {
		return shared.UpdateInteractionError(e, t.S("mgr.roles.cannot_delete_everyone", nil))
	}

	if err := e.Client().Rest.DeleteRole(guildID, role.ID); err != nil {
		return shared.UpdateInteractionError(
			e,
			t.S("mgr.roles.delete_error", map[string]any{"Role": discord.RoleMention(role.ID)}),
		)
	}
	return shared.UpdateInteractionSuccess(e, t.S("mgr.roles.delete_success", map[string]any{"Name": role.Name}))
}

func rolesAdd(
	e *events.ApplicationCommandInteractionCreate,
	guildID snowflake.ID,
	data discord.SlashCommandInteractionData,
	t commandapi.Translator,
) error {
	role := data.Role("role")
	member := data.User("member")

	if err := e.Client().Rest.AddMemberRole(guildID, member.ID, role.ID); err != nil {
		return shared.UpdateInteractionError(e, t.S("mgr.roles.add_error", map[string]any{
			"Role": discord.RoleMention(role.ID),
			"User": member.Mention(),
		}))
	}
	return shared.UpdateInteractionSuccess(e, t.S("mgr.roles.add_success", map[string]any{
		"Role": discord.RoleMention(role.ID),
		"User": member.Mention(),
	}))
}

func rolesRemove(
	e *events.ApplicationCommandInteractionCreate,
	guildID snowflake.ID,
	data discord.SlashCommandInteractionData,
	t commandapi.Translator,
) error {
	role := data.Role("role")
	member := data.User("member")

	if err := e.Client().Rest.RemoveMemberRole(guildID, member.ID, role.ID); err != nil {
		return shared.UpdateInteractionError(e, t.S("mgr.roles.remove_error", map[string]any{
			"Role": discord.RoleMention(role.ID),
			"User": member.Mention(),
		}))
	}
	return shared.UpdateInteractionSuccess(e, t.S("mgr.roles.remove_success", map[string]any{
		"Role": discord.RoleMention(role.ID),
		"User": member.Mention(),
	}))
}

func parseOptionalHexColor(raw string) (int, bool) {
	if raw == "" {
		return 0, true
	}
	v, ok := discordutil.ParseHexColor(raw)
	if !ok {
		return 0, false
	}
	return v, true
}
