package discordplatform

import "testing"

func TestParsePluginActionRejectsActionsForInteractions(t *testing.T) {
	t.Parallel()

	_, err := parsePluginAction("moderation", map[string]any{
		"actions": []any{
			map[string]any{
				"type": "send_dm",
				"message": map[string]any{
					"content": "hi",
				},
			},
		},
	}, false, pluginResponseSlash)
	if err == nil {
		t.Fatal("expected actions to be rejected for interaction responses")
	}
	if got := err.Error(); got != "actions are automation-only" {
		t.Fatalf("unexpected error: %q", got)
	}
}
