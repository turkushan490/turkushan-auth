package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/turkushan490/turkushan-auth/internal/auth"
	"github.com/turkushan490/turkushan-auth/internal/config"
	"github.com/turkushan490/turkushan-auth/internal/db"
	"github.com/turkushan490/turkushan-auth/internal/server"
	"github.com/turkushan490/turkushan-auth/internal/store"
	"github.com/turkushan490/turkushan-auth/web"
)

// version is set at build time: -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() {
	// Docker HEALTHCHECK runs the binary itself; the image has no shell or curl.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}

	log := newLogger()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version":
			fmt.Println(version)
			return
		case "reset-password":
			// Recovery when nobody can log in anymore:
			//   docker exec -it turkushan-auth /turkushan-auth reset-password <username>
			if err := resetPassword(os.Args[2:]); err != nil {
				fmt.Fprintln(os.Stderr, "reset-password:", err)
				os.Exit(1)
			}
			return
		case "backup":
			//   docker exec turkushan-auth /turkushan-auth backup
			if err := backup(os.Args[2:]); err != nil {
				fmt.Fprintln(os.Stderr, "backup:", err)
				os.Exit(1)
			}
			return
		}
	}
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// newLogger follows LOG_LEVEL (debug, info, warn, error) and LOG_FORMAT (text, json).
func newLogger() *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(os.Getenv("LOG_LEVEL"))); err != nil {
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}
	if strings.EqualFold(os.Getenv("LOG_FORMAT"), "json") {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.Version = version
	log.Info("turkushan-auth starting", "version", version)
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

// openStore opens the live database for a command run next to the server
// (docker exec). docker exec runs as root, so it first switches to the app
// user: no root-owned files end up in /data.
func openStore(ctx context.Context) (*store.Store, *config.Config, func(), error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, nil, err
	}
	if err := dropPrivileges(cfg, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		return nil, nil, nil, err
	}
	sqlDB, err := db.Open(filepath.Join(cfg.DataDir, "turkushan-auth.db"))
	if err != nil {
		return nil, nil, nil, err
	}
	if err := db.Migrate(ctx, sqlDB); err != nil {
		sqlDB.Close()
		return nil, nil, nil, err
	}
	return store.New(sqlDB), cfg, func() { sqlDB.Close() }, nil
}

// backup writes a consistent copy of the database, by default to
// /data/backups/turkushan-auth-<date>.db, while the portal keeps running.
func backup(args []string) error {
	if len(args) > 1 {
		return errors.New("usage: backup [file]")
	}
	ctx := context.Background()
	st, cfg, done, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer done()
	path := ""
	if len(args) == 1 {
		path = args[0]
	} else {
		dir := filepath.Join(cfg.DataDir, "backups")
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return err
		}
		path = filepath.Join(dir, "turkushan-auth-"+time.Now().Format("2006-01-02-150405")+".db")
	}
	if err := st.Backup(ctx, path); err != nil {
		return err
	}
	fmt.Println("Backup written to", path)
	return nil
}

// resetPassword sets a new password for a user, lifts any lock or block and
// signs them out everywhere. It runs next to the live server on the same database.
// Without a password argument it asks for one, so it stays out of the shell history.
func resetPassword(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return errors.New("usage: reset-password <username> [new password]  (leave the password out to type it)")
	}
	username := auth.NormalizeUsername(args[0])
	password := ""
	if len(args) == 2 {
		password = args[1]
	} else {
		fmt.Fprint(os.Stderr, "New password: ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return errors.New("no password given (use docker exec -it)")
		}
		password = strings.TrimRight(line, "\r\n")
	}
	if err := auth.ValidatePassword(password); err != nil {
		return err
	}
	ctx := context.Background()
	st, _, done, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer done()
	u, err := st.UserByUsername(ctx, username)
	if errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("no user %q", username)
	}
	if err != nil {
		return err
	}
	hash, err := auth.HashPassword(password)
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
	clean := func() {
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
	clean() // right away too, so a container that restarts often still cleans up
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			clean()
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
