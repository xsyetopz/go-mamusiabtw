package cmdadmin

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

	"github.com/xsyetopz/go-mamusiabtw/internal/discordapp/core"
	"github.com/xsyetopz/go-mamusiabtw/internal/discordapp/interactions"
	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
	pluginpkg "github.com/xsyetopz/go-mamusiabtw/internal/plugins"
	"github.com/xsyetopz/go-mamusiabtw/internal/store"
)

func Commands() []core.SlashCommand {
	return []core.SlashCommand{
		block(),
		unblock(),
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

func block() core.SlashCommand {
	return core.SlashCommand{
		Name:          "block",
		NameID:        "cmd.block.name",
		DescID:        "cmd.block.desc",
		CreateCommand: blockCreateCommand,
		Handle:        blockHandle,
	}
}

func blockCreateCommand(locales []string, t core.Translator) discord.ApplicationCommandCreate {
	maxLen := 255
	minLen := 1
	perm := discord.PermissionAdministrator
	loc := func(id string) map[discord.Locale]string {
		return core.LocalizeMap(locales, func(locale string) string {
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
	t core.Translator,
	s core.Services,
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
	t core.Translator,
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
	t core.Translator,
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

func unblock() core.SlashCommand {
	return core.SlashCommand{
		Name:          "unblock",
		NameID:        "cmd.unblock.name",
		DescID:        "cmd.unblock.desc",
		CreateCommand: unblockCreateCommand,
		Handle:        unblockHandle,
	}
}

func unblockCreateCommand(locales []string, t core.Translator) discord.ApplicationCommandCreate {
	maxLen := 255
	minLen := 1
	perm := discord.PermissionAdministrator
	loc := func(id string) map[discord.Locale]string {
		return core.LocalizeMap(locales, func(locale string) string {
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
	t core.Translator,
	s core.Services,
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

func plugins() core.SlashCommand {
	return core.SlashCommand{
		Name:          "plugins",
		NameID:        "cmd.plugins.name",
		DescID:        "cmd.plugins.desc",
		CreateCommand: pluginsCreateCommand,
		Handle:        pluginsHandle,
	}
}

func pluginsCreateCommand(locales []string, t core.Translator) discord.ApplicationCommandCreate {
	loc := func(id string) map[discord.Locale]string {
		return core.LocalizeMap(locales, func(locale string) string {
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
	t core.Translator,
	s core.Services,
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

func pluginsHandleList(t core.Translator, p core.PluginAdmin) interactions.SlashAction {
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

func formatPluginInfoLine(info pluginpkg.PluginInfo) string {
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
