package discordruntime

import (
	"testing"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

func TestResolveOwnerFromApplication(t *testing.T) {
	t.Parallel()

	t.Run("team owner wins", func(t *testing.T) {
		t.Parallel()

		application := &discord.Application{
			Owner: &discord.User{ID: snowflake.ID(11)},
			Team:  &discord.Team{OwnerID: snowflake.ID(22)},
		}

		ownerID, ok := resolveOwnerFromApplication(application)
		if !ok {
			t.Fatalf("expected owner to resolve")
		}
		if ownerID != 22 {
			t.Fatalf("unexpected owner id: %d", ownerID)
		}
	})

	t.Run("personal owner", func(t *testing.T) {
		t.Parallel()

		application := &discord.Application{
			Owner: &discord.User{ID: snowflake.ID(33)},
		}

		ownerID, ok := resolveOwnerFromApplication(application)
		if !ok {
			t.Fatalf("expected owner to resolve")
		}
		if ownerID != 33 {
			t.Fatalf("unexpected owner id: %d", ownerID)
		}
	})

	t.Run("missing owner", func(t *testing.T) {
		t.Parallel()

		ownerID, ok := resolveOwnerFromApplication(&discord.Application{})
		if ok {
			t.Fatalf("expected unresolved owner, got %d", ownerID)
		}
	})
}
