package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/xsyetopz/imotherbtw/internal/app"
	"github.com/xsyetopz/imotherbtw/internal/config"
	"github.com/xsyetopz/imotherbtw/internal/logging"
)

func main() {
	os.Exit(runMain())
}

func runMain() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
	imotherbtw, err := app.New(app.Dependencies{
		Logger: logger,
		Config: cfg,
	})
	if err != nil {
		return err
	}
	defer imotherbtw.Close()

	if startErr := imotherbtw.Start(ctx); startErr != nil {
		if errors.Is(startErr, context.Canceled) {
			return nil
		}
		return startErr
	}

	return nil
}
