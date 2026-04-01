package botengine

import (
	"strings"

	"github.com/disgoorg/disgo/discord"

	"github.com/xsyetopz/jagpda/internal/discordapp/core"
	"github.com/xsyetopz/jagpda/internal/discordapp/interactions"
	"github.com/xsyetopz/jagpda/internal/present"
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
