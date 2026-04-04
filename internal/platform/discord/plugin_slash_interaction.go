package discordplatform

import "github.com/disgoorg/disgo/events"

type pluginSlashInteraction struct {
	event    *events.ApplicationCommandInteractionCreate
	deferred bool
}

func (i *pluginSlashInteraction) Defer(ephemeral bool) error {
	if i.event == nil {
		return nil
	}
	if err := i.event.DeferCreateMessage(ephemeral); err != nil {
		return err
	}
	i.deferred = true
	return nil
}

func (i *pluginSlashInteraction) Deferred() bool {
	return i != nil && i.deferred
}
