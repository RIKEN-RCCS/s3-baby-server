//go:build linux

// fs_linux.go
// Copyright 2025-2025 RIKEN R-CCS.
// SPDX-License-Identifier: BSD-2-Clause

// This file is part of conditional builds for filesystem accesses.
// https://pkg.go.dev/cmd/go#hdr-Build_constraints

// This is to take file status atime/ctime/mtime as "syscall".  See
// https://pkg.go.dev/syscall?GOOS=linux

package server

import (
	"io/fs"
	"log"
	"syscall"
	"time"
)

func file_time(info fs.FileInfo) ([3]time.Time, bool) {
	var s, ok = info.Sys().(*syscall.Stat_t)
	if !ok {
		log.Print("fs.FileInfo.Sys() is not unix.")
		return [3]time.Time{}, false
	}
	var atime = time.Unix(int64(s.Atim.Sec), int64(s.Atim.Nsec))
	var ctime = time.Unix(int64(s.Ctim.Sec), int64(s.Ctim.Nsec))
	var mtime = time.Unix(int64(s.Mtim.Sec), int64(s.Mtim.Nsec))
	return [3]time.Time{atime, ctime, mtime}, true
}
