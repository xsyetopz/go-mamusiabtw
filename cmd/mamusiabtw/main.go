package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/xsyetopz/go-mamusiabtw/internal/app"
	"github.com/xsyetopz/go-mamusiabtw/internal/buildinfo"
	"github.com/xsyetopz/go-mamusiabtw/internal/config"
	"github.com/xsyetopz/go-mamusiabtw/internal/dotenv"
	"github.com/xsyetopz/go-mamusiabtw/internal/logging"
	migrate "github.com/xsyetopz/go-mamusiabtw/internal/migration"
	pluginhost "github.com/xsyetopz/go-mamusiabtw/internal/runtime/plugins"
)

func main() {
	os.Exit(runMain())
}

func runMain() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	loadedEnv, envErr := autoLoadEnvFile()
	if envErr != nil {
		_, _ = os.Stderr.WriteString(envErr.Error() + "\n")
		return 1
	}
	if loadedEnv.Path != "" {
		_ = os.Setenv("MAMUSIABTW_LOADED_ENV_FILE", loadedEnv.Path)
		_ = os.Setenv("MAMUSIABTW_LOADED_ENV_SOURCE", loadedEnv.Source)
	}

	// Low mental-load workflow: allow env files to be used without requiring users
	// to manually export variables before running.
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "init" {
		return runInitCommand(args[1:])
	}
	if len(args) > 0 && args[0] == "doctor" {
		return runDoctorCommand(args[1:])
	}
	if len(args) > 0 && args[0] == "dev" {
		return runDevCommand(ctx)
	}
	if len(args) > 0 && args[0] == "migrate" {
		return runMigrateCommand(ctx, args[1:])
	}
	if len(args) > 0 && args[0] == "version" {
		printVersion()
		return 0
	}
	if len(args) > 0 && args[0] == "sign-plugin" {
		return runSignPluginCommand(args[1:])
	}
	if len(args) > 0 && args[0] == "gen-signing-key" {
		return runGenSigningKeyCommand(args[1:])
	}

	cfg, err := config.LoadFromEnv()
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}

	logger, err := logging.New(cfg.LogLevel)
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}

	if runErr := run(ctx, logger, cfg); runErr != nil {
		logger.ErrorContext(ctx, "fatal", slog.String("err", runErr.Error()))
		return 1
	}

	return 0
}

