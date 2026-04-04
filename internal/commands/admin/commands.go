package admin

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/omit"
	"github.com/disgoorg/snowflake/v2"

	commandapi "github.com/xsyetopz/go-mamusiabtw/internal/commands/api"
	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
	"github.com/xsyetopz/go-mamusiabtw/internal/runtime/discord/interactions"
	pluginhost "github.com/xsyetopz/go-mamusiabtw/internal/runtime/plugins"
	store "github.com/xsyetopz/go-mamusiabtw/internal/storage"
)

func Commands() []commandapi.SlashCommand {
	return []commandapi.SlashCommand{
		block(),
		unblock(),
		modules(),
		plugins(),
	}
}

func parseUint64Base10(raw string) (uint64, bool) {
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func block() commandapi.SlashCommand {
	return commandapi.SlashCommand{
		Name:          "block",
		NameID:        "cmd.block.name",
		DescID:        "cmd.block.desc",
		CreateCommand: blockCreateCommand,
		Handle:        blockHandle,
	}
}

func blockCreateCommand(locales []string, t commandapi.Translator) discord.ApplicationCommandCreate {
	maxLen := 255
	minLen := 1
	perm := discord.PermissionAdministrator
	loc := func(id string) map[discord.Locale]string {
		return commandapi.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: id})
		})
	}

	return discord.SlashCommandCreate{
		Name:                     "block",
		NameLocalizations:        loc("cmd.block.name"),
		Description:              t.S("cmd.block.desc", nil),
		DescriptionLocalizations: loc("cmd.block.desc"),
		DefaultMemberPermissions: omit.NewPtr(perm),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:                     "user",
				NameLocalizations:        loc("cmd.block.sub.user.name"),
				Description:              t.S("cmd.block.sub.user.desc", nil),
				DescriptionLocalizations: loc("cmd.block.sub.user.desc"),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionUser{
						Name:                     "user",
						NameLocalizations:        loc("cmd.block.opt.user.name"),
						Description:              t.S("cmd.block.opt.user.desc", nil),
						DescriptionLocalizations: loc("cmd.block.opt.user.desc"),
						Required:                 true,
					},
					discord.ApplicationCommandOptionString{
						Name:                     "reason",
						NameLocalizations:        loc("cmd.block.opt.reason.name"),
						Description:              t.S("cmd.block.opt.reason.desc", nil),
						DescriptionLocalizations: loc("cmd.block.opt.reason.desc"),
						Required:                 true,
						MinLength:                &minLen,
						MaxLength:                &maxLen,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:                     "guild",
				NameLocalizations:        loc("cmd.block.sub.guild.name"),
				Description:              t.S("cmd.block.sub.guild.desc", nil),
				DescriptionLocalizations: loc("cmd.block.sub.guild.desc"),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:                     "guild_id",
						NameLocalizations:        loc("cmd.block.opt.guild_id.name"),
						Description:              t.S("cmd.block.opt.guild_id.desc", nil),
						DescriptionLocalizations: loc("cmd.block.opt.guild_id.desc"),
						Required:                 true,
						MinLength:                &minLen,
						MaxLength:                &maxLen,
					},
					discord.ApplicationCommandOptionString{
						Name:                     "reason",
						NameLocalizations:        loc("cmd.block.opt.reason.name"),
						Description:              t.S("cmd.block.opt.reason.desc", nil),
						DescriptionLocalizations: loc("cmd.block.opt.reason.desc"),
						Required:                 true,
						MinLength:                &minLen,
						MaxLength:                &maxLen,
					},
				},
			},
		},
	}
}

func blockHandle(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t commandapi.Translator,
	s commandapi.Services,
) (interactions.SlashAction, error) {
	actorID := uint64(e.User().ID)
	if s.IsOwner == nil || !s.IsOwner(actorID) {
		msg := discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("admin.not_owner", nil))
		return interactions.SlashMessage{Create: msg}, nil
	}
	if s.Store == nil {
		return nil, errors.New("store not configured")
	}

	data := e.SlashCommandInteractionData()
	sub := data.SubCommandName
	if sub == nil {
		msg := discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil))
		return interactions.SlashMessage{Create: msg}, nil
	}

	restrictions := s.Store.Restrictions()
	switch *sub {
	case "user":
		return blockUser(ctx, t, restrictions, actorID, data)
	case "guild":
		return blockGuild(ctx, e, t, restrictions, actorID, data)
	default:
		msg := discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil))
		return interactions.SlashMessage{Create: msg}, nil
	}
}

