package interactions

import (
	"strings"

	"github.com/disgoorg/disgo/discord"
)

func NoticeEmbed(kind Kind, title string, body string) discord.Embed {
	kind = Kind(strings.ToLower(strings.TrimSpace(string(kind))))
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

func NoticeMessage(kind Kind, title string, body string, ephemeral bool) discord.MessageCreate {
	msg := discord.MessageCreate{
		Embeds:          []discord.Embed{NoticeEmbed(kind, title, body)},
		AllowedMentions: &discord.AllowedMentions{},
	}
	if ephemeral {
		msg.Flags = discord.MessageFlagEphemeral
	}
	return msg
}

func NoticeUpdate(kind Kind, title string, body string) discord.MessageUpdate {
	embeds := []discord.Embed{NoticeEmbed(kind, title, body)}
	return discord.MessageUpdate{
		Embeds:          &embeds,
		AllowedMentions: &discord.AllowedMentions{},
	}
}

func noticeColor(kind Kind) int {
	switch kind {
	case KindSuccess:
		return noticeColorSuccess
	case KindWarning:
		return noticeColorWarning
	case KindError:
		return noticeColorError
	case KindInfo:
		fallthrough
	default:
		return noticeColorInfo
	}
}

func noticePrefix(kind Kind) string {
	switch kind {
	case KindSuccess:
		return "🌷 "
	case KindInfo:
		return "🫶 "
	case KindWarning:
		return "💛 "
	case KindError:
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
