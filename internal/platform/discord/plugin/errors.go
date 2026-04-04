package plugin

import (
	"strings"

	"github.com/disgoorg/disgo/discord"

	"github.com/xsyetopz/go-mamusiabtw/internal/features/commandapi"
	"github.com/xsyetopz/go-mamusiabtw/internal/platform/discord/interactions"
	"github.com/xsyetopz/go-mamusiabtw/internal/present"
)

func ErrorMessage(prodMode bool, t commandapi.Translator, err error) discord.MessageCreate {
	if prodMode {
		return interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true)
	}

	body := strings.TrimSpace(err.Error())
	if body == "" {
		body = unknownErrText
	}

	return interactions.NoticeMessage(present.KindError, "Plugin response rejected", body, true)
}

const unknownErrText = "UNKNOWN"
