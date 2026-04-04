package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DiscordToken string

	SQLitePath       string
	Migrations       string
	MigrationBackups string
	LocalesDir       string
	PluginsDir       string
	PermissionsFile  string
	ModulesFile      string
	LogLevel         string
	ProdMode         bool
	OwnerUserID      []uint64
	DevGuildID       *uint64

	CommandRegistrationMode  string
	CommandGuildIDs          []uint64
	CommandRegisterAllGuilds bool

	AllowUnsignedPlugins bool
	TrustedKeysFile      string

	SlashCooldown          time.Duration
	ComponentCooldown      time.Duration
	ModalCooldown          time.Duration
	SlashCooldownBypass    []string
	SlashCooldownOverrides map[string]time.Duration
}

const (
	defaultSQLitePath        = "./data/mamusiabtw.sqlite"
	defaultMigrationsDir     = "./migrations/sqlite"
	defaultMigrationBackups  = "./data/migration_backups"
	defaultLocalesDir        = "./locales"
	defaultPluginsDir        = "./plugins"
	defaultPermissionsFile   = "./config/permissions.json"
	defaultModulesFile       = "./config/modules.json"
	defaultTrustedKeysFile   = "./config/trusted_keys.json"
	defaultLogLevel          = "info"
	defaultCommandRegMode    = "global"
	defaultSlashCooldownMS   = 5000
	defaultComponentCooldown = 750
	defaultModalCooldownMS   = 1500
)

func LoadFromEnv() (Config, error) {
	return loadFromEnv(true)
}

func LoadStorageFromEnv() (Config, error) {
	return loadFromEnv(false)
}

func loadFromEnv(requireDiscordToken bool) (Config, error) {
	var (
		discordToken string
		err          error
	)
	if requireDiscordToken {
		discordToken, err = requiredEnv("DISCORD_TOKEN")
		if err != nil {
			return Config{}, err
		}
	}

	sqlitePath := envDefault("SQLITE_PATH", defaultSQLitePath)
	migrations := envDefault("MIGRATIONS_DIR", defaultMigrationsDir)
	migrationBackups := envDefault("MAMUSIABTW_MIGRATION_BACKUPS_DIR", defaultMigrationBackups)
	localesDir := envDefault("LOCALES_DIR", defaultLocalesDir)
	pluginsDir := envDefault("PLUGINS_DIR", defaultPluginsDir)
	permissionsFile := envDefault("MAMUSIABTW_PERMISSIONS_FILE", defaultPermissionsFile)
	modulesFile := envDefault("MAMUSIABTW_MODULES_FILE", defaultModulesFile)
	logLevel := envDefault("LOG_LEVEL", defaultLogLevel)

	prodMode := envBool1("MAMUSIABTW_PROD_MODE")
	allowUnsigned := envBool1("MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS")
	trustedKeysFile := envDefault("MAMUSIABTW_TRUSTED_KEYS_FILE", defaultTrustedKeysFile)

	owners, err := parseOwnerIDs(os.Getenv("OWNER_USER_IDS"))
	if err != nil {
		return Config{}, err
	}

	devGuildRaw := os.Getenv("DISCORD_DEV_GUILD_ID")
	devGuildVal, hasDevGuild, err := parseOptionalUint64(devGuildRaw)
	if err != nil {
		return Config{}, err
	}
	var devGuildID *uint64
	if hasDevGuild {
		v := devGuildVal
		devGuildID = &v
	}

	cmdRegMode := strings.ToLower(envDefault("MAMUSIABTW_COMMAND_REGISTRATION_MODE", defaultCommandRegMode))
	switch cmdRegMode {
	case "global", "guilds", "hybrid":
	default:
		return Config{}, fmt.Errorf("invalid MAMUSIABTW_COMMAND_REGISTRATION_MODE %q", cmdRegMode)
	}

	cmdGuildIDs, err := parseUint64List(os.Getenv("MAMUSIABTW_COMMAND_GUILD_IDS"), "MAMUSIABTW_COMMAND_GUILD_IDS")
	if err != nil {
		return Config{}, err
	}
	cmdRegisterAllGuilds := strings.TrimSpace(os.Getenv("MAMUSIABTW_COMMAND_REGISTER_ALL_GUILDS")) == "1"

	slashCooldown, err := parseDurationMS(os.Getenv("MAMUSIABTW_SLASH_COOLDOWN_MS"), defaultSlashCooldownMS)
	if err != nil {
		return Config{}, err
	}
	componentCooldown, err := parseDurationMS(os.Getenv("MAMUSIABTW_COMPONENT_COOLDOWN_MS"), defaultComponentCooldown)
	if err != nil {
		return Config{}, err
	}
	modalCooldown, err := parseDurationMS(os.Getenv("MAMUSIABTW_MODAL_COOLDOWN_MS"), defaultModalCooldownMS)
	if err != nil {
		return Config{}, err
	}
	slashBypass := parseCSV(os.Getenv("MAMUSIABTW_SLASH_COOLDOWN_BYPASS"))
	if len(slashBypass) == 0 {
		slashBypass = []string{"ping", "help", "plugins", "modules", "block", "unblock"}
	}
	slashOverrides, err := parseCooldownOverridesMS(os.Getenv("MAMUSIABTW_SLASH_COOLDOWN_OVERRIDES_MS"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		DiscordToken:     discordToken,
		SQLitePath:       sqlitePath,
		Migrations:       migrations,
		MigrationBackups: migrationBackups,
		LocalesDir:       localesDir,
		PluginsDir:       pluginsDir,
		PermissionsFile:  permissionsFile,
		ModulesFile:      modulesFile,
		LogLevel:         logLevel,
		ProdMode:         prodMode,
		OwnerUserID:      owners,
		DevGuildID:       devGuildID,

		CommandRegistrationMode:  cmdRegMode,
		CommandGuildIDs:          cmdGuildIDs,
		CommandRegisterAllGuilds: cmdRegisterAllGuilds,

		AllowUnsignedPlugins: allowUnsigned,
		TrustedKeysFile:      trustedKeysFile,

		SlashCooldown:          slashCooldown,
		ComponentCooldown:      componentCooldown,
		ModalCooldown:          modalCooldown,
		SlashCooldownBypass:    slashBypass,
		SlashCooldownOverrides: slashOverrides,
	}, nil
}

func parseOwnerIDs(raw string) ([]uint64, error) {
	return parseUint64List(raw, "OWNER_USER_IDS")
}

func parseOptionalUint64(raw string) (uint64, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false, nil
	}

	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("invalid uint64 %q: %w", raw, err)
	}

	return v, true, nil
}

