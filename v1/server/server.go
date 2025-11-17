// server.go
// Copyright 2025-2025 RIKEN R-CCS.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"
)

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

func Start(basePath, addr, logPath, authKey string) {
	var logger = slog.Default()

	// Run in UTC time zone instead of local time zone.

	time.Local = time.UTC

	// Convert a path to platform specific one.

	//var basepath1 = file.Clean(basePath)
	var basepath2 = filepath.Clean(basePath)

	// Change working directory to pool-directory.

	//r := mux.NewRouter()
	//r.Use(PanicRecovery)
	var sx = http.NewServeMux()

	logger.Info("Starting server", "address", addr)
	logger.Debug("options", "authKey", authKey)

	var bbs = Bb_server{pool_path: basepath2, Logger: logger, AuthKey: authKey}
	bbs.suffixes = make(map[string]suffix_record)
	bbs.server_quit = make(chan struct{})
	bbs.monitor1 = new_monitor()
	go bbs.monitor1.guard_loop()

	sx.HandleFunc("POST /bbs.ctl/{$}", func(w http.ResponseWriter, r *http.Request) {
		bbs.server_control(w, r)
	})

	register_dispatcher(&bbs, sx)
	var sv = prior_handler{&bbs, sx}
	var err1 = http.ListenAndServe(addr, &sv)
	if err1 != nil {
		logger.Error("", "error", err1)
	}
}
