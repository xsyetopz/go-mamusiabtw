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
	"strconv"
	"strings"
	"testing"
	"time"

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

type fakeDiscordExecutor struct {
	sendDMErr        error
	sendChannelErr   error
	timeoutErr       error
	sendDMCalls      int
	sendChannelCalls int
	timeoutCalls     int
	lastChannel      uint64
	lastGuild        uint64
	lastUser         uint64
	lastUntil        time.Time
	lastMessage      any
}

func (f *fakeDiscordExecutor) TimeoutMember(_ context.Context, guildID, userID uint64, until time.Time) error {
	f.timeoutCalls++
	f.lastGuild = guildID
	f.lastUser = userID
	f.lastUntil = until
	return f.timeoutErr
}

func (f *fakeDiscordExecutor) SendDM(_ context.Context, _ string, userID uint64, message any) (luaplugin.MessageResult, error) {
	f.sendDMCalls++
	f.lastUser = userID
	f.lastMessage = message
	if f.sendDMErr != nil {
		return luaplugin.MessageResult{}, f.sendDMErr
	}
	return luaplugin.MessageResult{
		MessageID: 101,
		ChannelID: 202,
		UserID:    userID,
	}, nil
}

func (f *fakeDiscordExecutor) SendChannel(_ context.Context, _ string, channelID uint64, message any) (luaplugin.MessageResult, error) {
	f.sendChannelCalls++
	f.lastChannel = channelID
	f.lastMessage = message
	if f.sendChannelErr != nil {
		return luaplugin.MessageResult{}, f.sendChannelErr
	}
	return luaplugin.MessageResult{
		MessageID: 303,
		ChannelID: channelID,
	}, nil
}

func (f *fakeDiscordExecutor) SetSlowmode(context.Context, uint64, int) error { return nil }

func (f *fakeDiscordExecutor) SetNickname(context.Context, uint64, uint64, *string) error { return nil }

func (f *fakeDiscordExecutor) CreateRole(context.Context, luaplugin.RoleCreateSpec) (luaplugin.RoleResult, error) {
	return luaplugin.RoleResult{}, nil
}

func (f *fakeDiscordExecutor) EditRole(context.Context, luaplugin.RoleEditSpec) (luaplugin.RoleResult, error) {
	return luaplugin.RoleResult{}, nil
}

func (f *fakeDiscordExecutor) DeleteRole(context.Context, uint64, uint64) error { return nil }

func (f *fakeDiscordExecutor) AddRole(context.Context, luaplugin.RoleMemberSpec) error { return nil }

func (f *fakeDiscordExecutor) RemoveRole(context.Context, luaplugin.RoleMemberSpec) error { return nil }

func (f *fakeDiscordExecutor) ListMessages(context.Context, luaplugin.MessageListSpec) ([]luaplugin.MessageInfo, error) {
	return nil, nil
}

func (f *fakeDiscordExecutor) DeleteMessage(context.Context, luaplugin.MessageDeleteSpec) error {
	return nil
}

func (f *fakeDiscordExecutor) BulkDeleteMessages(context.Context, uint64, []uint64) (int, error) {
	return 0, nil
}

func (f *fakeDiscordExecutor) PurgeMessages(context.Context, luaplugin.PurgeSpec) (int, error) {
	return 0, nil
}

func (f *fakeDiscordExecutor) CreateEmoji(context.Context, luaplugin.EmojiCreateSpec) (luaplugin.EmojiResult, error) {
	return luaplugin.EmojiResult{}, nil
}

func (f *fakeDiscordExecutor) EditEmoji(context.Context, luaplugin.EmojiEditSpec) (luaplugin.EmojiResult, error) {
	return luaplugin.EmojiResult{}, nil
}

func (f *fakeDiscordExecutor) DeleteEmoji(context.Context, luaplugin.EmojiDeleteSpec) error {
	return nil
}

