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

func TestCommandToCreate_ByType(t *testing.T) {
	t.Parallel()

	slash := commandToCreate("plugin", Command{
		Type:        CommandTypeSlash,
		Name:        "lookup",
		Description: "Lookup",
		Options: []CommandOption{{
			Name:         "query",
			Type:         "string",
			Description:  "Query",
			Autocomplete: "lookup_query",
			Choices: []OptionChoice{
				{Name: "stale", Value: "stale"},
			},
		}},
	}, nil, nil)

	slashCreate, ok := slash.(discord.SlashCommandCreate)
	if !ok {
		t.Fatalf("expected slash command create, got %T", slash)
	}
	if len(slashCreate.Options) != 1 {
		t.Fatalf("expected slash option to be present")
	}
	stringOpt, ok := slashCreate.Options[0].(discord.ApplicationCommandOptionString)
	if !ok {
		t.Fatalf("expected string option, got %T", slashCreate.Options[0])
	}
	if !stringOpt.Autocomplete {
		t.Fatalf("expected autocomplete to be enabled")
	}
	if len(stringOpt.Choices) != 0 {
		t.Fatalf("expected explicit choices to be cleared when autocomplete is enabled")
	}

	user := commandToCreate("plugin", Command{
		Type: CommandTypeUser,
		Name: "Inspect User",
	}, nil, nil)
	if _, ok := user.(discord.UserCommandCreate); !ok {
		t.Fatalf("expected user command create, got %T", user)
	}

	message := commandToCreate("plugin", Command{
		Type: CommandTypeMessage,
		Name: "Inspect Message",
	}, nil, nil)
	if _, ok := message.(discord.MessageCommandCreate); !ok {
		t.Fatalf("expected message command create, got %T", message)
	}
}