func runDoctorCommand(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	cfg, err := config.LoadFromEnvOptionalDiscordToken()
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}

	writeLine := func(format string, a ...any) {
		_, _ = fmt.Fprintf(os.Stdout, format+"\n", a...)
	}

	loadedEnv := strings.TrimSpace(os.Getenv("MAMUSIABTW_LOADED_ENV_FILE"))
	loadedEnvSource := strings.TrimSpace(os.Getenv("MAMUSIABTW_LOADED_ENV_SOURCE"))
	if loadedEnv == "" {
		writeLine("env_file_loaded: false")
	} else {
		writeLine("env_file_loaded: %s", loadedEnv)
		writeLine("env_file_source: %s", loadedEnvSource)
	}

	hasToken := strings.TrimSpace(cfg.DiscordToken) != ""
	writeLine("discord_token: %t", hasToken)
	writeLine("prod_mode: %t", cfg.ProdMode)
	writeLine("admin_api_enabled: %t", strings.TrimSpace(cfg.AdminAddr) != "")
	writeLine("allow_unsigned_plugins: %t", cfg.AllowUnsignedPlugins)
	writeLine("trusted_keys_file: %s", cfg.TrustedKeysFile)
	trustedKeysPath := strings.TrimSpace(cfg.TrustedKeysFile)
	trustedKeysExists := false
	trustedKeysCount := 0
	if trustedKeysPath != "" {
		if keys, err := pluginhost.ReadTrustedKeysFile(trustedKeysPath); err == nil {
			trustedKeysExists = true
			trustedKeysCount = len(keys)
		} else if !os.IsNotExist(err) {
			writeLine("trusted_keys_file_error: %s", err)
		}
	}
	writeLine("trusted_keys_file_exists: %t", trustedKeysExists)
	writeLine("trusted_keys_count_file: %d", trustedKeysCount)
	writeLine(
		"dashboard_signing_configured: %t",
		strings.TrimSpace(cfg.DashboardSigningKeyID) != "" && strings.TrimSpace(cfg.DashboardSigningKeyFile) != "",
	)
	if strings.TrimSpace(cfg.AdminAddr) != "" {
		writeLine("admin_addr: %s", cfg.AdminAddr)
		writeLine("setup_url: %s/api/setup", httpBaseFromAddr(cfg.AdminAddr))
	}

	if strings.TrimSpace(cfg.AdminAddr) != "" {
		base := httpBaseFromAddr(cfg.AdminAddr)
		writeLine("dashboard_base_url: %s", base)
		writeLine("dashboard_oauth_redirect_url: %s/api/auth/callback", base)
		writeLine("dashboard_client_id_set: %t", strings.TrimSpace(cfg.DashboardClientID) != "")
		writeLine("dashboard_client_secret_set: %t", strings.TrimSpace(cfg.DashboardClientSecret) != "")
		writeLine("dashboard_session_secret_set: %t", len(strings.TrimSpace(cfg.DashboardSessionSecret)) >= 32)
		if cfg.DashboardSessionSecretGenerated {
			writeLine("dashboard_session_secret_generated: true (dev-only, ephemeral)")
		}
	}

	if strings.TrimSpace(cfg.AdminAddr) != "" && cfg.ProdMode {
		if strings.TrimSpace(cfg.DashboardClientID) == "" ||
			strings.TrimSpace(cfg.DashboardClientSecret) == "" ||
			len(strings.TrimSpace(cfg.DashboardSessionSecret)) < 32 {
			writeLine("")
			writeLine("next: admin api is enabled in prod mode but oauth/session config is incomplete")
			writeLine("next: fill MAMUSIABTW_DASHBOARD_* vars (client id/secret/session secret)")
			return 1
		}
	}

	if !hasToken {
		writeLine("")
		writeLine("next: set DISCORD_TOKEN to start the bot")
		return 1
	}

	return 0
}

func autoLoadEnvFile() (dotenv.SearchResult, error) {
	if strings.TrimSpace(os.Getenv("MAMUSIABTW_DISABLE_DOTENV")) == "1" {
		return dotenv.SearchResult{}, nil
	}
	if bad := forbiddenDotenvFile(); bad != "" {
		return dotenv.SearchResult{}, fmt.Errorf("forbidden env file detected: %s (only .env.dev/.env.prod are allowed)", bad)
	}

	searchDirs := []dotenv.SearchResult{
		{Path: ".", Source: "working_dir"},
	}
	if execPath, err := os.Executable(); err == nil {
		if execDir := strings.TrimSpace(filepath.Dir(execPath)); execDir != "" && execDir != "." {
			searchDirs = append(searchDirs, dotenv.SearchResult{Path: execDir, Source: "executable_dir"})
		}
	}

	if explicit := strings.TrimSpace(os.Getenv("MAMUSIABTW_ENV_FILE")); explicit != "" {
		base := filepath.Base(explicit)
		if base != ".env.dev" && base != ".env.prod" {
			return dotenv.SearchResult{}, fmt.Errorf("refusing to load non-standard env file %s; use .env.dev or .env.prod instead", base)
		}
		return dotenv.LoadAutoWithSearch([]string{explicit}, searchDirs)
	}

	subcmd := ""
	if len(os.Args) > 1 {
		subcmd = strings.TrimSpace(os.Args[1])
	}

	switch subcmd {
	case "dev", "init":
		return dotenv.LoadAutoWithSearch([]string{".env.dev"}, searchDirs)
	case "doctor":
		return dotenv.LoadAutoWithSearch([]string{".env.prod", ".env.dev"}, searchDirs)
	default:
		return dotenv.LoadAutoWithSearch([]string{".env.prod"}, searchDirs)
	}
}

func httpBaseFromAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "http://127.0.0.1:8081"
	}
	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		return "http://" + addr
	}
	switch strings.TrimSpace(host) {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}
	return "http://" + host + ":" + port
}