func (f *fakeDiscordExecutor) CreateSticker(context.Context, luaplugin.StickerCreateSpec) (luaplugin.StickerResult, error) {
	return luaplugin.StickerResult{}, nil
}

func (f *fakeDiscordExecutor) EditSticker(context.Context, luaplugin.StickerEditSpec) (luaplugin.StickerResult, error) {
	return luaplugin.StickerResult{}, nil
}

func (f *fakeDiscordExecutor) DeleteSticker(context.Context, luaplugin.StickerDeleteSpec) error {
	return nil
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

func TestModerationPluginRoutes(t *testing.T) {
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
	if err = reg.LoadPluginLocales("moderation", filepath.FromSlash("../../../plugins/moderation/locales")); err != nil {
		t.Fatalf("plugin i18n: %v", err)
	}

	script := filepath.FromSlash("../../../plugins/moderation/plugin.lua")
	discordExecutor := &fakeDiscordExecutor{}
	vm, err := luaplugin.NewFromFile(script, luaplugin.Options{
		Logger:    logger,
		PluginID:  "moderation",
		PluginDir: filepath.Dir(script),
		Permissions: permissions.Permissions{
			Storage: permissions.StoragePermissions{
				Warnings: true,
				Audit:    true,
			},
			Discord: permissions.DiscordPermissions{
				SendDM:        true,
				TimeoutMember: true,
			},
		},
		Discord: discordExecutor,
		Store:   store,
		I18n:    &reg,
	})
	if err != nil {
		t.Fatalf("NewFromFile(moderation): %v", err)
	}
	t.Cleanup(vm.Close)

	definition, ok := vm.Definition()
	if !ok {
		t.Fatalf("expected descriptor definition")
	}
	if len(definition.Commands) != 2 {
		t.Fatalf("unexpected command count: %#v", definition.Commands)
	}
	if got := definition.Commands[0].DefaultMemberPermissions; len(got) == 0 {
		t.Fatalf("expected default member permissions on moderation commands")
	}

	ctx := context.Background()
	baseOptions := map[string]any{
		"user":   "99",
		"reason": "spam",
		"__resolved:user": map[string]any{
			"id":      "99",
			"bot":     false,
			"system":  false,
			"mention": "<@99>",
		},
	}

	for i := 0; i < 3; i++ {
		got, hasValue, callErr := vm.CallRoute(ctx, luaplugin.RouteCommand, "warn", luaplugin.Payload{
			GuildID: "7",
			UserID:  "42",
			Locale:  "en-US",
			Options: baseOptions,
		})
		if callErr != nil {
			t.Fatalf("CallRoute(warn %d): %v", i+1, callErr)
		}
		if !hasValue {
			t.Fatalf("expected warn response %d", i+1)
		}

		warnMap, ok := got.(map[string]any)
		if !ok {
			t.Fatalf("expected warn response object, got %T", got)
		}
		content, _ := warnMap["content"].(string)
		if !strings.Contains(content, "<@99>") {
			t.Fatalf("unexpected warn response: %#v", warnMap)
		}
		if _, hasActions := warnMap["actions"]; hasActions {
			t.Fatalf("expected no deferred actions in interaction response, got %#v", warnMap)
		}
		if i == 2 && !strings.Contains(content, "time-out for 10 minutes") {
			t.Fatalf("expected successful timeout branch, got %#v", warnMap)
		}
	}
	if discordExecutor.timeoutCalls != 1 {
		t.Fatalf("expected one synchronous timeout call, got %d", discordExecutor.timeoutCalls)
	}
	if discordExecutor.sendDMCalls != 3 {
		t.Fatalf("expected one synchronous DM send per warn, got %d", discordExecutor.sendDMCalls)
	}
	if discordExecutor.lastGuild != 7 || discordExecutor.lastUser != 99 {
		t.Fatalf("unexpected timeout target: guild=%d user=%d", discordExecutor.lastGuild, discordExecutor.lastUser)
	}

	warnings, err := store.Warnings().ListWarnings(ctx, 7, 99, 10)
	if err != nil || len(warnings) == 0 {
		t.Fatalf("expected stored warnings, got %#v (%v)", warnings, err)
	}
	var timeoutAuditCount int
	if scanErr := db.QueryRow(`SELECT COUNT(1) FROM audit_log WHERE action = 'warn.timeout'`).Scan(&timeoutAuditCount); scanErr != nil {
		t.Fatalf("count timeout audits: %v", scanErr)
	}
	if timeoutAuditCount != 1 {
		t.Fatalf("expected one timeout audit entry, got %d", timeoutAuditCount)
	}

	got, hasValue, err := vm.CallRoute(ctx, luaplugin.RouteCommand, "unwarn", luaplugin.Payload{
		GuildID: "7",
		UserID:  "42",
		Locale:  "en-US",
		Options: map[string]any{
			"user": "99",
			"__resolved:user": map[string]any{
				"id":      "99",
				"bot":     false,
				"system":  false,
				"mention": "<@99>",
			},
		},
	})
	if err != nil {
		t.Fatalf("CallRoute(unwarn): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected unwarn response")
	}
	unwarnMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected unwarn response object, got %T", got)
	}
	components, _ := unwarnMap["components"].([]any)
	if len(components) != 1 {
		t.Fatalf("expected unwarn select menu, got %#v", unwarnMap)
	}

	value := warnings[0].ID + "|42|99|" + strconv.FormatInt(time.Now().UTC().Unix(), 10)
	got, hasValue, err = vm.CallRoute(ctx, luaplugin.RouteComponent, "unwarn_select", luaplugin.Payload{
		GuildID: "7",
		UserID:  "42",
		Locale:  "en-US",
		Options: map[string]any{
			"type":   "string_select",
			"values": []any{value},
		},
	})
	if err != nil {
		t.Fatalf("CallRoute(unwarn_select): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected unwarn_select response")
	}
	updateMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected component response object, got %T", got)
	}
	updateContent, _ := updateMap["content"].(string)
	if !strings.Contains(updateContent, "<@99>") {
		t.Fatalf("unexpected unwarn update: %#v", updateMap)
	}
}

