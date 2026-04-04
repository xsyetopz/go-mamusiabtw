package interactions

import (
	"errors"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

type SlashAction interface {
	Execute(e *events.ApplicationCommandInteractionCreate) error
}

type SlashFunc func(e *events.ApplicationCommandInteractionCreate) error

func (f SlashFunc) Execute(e *events.ApplicationCommandInteractionCreate) error { return f(e) }

type ComponentAction interface {
	Execute(e *events.ComponentInteractionCreate) error
}

type ComponentFunc func(e *events.ComponentInteractionCreate) error

func (f ComponentFunc) Execute(e *events.ComponentInteractionCreate) error { return f(e) }

type ModalAction interface {
	Execute(e *events.ModalSubmitInteractionCreate) error
}

type ModalFunc func(e *events.ModalSubmitInteractionCreate) error

func (f ModalFunc) Execute(e *events.ModalSubmitInteractionCreate) error { return f(e) }

type SlashDefer struct {
	Ephemeral bool
}

func (a SlashDefer) Execute(e *events.ApplicationCommandInteractionCreate) error {
	return e.DeferCreateMessage(a.Ephemeral)
}

type SlashMessage struct {
	Create discord.MessageCreate
}

func (a SlashMessage) Execute(e *events.ApplicationCommandInteractionCreate) error {
	return e.CreateMessage(sanitizeMessageCreate(a.Create))
}

type SlashModal struct {
	Modal discord.ModalCreate
}

func (a SlashModal) Execute(e *events.ApplicationCommandInteractionCreate) error {
	return e.Modal(a.Modal)
}

type SlashUpdateInteractionResponse struct {
	Update discord.MessageUpdate
}

func (a SlashUpdateInteractionResponse) Execute(e *events.ApplicationCommandInteractionCreate) error {
	if e == nil || e.Client() == nil {
		return errors.New("nil event/client")
	}
	_, err := e.Client().Rest.UpdateInteractionResponse(e.ApplicationID(), e.Token(), sanitizeMessageUpdate(a.Update))
	return err
}

type SlashAcknowledge struct{}

func (a SlashAcknowledge) Execute(e *events.ApplicationCommandInteractionCreate) error {
	return e.Acknowledge()
}

type SlashSequence []SlashAction

func (s SlashSequence) Execute(e *events.ApplicationCommandInteractionCreate) error {
	for _, a := range s {
		if a == nil {
			continue
		}
		if err := a.Execute(e); err != nil {
			return err
		}
	}
	return nil
}

type ComponentAcknowledge struct{}

func (a ComponentAcknowledge) Execute(e *events.ComponentInteractionCreate) error {
	return e.Acknowledge()
}

type ComponentMessage struct {
	Create discord.MessageCreate
}

func (a ComponentMessage) Execute(e *events.ComponentInteractionCreate) error {
	return e.CreateMessage(sanitizeMessageCreate(a.Create))
}

type ComponentUpdate struct {
	Update discord.MessageUpdate
}

func (a ComponentUpdate) Execute(e *events.ComponentInteractionCreate) error {
	return e.UpdateMessage(sanitizeMessageUpdate(a.Update))
}

type ComponentModal struct {
	Modal discord.ModalCreate
}

func (a ComponentModal) Execute(e *events.ComponentInteractionCreate) error { return e.Modal(a.Modal) }

type ComponentSequence []ComponentAction

func (s ComponentSequence) Execute(e *events.ComponentInteractionCreate) error {
	for _, a := range s {
		if a == nil {
			continue
		}
		if err := a.Execute(e); err != nil {
			return err
		}
	}
	return nil
}

type ModalAcknowledge struct{}

func (a ModalAcknowledge) Execute(e *events.ModalSubmitInteractionCreate) error {
	return e.Acknowledge()
}

type ModalMessage struct {
	Create discord.MessageCreate
}

func (a ModalMessage) Execute(e *events.ModalSubmitInteractionCreate) error {
	return e.CreateMessage(sanitizeMessageCreate(a.Create))
}

type ModalUpdate struct {
	Update discord.MessageUpdate
}

func (a ModalUpdate) Execute(e *events.ModalSubmitInteractionCreate) error {
	return e.UpdateMessage(sanitizeMessageUpdate(a.Update))
}

type ModalSequence []ModalAction

func (s ModalSequence) Execute(e *events.ModalSubmitInteractionCreate) error {
	for _, a := range s {
		if a == nil {
			continue
		}
		if err := a.Execute(e); err != nil {
			return err
		}
	}
	return nil
}
