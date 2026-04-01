package discordapp

import "github.com/xsyetopz/jagpda/internal/discordapp/botengine"

type Dependencies = botengine.Dependencies
type Bot = botengine.Bot
type KawaiiConfig = botengine.KawaiiConfig

func New(deps Dependencies) (*Bot, error) {
	return botengine.New(deps)
}