func TestModerationPluginWarnTimeoutFailure(t *testing.T) {
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
	if err = reg.LoadPluginLocales("moderation", filepath.FromSlash("../../../plugins/moderation/locales")); err != nil {
		t.Fatalf("plugin i18n: %v", err)
	}

	discordExecutor := &fakeDiscordExecutor{timeoutErr: io.EOF}
	script := filepath.FromSlash("../../../plugins/moderation/plugin.lua")
	vm, err := luaplugin.NewFromFile(script, luaplugin.Options{
		Logger:    logger,
		PluginID:  "moderation",
		PluginDir: filepath.Dir(script),
		Permissions: permissions.Permissions{
			Storage: permissions.StoragePermissions{
				Warnings: true,
				Audit:    true,
			},
			Discord: permissions.DiscordPermissions{
				SendDM:        true,
				TimeoutMember: true,
			},
		},
		Discord: discordExecutor,
		Store:   store,
		I18n:    &reg,
	})
	if err != nil {
		t.Fatalf("NewFromFile(moderation): %v", err)
	}
	t.Cleanup(vm.Close)

	ctx := context.Background()
	options := map[string]any{
		"user":   "99",
		"reason": "spam",
		"__resolved:user": map[string]any{
			"id":      "99",
			"bot":     false,
			"system":  false,
			"mention": "<@99>",
		},
	}

	for i := 0; i < 3; i++ {
		got, hasValue, callErr := vm.CallRoute(ctx, luaplugin.RouteCommand, "warn", luaplugin.Payload{
			GuildID: "7",
			UserID:  "42",
			Locale:  "en-US",
			Options: options,
		})
		if callErr != nil {
			t.Fatalf("CallRoute(warn %d): %v", i+1, callErr)
		}
		if !hasValue {
			t.Fatalf("expected warn response %d", i+1)
		}
		if i != 2 {
			continue
		}

		warnMap, ok := got.(map[string]any)
		if !ok {
			t.Fatalf("expected warn response object, got %T", got)
		}
		content, _ := warnMap["content"].(string)
		if !strings.Contains(content, "I tried to time them out too, but I couldn't.") {
			t.Fatalf("expected timeout failure branch, got %#v", warnMap)
		}
		if _, hasActions := warnMap["actions"]; hasActions {
			t.Fatalf("expected no deferred actions on timeout failure, got %#v", warnMap)
		}
	}

	if discordExecutor.timeoutCalls != 1 {
		t.Fatalf("expected one synchronous timeout attempt, got %d", discordExecutor.timeoutCalls)
	}
	if discordExecutor.sendDMCalls != 3 {
		t.Fatalf("expected one synchronous DM send per warn, got %d", discordExecutor.sendDMCalls)
	}
	var timeoutAuditCount int
	if scanErr := db.QueryRow(`SELECT COUNT(1) FROM audit_log WHERE action = 'warn.timeout'`).Scan(&timeoutAuditCount); scanErr != nil {
		t.Fatalf("count timeout audits: %v", scanErr)
	}
	if timeoutAuditCount != 0 {
		t.Fatalf("expected no timeout audit entry on failure, got %d", timeoutAuditCount)
	}
}

