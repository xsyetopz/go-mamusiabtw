package interactions

import "github.com/disgoorg/disgo/discord"

func sanitizeMessageCreate(m discord.MessageCreate) discord.MessageCreate {
	if m.AllowedMentions == nil {
		m.AllowedMentions = &discord.AllowedMentions{}
	}
	return m
}

func sanitizeMessageUpdate(m discord.MessageUpdate) discord.MessageUpdate {
	if m.AllowedMentions == nil {
		m.AllowedMentions = &discord.AllowedMentions{}
	}
	return m
}
