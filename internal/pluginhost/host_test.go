package pluginhost

import (
	"testing"

	"github.com/disgoorg/disgo/discord"
)

func TestCommandPermissions_ExpressionsAliases(t *testing.T) {
	t.Parallel()

	got, ok := commandPermissions([]string{"manage_expressions", "create_expressions"})
	if !ok {
		t.Fatalf("expected expression permissions to map")
	}

	want := discord.PermissionManageGuildExpressions | discord.PermissionCreateGuildExpressions
	if got != want {
		t.Fatalf("unexpected permissions: got %v want %v", got, want)
	}
}