func TestDiscordSendAPIs(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{}))
	discordExecutor := &fakeDiscordExecutor{}

	dir := t.TempDir()
	script := filepath.Join(dir, "plugin.lua")
	if err := os.WriteFile(script, []byte(`
return bot.plugin({
  commands = {
    bot.command("sendtest", {
      description = "Send test messages.",
      run = function(ctx)
        local dm_result, dm_err = bot.discord.send_dm({
          user_id = ctx.user.id,
          message = { content = "dm test" },
        })
        if dm_result == nil or dm_err ~= nil then
          error("dm failed")
        end

        local channel_result, channel_err = bot.discord.send_channel({
          message = { content = "channel test" },
        })
        if channel_result == nil or channel_err ~= nil then
          error("channel failed")
        end

        return bot.ui.reply({
          content = dm_result.message_id .. ":" .. channel_result.channel_id,
          ephemeral = true,
        })
      end,
    }),
  },
})
`), 0o644); err != nil {
		t.Fatalf("write plugin: %v", err)
	}

	vm, err := luaplugin.NewFromFile(script, luaplugin.Options{
		Logger:    logger,
		PluginID:  "sendtest",
		PluginDir: dir,
		Permissions: permissions.Permissions{
			Discord: permissions.DiscordPermissions{
				SendDM:      true,
				SendChannel: true,
			},
		},
		Discord: discordExecutor,
	})
	if err != nil {
		t.Fatalf("NewFromFile(sendtest): %v", err)
	}
	t.Cleanup(vm.Close)

	got, hasValue, err := vm.CallRoute(context.Background(), luaplugin.RouteCommand, "sendtest", luaplugin.Payload{
		ChannelID: "77",
		UserID:    "42",
		Locale:    "en-US",
	})
	if err != nil {
		t.Fatalf("CallRoute(sendtest): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected sendtest value")
	}

	gotMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected object, got %T", got)
	}
	if gotMap["content"] != "101:77" {
		t.Fatalf("unexpected content: %#v", gotMap)
	}
	if discordExecutor.sendDMCalls != 1 || discordExecutor.sendChannelCalls != 1 {
		t.Fatalf("unexpected send call counts: dm=%d channel=%d", discordExecutor.sendDMCalls, discordExecutor.sendChannelCalls)
	}
	if discordExecutor.lastUser != 42 || discordExecutor.lastChannel != 77 {
		t.Fatalf("unexpected send targets: user=%d channel=%d", discordExecutor.lastUser, discordExecutor.lastChannel)
	}
}

