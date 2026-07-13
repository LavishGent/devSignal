// Command api runs the DevSignal backend: it loads configuration, connects
// to PostgreSQL, applies pending migrations, and serves the JSON API (plus
// the built frontend when STATIC_DIR is set).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"devsignal/internal/auth"
	"devsignal/internal/config"
	"devsignal/internal/database"
	"devsignal/internal/httpapi"
	"devsignal/internal/repo/postgres"
	"devsignal/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool, migrations.FS, logger); err != nil {
		return err
	}

	store := postgres.NewStore(pool)
	sessions := auth.NewSessionManager(store.Sessions, cfg.SessionTTL)
	base := httpapi.NewBaseHandler(logger)
	authH := httpapi.NewAuthHandler(base, store.Users, sessions, cfg.BcryptCost, cfg.Production())

	server := httpapi.NewServer(logger, cfg.Production(), cfg.StaticDir, authH)
	registerModules(server, base, store)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	logger.Info("devsignal api listening", "port", cfg.Port, "env", cfg.Env)
	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("devsignal api stopped")
	return nil
}