func runDevCommand(ctx context.Context) int {
	// Lowest-effort path: if you run "mamusiabtw dev" you get the admin API too.
	_ = os.Setenv("MAMUSIABTW_PROD_MODE", "0")
	if strings.TrimSpace(os.Getenv("MAMUSIABTW_ADMIN_ADDR")) == "" {
		_ = os.Setenv("MAMUSIABTW_ADMIN_ADDR", "127.0.0.1:8081")
	}
	if strings.TrimSpace(os.Getenv("MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS")) == "" {
		_ = os.Setenv("MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS", "1")
	}

	cfg, err := config.LoadFromEnv()
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}
	logger, err := logging.New(cfg.LogLevel)
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}
	base := httpBaseFromAddr(cfg.AdminAddr)
	_, _ = fmt.Fprintf(os.Stdout, "admin_setup_url: %s/api/setup\n", base)
	_, _ = fmt.Fprintf(os.Stdout, "dashboard_url: %s/\n", base)
	_, _ = os.Stdout.WriteString("dashboard_dev: cd apps/dashboard && bun run dev\n")
	_, _ = os.Stdout.WriteString("dashboard_dev_note: you can open dashboard_url (recommended) or http://127.0.0.1:5173/ (Vite proxies /api)\n")

	if runErr := run(ctx, logger, cfg); runErr != nil {
		logger.ErrorContext(ctx, "fatal", slog.String("err", runErr.Error()))
		return 1
	}
	return 0
}

func runInitCommand(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	mode := fs.String("mode", "dev", "mode: dev|prod")
	force := fs.Bool("force", false, "overwrite existing files")
	discordToken := fs.String("discord-token", "", "discord bot token")
	clientID := fs.String("client-id", "", "discord oauth client id")
	clientSecret := fs.String("client-secret", "", "discord oauth client secret")
	adminAddr := fs.String("admin-addr", "", "admin api listen addr (host:port)")
	sessionSecret := fs.String("session-secret", "", "session secret (32+ chars)")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	rawMode := strings.ToLower(strings.TrimSpace(*mode))
	modeKind := ""
	switch rawMode {
	case "dev":
		modeKind = "dev"
	case "prod":
		modeKind = "prod"
	default:
		_, _ = os.Stderr.WriteString("init: --mode must be dev|prod\n")
		return 1
	}

	rootEnv := ".env.dev"
	if modeKind == "prod" {
		rootEnv = ".env.prod"
	}

	if !*force {
		if _, err := os.Stat(rootEnv); err == nil {
			_, _ = os.Stderr.WriteString("init: " + rootEnv + " already exists (use --force to overwrite)\n")
			return 1
		}
	}

	if strings.TrimSpace(*adminAddr) == "" && modeKind == "dev" {
		*adminAddr = "127.0.0.1:8081"
	}
	if strings.TrimSpace(*sessionSecret) == "" {
		*sessionSecret = genHexSecret(32)
	}

	root := strings.Builder{}
	root.WriteString("# mamusiabtw\n")
	root.WriteString("DISCORD_TOKEN=" + strings.TrimSpace(*discordToken) + "\n")
	if modeKind == "prod" {
		root.WriteString("MAMUSIABTW_PROD_MODE=1\n")
		root.WriteString("MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=0\n")
	} else {
		root.WriteString("MAMUSIABTW_PROD_MODE=0\n")
		root.WriteString("MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=1\n")
	}
	if strings.TrimSpace(*adminAddr) != "" {
		root.WriteString("\n# Admin API + dashboard OAuth\n")
		root.WriteString("MAMUSIABTW_ADMIN_ADDR=" + strings.TrimSpace(*adminAddr) + "\n")
		root.WriteString("MAMUSIABTW_DASHBOARD_CLIENT_ID=" + strings.TrimSpace(*clientID) + "\n")
		root.WriteString("MAMUSIABTW_DASHBOARD_CLIENT_SECRET=" + strings.TrimSpace(*clientSecret) + "\n")
		root.WriteString("MAMUSIABTW_DASHBOARD_SESSION_SECRET=" + strings.TrimSpace(*sessionSecret) + "\n")
	}

	if err := os.WriteFile(rootEnv, []byte(root.String()), 0o600); err != nil {
		_, _ = os.Stderr.WriteString("init: write " + rootEnv + ": " + err.Error() + "\n")
		return 1
	}

	_, _ = fmt.Fprintf(os.Stdout, "wrote: %s\n", rootEnv)
	if modeKind == "dev" {
		_, _ = os.Stdout.WriteString("next: mamusiabtw dev\n")
		_, _ = os.Stdout.WriteString("next: cd apps/dashboard && bun install && bun run dev\n")
	}
	return 0
}

