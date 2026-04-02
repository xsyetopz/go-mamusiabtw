package botengine

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	"github.com/xsyetopz/imotherbtw/internal/plugins"
	"github.com/xsyetopz/imotherbtw/internal/store"
)

func (b *Bot) onGuildJoin(e *events.GuildJoin) {
	if b == nil || e == nil || b.store == nil {
		return
	}

	ctx := context.Background()
	guildID := uint64(e.GuildID)

	restrictions := b.store.Restrictions()
	if _, ok, err := restrictions.GetRestriction(ctx, store.TargetTypeGuild, guildID); err != nil {
		b.logger.Error(
			"guild restriction check failed",
			slog.String("err", err.Error()),
			slog.Uint64("guild_id", guildID),
		)
	} else if ok {
		b.logger.Warn("leaving restricted guild", slog.Uint64("guild_id", guildID))
		_ = e.Client().Rest.LeaveGuild(snowflake.ID(guildID))
		return
	}

	now := time.Now().UTC()

	guildName := strings.TrimSpace(e.Guild.Name)
	ownerID := uint64(e.Guild.OwnerID)

	_ = b.store.Guilds().UpsertGuildSeen(ctx, store.GuildSeen{
		GuildID:   guildID,
		OwnerID:   ownerID,
		CreatedAt: e.Guild.ID.Time().UTC(),
		JoinedAt:  now,
		LeftAt:    nil,
		Name:      guildName,
		UpdatedAt: now,
	})

	if ownerID != 0 {
		owner, _ := e.Client().Rest.GetUser(snowflake.ID(ownerID))
		isBot := false
		isSystem := false
		if owner != nil {
			isBot = owner.Bot
			isSystem = owner.System
		}

		_ = b.store.Users().UpsertUserSeen(ctx, store.UserSeen{
			UserID:      ownerID,
			CreatedAt:   snowflake.ID(ownerID).Time().UTC(),
			IsBot:       isBot,
			IsSystem:    isSystem,
			FirstSeenAt: now,
			LastSeenAt:  now,
		})
	}

	b.logger.Info(
		"joined guild",
		slog.Uint64("guild_id", guildID),
		slog.String("guild_name", guildName),
		slog.Uint64("owner_id", ownerID),
	)

	if b.devGuildID == nil && b.commandRegisterAllGuilds {
		locales := b.i18n.SupportedLocales()
		creates := b.commandCreates(locales)
		if _, err := b.client.Rest.SetGuildCommands(
			b.client.ApplicationID,
			snowflake.ID(guildID),
			creates,
		); err != nil {
			b.logger.Error(
				"register commands in joined guild failed",
				slog.String("err", err.Error()),
				slog.Uint64("guild_id", guildID),
			)
		}
	}
}

func (b *Bot) onGuildLeave(e *events.GuildLeave) {
	if b == nil || e == nil || b.store == nil {
		return
	}
	ctx := context.Background()
	now := time.Now().UTC()
	guildID := uint64(e.GuildID)

	_ = b.store.Guilds().MarkGuildLeft(ctx, guildID, now)

	guildName := strings.TrimSpace(e.Guild.Name)
	b.logger.Info("left guild", slog.Uint64("guild_id", guildID), slog.String("guild_name", guildName))
}

func (b *Bot) onGuildUpdate(e *events.GuildUpdate) {
	if b == nil || e == nil || b.store == nil {
		return
	}

	ctx := context.Background()
	guildID := uint64(e.GuildID)
	newOwner := uint64(e.Guild.OwnerID)
	now := time.Now().UTC()

	guildName := strings.TrimSpace(e.Guild.Name)
	_ = b.store.Guilds().UpsertGuildSeen(ctx, store.GuildSeen{
		GuildID:   guildID,
		OwnerID:   newOwner,
		CreatedAt: e.Guild.ID.Time().UTC(),
		JoinedAt:  now,
		LeftAt:    nil,
		Name:      guildName,
		UpdatedAt: now,
	})

	b.logger.Info(
		"guild updated",
		slog.Uint64("guild_id", guildID),
		slog.String("guild_name", guildName),
		slog.Uint64("owner_id", newOwner),
	)
}

func (b *Bot) onGuildMemberJoin(e *events.GuildMemberJoin) {
	if b == nil || e == nil || b.store == nil {
		return
	}

	user := e.Member.User
	if user.ID == 0 || user.Bot || user.System {
		return
	}

	ctx := context.Background()
	now := time.Now().UTC()
	guildID := uint64(e.GuildID)
	userID := uint64(user.ID)

	_ = b.store.Users().UpsertUserSeen(ctx, store.UserSeen{
		UserID:      userID,
		CreatedAt:   user.ID.Time().UTC(),
		IsBot:       user.Bot,
		IsSystem:    user.System,
		FirstSeenAt: now,
		LastSeenAt:  now,
	})
	_ = b.store.GuildMembers().MarkMemberJoined(ctx, guildID, userID, now)

	b.logger.Info(
		"member joined",
		slog.Uint64("guild_id", guildID),
		slog.Uint64("user_id", userID),
		slog.String("username", strings.TrimSpace(user.Username)),
	)

	if b.pluginAuto != nil {
		b.pluginAuto.FireEvent(pluginEventMemberJoin, plugins.Payload{
			GuildID:   snowflake.ID(guildID).String(),
			ChannelID: "",
			UserID:    user.ID.String(),
			Locale:    "",
			Options:   map[string]any{},
		})
	}
}

