package shared

import (
	"strings"

	"github.com/disgoorg/disgo/events"

	"github.com/xsyetopz/jagpda/internal/discordapp/interactions"
	"github.com/xsyetopz/jagpda/internal/present"
)

func UpdateInteractionSuccess(e *events.ApplicationCommandInteractionCreate, desc string) error {
	desc = strings.TrimSpace(desc)
	_, _ = e.Client().Rest.UpdateInteractionResponse(
		e.ApplicationID(),
		e.Token(),
		interactions.NoticeUpdate(present.KindSuccess, "", desc),
	)
	return nil
}

func UpdateInteractionError(e *events.ApplicationCommandInteractionCreate, desc string) error {
	desc = strings.TrimSpace(desc)
	_, _ = e.Client().Rest.UpdateInteractionResponse(
		e.ApplicationID(),
		e.Token(),
		interactions.NoticeUpdate(present.KindError, "", desc),
	)
	return nil
}
