package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xsyetopz/go-mamusiabtw/internal/config"
)

func TestLoadFromEnv_Defaults(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("DISCORD_TOKEN", "discord-token")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}

	if cfg.DiscordToken != "discord-token" {
		t.Fatalf("unexpected discord token: %q", cfg.DiscordToken)
	}
	if cfg.SQLitePath != "./data/mamusiabtw.sqlite" {
		t.Fatalf("unexpected sqlite path: %q", cfg.SQLitePath)
	}
	if cfg.Migrations != "./migrations/sqlite" {
		t.Fatalf("unexpected migrations dir: %q", cfg.Migrations)
	}
	if cfg.LocalesDir != "./locales" {
		t.Fatalf("unexpected locales dir: %q", cfg.LocalesDir)
	}
	if cfg.PluginsDir != "./plugins" {
		t.Fatalf("unexpected plugins dir: %q", cfg.PluginsDir)
	}
	if cfg.PermissionsFile != "./config/permissions.json" {
		t.Fatalf("unexpected permissions file: %q", cfg.PermissionsFile)
	}
	if cfg.ModulesFile != "./config/modules.json" {
		t.Fatalf("unexpected modules file: %q", cfg.ModulesFile)
	}
	if cfg.TrustedKeysFile != "./config/trusted_keys.json" {
		t.Fatalf("unexpected trusted keys file: %q", cfg.TrustedKeysFile)
	}
	if cfg.CommandRegistrationMode != "global" {
		t.Fatalf("unexpected command registration mode: %q", cfg.CommandRegistrationMode)
	}
	if cfg.SlashCooldown != 5*time.Second {
		t.Fatalf("unexpected slash cooldown: %s", cfg.SlashCooldown)
	}
	if cfg.ComponentCooldown != 750*time.Millisecond {
		t.Fatalf("unexpected component cooldown: %s", cfg.ComponentCooldown)
	}
	if cfg.ModalCooldown != 1500*time.Millisecond {
		t.Fatalf("unexpected modal cooldown: %s", cfg.ModalCooldown)
	}

	wantBypass := []string{"ping", "help", "plugins", "modules", "block", "unblock"}
	if !reflect.DeepEqual(cfg.SlashCooldownBypass, wantBypass) {
		t.Fatalf("unexpected bypass list: %#v", cfg.SlashCooldownBypass)
	}
	if len(cfg.SlashCooldownOverrides) != 0 {
		t.Fatalf("expected no cooldown overrides, got %#v", cfg.SlashCooldownOverrides)
	}
}

func TestLoadFromEnv_ParsesOverrides(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("DISCORD_TOKEN", "discord-token")
	t.Setenv("OWNER_USER_IDS", "11,22")
	t.Setenv("DISCORD_DEV_GUILD_ID", "33")
	t.Setenv("MAMUSIABTW_COMMAND_REGISTRATION_MODE", "hybrid")
	t.Setenv("MAMUSIABTW_COMMAND_GUILD_IDS", "44,55")
	t.Setenv("MAMUSIABTW_COMMAND_REGISTER_ALL_GUILDS", "1")
	t.Setenv("MAMUSIABTW_PROD_MODE", "1")
	t.Setenv("MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS", "1")
	t.Setenv("MAMUSIABTW_SLASH_COOLDOWN_MS", "9000")
	t.Setenv("MAMUSIABTW_COMPONENT_COOLDOWN_MS", "250")
	t.Setenv("MAMUSIABTW_MODAL_COOLDOWN_MS", "350")
	t.Setenv("MAMUSIABTW_SLASH_COOLDOWN_BYPASS", "ping, lookup:user")
	t.Setenv("MAMUSIABTW_SLASH_COOLDOWN_OVERRIDES_MS", "lookup:user=2500,manager:roles:add=1000")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}

	if !reflect.DeepEqual(cfg.OwnerUserID, []uint64{11, 22}) {
		t.Fatalf("unexpected owner ids: %#v", cfg.OwnerUserID)
	}
	if cfg.DevGuildID == nil || *cfg.DevGuildID != 33 {
		t.Fatalf("unexpected dev guild id: %v", cfg.DevGuildID)
	}
	if cfg.CommandRegistrationMode != "hybrid" {
		t.Fatalf("unexpected command registration mode: %q", cfg.CommandRegistrationMode)
	}
	if !reflect.DeepEqual(cfg.CommandGuildIDs, []uint64{44, 55}) {
		t.Fatalf("unexpected command guild ids: %#v", cfg.CommandGuildIDs)
	}
	if !cfg.CommandRegisterAllGuilds {
		t.Fatalf("expected register-all-guilds to be enabled")
	}
	if !cfg.ProdMode {
		t.Fatalf("expected prod mode to be enabled")
	}
	if !cfg.AllowUnsignedPlugins {
		t.Fatalf("expected unsigned plugins flag to be enabled")
	}
	if cfg.SlashCooldown != 9*time.Second {
		t.Fatalf("unexpected slash cooldown: %s", cfg.SlashCooldown)
	}
	if cfg.ComponentCooldown != 250*time.Millisecond {
		t.Fatalf("unexpected component cooldown: %s", cfg.ComponentCooldown)
	}
	if cfg.ModalCooldown != 350*time.Millisecond {
		t.Fatalf("unexpected modal cooldown: %s", cfg.ModalCooldown)
	}

	wantBypass := []string{"ping", "lookup:user"}
	if !reflect.DeepEqual(cfg.SlashCooldownBypass, wantBypass) {
		t.Fatalf("unexpected bypass list: %#v", cfg.SlashCooldownBypass)
	}

	wantOverrides := map[string]time.Duration{
		"lookup:user":       2500 * time.Millisecond,
		"manager:roles:add": 1000 * time.Millisecond,
	}
	if !reflect.DeepEqual(cfg.SlashCooldownOverrides, wantOverrides) {
		t.Fatalf("unexpected cooldown overrides: %#v", cfg.SlashCooldownOverrides)
	}
}

