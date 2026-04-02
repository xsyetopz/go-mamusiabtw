package discordapp

import "github.com/xsuetopz/go-mamusiabtw/internal/discordapp/botengine"

type Dependencies = botengine.Dependencies
type Bot = botengine.Bot
type KawaiiConfig = botengine.KawaiiConfig

func New(deps Dependencies) (*Bot, error) {
	return botengine.New(deps)
}
