package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/xsyetopz/imotherbtw/internal/config"
	"github.com/xsyetopz/imotherbtw/internal/discordapp"
	"github.com/xsyetopz/imotherbtw/internal/i18n"
	"github.com/xsyetopz/imotherbtw/internal/migrate"
	"github.com/xsyetopz/imotherbtw/internal/sqlite"
	"github.com/xsyetopz/imotherbtw/internal/store/sqlitestore"
)

type Dependencies struct {
	Logger *slog.Logger
	Config config.Config
}

type App struct {
	logger *slog.Logger
	cfg    config.Config

	store *sqlitestore.Store
	i18n  i18n.Registry
	bot   *discordapp.Bot
}

func New(deps Dependencies) (*App, error) {
	if deps.Logger == nil {
		return nil, errors.New("logger is required")
	}

	return &App{
		logger: deps.Logger,
		cfg:    deps.Config,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	if err := a.initStorage(ctx); err != nil {
		return err
	}
	if err := a.initI18n(); err != nil {
		return err
	}
	if err := a.initDiscordBot(); err != nil {
		return err
	}

	if err := a.bot.Start(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (a *App) Close() error {
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

	db, err := sqlite.Open(ctx, sqlite.Options{
		Path: a.cfg.SQLitePath,
	})
	if err != nil {
		return err
	}

	runner, err := migrate.New(a.cfg.Migrations)
	if err != nil {
		_ = db.Close()
		return err
	}
	if runErr := runner.Run(ctx, db); runErr != nil {
		_ = db.Close()
		return runErr
	}

	store, err := sqlitestore.New(db)
	if err != nil {
		_ = db.Close()
		return err
	}

	a.store = store
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

	bot, err := discordapp.New(discordapp.Dependencies{
		Logger: a.logger,
		Token:  a.cfg.DiscordToken,
		Kawaii: discordapp.KawaiiConfig{Token: a.cfg.KawaiiToken},

		Owners:                   a.cfg.OwnerUserID,
		DevGuildID:               a.cfg.DevGuildID,
		CommandRegistrationMode:  a.cfg.CommandRegistrationMode,
		CommandGuildIDs:          a.cfg.CommandGuildIDs,
		CommandRegisterAllGuilds: a.cfg.CommandRegisterAllGuilds,
		PluginsDir:               a.cfg.PluginsDir,
		PermissionsFile:          a.cfg.PermissionsFile,
		AllowUnsignedPlugins:     a.cfg.AllowUnsignedPlugins,
		ProdMode:                 a.cfg.ProdMode,
		TrustedKeysFile:          a.cfg.TrustedKeysFile,

		SlashCooldown:       a.cfg.SlashCooldown,
		ComponentCooldown:   a.cfg.ComponentCooldown,
		ModalCooldown:       a.cfg.ModalCooldown,
		SlashCooldownBypass: a.cfg.SlashCooldownBypass,

		I18n:  a.i18n,
		Store: a.store,
	})
	if err != nil {
		return err
	}

	a.bot = bot
	return nil
}
