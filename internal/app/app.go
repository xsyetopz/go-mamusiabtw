package app

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/xsyetopz/go-mamusiabtw/internal/config"
	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
	"github.com/xsyetopz/go-mamusiabtw/internal/migrate"
	"github.com/xsyetopz/go-mamusiabtw/internal/ops"
	discordplatform "github.com/xsyetopz/go-mamusiabtw/internal/platform/discord"
	"github.com/xsyetopz/go-mamusiabtw/internal/pluginhost"
	"github.com/xsyetopz/go-mamusiabtw/internal/sqlite"
	"github.com/xsyetopz/go-mamusiabtw/internal/store/sqlitestore"
)

type Dependencies struct {
	Logger *slog.Logger
	Config config.Config
}

type App struct {
	logger *slog.Logger
	cfg    config.Config

	store   *sqlitestore.Store
	i18n    i18n.Registry
	bot     *discordplatform.Bot
	ops     *ops.Server
	metrics *ops.Metrics

	startedAt        time.Time
	migrationVersion int
}

func New(deps Dependencies) (*App, error) {
	if deps.Logger == nil {
		return nil, errors.New("logger is required")
	}
	if deps.Config.ProdMode && deps.Config.AllowUnsignedPlugins {
		return nil, errors.New("prod mode requires signed plugins; set MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=0")
	}

	return &App{
		logger:  deps.Logger,
		cfg:     deps.Config,
		metrics: ops.NewMetrics(),
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	a.startedAt = time.Now()
	if err := a.initStorage(ctx); err != nil {
		return err
	}
	if err := a.validatePluginTrust(ctx); err != nil {
		return err
	}
	if err := a.initI18n(); err != nil {
		return err
	}
	if err := a.initDiscordBot(); err != nil {
		return err
	}
	if err := a.initOpsServer(); err != nil {
		return err
	}
	if a.ops != nil {
		if err := a.ops.Start(); err != nil {
			return err
		}
	}

	if err := a.bot.Start(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (a *App) Close() error {
	if a.ops != nil {
		_ = a.ops.Close(context.Background())
	}
	if a.bot != nil {
		a.bot.Close(context.Background())
	}
	if a.store != nil {
		return a.store.Close()
	}
	return nil
}

func (a *App) initStorage(ctx context.Context) error {
	if a.store != nil {
		return nil
	}

	runner, err := migrate.New(migrate.Options{
		Dir:       a.cfg.Migrations,
		BackupDir: a.cfg.MigrationBackups,
	})
	if err != nil {
		return err
	}
	status, runErr := runner.UpPath(ctx, a.cfg.SQLitePath)
	if runErr != nil {
		return runErr
	}
	a.migrationVersion = status.CurrentVersion

	db, err := sqlite.Open(ctx, sqlite.Options{
		Path: a.cfg.SQLitePath,
	})
	if err != nil {
		return err
	}

	store, err := sqlitestore.New(db)
	if err != nil {
		_ = db.Close()
		return err
	}

	a.store = store
	return nil
}

func (a *App) validatePluginTrust(ctx context.Context) error {
	if !a.cfg.ProdMode || a.cfg.AllowUnsignedPlugins || a.store == nil {
		return nil
	}

	fileKeys := 0
	path := strings.TrimSpace(a.cfg.TrustedKeysFile)
	if path != "" {
		keys, err := pluginhost.ReadTrustedKeysFile(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		fileKeys = len(keys)
	}

	signers, err := a.store.TrustedSigners().ListTrustedSigners(ctx)
	if err != nil {
		return err
	}
	if fileKeys == 0 && len(signers) == 0 {
		return errors.New("prod mode requires at least one trusted signer in MAMUSIABTW_TRUSTED_KEYS_FILE or SQLite")
	}
	return nil
}

func (a *App) initI18n() error {
	reg, err := i18n.LoadCore(a.cfg.LocalesDir)
	if err != nil {
		return err
	}

	a.i18n = reg
	return nil
}

func (a *App) initDiscordBot() error {
	if a.bot != nil {
		return nil
	}
	if a.store == nil {
		return errors.New("store must be initialized before discord bot")
	}

	bot, err := discordplatform.New(discordplatform.Dependencies{
		Logger: a.logger,
		Token:  a.cfg.DiscordToken,

		Owners:                   a.cfg.OwnerUserID,
		DevGuildID:               a.cfg.DevGuildID,
		CommandRegistrationMode:  a.cfg.CommandRegistrationMode,
		CommandGuildIDs:          a.cfg.CommandGuildIDs,
		CommandRegisterAllGuilds: a.cfg.CommandRegisterAllGuilds,
		PluginsDir:               a.cfg.PluginsDir,
		PermissionsFile:          a.cfg.PermissionsFile,
		ModulesFile:              a.cfg.ModulesFile,
		AllowUnsignedPlugins:     a.cfg.AllowUnsignedPlugins,
		ProdMode:                 a.cfg.ProdMode,
		TrustedKeysFile:          a.cfg.TrustedKeysFile,

		SlashCooldown:          a.cfg.SlashCooldown,
		ComponentCooldown:      a.cfg.ComponentCooldown,
		ModalCooldown:          a.cfg.ModalCooldown,
		SlashCooldownBypass:    a.cfg.SlashCooldownBypass,
		SlashCooldownOverrides: a.cfg.SlashCooldownOverrides,

		I18n:    a.i18n,
		Store:   a.store,
		Metrics: a.metrics,
	})
	if err != nil {
		return err
	}

	a.bot = bot
	return nil
}

func (a *App) initOpsServer() error {
	if a.ops != nil || a.cfg.OpsAddr == "" {
		return nil
	}

	server, err := ops.New(a.cfg.OpsAddr, a.logger, a.opsSnapshot)
	if err != nil {
		return err
	}
	a.ops = server
	return nil
}

func (a *App) opsSnapshot() ops.Snapshot {
	snap := ops.Snapshot{
		StartedAt:        a.startedAt,
		MigrationVersion: a.migrationVersion,
		ProdMode:         a.cfg.ProdMode,
	}
	if a.bot == nil {
		if a.metrics != nil {
			a.metrics.FillSnapshot(&snap)
		}
		return snap
	}

	stats := a.bot.Stats()
	snap.Ready = stats.Ready
	snap.ModuleCount = stats.ModuleCount
	snap.EnabledModuleCount = stats.EnabledModuleCount
	snap.PluginCount = stats.PluginCount
	snap.EnabledPluginCount = stats.EnabledPluginCount
	snap.BuiltinCommandCount = stats.BuiltinCommandCount
	snap.SlashCommandCount = stats.SlashCommandCount
	snap.UserCommandCount = stats.UserCommandCount
	snap.MessageCommandCount = stats.MessageCommandCount
	if a.metrics != nil {
		a.metrics.FillSnapshot(&snap)
	}
	return snap
}
