package discordruntime

import (
	"context"
)

func (b *Bot) Start(ctx context.Context) error {
	if err := b.reloadModules(ctx); err != nil {
		return err
	}

	if err := b.client.OpenGateway(ctx); err != nil {
		return decorateGatewayOpenError(err, requestedGatewayIntentsMask())
	}

	if b.pluginAuto != nil {
		b.pluginAuto.Start(ctx)
	}

	b.startReminderScheduler(ctx)
	b.ready.Store(true)
	return nil
}

func (b *Bot) Close(ctx context.Context) {
	b.ready.Store(false)
	if b.client != nil {
		b.client.Close(ctx)
	}
	if b.pluginAuto != nil {
		b.pluginAuto.Stop()
	}
}

func (b *Bot) registerCommands(ctx context.Context) error {
	return b.commandRegistrar().Register(ctx, b.commandRegistrationMode, b.devGuildID, b.commandGuildIDs)
}