func TestManagerPluginRoutes(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{}))

	reg, err := i18n.LoadCore(filepath.FromSlash("../../../locales"))
	if err != nil {
		t.Fatalf("i18n: %v", err)
	}
	if err = reg.LoadPluginLocales("manager", filepath.FromSlash("../../../plugins/manager/locales")); err != nil {
		t.Fatalf("plugin i18n: %v", err)
	}

	discordExecutor := &fakeDiscordExecutor{}
	script := filepath.FromSlash("../../../plugins/manager/plugin.lua")
	vm, err := luaplugin.NewFromFile(script, luaplugin.Options{
		Logger:    logger,
		PluginID:  "manager",
		PluginDir: filepath.Dir(script),
		Permissions: permissions.Permissions{
			Discord: permissions.DiscordPermissions{
				SetSlowmode:   true,
				SetNickname:   true,
				CreateRole:    true,
				EditRole:      true,
				DeleteRole:    true,
				AddRole:       true,
				RemoveRole:    true,
				PurgeMessages: true,
				CreateEmoji:   true,
				EditEmoji:     true,
				DeleteEmoji:   true,
				CreateSticker: true,
				EditSticker:   true,
				DeleteSticker: true,
			},
		},
		Discord: discordExecutor,
		I18n:    &reg,
	})
	if err != nil {
		t.Fatalf("NewFromFile(manager): %v", err)
	}
	t.Cleanup(vm.Close)

	definition, ok := vm.Definition()
	if !ok {
		t.Fatalf("expected descriptor definition")
	}
	if len(definition.Commands) != 6 {
		t.Fatalf("unexpected command count: %#v", definition.Commands)
	}

	ctx := context.Background()

	got, hasValue, err := vm.CallRoute(ctx, luaplugin.RouteCommand, "slowmode", luaplugin.Payload{
		GuildID:   "1",
		ChannelID: "9",
		UserID:    "42",
		Locale:    "en-US",
		Options: map[string]any{
			"seconds": 5,
		},
	})
	if err != nil {
		t.Fatalf("CallRoute(slowmode): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected slowmode value")
	}

	slowmodeMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected slowmode object, got %T", got)
	}
	slowmodeEmbeds, _ := slowmodeMap["embeds"].([]any)
	slowmodeEmbed, _ := slowmodeEmbeds[0].(map[string]any)
	slowmodeDesc, _ := slowmodeEmbed["description"].(string)
	if !strings.Contains(slowmodeDesc, "<#9>") || !strings.Contains(slowmodeDesc, "5s") {
		t.Fatalf("unexpected slowmode description: %#v", slowmodeEmbed)
	}

	got, hasValue, err = vm.CallRoute(ctx, luaplugin.RouteCommand, "nick", luaplugin.Payload{
		GuildID: "1",
		UserID:  "42",
		Locale:  "en-US",
		Options: map[string]any{
			"user":     "99",
			"nickname": "Captain",
			"__resolved:user": map[string]any{
				"id":      "99",
				"mention": "<@99>",
				"bot":     false,
				"system":  false,
			},
		},
	})
	if err != nil {
		t.Fatalf("CallRoute(nick): %v", err)
	}
	if !hasValue {
		t.Fatalf("expected nick value")
	}

	nickMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected nick object, got %T", got)
	}
	nickEmbeds, _ := nickMap["embeds"].([]any)
	nickEmbed, _ := nickEmbeds[0].(map[string]any)
	nickDesc, _ := nickEmbed["description"].(string)
	if !strings.Contains(nickDesc, "<@99>") || !strings.Contains(nickDesc, "Captain") {
		t.Fatalf("unexpected nick description: %#v", nickEmbed)
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
