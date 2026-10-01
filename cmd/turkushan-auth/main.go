package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/turkushan490/turkushan-auth/internal/auth"
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

	// Recovery when nobody can log in anymore:
	//   docker exec turkushan-auth /turkushan-auth reset-password <username> <new password>
	if len(os.Args) > 1 && os.Args[1] == "reset-password" {
		if err := resetPassword(log, os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "reset-password:", err)
			os.Exit(1)
		}
		return
	}
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
	if res == store.BootstrapExisting && cfg.AdminPassword != "" {
		log.Info("ADMIN_PASSWORD is only used to create the admin account; you can remove it from the container settings")
	}

	go cleanupSessions(ctx, st, log)

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
		ReadTimeout:       60 * time.Second, // leaves room for an 8 MB background upload on a slow line
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

// resetPassword sets a new password for a user, lifts any lock or block and
// signs them out everywhere. It runs next to the live server on the same database.
func resetPassword(log *slog.Logger, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: reset-password <username> <new password>")
	}
	username := auth.NormalizeUsername(args[0])
	if err := auth.ValidatePassword(args[1]); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	// docker exec runs as root; switch to the app user so no root-owned files end up in /data.
	if err := dropPrivileges(cfg, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		return err
	}
	sqlDB, err := db.Open(filepath.Join(cfg.DataDir, "turkushan-auth.db"))
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	ctx := context.Background()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		return err
	}
	st := store.New(sqlDB)
	u, err := st.UserByUsername(ctx, username)
	if errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("no user %q", username)
	}
	if err != nil {
		return err
	}
	hash, err := auth.HashPassword(args[1])
	if err != nil {
		return err
	}
	if err := st.SetPassword(ctx, u.ID, hash); err != nil {
		return err
	}
	if err := st.SetUserStatus(ctx, u.ID, "active"); err != nil {
		return err
	}
	if err := st.Audit(ctx, "system", "user.password_reset", u.Username, "reset-password command", ""); err != nil {
		return err
	}
	fmt.Printf("Password for %s changed. They can sign in now.\n", u.Username)
	return nil
}

func cleanupSessions(ctx context.Context, st *store.Store, log *slog.Logger) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n, err := st.DeleteExpiredSessions(ctx)
			if err != nil {
				log.Error("session cleanup", "err", err)
			} else if n > 0 {
				log.Info("expired sessions removed", "count", n)
			}
			if err := st.DeleteExpiredTokens(ctx); err != nil {
				log.Error("token cleanup", "err", err)
			}
			if err := st.DeleteExpiredPending(ctx); err != nil {
				log.Error("pending sign-up cleanup", "err", err)
			}
		}
	}
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
