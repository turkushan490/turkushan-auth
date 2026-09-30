//go:build !linux

package main

import (
	"log/slog"

	"github.com/turkushan490/turkushan-auth/internal/config"
)

func dropPrivileges(*config.Config, *slog.Logger) error { return nil }