func blockUser(
	ctx context.Context,
	t commandapi.Translator,
	restrictions store.RestrictionStore,
	actorID uint64,
	data discord.SlashCommandInteractionData,
) (interactions.SlashAction, error) {
	user := data.User("user")
	reason := strings.TrimSpace(data.String("reason"))

	if err := restrictions.PutRestriction(ctx, store.Restriction{
		TargetType: store.TargetTypeUser,
		TargetID:   uint64(user.ID),
		Reason:     reason,
		CreatedBy:  actorID,
		CreatedAt:  time.Now().UTC(),
	}); err != nil {
		return nil, err
	}
	msg := discord.NewMessageCreate().
		WithEphemeral(true).
		WithContent(t.S("admin.block.user.success", map[string]any{"User": user.Mention()}))
	return interactions.SlashMessage{Create: msg}, nil
}

func blockGuild(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t commandapi.Translator,
	restrictions store.RestrictionStore,
	actorID uint64,
	data discord.SlashCommandInteractionData,
) (interactions.SlashAction, error) {
	raw := strings.TrimSpace(data.String("guild_id"))
	guildID, ok := parseUint64Base10(raw)
	if !ok {
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().
				WithEphemeral(true).
				WithContent(t.S("admin.block.guild.invalid", map[string]any{"GuildID": raw})),
		}, nil
	}

	reason := strings.TrimSpace(data.String("reason"))
	if err := restrictions.PutRestriction(ctx, store.Restriction{
		TargetType: store.TargetTypeGuild,
		TargetID:   guildID,
		Reason:     reason,
		CreatedBy:  actorID,
		CreatedAt:  time.Now().UTC(),
	}); err != nil {
		return nil, err
	}

	if currentGuild := e.GuildID(); currentGuild != nil && uint64(*currentGuild) == guildID {
		go func() { _ = e.Client().Rest.LeaveGuild(snowflake.ID(guildID)) }()
	}

	msg := discord.NewMessageCreate().
		WithEphemeral(true).
		WithContent(t.S("admin.block.guild.success", map[string]any{"GuildID": raw}))
	return interactions.SlashMessage{Create: msg}, nil
}

func unblock() commandapi.SlashCommand {
	return commandapi.SlashCommand{
		Name:          "unblock",
		NameID:        "cmd.unblock.name",
		DescID:        "cmd.unblock.desc",
		CreateCommand: unblockCreateCommand,
		Handle:        unblockHandle,
	}
}

func unblockCreateCommand(locales []string, t commandapi.Translator) discord.ApplicationCommandCreate {
	maxLen := 255
	minLen := 1
	perm := discord.PermissionAdministrator
	loc := func(id string) map[discord.Locale]string {
		return commandapi.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: id})
		})
	}

	return discord.SlashCommandCreate{
		Name:                     "unblock",
		NameLocalizations:        loc("cmd.unblock.name"),
		Description:              t.S("cmd.unblock.desc", nil),
		DescriptionLocalizations: loc("cmd.unblock.desc"),
		DefaultMemberPermissions: omit.NewPtr(perm),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:                     "user",
				NameLocalizations:        loc("cmd.unblock.sub.user.name"),
				Description:              t.S("cmd.unblock.sub.user.desc", nil),
				DescriptionLocalizations: loc("cmd.unblock.sub.user.desc"),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionUser{
						Name:                     "user",
						NameLocalizations:        loc("cmd.unblock.opt.user.name"),
						Description:              t.S("cmd.unblock.opt.user.desc", nil),
						DescriptionLocalizations: loc("cmd.unblock.opt.user.desc"),
						Required:                 true,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:                     "guild",
				NameLocalizations:        loc("cmd.unblock.sub.guild.name"),
				Description:              t.S("cmd.unblock.sub.guild.desc", nil),
				DescriptionLocalizations: loc("cmd.unblock.sub.guild.desc"),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:                     "guild_id",
						NameLocalizations:        loc("cmd.unblock.opt.guild_id.name"),
						Description:              t.S("cmd.unblock.opt.guild_id.desc", nil),
						DescriptionLocalizations: loc("cmd.unblock.opt.guild_id.desc"),
						Required:                 true,
						MinLength:                &minLen,
						MaxLength:                &maxLen,
					},
				},
			},
		},
	}
}