func TestLoadFromEnv_RejectsInvalidInputs(t *testing.T) {
	t.Run("registration mode", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("DISCORD_TOKEN", "discord-token")
		t.Setenv("MAMUSIABTW_COMMAND_REGISTRATION_MODE", "broken")

		if _, err := config.LoadFromEnv(); err == nil {
			t.Fatalf("expected invalid registration mode error")
		}
	})

	t.Run("cooldown override", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("DISCORD_TOKEN", "discord-token")
		t.Setenv("MAMUSIABTW_SLASH_COOLDOWN_OVERRIDES_MS", "lookup:user=nope")

		if _, err := config.LoadFromEnv(); err == nil {
			t.Fatalf("expected invalid cooldown override error")
		}
	})

	t.Run("owner ids", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("DISCORD_TOKEN", "discord-token")
		t.Setenv("OWNER_USER_IDS", "11,nope")

		if _, err := config.LoadFromEnv(); err == nil {
			t.Fatalf("expected invalid owner ids error")
		}
	})
}

func TestShippedSchemaURLs(t *testing.T) {
	t.Parallel()

	const schemaBaseURL = "https://raw.githubusercontent.com/xsyetopz/go-mamusiabtw/refs/heads/main/schemas/"

	cases := []struct {
		path string
		key  string
		want string
	}{
		{path: "config/permissions.json", key: "$schema", want: schemaBaseURL + "permissions.schema.v1.json"},
		{path: "config/modules.json", key: "$schema", want: schemaBaseURL + "modules.schema.v1.json"},
		{path: "examples/plugins/example/plugin.json", key: "$schema", want: schemaBaseURL + "plugin.schema.v1.json"},
		{path: "plugins/fun/plugin.json", key: "$schema", want: schemaBaseURL + "plugin.schema.v1.json"},
		{path: "plugins/wellness/plugin.json", key: "$schema", want: schemaBaseURL + "plugin.schema.v1.json"},
		{path: "schemas/messages.schema.v1.json", key: "$id", want: schemaBaseURL + "messages.schema.v1.json"},
		{path: "schemas/modules.schema.v1.json", key: "$id", want: schemaBaseURL + "modules.schema.v1.json"},
		{path: "schemas/permissions.schema.v1.json", key: "$id", want: schemaBaseURL + "permissions.schema.v1.json"},
		{path: "schemas/plugin.schema.v1.json", key: "$id", want: schemaBaseURL + "plugin.schema.v1.json"},
		{path: "schemas/signature.schema.v1.json", key: "$id", want: schemaBaseURL + "signature.schema.v1.json"},
		{path: "schemas/trusted_keys.schema.v1.json", key: "$id", want: schemaBaseURL + "trusted_keys.schema.v1.json"},
	}

	repoRoot := filepath.Clean(filepath.Join("..", ".."))

	for _, tc := range cases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()

			filePath := filepath.Join(repoRoot, tc.path)
			bytes, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("ReadFile(%q): %v", filePath, err)
			}

			var payload map[string]any
			if err := json.Unmarshal(bytes, &payload); err != nil {
				t.Fatalf("json.Unmarshal(%q): %v", filePath, err)
			}

			got, ok := payload[tc.key].(string)
			if !ok {
				t.Fatalf("missing %q in %s", tc.key, tc.path)
			}
			if got != tc.want {
				t.Fatalf("unexpected %s in %s: got %q want %q", tc.key, tc.path, got, tc.want)
			}
		})
	}
}

