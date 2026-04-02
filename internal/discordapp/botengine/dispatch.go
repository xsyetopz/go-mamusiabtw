package botengine

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	cmdmoderation "github.com/xsyetopz/go-mamusiabtw/internal/discordapp/commands/moderation"
	cmdwellness "github.com/xsyetopz/go-mamusiabtw/internal/discordapp/commands/wellness"
	"github.com/xsyetopz/go-mamusiabtw/internal/discordapp/core"
	"github.com/xsyetopz/go-mamusiabtw/internal/discordapp/interactions"
	"github.com/xsyetopz/go-mamusiabtw/internal/plugins"
	"github.com/xsyetopz/go-mamusiabtw/internal/present"
)

func (b *Bot) onCommand(e *events.ApplicationCommandInteractionCreate) {
	ctx := context.Background()

	locale := e.Locale()
	t := core.Translator{Registry: b.i18n, Locale: locale, UserID: uint64(e.User().ID)}

	data := e.SlashCommandInteractionData()
	cmdName := data.CommandName()

	if !b.preflightSlash(ctx, e, t) {
		return
	}

	guildID := e.GuildID()
	guildName := ""
	if guildID != nil {
		if guild, ok := e.Client().Caches.Guild(*guildID); ok {
			guildName = strings.TrimSpace(guild.Name)
		}
	}
	b.logger.Info(
		"command used",
		slog.String("cmd", cmdName),
		slog.Uint64("user_id", uint64(e.User().ID)),
		slog.String("username", strings.TrimSpace(e.User().Username)),
		slog.String("guild_name", guildName),
		slog.String("guild_id", snowflakePtrToString(guildID)),
	)

	if !b.takeCommandCooldown(e, t, cmdName, time.Now()) {
		return
	}

	if b.handleRegisteredSlash(ctx, e, t, locale, cmdName) {
		return
	}

	b.handlePluginSlash(ctx, e, t, locale, cmdName, data)
}

