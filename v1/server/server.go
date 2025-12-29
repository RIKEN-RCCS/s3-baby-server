// server.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// S3-Baby-Server main.  The main entry is Start_server().

package server

import (
	"encoding/json"
	"fmt"
	"github.com/riken-rccs/s3-baby-server/pkg/awss3aide"
	"github.com/riken-rccs/s3-baby-server/pkg/httpaide"
	//"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	//"strconv"
	"sync"
	"time"
)

const Bb_version = "v1.2.1"

type Bb_server struct {
	server    *http.Server
	cert_pair [2]string
	cred_pair [2]string
	config    Bb_configuration
	logger    *slog.Logger

	rid_past uint64
	suffixes map[string]suffix_record
	monitor1 *monitor
	mutex    sync.Mutex

	server_quit chan struct{}

	// POOL_PATH is a path where buckets reside.  Baby-server changes
	// the working directory to that path, and this is only a record.
	pool_path string
}

type Bb_configuration struct {
	// Access_logging bool
	// Pending_upload_expiration time.Duration

	Server_control_path     string
	Site_base_url           *string
	Verify_fs_write         bool
	Limit_of_xml_parameters int64

	// Anonymize_ower bool
	// File_creation_mode fs.FileMode

	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
}

// HANDLER_DATA is a record of handler context.
type Handler_data struct {
	Request_id     uint64
	Action_name    string
	ResponseWriter http.ResponseWriter
	Request        *http.Request
}

// H_LIMIT_OF_XML_PARAMETERS limits the size of XML parameters
// in a request body.
var h_limit_of_xml_parameters int64 = (2 * 1024 * 1024)

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

func Start_server(pool_directory, addr, cred, cert, conf, logs string) {

	// Run in UTC time zone instead of local time zone.

	time.Local = time.UTC

	var loglevel = new(slog.LevelVar)
	switch logs {
	case "debug":
		loglevel.Set(slog.LevelDebug)
	case "info":
		loglevel.Set(slog.LevelInfo)
	case "warn":
		loglevel.Set(slog.LevelWarn)
	default:
		loglevel.Set(slog.LevelInfo)
	}
	var logger = slog.New(slog.NewTextHandler(os.Stdout,
		&slog.HandlerOptions{Level: loglevel}))

	var credpair [2]string
	{
		var access, secret, ok = strings.Cut(cred, ",")
		if !ok || len(access) == 0 || len(secret) == 0 {
			logger.Info("Bad authorization key pair", "pair", cred)
			return
		}
		credpair = [2]string{access, secret}
	}

	var certpair [2]string
	if cert != "" {
		var crt, key, ok = strings.Cut(cert, ",")
		if !ok || len(crt) == 0 || len(key) == 0 {
			logger.Error("Bad certificate and key pair for https",
				"pair", cert)
			return
		}
		certpair = [2]string{crt, key}
	}

	// Set default configurations, then read a configuration file.

	var config = Bb_configuration{
		Server_control_path: "bbs.ctl",
	}
	if conf != "" {
		var f1, err1 = os.Open(conf)
		if err1 != nil {
			logger.Error("os.Open() failed for conf",
				"path", conf, "error", err1)
			return
		}
		var d = json.NewDecoder(f1)
		var err2 = d.Decode(&config)
		if err2 != nil {
			logger.Error("json.Decode() failed",
				"path", conf, "error", err2)
			return
		}
		var err3 = f1.Close()
		if err3 != nil {
			logger.Error("op.Close() failed",
				"path", conf, "error", err3)
			return
		}
	}

	// Change the working directory to the pool-directory.  It is to
	// avoid accidentally disclosing the full path (which may include
	// a user name or a project name)

	var wd = path.Clean(pool_directory)
	var err1 = os.Chdir(wd)
	if err1 != nil {
		logger.Info("os.Chdir() failed", "directory", wd, "error", err1)
		return
	}

	var bbs = &Bb_server{
		pool_path: wd,
		cert_pair: certpair,
		cred_pair: credpair,
		config:    config,
		logger:    logger}
	bbs.suffixes = make(map[string]suffix_record)
	bbs.server_quit = make(chan struct{})
	bbs.monitor1 = new_monitor()
	go bbs.monitor1.guard_loop()

	if bbs.config.Limit_of_xml_parameters != 0 {
		h_limit_of_xml_parameters = bbs.config.Limit_of_xml_parameters
	}

	var control = "POST /" + config.Server_control_path + "/{$}"
	var sx = http.NewServeMux()
	sx.HandleFunc(control, func(w http.ResponseWriter, r *http.Request) {
		bbs.server_control(w, r)
	})
	register_dispatcher(bbs, sx)
	var sv = &prior_handler{bbs, sx}

	bbs.server = &http.Server{
		Addr:     addr,
		Handler:  sv,
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),

		ReadTimeout:       bbs.config.ReadTimeout,
		ReadHeaderTimeout: bbs.config.ReadHeaderTimeout,
		WriteTimeout:      bbs.config.WriteTimeout,
		IdleTimeout:       bbs.config.IdleTimeout,
		MaxHeaderBytes:    bbs.config.MaxHeaderBytes,
	}

	var proto string
	if certpair[0] != "" {
		proto = "(https)"
	} else {
		proto = "(http)"
	}
	logger.Info(("Starting Baby-server " + proto), "address", addr,
		"access-key", credpair[0], "version", Bb_version)

	var err2 error
	if certpair[0] != "" {
		err2 = bbs.server.ListenAndServeTLS(certpair[0], certpair[1])
	} else {
		err2 = bbs.server.ListenAndServe()
	}
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
	var key, reason = awss3aide.Check_credential_is_okay(r, bbs.cred_pair)
	if reason != nil {
		bbs.logger.Info("Bad authorization", "key", key, "reason", reason)
		time.Sleep(1 * time.Second)
		http.Error(w, "Bad authorization", 401)
	}
	return key, reason
}
