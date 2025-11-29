// server.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path"
	"github.com/riken-rccs/s3-baby-server/pkg/awss3aide"
	"strings"
	"sync"
	"time"
)

const Bb_version = "v1.2.1"

type Bb_configuration struct {
	Access_logging            bool
	Pending_upload_expiration time.Duration
	Server_control_path       string

	// Anonymize_ower            bool
	Verify_fs_write bool
	// File_follow_link   bool

	request_processing_timeout time.Duration

	File_creation_mode fs.FileMode

	Site_base_url *string
}

type Bb_server struct {
	pool_path string
	Logger    *slog.Logger
	AuthKey   string
	keypair   [2]string

	conf Bb_configuration

	rid      int64
	suffixes map[string]suffix_record
	monitor1 *monitor
	mutex    sync.Mutex

	server_quit chan struct{}
}

// PRIOR_HANDLER is an http.Handler and it checks an authorization
// header in a request before passing it to actual handlers.  See its
// ServeHTTP() method.
type prior_handler struct {
	bbs *Bb_server
	sx  *http.ServeMux
}

func (sv *prior_handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("prior_handler does nothing.\n")
	var err1 = sv.bbs.check_authorization_header(w, r)
	if err1 != nil {
		return
	}
	sv.sx.ServeHTTP(w, r)
}

func Start_server(pool_directory, addr, logPath, authKey string) {
	// Run in UTC time zone instead of local time zone.

	time.Local = time.UTC
	var logger = slog.Default()

	logger.Info("Starting server", "address", addr)

	var access, secret, ok = strings.Cut(authKey, ",")
	if !ok || len(access) == 0 || len(secret) == 0 {
		logger.Info("Bad authentication key pair", "pair", authKey)
		return
	}
	var keypair = [2]string{access, secret}

	// Change working directory to the pool-directory.  It is to avoid
	// accidentally disclose the full path (it may include a user or
	// project name)

	var wd = path.Clean(pool_directory)
	var err1 = os.Chdir(wd)
	if err1 != nil {
		logger.Info("os.Chdir() failed", "directory", wd, "error", err1)
		return
	}

	var bbs = &Bb_server{pool_path: ".", Logger: logger, keypair: keypair}
	bbs.suffixes = make(map[string]suffix_record)
	bbs.server_quit = make(chan struct{})
	bbs.monitor1 = new_monitor()
	go bbs.monitor1.guard_loop()

	var sx = http.NewServeMux()
	sx.HandleFunc("POST /bbs.ctl/{$}", func(w http.ResponseWriter, r *http.Request) {
		bbs.server_control(w, r)
	})

	register_dispatcher(bbs, sx)
	var sv = &prior_handler{bbs, sx}
	var err2 = http.ListenAndServe(addr, sv)
	if err2 != nil {
		bbs.Logger.Info("http.ListenAndServe() returns", "error", err2)
	}
}

// SERVER_CONTROL handles requests to shutdown.  It is hooked at
// "POST_/bbs.ctl/".
func (bbs *Bb_server) server_control(w http.ResponseWriter, r *http.Request) {
	var q = r.URL.Query()
	if q.Has("shutdown") {
		log.Fatal("SHUTDOWN")
	}
}

func (bbs *Bb_server) check_authorization_header(w http.ResponseWriter, r *http.Request) error {
	var keypair = bbs.keypair
	var ok, _ = awss3aide.Check_credential_is_okay(r, keypair)
	if !ok {
		time.Sleep(1 * time.Second)
		http.Error(w, "msg", 401)
	}
	return nil
}
