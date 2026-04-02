package i18n_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xsuetopz/go-mamusiabtw/internal/i18n"
)

func TestLoadPluginLocales_OnlyLoadsSupportedDiscordLocales(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	mustWrite := func(path string, b []byte) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, b, 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	mustWrite(filepath.Join(dir, "en-US", "messages.json"), []byte(`[
  {"id":"ok.en_us","translation":"ok"}
]`))
	mustWrite(filepath.Join(dir, "xx-YY", "messages.json"), []byte(`[
  {"id":"only.invalid","translation":"should_not_load"}
]`))

	r, err := i18n.LoadCore(filepath.Join("..", "..", "locales"))
	if err != nil {
		t.Fatalf("LoadCore: %v", err)
	}

	if loadErr := r.LoadPluginLocales("p1", dir); loadErr != nil {
		t.Fatalf("LoadPluginLocales: %v", loadErr)
	}

	if _, ok := r.TryLocalize(i18n.Config{PluginID: "p1", Locale: "en-US", MessageID: "ok.en_us"}); !ok {
		t.Fatalf("expected en-US plugin message to be loaded")
	}

	if _, ok := r.TryLocalize(i18n.Config{PluginID: "p1", Locale: "xx-YY", MessageID: "only.invalid"}); ok {
		t.Fatalf("expected unsupported locale folder to be ignored")
	}
}
