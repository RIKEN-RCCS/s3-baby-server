//go:build unix && !linux
// fs_linux.go
// Copyright 2025-2025 RIKEN R-CCS.
// SPDX-License-Identifier: BSD-2-Clause

// Name "_unix.go" is not a proper build constrant on file names.

// This file is part of conditional builds for filesystem accesses.
// https://pkg.go.dev/cmd/go#hdr-Build_constraints

// This is to take file status atime/ctime/mtime as "syscall".  See
// https://pkg.go.dev/syscall?GOOS=darwin
// (No documents found specific to *bsd nor solaris nor illumos).

package server

import (
	"log"
	"io/fs"
	"syscall"
	"time"
)

func file_time(info fs.FileInfo) ([3]time.Time, bool) {
	var s, ok = info.Sys().(*syscall.Stat_t)
	if !ok {
		log.Print("fs.FileInfo.Sys() is not unix.")
		return [3]time.Time{}, false
	}
	var atime = time.Unix(int64(s.Atimespec.Sec), int64(s.Atimespec.Nsec))
	var ctime = time.Unix(int64(s.Ctimespec.Sec), int64(s.Ctimespec.Nsec))
	var mtime = time.Unix(int64(s.Mtimespec.Sec), int64(s.Mtimespec.Nsec))
	return [3]time.Time{atime, ctime, mtime}, true
}
