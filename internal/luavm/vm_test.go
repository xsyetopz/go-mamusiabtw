package luavm_test

import (
	"bytes"
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/xsyetopz/jagpda/internal/i18n"
	"github.com/xsyetopz/jagpda/internal/luavm"
	"github.com/xsyetopz/jagpda/internal/permissions"
	"github.com/xsyetopz/jagpda/internal/store/sqlitestore"
)

func TestHandleAndKV(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	if err := initPluginKVSchema(db); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	store, err := sqlitestore.New(db)
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{}))

	reg, err := i18n.LoadCore(filepath.FromSlash("../../locales"))
	if err != nil {
		t.Fatalf("i18n: %v", err)
	}
	if err = reg.LoadPluginLocales("example", filepath.FromSlash("../../plugins/example/locales")); err != nil {
		t.Fatalf("plugin i18n: %v", err)
	}
	s, locErr := reg.Localize(i18n.Config{
		Locale:       "en-GB",
		PluginID:     "example",
		MessageID:    "example.counter",
		TemplateData: map[string]any{"Count": 1},
	})
	if locErr != nil || !strings.Contains(s, "Counter") {
		t.Fatalf("plugin localize failed: %q (%v)", s, locErr)
	}

	script := filepath.FromSlash("../../plugins/example/plugin.lua")
	vm, err := luavm.NewFromFile(script, luavm.Options{
		Logger:    logger,
		PluginID:  "example",
		PluginDir: filepath.Dir(script),
		Permissions: permissions.Permissions{
			Storage: permissions.StoragePermissions{KV: true},
		},
		Store: store.PluginKV(),
		I18n:  &reg,
	})
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	t.Cleanup(vm.Close)

	ctx := context.Background()

	got, hasValue, err := vm.CallHandle(ctx, "Handle", "example", luavm.Payload{
		GuildID: "1",
		Locale:  "en-GB",
	})
	if err != nil {
		t.Fatalf("CallHandle: %v", err)
	}
	if !hasValue {
		t.Fatalf("expected a value")
	}
	gotMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected object, got %T", got)
	}
	content, _ := gotMap["content"].(string)
	if !strings.Contains(content, "Counter: 1") {
		t.Fatalf("expected counter=1, got %#v", gotMap)
	}

	got, hasValue, err = vm.CallHandle(ctx, "Handle", "example", luavm.Payload{
		GuildID: "1",
		Locale:  "en-GB",
	})
	if err != nil {
		t.Fatalf("CallHandle(2): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected a value")
	}
	gotMap, ok = got.(map[string]any)
	if !ok {
		t.Fatalf("expected object, got %T", got)
	}
	content, _ = gotMap["content"].(string)
	if !strings.Contains(content, "Counter: 2") {
		t.Fatalf("expected counter=2, got %#v", gotMap)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

func initPluginKVSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS plugin_kv (
    guild_id INTEGER NOT NULL,
    plugin_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value_json TEXT NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (guild_id, plugin_id, key)
);`)
	return err
}
