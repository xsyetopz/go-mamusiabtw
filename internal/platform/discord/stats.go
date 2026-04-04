package discordplatform

import "github.com/xsyetopz/go-mamusiabtw/internal/platform/discord/catalog"

type Stats = catalog.Stats

func (b *Bot) Stats() Stats {
	stats, _ := b.stats.Load().(Stats)
	stats.Ready = b.ready.Load()
	return stats
}
