// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package utils

import (
	"log/slog"
	"os"
	"strings"
)

func RemoveAndMakeDir(path string, logger *slog.Logger) {
	if err := os.RemoveAll(path); err != nil {
		logger.Debug("Failed to delete directory", "path", path)
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		logger.Debug("Failed to create directory", "path", path)
	}
}

func CloseFile(file *os.File, logger *slog.Logger) {
	if err := file.Close(); err != nil {
		logger.Error("", "error", err)
	}
}

func GetDirOnly(entries []os.DirEntry) []os.DirEntry {
	var filtered []os.DirEntry
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".") { // 隠しディレクトリは追加しない
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