func (b *Bot) onGuildMemberLeave(e *events.GuildMemberLeave) {
	if b == nil || e == nil || b.store == nil {
		return
	}

	user := e.User
	if user.ID == 0 || user.Bot || user.System {
		return
	}

	ctx := context.Background()
	now := time.Now().UTC()
	guildID := uint64(e.GuildID)
	userID := uint64(user.ID)

	_ = b.store.GuildMembers().MarkMemberLeft(ctx, guildID, userID, now)
	_ = b.store.Users().TouchUserSeen(ctx, userID, now)

	b.logger.Info(
		"member left",
		slog.Uint64("guild_id", guildID),
		slog.Uint64("user_id", userID),
		slog.String("username", strings.TrimSpace(user.Username)),
	)

	if b.pluginAuto != nil {
		b.pluginAuto.FireEvent(pluginEventMemberLeave, plugins.Payload{
			GuildID:   snowflake.ID(guildID).String(),
			ChannelID: "",
			UserID:    user.ID.String(),
			Locale:    "",
			Options:   map[string]any{},
		})
	}
}

func (b *Bot) onGuildBan(e *events.GuildBan) {
	if b == nil || e == nil {
		return
	}
	b.logger.Info(
		"user banned",
		slog.Uint64("guild_id", uint64(e.GuildID)),
		slog.Uint64("user_id", uint64(e.User.ID)),
		slog.String("username", strings.TrimSpace(e.User.Username)),
	)

	if b.pluginAuto != nil {
		b.pluginAuto.FireEvent(pluginEventGuildBan, plugins.Payload{
			GuildID:   e.GuildID.String(),
			ChannelID: "",
			UserID:    e.User.ID.String(),
			Locale:    "",
			Options:   map[string]any{},
		})
	}
}

func (b *Bot) onGuildUnban(e *events.GuildUnban) {
	if b == nil || e == nil {
		return
	}
	b.logger.Info(
		"user unbanned",
		slog.Uint64("guild_id", uint64(e.GuildID)),
		slog.Uint64("user_id", uint64(e.User.ID)),
		slog.String("username", strings.TrimSpace(e.User.Username)),
	)

	if b.pluginAuto != nil {
		b.pluginAuto.FireEvent(pluginEventGuildUnban, plugins.Payload{
			GuildID:   e.GuildID.String(),
			ChannelID: "",
			UserID:    e.User.ID.String(),
			Locale:    "",
			Options:   map[string]any{},
		})
	}
}

func (b *Bot) onGuildChannelCreate(e *events.GuildChannelCreate) {
	if b == nil || e == nil {
		return
	}
	b.logger.Info(
		"channel created",
		slog.Uint64("guild_id", uint64(e.GuildID)),
		slog.Uint64("channel_id", uint64(e.ChannelID)),
		slog.String("channel_name", strings.TrimSpace(e.Channel.Name())),
	)
}

func (b *Bot) onGuildChannelDelete(e *events.GuildChannelDelete) {
	if b == nil || e == nil {
		return
	}
	b.logger.Info(
		"channel deleted",
		slog.Uint64("guild_id", uint64(e.GuildID)),
		slog.Uint64("channel_id", uint64(e.ChannelID)),
		slog.String("channel_name", strings.TrimSpace(e.Channel.Name())),
	)
}

func (b *Bot) onRoleCreate(e *events.RoleCreate) {
	if b == nil || e == nil {
		return
	}
	b.logger.Info(
		"role created",
		slog.Uint64("guild_id", uint64(e.GuildID)),
		slog.Uint64("role_id", uint64(e.RoleID)),
		slog.String("role_name", strings.TrimSpace(e.Role.Name)),
	)
}

func (b *Bot) onRoleDelete(e *events.RoleDelete) {
	if b == nil || e == nil {
		return
	}
	b.logger.Info(
		"role deleted",
		slog.Uint64("guild_id", uint64(e.GuildID)),
		slog.Uint64("role_id", uint64(e.RoleID)),
		slog.String("role_name", strings.TrimSpace(e.Role.Name)),
	)
}

func (b *Bot) onInviteCreate(e *events.InviteCreate) {
	if b == nil || e == nil {
		return
	}

	guildID := uint64(0)
	if e.GuildID != nil {
		guildID = uint64(*e.GuildID)
	}
	inviterID := uint64(0)
	inviterName := ""
	if e.Inviter != nil {
		inviterID = uint64(e.Inviter.ID)
		inviterName = strings.TrimSpace(e.Inviter.Username)
	}

	b.logger.Info(
		"invite created",
		slog.Uint64("guild_id", guildID),
		slog.Uint64("channel_id", uint64(e.ChannelID)),
		slog.String("code", strings.TrimSpace(e.Code)),
		slog.Uint64("inviter_id", inviterID),
		slog.String("inviter_name", inviterName),
	)
}

func (b *Bot) onInviteDelete(e *events.InviteDelete) {
	if b == nil || e == nil {
		return
	}

	guildID := uint64(0)
	if e.GuildID != nil {
		guildID = uint64(*e.GuildID)
	}

	b.logger.Info(
		"invite deleted",
		slog.Uint64("guild_id", guildID),
		slog.Uint64("channel_id", uint64(e.ChannelID)),
		slog.String("code", strings.TrimSpace(e.Code)),
	)
}
