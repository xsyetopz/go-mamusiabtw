package interactions

import (
	"strings"

	"github.com/disgoorg/disgo/discord"

	"github.com/xsyetopz/go-mamusiabtw/internal/present"
)

func NoticeEmbed(kind present.Kind, title string, body string) discord.Embed {
	kind = present.Kind(strings.ToLower(strings.TrimSpace(string(kind))))
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)

	desc := body
	if body != "" {
		if prefix := noticePrefix(kind); prefix != "" {
			desc = prefix + body
		}
	}

	e := discord.Embed{
		Title:       title,
		Description: desc,
		Color:       noticeColor(kind),
	}
	if title == "" {
		e.Title = ""
	}
	if desc == "" {
		e.Description = ""
	}
	return e
}

func NoticeMessage(kind present.Kind, title string, body string, ephemeral bool) discord.MessageCreate {
	msg := discord.MessageCreate{
		Embeds:          []discord.Embed{NoticeEmbed(kind, title, body)},
		AllowedMentions: &discord.AllowedMentions{},
	}
	if ephemeral {
		msg.Flags = discord.MessageFlagEphemeral
	}
	return msg
}

func NoticeUpdate(kind present.Kind, title string, body string) discord.MessageUpdate {
	embeds := []discord.Embed{NoticeEmbed(kind, title, body)}
	return discord.MessageUpdate{
		Embeds:          &embeds,
		AllowedMentions: &discord.AllowedMentions{},
	}
}

func noticeColor(kind present.Kind) int {
	switch kind {
	case present.KindSuccess:
		return noticeColorSuccess
	case present.KindWarning:
		return noticeColorWarning
	case present.KindError:
		return noticeColorError
	case present.KindInfo:
		fallthrough
	default:
		return noticeColorInfo
	}
}

func noticePrefix(kind present.Kind) string {
	switch kind {
	case present.KindSuccess:
		return "🌷 "
	case present.KindInfo:
		return "🫶 "
	case present.KindWarning:
		return "💛 "
	case present.KindError:
		return "😾 "
	default:
		return ""
	}
}

const (
	noticeColorSuccess = 0xA7F3D0
	noticeColorWarning = 0xFDE68A
	noticeColorError   = 0xFCA5A5
	noticeColorInfo    = 0xC4B5FD
)
