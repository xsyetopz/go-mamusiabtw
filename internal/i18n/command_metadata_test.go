package i18n_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestCommandMetadataLocalizationRules(t *testing.T) {
	t.Parallel()

	localesDir := filepath.FromSlash("../../locales")
	baseLocale := "en-US"

	basePath := filepath.Join(localesDir, baseLocale, "messages.json")
	baseMessages := mustLoadMessages(t, basePath)
	baseByID := mapByID(t, baseMessages)
	cmdNameIDs, cmdDescIDs := commandNameAndDescIDs(baseByID)

	entries, err := os.ReadDir(localesDir)
	if err != nil {
		t.Fatalf("read locales dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		locale := entry.Name()
		if locale == baseLocale {
			continue
		}

		path := filepath.Join(localesDir, locale, "messages.json")
		if _, statErr := os.Stat(path); statErr != nil {
			continue
		}

		messages := mustLoadMessages(t, path)
		if len(messages) == 0 {
			continue
		}
		gotByID := mapByID(t, messages)

		assertCommandNamesMatchBase(t, locale, baseLocale, baseByID, gotByID, cmdNameIDs)
		if locale != "en-GB" {
			assertCommandDescriptionsLocalized(t, locale, baseLocale, baseByID, gotByID, cmdDescIDs)
		}
	}
}

func commandNameAndDescIDs(baseByID map[string]string) ([]string, []string) {
	var nameIDs []string
	var descIDs []string
	for id := range baseByID {
		if !strings.HasPrefix(id, "cmd.") {
			continue
		}
		if strings.HasSuffix(id, ".name") {
			nameIDs = append(nameIDs, id)
		}
		if strings.HasSuffix(id, ".desc") {
			descIDs = append(descIDs, id)
		}
	}
	sort.Strings(nameIDs)
	sort.Strings(descIDs)
	return nameIDs, descIDs
}

func assertCommandNamesMatchBase(
	t *testing.T,
	locale string,
	baseLocale string,
	baseByID map[string]string,
	gotByID map[string]string,
	nameIDs []string,
) {
	t.Helper()

	for _, id := range nameIDs {
		if gotByID[id] != baseByID[id] {
			t.Fatalf("locale %q: %q must match %q for Discord command/option names", locale, id, baseLocale)
		}
	}
}

func assertCommandDescriptionsLocalized(
	t *testing.T,
	locale string,
	baseLocale string,
	baseByID map[string]string,
	gotByID map[string]string,
	descIDs []string,
) {
	t.Helper()

	for _, id := range descIDs {
		if gotByID[id] == baseByID[id] {
			t.Fatalf("locale %q: %q must be localized (must not equal %q)", locale, id, baseLocale)
		}
	}
}
