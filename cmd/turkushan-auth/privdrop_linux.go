//go:build linux

package main

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"

	"github.com/turkushan490/turkushan-auth/internal/config"
)

// dropPrivileges hands the data folder to PUID:PGID and switches to that user.
// The container starts as root so it can fix ownership of an appdata folder that
// Docker created as root; after this nothing runs as root.
func dropPrivileges(cfg *config.Config, log *slog.Logger) error {
	if os.Getuid() != 0 || cfg.PUID == 0 {
		return nil
	}
	if err := os.MkdirAll(cfg.DataDir, 0o750); err != nil {
		return err
	}
	err := filepath.WalkDir(cfg.DataDir, func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Lchown(p, cfg.PUID, cfg.PGID)
	})
	if err != nil {
		return err
	}
	if err := syscall.Setgroups([]int{}); err != nil {
		return err
	}
	if err := syscall.Setgid(cfg.PGID); err != nil {
		return err
	}
	if err := syscall.Setuid(cfg.PUID); err != nil {
		return err
	}
	log.Info("running as", "uid", cfg.PUID, "gid", cfg.PGID)
	return nil
}
