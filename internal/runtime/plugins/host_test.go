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

func TestCommandToCreate_NormalizesRequiredOptionsFirst(t *testing.T) {
	t.Parallel()

	createAny := commandToCreate("plugin", Command{
		Type:        CommandTypeSlash,
		Name:        "stickers",
		Description: "Manage stickers",
		Subcommands: []Subcommand{{
			Name:        "create",
			Description: "Create",
			Options: []CommandOption{
				// Bad order on purpose: optional first, required later.
				{Name: "description", Type: "string", Description: "Sticker description", Required: false},
				{Name: "name", Type: "string", Description: "Sticker name", Required: true},
				{Name: "emoji_tag", Type: "string", Description: "Emoji tag", Required: true},
			},
		}},
	}, nil, nil)

	create, ok := createAny.(discord.SlashCommandCreate)
	if !ok {
		t.Fatalf("expected slash create, got %T", createAny)
	}
	if len(create.Options) != 1 {
		t.Fatalf("expected 1 top-level option, got %d", len(create.Options))
	}

	sub, ok := create.Options[0].(discord.ApplicationCommandOptionSubCommand)
	if !ok {
		t.Fatalf("expected subcommand option, got %T", create.Options[0])
	}
	if len(sub.Options) != 3 {
		t.Fatalf("expected 3 sub options, got %d", len(sub.Options))
	}

	first, ok := sub.Options[0].(discord.ApplicationCommandOptionString)
	if !ok {
		t.Fatalf("expected first option to be string, got %T", sub.Options[0])
	}
	if !first.Required || first.Name != "name" {
		t.Fatalf("expected first option to be required name, got name=%q required=%v", first.Name, first.Required)
	}

	second, ok := sub.Options[1].(discord.ApplicationCommandOptionString)
	if !ok {
		t.Fatalf("expected second option to be string, got %T", sub.Options[1])
	}
	if !second.Required || second.Name != "emoji_tag" {
		t.Fatalf("expected second option to be required emoji_tag, got name=%q required=%v", second.Name, second.Required)
	}

	third, ok := sub.Options[2].(discord.ApplicationCommandOptionString)
	if !ok {
		t.Fatalf("expected third option to be string, got %T", sub.Options[2])
	}
	if third.Required || third.Name != "description" {
		t.Fatalf("expected third option to be optional description, got name=%q required=%v", third.Name, third.Required)
	}
}
