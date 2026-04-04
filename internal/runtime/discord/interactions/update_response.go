package interactions

import (
	"strings"

	"github.com/disgoorg/disgo/events"
)

func UpdateInteractionSuccess(e *events.ApplicationCommandInteractionCreate, desc string) error {
	desc = strings.TrimSpace(desc)
	_, _ = e.Client().Rest.UpdateInteractionResponse(
		e.ApplicationID(),
		e.Token(),
		NoticeUpdate(KindSuccess, "", desc),
	)
	return nil
}

func UpdateInteractionError(e *events.ApplicationCommandInteractionCreate, desc string) error {
	desc = strings.TrimSpace(desc)
	_, _ = e.Client().Rest.UpdateInteractionResponse(
		e.ApplicationID(),
		e.Token(),
		NoticeUpdate(KindError, "", desc),
	)
	return nil
}
