package i18n_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"text/template"
)

type localeMessage struct {
	ID          string `json:"id"`
	Translation string `json:"translation"`
}

func TestLocaleParityAndTemplateFields(t *testing.T) {
	t.Parallel()

	localesDir := filepath.FromSlash("../../locales")
	baseLocale := "en-US"

	basePath := filepath.Join(localesDir, baseLocale, "messages.json")
	baseMessages := mustLoadMessages(t, basePath)

	baseByID := mapByID(t, baseMessages)

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
		if locale == baseLocale {
			continue
		}
		if len(messages) == 0 {
			continue
		}
		gotByID := mapByID(t, messages)

		assertSameMessageIDs(t, baseLocale, baseByID, locale, gotByID)
		assertTemplateFieldsMatch(t, baseLocale, baseByID, locale, gotByID)
	}
}

func mustLoadMessages(t *testing.T, path string) []localeMessage {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}

	var msgs []localeMessage
	if err = json.Unmarshal(b, &msgs); err != nil {
		t.Fatalf("unmarshal %q: %v", path, err)
	}

	for i := range msgs {
		msgs[i].ID = strings.TrimSpace(msgs[i].ID)
		msgs[i].Translation = strings.TrimSpace(msgs[i].Translation)
		if msgs[i].ID == "" {
			t.Fatalf("empty message id in %q at index %d", path, i)
		}
		if msgs[i].Translation == "" {
			t.Fatalf("empty translation for %q in %q", msgs[i].ID, path)
		}
		if strings.HasPrefix(msgs[i].ID, "cmd.") &&
			(strings.HasSuffix(msgs[i].ID, ".name") || strings.HasSuffix(msgs[i].ID, ".desc")) {
			if strings.Contains(msgs[i].Translation, "{{") {
				t.Fatalf("command metadata must not be templated: %q in %q", msgs[i].ID, path)
			}
		}
	}
	return msgs
}

func mapByID(t *testing.T, msgs []localeMessage) map[string]string {
	t.Helper()
	m := make(map[string]string, len(msgs))
	for _, msg := range msgs {
		if _, exists := m[msg.ID]; exists {
			t.Fatalf("duplicate message id %q", msg.ID)
		}
		m[msg.ID] = msg.Translation
	}
	return m
}

func assertSameMessageIDs(
	t *testing.T,
	baseLocale string,
	base map[string]string,
	locale string,
	got map[string]string,
) {
	t.Helper()

	var missing []string
	for id := range base {
		if _, ok := got[id]; !ok {
			missing = append(missing, id)
		}
	}
	var extra []string
	for id := range got {
		if _, ok := base[id]; !ok {
			extra = append(extra, id)
		}
	}

	sort.Strings(missing)
	sort.Strings(extra)

	if len(missing) == 0 && len(extra) == 0 {
		return
	}

	const maxList = 12
	if len(missing) > maxList {
		missing = append(missing[:maxList], fmt.Sprintf("... (+%d more)", len(missing)-maxList))
	}
	if len(extra) > maxList {
		extra = append(extra[:maxList], fmt.Sprintf("... (+%d more)", len(extra)-maxList))
	}

	t.Fatalf(
		"locale %q message IDs differ from %q: missing=%v extra=%v",
		locale,
		baseLocale,
		missing,
		extra,
	)
}

func assertTemplateFieldsMatch(
	t *testing.T,
	baseLocale string,
	base map[string]string,
	locale string,
	got map[string]string,
) {
	t.Helper()

	allowedExtra := map[string]struct{}{
		"Mommy": {},
		"Pet":   {},
	}
	allowedMissing := allowedExtra

	for id, baseText := range base {
		gotText, ok := got[id]
		if !ok {
			continue
		}

		baseFields, err := templateFields(baseText)
		if err != nil {
			t.Fatalf("parse %s/%s: %v", baseLocale, id, err)
		}
		gotFields, err := templateFields(gotText)
		if err != nil {
			t.Fatalf("parse %s/%s: %v", locale, id, err)
		}

		if missing := missingFields(baseFields, gotFields, allowedMissing); len(missing) > 0 {
			t.Fatalf(
				"missing template placeholders for %s (base=%s locale=%s): missing=%v base=%v got=%v",
				id,
				baseLocale,
				locale,
				missing,
				setToSortedSlice(baseFields),
				setToSortedSlice(gotFields),
			)
		}
		if extra := extraFields(gotFields, baseFields, allowedExtra); len(extra) > 0 {
			t.Fatalf(
				"unexpected template placeholders for %s (base=%s locale=%s): extra=%v base=%v got=%v",
				id,
				baseLocale,
				locale,
				extra,
				setToSortedSlice(baseFields),
				setToSortedSlice(gotFields),
			)
		}
	}
}

func missingFields(base, got, allowedMissing map[string]struct{}) []string {
	var missing []string
	for k := range base {
		if _, ok := got[k]; ok {
			continue
		}
		if _, ok := allowedMissing[k]; ok {
			continue
		}
		missing = append(missing, k)
	}
	sort.Strings(missing)
	return missing
}

func templateFields(s string) (map[string]struct{}, error) {
	// Parse once to ensure the string is a valid Go template.
	if _, err := template.New("msg").Option("missingkey=error").Parse(s); err != nil {
		return nil, err
	}

	// Extract placeholder field paths from within template actions.
	//
	// This keeps the check lightweight (and lint-friendly) while still catching
	// accidental placeholder additions like `{{.Foo}}`.
	actionRe := regexp.MustCompile(`\{\{[^}]*\}\}`)
	fieldRe := regexp.MustCompile(`\.[A-Za-z][A-Za-z0-9_]*(?:\.[A-Za-z][A-Za-z0-9_]*)*`)

	fields := map[string]struct{}{}
	for _, action := range actionRe.FindAllString(s, -1) {
		for _, match := range fieldRe.FindAllString(action, -1) {
			fields[strings.TrimPrefix(match, ".")] = struct{}{}
		}
	}
	return fields, nil
}

func extraFields(got, base, allowedExtra map[string]struct{}) []string {
	var extra []string
	for k := range got {
		if _, ok := base[k]; ok {
			continue
		}
		if _, ok := allowedExtra[k]; ok {
			continue
		}
		extra = append(extra, k)
	}
	sort.Strings(extra)
	return extra
}

func setToSortedSlice(s map[string]struct{}) []string {
	out := make([]string, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
