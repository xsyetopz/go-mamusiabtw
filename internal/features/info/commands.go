package info

import "github.com/xsyetopz/go-mamusiabtw/internal/features/commandapi"

const (
	infoEmbedColor      = 0x5865F2
	infoErrorEmbedColor = 0xED4245
)

func Commands() []commandapi.SlashCommand {
	return []commandapi.SlashCommand{
		about(),
		lookup(),
	}
}
