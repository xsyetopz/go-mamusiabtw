package luaplugin_test

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
	"github.com/xsyetopz/go-mamusiabtw/internal/permissions"
	"github.com/xsyetopz/go-mamusiabtw/internal/pluginhost/lua"
	"github.com/xsyetopz/go-mamusiabtw/internal/store/sqlitestore"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestDescriptorRoutesAndKV(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	if err := initSQLiteSchema(db, "../../../migrations/sqlite/001_init.sql"); err != nil {
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
		Store: store,
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
			Network: permissions.NetworkPermissions{HTTP: true},
		},
		I18n: &reg,
		HTTPClient: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != "https://kawaii.red/api/gif/hug?token=anonymous" &&
					req.URL.String() != "https://kawaii.red/api/gif/pat?token=anonymous" &&
					req.URL.String() != "https://kawaii.red/api/gif/poke?token=anonymous" &&
					req.URL.String() != "https://kawaii.red/api/gif/shrug?token=anonymous" {
					t.Fatalf("unexpected http url: %s", req.URL.String())
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"response":"https://kawaii.red/demo.gif"}`)),
					Request:    req,
				}, nil
			}),
		},
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

func TestWellnessPluginRoutes(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	if err := initSQLiteSchema(db, "../../../migrations/sqlite/001_init.sql", "../../../migrations/sqlite/003_wellness.sql"); err != nil {
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
	if err = reg.LoadPluginLocales("wellness", filepath.FromSlash("../../../plugins/wellness/locales")); err != nil {
		t.Fatalf("plugin i18n: %v", err)
	}

	script := filepath.FromSlash("../../../plugins/wellness/plugin.lua")
	vm, err := luaplugin.NewFromFile(script, luaplugin.Options{
		Logger:    logger,
		PluginID:  "wellness",
		PluginDir: filepath.Dir(script),
		Permissions: permissions.Permissions{
			Storage: permissions.StoragePermissions{
				UserSettings: true,
				CheckIns:     true,
				Reminders:    true,
			},
		},
		Store: store,
		I18n:  &reg,
	})
	if err != nil {
		t.Fatalf("NewFromFile(wellness): %v", err)
	}
	t.Cleanup(vm.Close)

	ctx := context.Background()

	got, hasValue, err := vm.CallRoute(ctx, luaplugin.RouteCommand, "timezone", luaplugin.Payload{
		UserID: "42",
		Locale: "en-US",
		Options: map[string]any{
			"__subcommand": "set",
			"iana":         "Europe/Tallinn",
		},
	})
	if err != nil {
		t.Fatalf("CallRoute(timezone set): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected timezone response")
	}
	timezoneMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected timezone object, got %T", got)
	}
	if content, _ := timezoneMap["content"].(string); !strings.Contains(content, "Europe/Tallinn") {
		t.Fatalf("unexpected timezone response: %#v", timezoneMap)
	}

	got, hasValue, err = vm.CallRoute(ctx, luaplugin.RouteCommand, "checkin", luaplugin.Payload{
		UserID: "42",
		Locale: "en-US",
		Options: map[string]any{
			"mood": 5,
		},
	})
	if err != nil {
		t.Fatalf("CallRoute(checkin): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected checkin response")
	}

	got, hasValue, err = vm.CallRoute(ctx, luaplugin.RouteCommand, "remind", luaplugin.Payload{
		GuildID:   "7",
		ChannelID: "11",
		UserID:    "42",
		Locale:    "en-US",
		Options: map[string]any{
			"__subcommand": "create",
			"schedule":     "0 9 * * *",
			"kind":         "hydrate",
			"delivery":     "dm",
		},
	})
	if err != nil {
		t.Fatalf("CallRoute(remind create): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected remind create response")
	}
	createMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected remind create object, got %T", got)
	}
	createContent, _ := createMap["content"].(string)
	if !strings.Contains(createContent, "hydrate") {
		t.Fatalf("unexpected remind create response: %#v", createMap)
	}

	listed, err := store.Reminders().ListReminders(ctx, 42, 10)
	if err != nil {
		t.Fatalf("ListReminders: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected one reminder, got %d", len(listed))
	}

	got, hasValue, err = vm.CallRoute(ctx, luaplugin.RouteCommand, "remind", luaplugin.Payload{
		UserID: "42",
		Locale: "en-US",
		Options: map[string]any{
			"__subcommand": "delete",
		},
	})
	if err != nil {
		t.Fatalf("CallRoute(remind delete): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected remind delete prompt")
	}
	deleteMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected remind delete object, got %T", got)
	}
	rows, _ := deleteMap["components"].([]any)
	if len(rows) != 1 {
		t.Fatalf("expected one component row, got %#v", deleteMap)
	}

	got, hasValue, err = vm.CallRoute(ctx, luaplugin.RouteComponent, "delete_reminder", luaplugin.Payload{
		UserID: "42",
		Locale: "en-US",
		Options: map[string]any{
			"type":   "string_select",
			"values": []any{listed[0].ID},
		},
	})
	if err != nil {
		t.Fatalf("CallRoute(delete_reminder): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected delete component response")
	}
	deleteResult, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected delete component object, got %T", got)
	}
	if content, _ := deleteResult["content"].(string); !strings.Contains(content, "Deleted") {
		t.Fatalf("unexpected delete component response: %#v", deleteResult)
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

func initSQLiteSchema(db *sql.DB, relPaths ...string) error {
	for _, relPath := range relPaths {
		scriptPath := filepath.FromSlash(relPath)
		bytes, err := os.ReadFile(scriptPath)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(bytes)); err != nil {
			return err
		}
	}
	return nil
}