func envBool1(name string) bool {
	return strings.TrimSpace(os.Getenv(name)) == "1"
}

func envDefault(name, def string) string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def
	}
	return raw
}

func requiredEnv(name string) (string, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return "", errors.New(name + " is required")
	}
	return raw, nil
}

func parseDurationMS(raw string, def int) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Duration(def) * time.Millisecond, nil
	}

	ms, err := strconv.Atoi(raw)
	if err != nil || ms < 0 {
		return 0, fmt.Errorf("invalid milliseconds %q", raw)
	}
	return time.Duration(ms) * time.Millisecond, nil
}

func parseCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		s := strings.TrimSpace(part)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func parseCooldownOverridesMS(raw string) (map[string]time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]time.Duration{}, nil
	}
	items := parseCSV(raw)
	if len(items) == 0 {
		return map[string]time.Duration{}, nil
	}

	out := make(map[string]time.Duration, len(items))
	for _, item := range items {
		key, msRaw, ok := strings.Cut(item, "=")
		if !ok {
			return nil, fmt.Errorf("invalid cooldown override %q (expected name=ms)", item)
		}

		key = strings.ToLower(strings.TrimSpace(key))
		if key == "" {
			return nil, fmt.Errorf("invalid cooldown override %q (empty name)", item)
		}

		msRaw = strings.TrimSpace(msRaw)
		ms, err := strconv.Atoi(msRaw)
		if err != nil || ms < 0 {
			return nil, fmt.Errorf("invalid cooldown override %q (invalid ms %q)", item, msRaw)
		}

		out[key] = time.Duration(ms) * time.Millisecond
	}
	return out, nil
}

func parseUint64List(raw string, envName string) ([]uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	out := make([]uint64, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s contains invalid snowflake %q: %w", strings.TrimSpace(envName), part, err)
		}

		out = append(out, id)
	}

	return out, nil
}
