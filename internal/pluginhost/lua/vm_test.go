package luaplugin_test

import (
	"bytes"
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
	"github.com/xsyetopz/go-mamusiabtw/internal/integrations/kawaii"
	"github.com/xsyetopz/go-mamusiabtw/internal/permissions"
	"github.com/xsyetopz/go-mamusiabtw/internal/pluginhost/lua"
	"github.com/xsyetopz/go-mamusiabtw/internal/store/sqlitestore"
)

type fakeKawaii struct {
	url string
}

func (f fakeKawaii) FetchGIF(_ context.Context, endpoint kawaii.Endpoint) (string, error) {
	if f.url == "" {
		return "https://kawaii.red/" + string(endpoint) + ".gif", nil
	}
	return f.url, nil
}

func TestDescriptorRoutesAndKV(t *testing.T) {
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

	reg, err := i18n.LoadCore(filepath.FromSlash("../../../locales"))
	if err != nil {
		t.Fatalf("i18n: %v", err)
	}
	if err = reg.LoadPluginLocales("example", filepath.FromSlash("../../../examples/plugins/example/locales")); err != nil {
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

	script := filepath.FromSlash("../../../examples/plugins/example/plugin.lua")
	vm, err := luaplugin.NewFromFile(script, luaplugin.Options{
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

	definition, ok := vm.Definition()
	if !ok {
		t.Fatalf("expected descriptor definition")
	}
	if len(definition.Commands) != 1 || definition.Commands[0].Name != "example" {
		t.Fatalf("unexpected commands: %#v", definition.Commands)
	}
	if definition.Commands[0].DescriptionID != "cmd.example.desc" {
		t.Fatalf("unexpected command description id: %#v", definition.Commands[0])
	}
	if len(definition.Modals) != 1 || definition.Modals[0] != "set_counter" {
		t.Fatalf("unexpected modal routes: %#v", definition.Modals)
	}

	got, hasValue, err := vm.CallRoute(ctx, luaplugin.RouteCommand, "example", luaplugin.Payload{
		GuildID: "1",
		Locale:  "en-GB",
	})
	if err != nil {
		t.Fatalf("CallRoute(command): %v", err)
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

	got, hasValue, err = vm.CallRoute(ctx, luaplugin.RouteComponent, "inc", luaplugin.Payload{
		GuildID: "1",
		Locale:  "en-GB",
		Options: map[string]any{"type": "button"},
	})
	if err != nil {
		t.Fatalf("CallRoute(component): %v", err)
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

	got, hasValue, err = vm.CallRoute(ctx, luaplugin.RouteModal, "set_counter", luaplugin.Payload{
		GuildID: "1",
		Locale:  "en-GB",
		Options: map[string]any{
			"fields": map[string]any{"value": "5"},
		},
	})
	if err != nil {
		t.Fatalf("CallRoute(modal): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected modal value")
	}
	gotMap, ok = got.(map[string]any)
	if !ok {
		t.Fatalf("expected modal object, got %T", got)
	}
	content, _ = gotMap["content"].(string)
	if !strings.Contains(content, "Counter: 5") {
		t.Fatalf("expected counter=5, got %#v", gotMap)
	}
}

func TestFunPluginRoutes(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{}))

	reg, err := i18n.LoadCore(filepath.FromSlash("../../../locales"))
	if err != nil {
		t.Fatalf("i18n: %v", err)
	}
	if err = reg.LoadPluginLocales("fun", filepath.FromSlash("../../../plugins/fun/locales")); err != nil {
		t.Fatalf("plugin i18n: %v", err)
	}

	script := filepath.FromSlash("../../../plugins/fun/plugin.lua")
	vm, err := luaplugin.NewFromFile(script, luaplugin.Options{
		Logger:    logger,
		PluginID:  "fun",
		PluginDir: filepath.Dir(script),
		Permissions: permissions.Permissions{
			Integrations: permissions.IntegrationsPermissions{Kawaii: true},
		},
		I18n:   &reg,
		Kawaii: fakeKawaii{url: "https://kawaii.red/demo.gif"},
	})
	if err != nil {
		t.Fatalf("NewFromFile(fun): %v", err)
	}
	t.Cleanup(vm.Close)

	definition, ok := vm.Definition()
	if !ok {
		t.Fatalf("expected descriptor definition")
	}
	if len(definition.Commands) != 7 {
		t.Fatalf("unexpected command count: %#v", definition.Commands)
	}

	ctx := context.Background()

	got, hasValue, err := vm.CallRoute(ctx, luaplugin.RouteCommand, "flip", luaplugin.Payload{
		UserID: "42",
		Locale: "en-US",
	})
	if err != nil {
		t.Fatalf("CallRoute(flip): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected flip value")
	}
	flipMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected flip object, got %T", got)
	}
	flipEmbeds, ok := flipMap["embeds"].([]any)
	if !ok || len(flipEmbeds) != 1 {
		t.Fatalf("expected flip embeds, got %#v", flipMap)
	}
	flipEmbed, ok := flipEmbeds[0].(map[string]any)
	if !ok {
		t.Fatalf("expected flip embed object, got %#v", flipEmbeds[0])
	}
	flipDesc, _ := flipEmbed["description"].(string)
	if !strings.Contains(flipDesc, "<@42>") || !strings.Contains(flipDesc, "flipped and got") {
		t.Fatalf("unexpected flip description: %#v", flipEmbed)
	}

	got, hasValue, err = vm.CallRoute(ctx, luaplugin.RouteCommand, "roll", luaplugin.Payload{
		UserID: "42",
		Locale: "en-US",
		Options: map[string]any{
			"number": 2,
			"sides":  6,
		},
	})
	if err != nil {
		t.Fatalf("CallRoute(roll): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected roll value")
	}
	rollMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected roll object, got %T", got)
	}
	rollEmbeds, _ := rollMap["embeds"].([]any)
	rollEmbed, _ := rollEmbeds[0].(map[string]any)
	rollDesc, _ := rollEmbed["description"].(string)
	if !strings.Contains(rollDesc, "rolled `2d6`") {
		t.Fatalf("unexpected roll description: %#v", rollEmbed)
	}
	if _, ok := rollEmbed["footer"].(string); !ok {
		t.Fatalf("expected roll footer: %#v", rollEmbed)
	}

	got, hasValue, err = vm.CallRoute(ctx, luaplugin.RouteCommand, "hug", luaplugin.Payload{
		GuildID: "1",
		UserID:  "42",
		Locale:  "en-US",
		Options: map[string]any{
			"user": "99",
		},
	})
	if err != nil {
		t.Fatalf("CallRoute(hug): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected hug value")
	}
	hugMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected hug object, got %T", got)
	}
	hugEmbeds, _ := hugMap["embeds"].([]any)
	hugEmbed, _ := hugEmbeds[0].(map[string]any)
	hugDesc, _ := hugEmbed["description"].(string)
	hugImage, _ := hugEmbed["image_url"].(string)
	if !strings.Contains(hugDesc, "<@99>") || hugImage != "https://kawaii.red/demo.gif" {
		t.Fatalf("unexpected hug embed: %#v", hugEmbed)
	}

	got, hasValue, err = vm.CallRoute(ctx, luaplugin.RouteCommand, "shrug", luaplugin.Payload{
		GuildID: "1",
		UserID:  "42",
		Locale:  "en-US",
		Options: map[string]any{
			"message": "maybe",
		},
	})
	if err != nil {
		t.Fatalf("CallRoute(shrug): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected shrug value")
	}
	shrugMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected shrug object, got %T", got)
	}
	content, _ := shrugMap["content"].(string)
	shrugEmbeds, _ := shrugMap["embeds"].([]any)
	shrugEmbed, _ := shrugEmbeds[0].(map[string]any)
	if content != "maybe" || shrugEmbed["image_url"] != "https://kawaii.red/demo.gif" {
		t.Fatalf("unexpected shrug payload: %#v", shrugMap)
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