func TestAuthoringAssetsLayout(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Clean(filepath.Join("..", ".."))

	luarcPath := filepath.Join(repoRoot, ".luarc.json")
	luarcBytes, err := os.ReadFile(luarcPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", luarcPath, err)
	}

	var luarc map[string]any
	if err := json.Unmarshal(luarcBytes, &luarc); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", luarcPath, err)
	}

	libraryEntries, ok := luarc["workspace.library"].([]any)
	if !ok {
		t.Fatalf("workspace.library missing or invalid in %s", luarcPath)
	}

	var hasBotAPI bool
	for _, entry := range libraryEntries {
		pathValue, ok := entry.(string)
		if !ok || pathValue != "./sdk/lua/bot_api.lua" {
			continue
		}
		hasBotAPI = true

		fullPath := filepath.Join(repoRoot, pathValue)
		if _, err := os.Stat(fullPath); err != nil {
			t.Fatalf("Stat(%q): %v", fullPath, err)
		}
	}
	if !hasBotAPI {
		t.Fatalf("workspace.library does not include ./sdk/lua/bot_api.lua")
	}

	for _, relPath := range []string{
		"examples/plugins/example/plugin.json",
		"examples/plugins/example/plugin.lua",
		"examples/plugins/example/lib/counter.lua",
		"examples/plugins/example/locales/en-US/messages.json",
		"examples/plugins/example/locales/en-GB/messages.json",
		"plugins/fun/plugin.json",
		"plugins/fun/plugin.lua",
		"plugins/wellness/plugin.json",
		"plugins/wellness/plugin.lua",
	} {
		fullPath := filepath.Join(repoRoot, relPath)
		if _, err := os.Stat(fullPath); err != nil {
			t.Fatalf("Stat(%q): %v", fullPath, err)
		}
	}

	localeEntries, err := os.ReadDir(filepath.Join(repoRoot, "locales"))
	if err != nil {
		t.Fatalf("ReadDir(locales): %v", err)
	}
	for _, entry := range localeEntries {
		if !entry.IsDir() {
			continue
		}

		funLocalePath := filepath.Join(repoRoot, "plugins", "fun", "locales", entry.Name(), "messages.json")
		if _, err := os.Stat(funLocalePath); err != nil {
			t.Fatalf("Stat(%q): %v", funLocalePath, err)
		}

		wellnessLocalePath := filepath.Join(repoRoot, "plugins", "wellness", "locales", entry.Name(), "messages.json")
		if _, err := os.Stat(wellnessLocalePath); err != nil {
			t.Fatalf("Stat(%q): %v", wellnessLocalePath, err)
		}

		coreLocalePath := filepath.Join(repoRoot, "locales", entry.Name(), "messages.json")
		coreBytes, err := os.ReadFile(coreLocalePath)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", coreLocalePath, err)
		}

		var coreMessages []map[string]any
		if err := json.Unmarshal(coreBytes, &coreMessages); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", coreLocalePath, err)
		}
		for _, message := range coreMessages {
			id, _ := message["id"].(string)
			if strings.HasPrefix(id, "cmd.flip") ||
				strings.HasPrefix(id, "cmd.roll") ||
				strings.HasPrefix(id, "cmd.8ball") ||
				strings.HasPrefix(id, "cmd.hug") ||
				strings.HasPrefix(id, "cmd.pat") ||
				strings.HasPrefix(id, "cmd.poke") ||
				strings.HasPrefix(id, "cmd.shrug") ||
				strings.HasPrefix(id, "fun.") {
				t.Fatalf("core locale %q still contains migrated fun id %q", coreLocalePath, id)
			}
			if strings.HasPrefix(id, "cmd.timezone") ||
				strings.HasPrefix(id, "cmd.checkin") ||
				strings.HasPrefix(id, "cmd.remind") ||
				strings.HasPrefix(id, "wellness.") {
				t.Fatalf("core locale %q still contains migrated wellness id %q", coreLocalePath, id)
			}
		}
	}
}

func resetConfigEnv(t *testing.T) {
	t.Helper()

	for _, name := range []string{
		"DISCORD_TOKEN",
		"SQLITE_PATH",
		"MIGRATIONS_DIR",
		"LOCALES_DIR",
		"PLUGINS_DIR",
		"MAMUSIABTW_PERMISSIONS_FILE",
		"MAMUSIABTW_MODULES_FILE",
		"LOG_LEVEL",
		"MAMUSIABTW_PROD_MODE",
		"MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS",
		"MAMUSIABTW_TRUSTED_KEYS_FILE",
		"OWNER_USER_IDS",
		"DISCORD_DEV_GUILD_ID",
		"MAMUSIABTW_COMMAND_REGISTRATION_MODE",
		"MAMUSIABTW_COMMAND_GUILD_IDS",
		"MAMUSIABTW_COMMAND_REGISTER_ALL_GUILDS",
		"MAMUSIABTW_SLASH_COOLDOWN_MS",
		"MAMUSIABTW_COMPONENT_COOLDOWN_MS",
		"MAMUSIABTW_MODAL_COOLDOWN_MS",
		"MAMUSIABTW_SLASH_COOLDOWN_BYPASS",
		"MAMUSIABTW_SLASH_COOLDOWN_OVERRIDES_MS",
	} {
		t.Setenv(name, "")
	}
}
