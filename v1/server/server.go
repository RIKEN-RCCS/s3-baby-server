// server.go
// Copyright 2025-2025 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"io/fs"
	"os"
	"path"
	//"path/filepath"
	"time"
	"sync"
)

const Bb_version = "v1.2.1"

type Bb_configuration struct {
	Access_logging            bool
	Anonymize_ower            bool
	Verify_fs_write           bool
	Pending_upload_expiration time.Duration
	Server_controler_path     string

	request_processing_timeout time.Duration

	File_follow_link   bool
	File_creation_mode fs.FileMode
}

type Bb_server struct {
	pool_path string
	Logger    *slog.Logger
	AuthKey   string

	Bb_config Bb_configuration

	rid      int64
	suffixes map[string]suffix_record
	monitor1 *monitor
	mutex    sync.Mutex

	server_quit chan struct{}
}

type prior_handler struct {
	bbs *Bb_server
	sx *http.ServeMux
}

// PRIOR_HANDLER checks an authorization header in a request before
// passing it to actual handlers.
func (sv *prior_handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//var logger = bbs.Logger
	//var authKey = bbs.AuthKey

	//option := newHTTPS3Options(r, logger)
	//if !option.checkAuthorization(r, authKey) {
	//if !option.CheckErrorHeader() {
	//if !option.CheckKeyPath(s3.RootPath, option.GetPath()) {

	fmt.Printf("prior_handler does nothing.\n")
	sv.sx.ServeHTTP(w, r)

	//option.Logger.Error(err.Message, "status code", err.Status)
	//w.Header().Set("Content-Type", "application/xml")
	//w.WriteHeader(err.Status)
	//if err := xml.NewEncoder(w).Encode(err); err != nil {
}

// SERVER_CONTROL handles requests to shutdown.  It is hooked at the
// url "/bbs.ctl".
func (bbs *Bb_server) server_control(w http.ResponseWriter, r *http.Request) {
	var q = r.URL.Query()
	var delete = q.Has("delete")
	if delete {
		log.Fatal("SHUTDOWN")
	}
}

func Start_server(pool_directory, addr, logPath, authKey string) {
	// Run in UTC time zone instead of local time zone.

	time.Local = time.UTC
	var logger = slog.Default()

	logger.Info("Starting server", "address", addr)

	// Change working directory to the pool-directory.  It is to avoid
	// accidentally disclose the full path (it may include a user or
	// project name)

	var wd = path.Clean(pool_directory)
	var err1 = os.Chdir(wd)
	if err1 != nil {
		logger.Info("os.Chdir() failed", "directory", wd, "error", err1)
		return
	}

	var bbs = &Bb_server{pool_path: ".", Logger: logger, AuthKey: authKey}
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
		logger.Info("http.ListenAndServe() returns", "error", err2)
	}
}