func genHexSecret(nBytes int) string {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return strings.Repeat("x", nBytes)
	}
	return hex.EncodeToString(buf)
}

func forbiddenDotenvFile() string {
	// Hardline policy: only .env.dev and .env.prod are permitted.
	forbidden := []string{
		".env.local",
		".env.development",
		".env.production",
		".env.production.local",
		".env.dev.local",
		".env.prod.local",
		".env",
	}
	for _, path := range forbidden {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func run(ctx context.Context, logger *slog.Logger, cfg config.Config) error {
	mamusiabtw, err := app.New(app.Dependencies{
		Logger: logger,
		Config: cfg,
	})
	if err != nil {
		return err
	}
	defer mamusiabtw.Close()

	if startErr := mamusiabtw.Start(ctx); startErr != nil {
		if errors.Is(startErr, context.Canceled) {
			return nil
		}
		return startErr
	}

	return nil
}

func runMigrateCommand(ctx context.Context, args []string) int {
	cfg, err := config.LoadStorageFromEnv()
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}

	runner, err := migrate.New(migrate.Options{
		Dir:       cfg.Migrations,
		BackupDir: cfg.MigrationBackups,
	})
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}

	if len(args) == 0 {
		printMigrateUsage()
		return 1
	}

	switch args[0] {
	case "status":
		status, err := runner.StatusPath(ctx, cfg.SQLitePath)
		if err != nil {
			_, _ = os.Stderr.WriteString(err.Error() + "\n")
			return 1
		}
		printStatus(status)
		return 0
	case "up":
		status, err := runner.UpPath(ctx, cfg.SQLitePath)
		if err != nil {
			_, _ = os.Stderr.WriteString(err.Error() + "\n")
			return 1
		}
		printStatus(status)
		return 0
	case "backup":
		backupPath, err := runner.BackupPath(ctx, cfg.SQLitePath)
		if err != nil {
			_, _ = os.Stderr.WriteString(err.Error() + "\n")
			return 1
		}
		_, _ = fmt.Fprintf(os.Stdout, "backup: %s\n", backupPath)
		return 0
	default:
		printMigrateUsage()
		return 1
	}
}

func printStatus(status migrate.Status) {
	_, _ = fmt.Fprintf(os.Stdout, "current_version: %d\n", status.CurrentVersion)
	if len(status.Applied) == 0 {
		_, _ = os.Stdout.WriteString("applied: none\n")
	} else {
		_, _ = os.Stdout.WriteString("applied:\n")
		for _, item := range status.Applied {
			_, _ = fmt.Fprintf(os.Stdout, "  - %03d %s (%s)\n", item.Version, item.Name, item.Kind)
		}
	}
	if len(status.Pending) == 0 {
		_, _ = os.Stdout.WriteString("pending: none\n")
		return
	}
	_, _ = os.Stdout.WriteString("pending:\n")
	for _, item := range status.Pending {
		_, _ = fmt.Fprintf(os.Stdout, "  - %03d %s (%s)\n", item.Version, item.Name, item.Kind)
	}
}

func printMigrateUsage() {
	_, _ = os.Stderr.WriteString(
		"usage:\n" +
			"  mamusiabtw migrate status\n" +
			"  mamusiabtw migrate up\n" +
			"  mamusiabtw migrate backup\n" +
			"",
	)
}

func printVersion() {
	info := buildinfo.Current()
	_, _ = fmt.Fprintf(os.Stdout, "version: %s\n", info.Version)
	_, _ = fmt.Fprintf(os.Stdout, "description: %s\n", info.Description)
	if info.Repository != "" {
		_, _ = fmt.Fprintf(os.Stdout, "repository: %s\n", info.Repository)
	}
	if info.DeveloperURL != "" {
		_, _ = fmt.Fprintf(os.Stdout, "developer_url: %s\n", info.DeveloperURL)
	}
	if info.SupportServerURL != "" {
		_, _ = fmt.Fprintf(os.Stdout, "support_server_url: %s\n", info.SupportServerURL)
	}
	if info.MascotImageURL != "" {
		_, _ = fmt.Fprintf(os.Stdout, "mascot_image_url: %s\n", info.MascotImageURL)
	}
}

