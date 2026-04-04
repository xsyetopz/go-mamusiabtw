package discordruntime

func toSet(ids []uint64) map[uint64]struct{} {
	out := map[uint64]struct{}{}
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}

func (b *Bot) isOwner(userID uint64) bool {
	_, ok := b.owners[userID]
	return ok
}
