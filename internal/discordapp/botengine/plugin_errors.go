package botengine

import (
	"strings"

	"github.com/disgoorg/disgo/discord"

	"github.com/xsuetopz/go-mamusiabtw/internal/discordapp/core"
	"github.com/xsuetopz/go-mamusiabtw/internal/discordapp/interactions"
	"github.com/xsuetopz/go-mamusiabtw/internal/present"
)

func (b *Bot) pluginResponseErrorMessage(t core.Translator, err error) discord.MessageCreate {
	if b.prodMode {
		return interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true)
	}

	body := strings.TrimSpace(err.Error())
	if body == "" {
		body = unknownErrText
	}

	return interactions.NoticeMessage(present.KindError, "Plugin response rejected", body, true)
}

const unknownErrText = "UNKNOWN"