func runSignPluginCommand(args []string) int {
	fs := flag.NewFlagSet("sign-plugin", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	dir := fs.String("dir", "", "plugin directory to sign")
	keyID := fs.String("key-id", "", "signer key id")
	privateKeyFile := fs.String("private-key-file", "", "file containing base64 ed25519 private key bytes or seed")
	out := fs.String("out", "", "output signature path (default: <dir>/signature.json)")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *dir == "" || *keyID == "" || *privateKeyFile == "" {
		_, _ = os.Stderr.WriteString("usage: mamusiabtw sign-plugin --dir <plugin_dir> --key-id <key_id> --private-key-file <path> [--out <signature.json>]\n")
		return 1
	}

	privateKey, err := pluginhost.ReadEd25519PrivateKeyFile(*privateKeyFile)
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}

	sig, publicKey, err := pluginhost.SignDir(*dir, *keyID, privateKey)
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}

	target := *out
	if target == "" {
		target = *dir + "/signature.json"
	}

	payload := map[string]any{
		"$schema":       pluginhost.SignatureSchemaURL,
		"key_id":        sig.KeyID,
		"hash_b64":      sig.HashB64,
		"signature_b64": sig.SignatureB64,
		"algorithm":     sig.Algorithm,
	}
	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}
	bytes = append(bytes, '\n')
	if err := os.WriteFile(target, bytes, 0o644); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}

	_, _ = fmt.Fprintf(os.Stdout, "signature: %s\n", target)
	_, _ = fmt.Fprintf(os.Stdout, "public_key_b64: %s\n", base64.StdEncoding.EncodeToString(publicKey))
	return 0
}

func runGenSigningKeyCommand(args []string) int {
	fs := flag.NewFlagSet("gen-signing-key", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	keyID := fs.String("key-id", "", "signer key id")
	privateKeyFile := fs.String("private-key-file", "", "output private key file (default: ./data/keys/<key_id>.key)")
	trustedKeysFile := fs.String("trusted-keys-file", "", "trusted keys file to create/update (default: ./config/trusted_keys.json)")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if strings.TrimSpace(*keyID) == "" {
		_, _ = os.Stderr.WriteString("usage: mamusiabtw gen-signing-key --key-id <key_id> [--private-key-file <path>] [--trusted-keys-file <path>]\n")
		return 1
	}

	keyPath := strings.TrimSpace(*privateKeyFile)
	if keyPath == "" {
		keyPath = filepath.Join(".", "data", "keys", strings.TrimSpace(*keyID)+".key")
	}
	trustPath := strings.TrimSpace(*trustedKeysFile)
	if trustPath == "" {
		trustPath = strings.TrimSpace(os.Getenv("MAMUSIABTW_TRUSTED_KEYS_FILE"))
	}
	if trustPath == "" {
		trustPath = "./config/trusted_keys.json"
	}

	publicKey, privateKey, err := pluginhost.GenerateEd25519Key()
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}
	if err := pluginhost.WriteEd25519PrivateKeyFile(keyPath, privateKey); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}

	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKey)
	if err := pluginhost.UpsertTrustedKeyFile(trustPath, pluginhost.TrustedKey{
		KeyID:        strings.TrimSpace(*keyID),
		PublicKeyB64: publicKeyB64,
	}); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}

	_, _ = fmt.Fprintf(os.Stdout, "key_id: %s\n", strings.TrimSpace(*keyID))
	_, _ = fmt.Fprintf(os.Stdout, "private_key_file: %s\n", keyPath)
	_, _ = fmt.Fprintf(os.Stdout, "trusted_keys_file: %s\n", trustPath)
	_, _ = fmt.Fprintf(os.Stdout, "public_key_b64: %s\n", publicKeyB64)
	return 0
}
