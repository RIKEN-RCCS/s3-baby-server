// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"context"
	"log/slog"
	"os"
)

type CustomHandler struct {
	fileHandler   slog.Handler
	stdoutHandler slog.Handler
}

func (h *CustomHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.fileHandler.Enabled(ctx, level) || h.stdoutHandler.Enabled(ctx, level)
}

func (h *CustomHandler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.stdoutHandler.Handle(ctx, r); err != nil {
		return err
	}
	if r.Level >= slog.LevelInfo {
		if err := h.fileHandler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CustomHandler{
		fileHandler:   h.fileHandler.WithAttrs(attrs),
		stdoutHandler: h.stdoutHandler.WithAttrs(attrs),
	}
}

func (h *CustomHandler) WithGroup(name string) slog.Handler {
	return &CustomHandler{
		fileHandler:   h.fileHandler.WithGroup(name),
		stdoutHandler: h.stdoutHandler.WithGroup(name),
	}
}

func Init(logFilePath string) *slog.Logger {
	file, _ := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
	fileHandler := slog.NewTextHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo})
	stdoutHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler := &CustomHandler{
		fileHandler:   fileHandler,
		stdoutHandler: stdoutHandler,
	}
	return slog.New(handler)
}