func (b *Bot) preflightSlash(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
) bool {
	restricted, err := b.checkRestrictions(ctx, e, t)
	if err != nil {
		b.logger.ErrorContext(ctx, "restriction check failed", slog.String("err", err.Error()))
		_ = e.CreateMessage(discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.generic", nil)))
		return false
	}
	return !restricted
}

func (b *Bot) takeCommandCooldown(
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	cmdName string,
	now time.Time,
) bool {
	key := slashCooldownKey(e, cmdName)
	if d := b.commandCooldown(key); d > 0 {
		if remaining, ok := b.cooldowns.Take(uint64(e.User().ID), key, d, now); !ok {
			msg := interactions.NoticeMessage(
				present.KindWarning,
				"",
				t.S("err.cooldown", map[string]any{"Seconds": cooldownSecs(remaining)}),
				true,
			)
			_ = e.CreateMessage(msg)
			return false
		}
	}
	return true
}

func slashCooldownKey(e *events.ApplicationCommandInteractionCreate, cmdName string) string {
	key := strings.ToLower(strings.TrimSpace(cmdName))
	if key == "" || e == nil {
		return key
	}
	data := e.SlashCommandInteractionData()

	group := ""
	if data.SubCommandGroupName != nil {
		group = strings.ToLower(strings.TrimSpace(*data.SubCommandGroupName))
	}

	sub := ""
	if data.SubCommandName != nil {
		sub = strings.ToLower(strings.TrimSpace(*data.SubCommandName))
	}
	if sub == "" {
		return key
	}

	if group != "" {
		return key + ":" + group + ":" + sub
	}
	return key + ":" + sub
}

func (b *Bot) handleRegisteredSlash(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	locale discord.Locale,
	cmdName string,
) bool {
	cmd, ok := b.commands[cmdName]
	if !ok {
		return false
	}

	action, err := cmd.Handle(ctx, e, t, b.services(locale))
	if err != nil {
		b.logger.ErrorContext(ctx, "command failed", slog.String("cmd", cmdName), slog.String("err", err.Error()))
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
		return true
	}
	if action == nil {
		_ = e.Acknowledge()
		return true
	}
	if execErr := action.Execute(e); execErr != nil {
		b.logger.ErrorContext(
			ctx,
			"command action failed",
			slog.String("cmd", cmdName),
			slog.String("err", execErr.Error()),
		)
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
	}
	return true
}

func (b *Bot) handlePluginSlash(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	locale discord.Locale,
	cmdName string,
	data discord.SlashCommandInteractionData,
) {
	// Plugin commands.
	pluginManager := b.plugins
	if pluginManager == nil {
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
		return
	}
	if _, ok := pluginManager.Commands()[cmdName]; !ok {
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
		return
	}

	res, defaultEphemeral, pluginID, err := pluginManager.HandleSlash(ctx, cmdName, plugins.Payload{
		GuildID:   snowflakePtrToString(e.GuildID()),
		ChannelID: e.Channel().ID().String(),
		UserID:    e.User().ID.String(),
		Locale:    locale.Code(),
		Options:   pluginOptions(data),
	})
	if err != nil {
		b.logger.ErrorContext(
			ctx,
			"plugin command failed",
			slog.String("cmd", cmdName),
			slog.String("err", err.Error()),
		)
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
		return
	}

	action, err := parsePluginAction(pluginID, res, defaultEphemeral, pluginResponseSlash)
	if err != nil {
		b.logger.ErrorContext(
			ctx,
			"plugin response parse failed",
			slog.String("cmd", cmdName),
			slog.String("err", err.Error()),
		)
		_ = e.CreateMessage(b.pluginResponseErrorMessage(t, err))
		return
	}

	b.executePluginActionFromSlash(e, t, action)
}

func (b *Bot) executePluginActionFromSlash(
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	action pluginAction,
) {
	switch action.Kind {
	case pluginActionNone:
		_ = e.Acknowledge()
	case pluginActionModal:
		_ = e.Modal(action.Modal)
	case pluginActionUpdate:
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
	case pluginActionMessage:
		_ = e.CreateMessage(action.Create)
	default:
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
	}
}

func (b *Bot) onComponent(e *events.ComponentInteractionCreate) {
	ctx := context.Background()

	locale := e.Locale()
	t := core.Translator{Registry: b.i18n, Locale: locale, UserID: uint64(e.User().ID)}

	customID := e.Data.CustomID()
	if !b.takeComponentCooldown(e, t, customID, time.Now()) {
		return
	}

	if b.handleUnwarnComponent(ctx, e, t, locale, customID) {
		return
	}

	if b.handleRemindDeleteComponent(ctx, e, t, customID) {
		return
	}

	b.handlePluginComponent(ctx, e, t, locale, customID)
}

func (b *Bot) takeComponentCooldown(
	e *events.ComponentInteractionCreate,
	t core.Translator,
	customID string,
	now time.Time,
) bool {
	if d := b.componentCooldown(customID); d > 0 {
		if remaining, ok := b.cooldowns.Take(uint64(e.User().ID), componentCooldownKey(customID), d, now); !ok {
			msg := interactions.NoticeMessage(
				present.KindWarning,
				"",
				t.S("err.cooldown", map[string]any{"Seconds": cooldownSecs(remaining)}),
				true,
			)
			_ = e.CreateMessage(msg)
			return false
		}
	}
	return true
}

func (b *Bot) handleUnwarnComponent(
	ctx context.Context,
	e *events.ComponentInteractionCreate,
	t core.Translator,
	locale discord.Locale,
	customID string,
) bool {
	if !strings.HasPrefix(customID, "mamusiabtw:unwarn:") {
		return false
	}

	data := e.StringSelectMenuInteractionData()
	action, err := cmdmoderation.HandleUnwarnSelection(ctx, e, t, b.services(locale), customID, data.Values)
	if err != nil {
		b.logger.ErrorContext(
			ctx,
			"component failed",
			slog.String("custom_id", customID),
			slog.String("err", err.Error()),
		)
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
		return true
	}
	if action == nil {
		_ = e.Acknowledge()
		return true
	}
	if execErr := action.Execute(e); execErr != nil {
		b.logger.ErrorContext(
			ctx,
			"component action failed",
			slog.String("custom_id", customID),
			slog.String("err", execErr.Error()),
		)
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
	}
	return true
}

func (b *Bot) handleRemindDeleteComponent(
	ctx context.Context,
	e *events.ComponentInteractionCreate,
	t core.Translator,
	customID string,
) bool {
	if !strings.HasPrefix(customID, "mamusiabtw:reminddel:") {
		return false
	}

	if b.store == nil {
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
		return true
	}

	data := e.StringSelectMenuInteractionData()
	action, err := cmdwellness.HandleRemindDeleteSelection(ctx, e, t, b.store, customID, data.Values)
	if err != nil {
		b.logger.ErrorContext(
			ctx,
			"component failed",
			slog.String("custom_id", customID),
			slog.String("err", err.Error()),
		)
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
		return true
	}
	if action == nil {
		_ = e.Acknowledge()
		return true
	}
	if execErr := action.Execute(e); execErr != nil {
		b.logger.ErrorContext(
			ctx,
			"component action failed",
			slog.String("custom_id", customID),
			slog.String("err", execErr.Error()),
		)
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
	}
	return true
}

func (b *Bot) handlePluginComponent(
	ctx context.Context,
	e *events.ComponentInteractionCreate,
	t core.Translator,
	locale discord.Locale,
	customID string,
) {
	pluginManager := b.plugins
	if pluginManager == nil {
		_ = e.Acknowledge()
		return
	}

	pluginID, localID, ok := plugins.ParseCustomID(customID)
	if !ok {
		_ = e.Acknowledge()
		return
	}

	res, hasValue, err := pluginManager.HandleComponent(ctx, pluginID, localID, plugins.Payload{
		GuildID:   snowflakePtrToString(e.GuildID()),
		ChannelID: e.Channel().ID().String(),
		UserID:    e.User().ID.String(),
		Locale:    locale.Code(),
		Options:   componentOptions(e),
	})
	if err != nil {
		b.logger.ErrorContext(
			ctx,
			"plugin component failed",
			slog.String("custom_id", customID),
			slog.String("err", err.Error()),
		)
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
		return
	}

	if !hasValue {
		_ = e.Acknowledge()
		return
	}

	action, err := parsePluginAction(pluginID, res, false, pluginResponseComponent)
	if err != nil {
		b.logger.ErrorContext(
			ctx,
			"plugin component response parse failed",
			slog.String("custom_id", customID),
			slog.String("err", err.Error()),
		)
		_ = e.CreateMessage(b.pluginResponseErrorMessage(t, err))
		return
	}

	b.executePluginActionFromComponent(e, action)
}

func (b *Bot) executePluginActionFromComponent(e *events.ComponentInteractionCreate, action pluginAction) {
	switch action.Kind {
	case pluginActionNone:
		_ = e.Acknowledge()
	case pluginActionModal:
		_ = e.Modal(action.Modal)
	case pluginActionUpdate:
		_ = e.UpdateMessage(action.Update)
	case pluginActionMessage:
		_ = e.CreateMessage(action.Create)
	default:
		_ = e.Acknowledge()
	}
}

func (b *Bot) onModal(e *events.ModalSubmitInteractionCreate) {
	ctx := context.Background()

	locale := e.Locale()
	t := core.Translator{Registry: b.i18n, Locale: locale, UserID: uint64(e.User().ID)}

	customID := strings.TrimSpace(e.Data.CustomID)
	if d := b.modalCooldown(customID); d > 0 {
		if remaining, ok := b.cooldowns.Take(uint64(e.User().ID), modalCooldownKey(customID), d, time.Now()); !ok {
			_ = e.CreateMessage(
				interactions.NoticeMessage(
					present.KindWarning,
					"",
					t.S("err.cooldown", map[string]any{"Seconds": cooldownSecs(remaining)}),
					true,
				),
			)
			return
		}
	}

	if b.plugins == nil {
		_ = e.Acknowledge()
		return
	}

	pluginID, localID, ok := plugins.ParseCustomID(customID)
	if !ok {
		_ = e.Acknowledge()
		return
	}

	res, hasValue, err := b.plugins.HandleModal(ctx, pluginID, localID, plugins.Payload{
		GuildID:   snowflakePtrToString(e.GuildID()),
		ChannelID: e.Channel().ID().String(),
		UserID:    e.User().ID.String(),
		Locale:    locale.Code(),
		Options:   modalOptions(e, pluginID),
	})
	if err != nil {
		b.logger.Error("plugin modal failed", slog.String("custom_id", customID), slog.String("err", err.Error()))
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
		return
	}

	if !hasValue {
		_ = e.Acknowledge()
		return
	}

	action, err := parsePluginAction(pluginID, res, false, pluginResponseModalSubmit)
	if err != nil {
		b.logger.Error(
			"plugin modal response parse failed",
			slog.String("custom_id", customID),
			slog.String("err", err.Error()),
		)
		_ = e.CreateMessage(b.pluginResponseErrorMessage(t, err))
		return
	}

	switch action.Kind {
	case pluginActionNone:
		_ = e.Acknowledge()
	case pluginActionModal:
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
	case pluginActionUpdate:
		_ = e.UpdateMessage(action.Update)
	case pluginActionMessage:
		_ = e.CreateMessage(action.Create)
	default:
		_ = e.CreateMessage(interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true))
	}
}
