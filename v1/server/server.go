// server.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// S3-Baby-Server main.  Call Start_server().

package server

import (
	//"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"fmt"
	"path"
	"github.com/riken-rccs/s3-baby-server/pkg/awss3aide"
	"github.com/riken-rccs/s3-baby-server/pkg/httpaide"
	"strings"
	//"strconv"
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

	File_creation_mode fs.FileMode

	Site_base_url *string

	request_processing_timeout time.Duration
}

type Bb_server struct {
	server *http.Server
	keypair   [2]string
	logger    *slog.Logger

	conf Bb_configuration

	rid_past      int64
	suffixes map[string]suffix_record
	monitor1 *monitor
	mutex    sync.Mutex

	server_quit chan struct{}

	// POOL_PATH is a path passed to the server command.  Baby-server
	// changes working directory to that path, and this is only a
	// record.
	pool_path string
}

// PRIOR_HANDLER is an http.Handler and it checks an authorization
// header in a request before passing it to actual handlers.  See its
// ServeHTTP() method.
type prior_handler struct {
	bbs *Bb_server
	sx  *http.ServeMux
}

func (sv *prior_handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var auth, err1 = sv.bbs.attest_authorization(w, r)
	if err1 != nil {
		return
	}
	var w2 = &httpaide.ResponseWriter2{ResponseWriter: w}
	sv.sx.ServeHTTP(w2, r)

	{
		var user = auth[:min(len(auth), 16)]
		var code = w2.Status_code
		var length = w2.Content_length
		var m = httpaide.Log_access(r, code, length, user)
		fmt.Printf("%s\n", m)
	}
}

func Start_server(pool_directory, addr, logPath, authKey string) {
	// Run in UTC time zone instead of local time zone.

	time.Local = time.UTC

	//var logger = slog.Default()
	var loglevel = new(slog.LevelVar)
	loglevel.Set(slog.LevelDebug)
	var logger = slog.New(slog.NewTextHandler(os.Stdout,
		&slog.HandlerOptions{Level: loglevel}))

	logger.Info("Starting baby-server", "address", addr)

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

	var bbs = &Bb_server{pool_path: wd, logger: logger, keypair: keypair}
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

	bbs.server = &http.Server{Addr: addr, Handler: sv}
	//var err2 = http.ListenAndServe(addr, sv)
	var err2 = bbs.server.ListenAndServe()
	if err2 != nil {
		bbs.logger.Info("http.ListenAndServe() returns", "error", err2)
	}
}

// SERVER_CONTROL handles requests to shutdown.  It is hooked at
// "POST_/bbs.ctl/".
func (bbs *Bb_server) server_control(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	var q = r.URL.Query()
	if q.Has("quit") {
		bbs.logger.Info("Shutdown requested")
		var err1 = bbs.server.Shutdown(ctx)
		if err1 != nil {
			bbs.logger.Info("Shutdown failed", "error", err1)
			log.Fatal("SHUTDOWN FORCED")
		}
	}
}

func (bbs *Bb_server) attest_authorization(w http.ResponseWriter, r *http.Request) (string, error) {
	var keypair = bbs.keypair
	var key, reason = awss3aide.Check_credential_is_okay(r, keypair)
	if reason != nil {
		bbs.logger.Info("Bad authorization", "key", key, "reason", reason)
		time.Sleep(1 * time.Second)
		http.Error(w, "Bad authorization", 401)
	}
	return key, reason
}
