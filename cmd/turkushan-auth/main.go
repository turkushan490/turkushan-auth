package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/turkushan490/turkushan-auth/internal/config"
	"github.com/turkushan490/turkushan-auth/internal/db"
	"github.com/turkushan490/turkushan-auth/internal/server"
	"github.com/turkushan490/turkushan-auth/internal/store"
	"github.com/turkushan490/turkushan-auth/web"
)

func main() {
	// Docker HEALTHCHECK runs the binary itself; the image has no shell or curl.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := dropPrivileges(cfg, log); err != nil {
		return fmt.Errorf("drop privileges: %w", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o750); err != nil {
		return err
	}
	if err := cfg.LoadSecret(); err != nil {
		return fmt.Errorf("session secret: %w", err)
	}

	sqlDB, err := db.Open(filepath.Join(cfg.DataDir, "turkushan-auth.db"))
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := db.Migrate(ctx, sqlDB); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	st := store.New(sqlDB)
	res, err := st.BootstrapAdmin(ctx, cfg.AdminUser, cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("admin bootstrap: %w", err)
	}
	if res == store.BootstrapNoAdmin {
		log.Warn("no admin account yet: set ADMIN_USER and ADMIN_PASSWORD and restart")
	} else {
		log.Info("admin account", "result", string(res))
	}

	dist, err := web.Dist()
	if err != nil {
		return err
	}
	handler, err := server.New(cfg, st, log, dist)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", srv.Addr, "app_url", cfg.AppURL.String())
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func healthcheck() int {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3010"
	}
	c := http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		return 1
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
