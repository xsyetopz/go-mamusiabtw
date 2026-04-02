package cmdinfo

import "github.com/xsuetopz/go-mamusiabtw/internal/discordapp/core"

const (
	infoEmbedColor      = 0x5865F2
	infoErrorEmbedColor = 0xED4245
)

func Commands() []core.SlashCommand {
	return []core.SlashCommand{
		about(),
		lookup(),
	}
}
