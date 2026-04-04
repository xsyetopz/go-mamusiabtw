package router

import (
	"testing"

	"github.com/disgoorg/disgo/discord"
)

func TestParsePluginAutocompleteChoices(t *testing.T) {
	t.Parallel()

	choices, err := ParsePluginAutocompleteChoices("test", []any{
		map[string]any{"name": "alpha", "value": "a"},
		map[string]any{"name": "beta", "value": float64(2)},
		map[string]any{"name": "gamma", "value": 2.5},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(choices) != 3 {
		t.Fatalf("unexpected choice count: got %d want 3", len(choices))
	}
	if _, ok := choices[0].(discord.AutocompleteChoiceString); !ok {
		t.Fatalf("expected first choice to be string, got %T", choices[0])
	}
	if _, ok := choices[1].(discord.AutocompleteChoiceInt); !ok {
		t.Fatalf("expected second choice to be int, got %T", choices[1])
	}
	if _, ok := choices[2].(discord.AutocompleteChoiceFloat); !ok {
		t.Fatalf("expected third choice to be float, got %T", choices[2])
	}
}

func TestParsePluginAutocompleteChoicesFromObject(t *testing.T) {
	t.Parallel()

	choices, err := ParsePluginAutocompleteChoices("test", map[string]any{
		"choices": []any{
			map[string]any{"name": "delta", "value": "d"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(choices) != 1 {
		t.Fatalf("unexpected choice count: got %d want 1", len(choices))
	}
}
