package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/xsyetopz/go-mamusiabtw/internal/app"
	"github.com/xsyetopz/go-mamusiabtw/internal/buildinfo"
	"github.com/xsyetopz/go-mamusiabtw/internal/config"
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

	args := os.Args[1:]
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
	case "down":
		return runMigrateDown(ctx, runner, cfg.SQLitePath, args[1:])
	default:
		printMigrateUsage()
		return 1
	}
}

func runMigrateDown(ctx context.Context, runner migrate.Runner, dbPath string, args []string) int {
	fs := flag.NewFlagSet("migrate down", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	to := fs.Int("to", -1, "target migration version")
	steps := fs.Int("steps", 0, "number of applied migrations to roll back")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if (*to >= 0 && *steps > 0) || (*to < 0 && *steps <= 0) {
		_, _ = os.Stderr.WriteString("specify exactly one of --to or --steps for migrate down\n")
		return 1
	}

	var (
		status migrate.Status
		err    error
	)
	if *to >= 0 {
		status, err = runner.DownToPath(ctx, dbPath, *to)
	} else {
		status, err = runner.DownStepsPath(ctx, dbPath, *steps)
	}
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}

	printStatus(status)
	return 0
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
			"  mamusiabtw migrate down --to <version>\n" +
			"  mamusiabtw migrate down --steps <n>\n",
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
