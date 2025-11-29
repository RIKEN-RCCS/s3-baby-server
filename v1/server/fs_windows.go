//go:build windows

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// This file is part of conditional builds for filesystem accesses.
// https://pkg.go.dev/cmd/go#hdr-Build_constraints

// This is to take file status atime/ctime/mtime as "syscall".  See
// https://pkg.go.dev/syscall?GOOS=windows

package server

import (
	"io/fs"
	"log"
	"syscall"
	"time"
)

func file_time(info fs.FileInfo) ([3]time.Time, bool) {
	var s, ok = info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		log.Print("fs.FileInfo.Sys() is not windows.")
		return [3]time.Time{}, false
	}
	var atime = time.Unix(0, s.LastAccessTime.Nanoseconds())
	var ctime = time.Unix(0, s.CreationTime.Nanoseconds())
	var mtime = time.Unix(0, s.LastWriteTime.Nanoseconds())
	return [3]time.Time{atime, ctime, mtime}, true
}
