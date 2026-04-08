package interactions

import (
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
)

const (
	ThemeColorBrand   = 0x0F62FE // IBM Blue 60
	ThemeColorSuccess = 0x009E73 // Wong bluish green
	ThemeColorWarning = 0xE69F00 // Wong orange
	ThemeColorError   = 0xD55E00 // Wong vermillion
)

func boolPtr(v bool) *bool { return &v }

func Bool(v bool) *bool { return boolPtr(v) }

func MessageEmbeds(embeds []discord.Embed, ephemeral bool) discord.MessageCreate {
	msg := discord.MessageCreate{
		Embeds:          embeds,
		AllowedMentions: &discord.AllowedMentions{},
	}
	if ephemeral {
		msg.Flags = discord.MessageFlagEphemeral
	}
	return msg
}

func Embed(title, description string, color int) discord.Embed {
	return discord.Embed{
		Title:       strings.TrimSpace(title),
		Description: strings.TrimSpace(description),
		Color:       color,
	}
}

func JoinLines(lines []string) string {
	var b strings.Builder
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if b.Len() != 0 {
			b.WriteByte('\n')
		}
		_, _ = fmt.Fprint(&b, line)
		if i == len(lines)-1 {
			// nothing
		}
	}
	return b.String()
}

// EmbedFieldChunked splits "lines" into a sequence of fields whose Value stays under
// Discord's 1024 character field limit. Empty result is valid.
func EmbedFieldChunked(name string, lines []string, inline bool) []discord.EmbedField {
	const limit = 1024
	name = strings.TrimSpace(name)
	if name == "" {
		name = "\u200b"
	}

	out := []discord.EmbedField{}
	cur := []string{}
	curLen := 0

	flush := func() {
		if len(cur) == 0 {
			return
		}
		out = append(out, discord.EmbedField{
			Name:   name,
			Value:  JoinLines(cur),
			Inline: boolPtr(inline),
		})
		cur = nil
		curLen = 0
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		// +1 accounts for newline when joined.
		add := len(line)
		if curLen > 0 {
			add++
		}
		if curLen > 0 && curLen+add > limit {
			flush()
		}
		if len(line) > limit {
			// Single line too long. Keep it safe and still readable.
			line = line[:limit-1] + "…"
			add = len(line)
		}
		cur = append(cur, line)
		curLen += add
	}
	flush()
	return out
}
