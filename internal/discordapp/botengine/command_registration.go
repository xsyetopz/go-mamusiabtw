package botengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	"github.com/xsuetopz/go-mamusiabtw/internal/discordapp/core"
	"github.com/xsuetopz/go-mamusiabtw/internal/i18n"
)

func (b *Bot) commandCreates(locales []string) []discord.ApplicationCommandCreate {
	const extraCreatesCapacity = 8
	creates := make([]discord.ApplicationCommandCreate, 0, len(b.order)+extraCreatesCapacity)
	for _, cmd := range b.order {
		creates = append(
			creates,
			cmd.CreateCommand(locales, core.Translator{Registry: b.i18n, Locale: discord.LocaleEnglishUS}),
		)
	}
	if b.plugins != nil {
		creates = append(
			creates,
			b.plugins.CommandCreatesWithLocalizations(locales, func(pluginID, locale, messageID string) (string, bool) {
				return b.i18n.TryLocalize(i18n.Config{
					Locale:    locale,
					PluginID:  pluginID,
					MessageID: messageID,
				})
			})...)
	}
	return creates
}

func (b *Bot) setCommandsInGuilds(
	_ context.Context,
	creates []discord.ApplicationCommandCreate,
	guildIDs []uint64,
) error {
	if b == nil || b.client == nil {
		return errors.New("discord client not initialized")
	}
	if len(guildIDs) == 0 {
		return nil
	}

	for _, guildID := range guildIDs {
		if guildID == 0 {
			continue
		}
		_, err := b.client.Rest.SetGuildCommands(b.client.ApplicationID, snowflake.ID(guildID), creates)
		if err != nil {
			return fmt.Errorf("set commands for guild %d: %w", guildID, err)
		}
	}
	return nil
}

func (b *Bot) registerCommandsInCachedGuilds(_ context.Context) error {
	if b == nil || b.client == nil {
		return errors.New("discord client not initialized")
	}

	locales := b.i18n.SupportedLocales()
	creates := b.commandCreates(locales)

	for guild := range b.client.Caches.Guilds() {
		guildID := uint64(guild.ID)
		if guildID == 0 {
			continue
		}
		_, err := b.client.Rest.SetGuildCommands(b.client.ApplicationID, snowflake.ID(guildID), creates)
		if err != nil {
			return fmt.Errorf("set commands for cached guild %d: %w", guildID, err)
		}
	}

	return nil
}

func (b *Bot) onGuildsReady(e *events.GuildsReady) {
	if b == nil || e == nil {
		return
	}
	if b.devGuildID != nil {
		return
	}
	if !b.commandRegisterAllGuilds {
		return
	}

	ctx := context.Background()
	if err := b.registerCommandsInCachedGuilds(ctx); err != nil {
		b.logger.Error("register commands in cached guilds failed", "err", err.Error())
	}
}