func unblockHandle(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t commandapi.Translator,
	s commandapi.Services,
) (interactions.SlashAction, error) {
	actorID := uint64(e.User().ID)
	if s.IsOwner == nil || !s.IsOwner(actorID) {
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("admin.not_owner", nil)),
		}, nil
	}
	if s.Store == nil {
		return nil, errors.New("store not configured")
	}

	data := e.SlashCommandInteractionData()
	sub := data.SubCommandName
	if sub == nil {
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil)),
		}, nil
	}

	restrictions := s.Store.Restrictions()

	switch *sub {
	case "user":
		user := data.User("user")
		if err := restrictions.DeleteRestriction(ctx, store.TargetTypeUser, uint64(user.ID)); err != nil {
			return nil, err
		}
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().
				WithEphemeral(true).
				WithContent(t.S("admin.unblock.user.success", map[string]any{"User": user.Mention()})),
		}, nil

	case "guild":
		raw := strings.TrimSpace(data.String("guild_id"))
		guildID, ok := parseUint64Base10(raw)
		if !ok {
			return interactions.SlashMessage{
				Create: discord.NewMessageCreate().
					WithEphemeral(true).
					WithContent(t.S("admin.unblock.guild.invalid", map[string]any{"GuildID": raw})),
			}, nil
		}
		if err := restrictions.DeleteRestriction(ctx, store.TargetTypeGuild, guildID); err != nil {
			return nil, err
		}
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().
				WithEphemeral(true).
				WithContent(t.S("admin.unblock.guild.success", map[string]any{"GuildID": raw})),
		}, nil
	default:
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil)),
		}, nil
	}
}

func plugins() commandapi.SlashCommand {
	return commandapi.SlashCommand{
		Name:          "plugins",
		NameID:        "cmd.plugins.name",
		DescID:        "cmd.plugins.desc",
		CreateCommand: pluginsCreateCommand,
		Handle:        pluginsHandle,
	}
}

func modules() commandapi.SlashCommand {
	return commandapi.SlashCommand{
		Name: "modules",
		CreateCommand: func(_ []string, _ commandapi.Translator) discord.ApplicationCommandCreate {
			return discord.SlashCommandCreate{
				Name:        "modules",
				Description: "Inspect and manage modules",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommand{
						Name:        "list",
						Description: "List built-ins and plugins",
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "info",
						Description: "Show one module",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "module",
								Description: "Module ID",
								Required:    true,
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "enable",
						Description: "Enable one module",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "module",
								Description: "Module ID",
								Required:    true,
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "disable",
						Description: "Disable one module",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "module",
								Description: "Module ID",
								Required:    true,
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "reset",
						Description: "Reset one module to its default",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "module",
								Description: "Module ID",
								Required:    true,
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "reload",
						Description: "Reload official and user plugins",
					},
				},
			}
		},
		Handle: modulesHandle,
	}
}

func modulesHandle(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	_ commandapi.Translator,
	s commandapi.Services,
) (interactions.SlashAction, error) {
	actorID := uint64(e.User().ID)
	if s.IsOwner == nil || !s.IsOwner(actorID) {
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent("Only the bot owner can manage modules."),
		}, nil
	}
	if s.Modules == nil || !s.Modules.Configured() {
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent("Modules are not configured."),
		}, nil
	}

	data := e.SlashCommandInteractionData()
	sub := data.SubCommandName
	if sub == nil {
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent("Missing subcommand."),
		}, nil
	}

	switch *sub {
	case "list":
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(formatModuleList(s.Modules.Infos())),
		}, nil
	case "info":
		moduleID := strings.TrimSpace(data.String("module"))
		for _, info := range s.Modules.Infos() {
			if info.ID != moduleID {
				continue
			}
			return interactions.SlashMessage{
				Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(formatModuleDetails(info)),
			}, nil
		}
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent("Unknown module: " + moduleID),
		}, nil
	case "enable":
		moduleID := strings.TrimSpace(data.String("module"))
		if err := s.Modules.SetEnabled(ctx, moduleID, true, actorID); err != nil {
			return interactions.SlashMessage{
				Create: discord.NewMessageCreate().WithEphemeral(true).WithContent("Enable failed: " + err.Error()),
			}, nil
		}
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent("Enabled module: " + moduleID),
		}, nil
	case "disable":
		moduleID := strings.TrimSpace(data.String("module"))
		if err := s.Modules.SetEnabled(ctx, moduleID, false, actorID); err != nil {
			return interactions.SlashMessage{
				Create: discord.NewMessageCreate().WithEphemeral(true).WithContent("Disable failed: " + err.Error()),
			}, nil
		}
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent("Disabled module: " + moduleID),
		}, nil
	case "reset":
		moduleID := strings.TrimSpace(data.String("module"))
		if err := s.Modules.Reset(ctx, moduleID); err != nil {
			return interactions.SlashMessage{
				Create: discord.NewMessageCreate().WithEphemeral(true).WithContent("Reset failed: " + err.Error()),
			}, nil
		}
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent("Reset module to default: " + moduleID),
		}, nil
	case "reload":
		if err := s.Modules.Reload(ctx); err != nil {
			return interactions.SlashMessage{
				Create: discord.NewMessageCreate().WithEphemeral(true).WithContent("Reload failed: " + err.Error()),
			}, nil
		}
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent("Reloaded modules."),
		}, nil
	default:
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent("Unknown subcommand."),
		}, nil
	}
}

