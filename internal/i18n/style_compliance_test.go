package i18n_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocaleStyleCompliance(t *testing.T) {
	t.Parallel()

	localesDir := filepath.FromSlash("../../locales")

	heartIDs := map[string]struct{}{
		"ok.pong":                        {},
		"mod.unwarn.success":             {},
		"mgr.slowmode.removed":           {},
		"mgr.stickers.create_success":    {},
		"mgr.stickers.edit_success":      {},
		"mgr.stickers.delete_success":    {},
		"wellness.timezone.set":          {},
		"wellness.timezone.cleared":      {},
		"wellness.checkin.saved":         {},
		"wellness.remind.created":        {},
		"wellness.remind.delete.success": {},
	}

	noTildeIDs := map[string]struct{}{
		"mod.unwarn.placeholder":             {},
		"wellness.remind.delete.placeholder": {},
	}

	entries, err := os.ReadDir(localesDir)
	if err != nil {
		t.Fatalf("read locales dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		locale := entry.Name()
		path := filepath.Join(localesDir, locale, "messages.json")
		if _, statErr := os.Stat(path); statErr != nil {
			continue
		}

		messages := mustLoadMessages(t, path)
		if len(messages) == 0 {
			continue
		}

		byID := mapByID(t, messages)

		assertLocaleHearts(t, locale, byID, heartIDs)
		assertNoTildePrompts(t, locale, byID, noTildeIDs)
		assertNoStackedToneMarks(t, locale, byID)
		assertLocaleTildeGlyphs(t, locale, byID)
	}
}

func assertLocaleHearts(t *testing.T, locale string, byID map[string]string, heartIDs map[string]struct{}) {
	t.Helper()

	wantTilde := "~"
	if locale == "ja" {
		wantTilde = "〜"
	}
	if locale == "zh-CN" || locale == "zh-TW" {
		wantTilde = "～"
	}

	for id, text := range byID {
		if _, ok := heartIDs[id]; ok {
			if strings.Count(text, "❤️") != 1 {
				t.Fatalf("locale %q: %q must include exactly one ❤️", locale, id)
			}
			if !strings.Contains(text, wantTilde) {
				t.Fatalf("locale %q: %q must include a %q tone mark", locale, id, wantTilde)
			}
			continue
		}
		if strings.Contains(text, "❤️") {
			t.Fatalf("locale %q: unexpected ❤️ in %q", locale, id)
		}
	}
}

func assertNoTildePrompts(t *testing.T, locale string, byID map[string]string, noTildeIDs map[string]struct{}) {
	t.Helper()

	for id := range noTildeIDs {
		text, ok := byID[id]
		if !ok {
			continue
		}
		if strings.ContainsAny(text, "~〜～") {
			t.Fatalf("locale %q: %q must not include any tilde tone marks", locale, id)
		}
		if !strings.HasSuffix(text, "...") {
			t.Fatalf("locale %q: %q must end with an ellipsis", locale, id)
		}
	}
}

func assertNoStackedToneMarks(t *testing.T, locale string, byID map[string]string) {
	t.Helper()

	for id, text := range byID {
		if strings.Contains(text, "~~") || strings.Contains(text, "〜〜") || strings.Contains(text, "～～") {
			t.Fatalf("locale %q: %q contains stacked tone marks", locale, id)
		}
	}
}

func assertLocaleTildeGlyphs(t *testing.T, locale string, byID map[string]string) {
	t.Helper()

	for id, text := range byID {
		switch locale {
		case "ja":
			if strings.Contains(text, "~") || strings.Contains(text, "～") {
				t.Fatalf("locale %q: %q must not use ASCII/fullwidth tilde", locale, id)
			}
		case "zh-CN", "zh-TW":
			if strings.Contains(text, "~") || strings.Contains(text, "〜") {
				t.Fatalf("locale %q: %q must not use ASCII/wave tilde", locale, id)
			}
		default:
			if strings.Contains(text, "〜") || strings.Contains(text, "～") {
				t.Fatalf("locale %q: %q must not use wave/fullwidth tilde", locale, id)
			}
		}
	}
}
