// Command server runs the marketplace API.
//
//	server serve              migrate, seed (unless SEED_ON_BOOT=false) and listen
//	server migrate            apply pending migrations and exit
//	server seed               seed demo data and exit
//	server seed:demo-orders   seed only the back-dated demo orders
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

	"github.com/asihat/b2b-marketplace/backend/internal/config"
	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/seed"
	"github.com/asihat/b2b-marketplace/backend/internal/server"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if err := run(cmd); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(cmd string) error {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL, time.Duration(cfg.DBWaitSecs)*time.Second)
	if err != nil {
		return err
	}
	defer pool.Close()
	app := services.NewApp(cfg, pool)

	switch cmd {
	case "migrate":
		return db.Migrate(ctx, pool)
	case "seed":
		if err := db.Migrate(ctx, pool); err != nil {
			return err
		}
		return seed.Run(ctx, app)
	case "seed:demo-orders":
		return seed.RunDemoOrders(ctx, app)
	case "serve":
		if err := db.Migrate(ctx, pool); err != nil {
			return err
		}
		if cfg.SeedOnBoot {
			if err := seed.Run(ctx, app); err != nil {
				// Like the former `db:seed || true`: a seeding problem must not keep the API down.
				slog.Error("seeding failed", "error", err)
			}
			app.FlushCaches()
		}
		return serve(ctx, cfg, app)
	default:
		return fmt.Errorf("unknown command %q (expected serve, migrate, seed or seed:demo-orders)", cmd)
	}
}

func serve(ctx context.Context, cfg config.Config, app *services.App) error {
	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:           server.NewRouter(app),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr, "app_url", cfg.AppURL)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