func formatModuleList(infos []commandapi.ModuleInfo) string {
	if len(infos) == 0 {
		return "No modules are loaded."
	}

	lines := make([]string, 0, len(infos)+1)
	lines = append(lines, "Loaded modules:")
	for _, info := range infos {
		state := "disabled"
		if info.Enabled {
			state = "enabled"
		}
		toggle := "fixed"
		if info.Toggleable {
			toggle = "toggleable"
		}
		lines = append(lines, "- "+info.ID+" ["+string(info.Kind)+", "+string(info.Runtime)+", "+state+", "+toggle+"]")
	}
	return strings.Join(lines, "\n")
}

func formatModuleDetails(info commandapi.ModuleInfo) string {
	commands := "none"
	if len(info.Commands) != 0 {
		commands = strings.Join(info.Commands, ", ")
	}
	return strings.Join([]string{
		"Module: " + info.ID,
		"Name: " + info.Name,
		"Kind: " + string(info.Kind),
		"Runtime: " + string(info.Runtime),
		"Enabled: " + strconv.FormatBool(info.Enabled),
		"Default enabled: " + strconv.FormatBool(info.DefaultEnabled),
		"Toggleable: " + strconv.FormatBool(info.Toggleable),
		"Signed: " + strconv.FormatBool(info.Signed),
		"Source: " + info.Source,
		"Commands: " + commands,
	}, "\n")
}

func pluginsCreateCommand(locales []string, t commandapi.Translator) discord.ApplicationCommandCreate {
	loc := func(id string) map[discord.Locale]string {
		return commandapi.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: id})
		})
	}

	return discord.SlashCommandCreate{
		Name:                     "plugins",
		NameLocalizations:        loc("cmd.plugins.name"),
		Description:              t.S("cmd.plugins.desc", nil),
		DescriptionLocalizations: loc("cmd.plugins.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:                     "list",
				NameLocalizations:        loc("cmd.plugins.sub.list.name"),
				Description:              t.S("cmd.plugins.sub.list.desc", nil),
				DescriptionLocalizations: loc("cmd.plugins.sub.list.desc"),
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:                     "reload",
				NameLocalizations:        loc("cmd.plugins.sub.reload.name"),
				Description:              t.S("cmd.plugins.sub.reload.desc", nil),
				DescriptionLocalizations: loc("cmd.plugins.sub.reload.desc"),
			},
		},
	}
}

func pluginsHandle(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t commandapi.Translator,
	s commandapi.Services,
) (interactions.SlashAction, error) {
	actorID := uint64(e.User().ID)
	if s.IsOwner == nil || !s.IsOwner(actorID) {
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("admin.not_owner", nil)),
		}, nil
	}
	if s.Plugins == nil || !s.Plugins.Configured() {
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().
				WithEphemeral(true).
				WithContent(t.S("admin.plugins.not_configured", nil)),
		}, nil
	}

	data := e.SlashCommandInteractionData()
	sub := data.SubCommandName
	if sub == nil {
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil)),
		}, nil
	}

	switch *sub {
	case "list":
		return pluginsHandleList(t, s.Plugins), nil
	case "reload":
		if err := s.Plugins.Reload(ctx); err != nil {
			return nil, err
		}
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("admin.plugins.reloaded", nil)),
		}, nil
	default:
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil)),
		}, nil
	}
}

func pluginsHandleList(t commandapi.Translator, p commandapi.PluginAdmin) interactions.SlashAction {
	infos := p.Infos()
	if len(infos) == 0 {
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("admin.plugins.none", nil)),
		}
	}

	lines := make([]string, 0, len(infos))
	for _, info := range infos {
		lines = append(lines, formatPluginInfoLine(info))
	}

	return interactions.SlashMessage{
		Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("admin.plugins.list", map[string]any{
			"Count": len(infos),
			"List":  strings.Join(lines, "\n"),
		})),
	}
}

func formatPluginInfoLine(info pluginhost.PluginInfo) string {
	cmdNames := make([]string, 0, len(info.Commands))
	for _, cmd := range info.Commands {
		if strings.TrimSpace(cmd.Name) != "" {
			cmdNames = append(cmdNames, cmd.Name)
		}
	}

	sig := "unsigned"
	if info.Signed {
		sig = "signed"
	}

	name := strings.TrimSpace(info.Name)
	version := strings.TrimSpace(info.Version)
	return "- " + info.ID + " (" + name + " " + version + ", " + sig + ") cmds: " + strings.Join(cmdNames, ", ")
}
