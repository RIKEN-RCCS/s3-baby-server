// server.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// S3-Baby-Server main.  The main entry is Start_server().

package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
	//runtimepprof "runtime/pprof"

	"github.com/riken-rccs/s3-baby-server/pkg/awss3aide"
	"github.com/riken-rccs/s3-baby-server/pkg/httpaide"
)

const Bb_version = "v1.2.1"

type Bb_server struct {
	server    *http.Server
	cert_pair [2]string
	cred_pair [2]string
	config    Bb_configuration
	logger    *slog.Logger

	access_logging *os.File

	rid_past uint64
	suffixes map[string]suffix_record
	monitor1 *monitor
	mutex    sync.Mutex

	server_quit chan struct{}

	// POOL_PATH is a path where buckets reside.  Baby-server changes
	// the working directory to that path, and this is only a record.
	pool_path string
}

type msec_duration int64

func time_duration(v msec_duration) time.Duration {
	return time.Duration(v) * time.Millisecond
}

// BB_CONFIGURATION is the configuration.  It may be loaded from a
// specified file.  Parameters from "ReadTimeout" to "MaxHeaderBytes"
// are set to Golang's http.Server.  Time values are in msec duration,
// because time.Duration are in large numbers that are not an
// appropriate representation in a configuration file.
type Bb_configuration struct {
	Server_control_path     string        `json:"server_control_path"`
	Site_base_url           *string       `json:"site_base_url"`
	Exclusion_wait          msec_duration `json:"exclusion_wait"`
	Record_etag_threshold   int64         `json:"record_etag_threshold"`
	Limit_of_xml_parameters int64         `json:"limit_of_xml_parameters"`
	Verify_fs_write         bool          `json:"verify_fs_write"`

	// Anonymize_ower bool
	// File_creation_mode fs.FileMode

	ReadTimeout       msec_duration `json:"ReadTimeout"`
	ReadHeaderTimeout msec_duration `json:"ReadHeaderTimeout"`
	WriteTimeout      msec_duration `json:"WriteTimeout"`
	IdleTimeout       msec_duration `json:"IdleTimeout"`
	MaxHeaderBytes    int           `json:"MaxHeaderBytes"`
}

// HANDLER_DATA is a record of handler context.
type Handler_data struct {
	Request_id     uint64
	ResponseWriter http.ResponseWriter
	Request        *http.Request
}

// H_LIMIT_OF_XML_PARAMETERS limits the size of XML parameters
// in a request body.
var h_limit_of_xml_parameters int64 = (2 * 1024 * 1024)

// PRIOR_HANDLER is an http.Handler and it checks an authorization
// header in a request before passing it to actual handlers.  It also
// prints access logs.  See its ServeHTTP() method.
type prior_handler struct {
	bbs *Bb_server
	sx  *http.ServeMux
}

func (sv *prior_handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rid = sv.bbs.make_request_id()
	var start_time = time.Now()

	var auth, err1 = sv.bbs.attest_authorization(w, r)
	if err1 != nil {
		return
	}

	var w2 = &httpaide.ResponseWriter2{ResponseWriter: w}
	var ctx1 = r.Context()
	var frame = &Handler_data{
		Request_id:     rid,
		ResponseWriter: w,
		Request:        r}
	var ctx2 = context.WithValue(ctx1, "handler-data", frame)
	var r2 = r.WithContext(ctx2)

	// Call the service dispatcher.

	sv.sx.ServeHTTP(w2, r2)

	var user = auth[:min(len(auth), 16)]
	var code = w2.Status_code
	var length = w2.Content_length

	var elapse_time = time.Since(start_time)
	var request = fmt.Sprintf("%s %s", r.Method, r.URL)
	sv.bbs.logger.Debug("Handling time",
		"request", request, "rid", rid, "code", code, "length", length,
		"elapse", elapse_time)

	if sv.bbs.access_logging != nil {
		var m = httpaide.Log_access(r, code, length, user)
		fmt.Fprintf(sv.bbs.access_logging, "%s\n", m)
	}
}

func Start_server(dump_conf bool, cred, cert [2]string, pool_directory, addr, conf, logs string, loga bool, prof int) {

	// Run in GMT time zone instead of the local time zone.  The time
	// format RFC-1123 requires GMT, and time.UTC does not work here.

	// time.Local = time.UTC
	time.Local = time.FixedZone("GMT", 0)

	// Create logger.

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

	// Set default configurations, or read it from a file.

	var config = Bb_configuration{
		Server_control_path:     "bbs.ctl",
		Site_base_url:           nil,
		Exclusion_wait:          100,
		Record_etag_threshold:   20,
		Limit_of_xml_parameters: 2,
		Verify_fs_write:         false,
	}

	if conf != "" {
		var path = filepath.Clean(conf)
		var f1, err1 = os.Open(path)
		if err1 != nil {
			logger.Error("os.Open() failed for conf",
				"path", path, "error", err1)
			return
		}
		defer func() {
			var err3 = f1.Close()
			if err3 != nil {
				// IGNORE-ERRORS.
			}
		}()
		var d = json.NewDecoder(f1)
		var err2 = d.Decode(&config)
		if err2 != nil {
			logger.Error("json.Decode() failed",
				"path", path, "error", err2)
			return
		}
		var err3 = f1.Close()
		if err3 != nil {
			logger.Error("op.Close() failed",
				"path", path, "error", err3)
			return
		}
	}

	// Dump configuration and exit.

	if dump_conf {
		var e = json.NewEncoder(os.Stdout)
		e.SetIndent("", "  ")
		var err4 = e.Encode(&config)
		if err4 != nil {
			logger.Error("json.Encoder.Encode() failed",
				"error", err4)
			return
		}
		return
	}

	// Change the working directory to the pool-directory.  It is to
	// avoid accidentally disclosing the full path (which may include
	// a user name or a project name)

	var wd = filepath.Clean(pool_directory)
	var err1 = os.Chdir(wd)
	if err1 != nil {
		logger.Info("os.Chdir() failed", "directory", wd, "error", err1)
		return
	}

	// Check the directory to store access logs.

	var access_logging *os.File
	{
		var dir1 = ".s3bbs"
		var dir2 = "log"
		var path1 = filepath.Join(".", dir1, dir2)
		var _, err1 = os.Lstat(path1)
		if err1 != nil && !errors.Is(err1, fs.ErrNotExist) {
			logger.Info("os.Lstat() failed in checking .s3bbs/log",
				"path", path1, "error", err1)
			return
		}
		if err1 == nil {
			var path2 = filepath.Join(".", dir1, dir2, "access-log")
			var oappend = os.O_APPEND | os.O_CREATE | os.O_WRONLY
			var f, err2 = os.OpenFile(path2, oappend, 0644)
			if err2 != nil {
				logger.Info("os.OpenFile() failed for access-log",
					"path", path2, "error", err2)
				return
			}
			access_logging = f
		} else if loga {
			access_logging = os.Stdout
		}
	}

	var bbs = &Bb_server{
		pool_path:      wd,
		cert_pair:      cert,
		cred_pair:      cred,
		config:         config,
		logger:         logger,
		access_logging: access_logging}
	bbs.suffixes = make(map[string]suffix_record)
	bbs.server_quit = make(chan struct{})
	bbs.monitor1 = new_monitor()
	go bbs.monitor1.guard_loop()

	h_limit_of_xml_parameters = (config.Limit_of_xml_parameters * 1024 * 1024)

	var sx = http.NewServeMux()
	var control = "POST /" + config.Server_control_path + "/{command}"
	sx.HandleFunc(control, func(w http.ResponseWriter, r *http.Request) {
		bbs.server_control(w, r)
	})
	register_dispatcher(bbs, sx)
	var sv = &prior_handler{bbs, sx}

	bbs.server = &http.Server{
		Addr:     addr,
		Handler:  sv,
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),

		ReadTimeout:       time_duration(bbs.config.ReadTimeout),
		ReadHeaderTimeout: time_duration(bbs.config.ReadHeaderTimeout),
		WriteTimeout:      time_duration(bbs.config.WriteTimeout),
		IdleTimeout:       time_duration(bbs.config.IdleTimeout),
		MaxHeaderBytes:    bbs.config.MaxHeaderBytes,
	}

	if prof != 0 {
		go service_profiler(logger, prof)
	}

	var proto string
	if cert[0] != "" {
		proto = "(https)"
	} else {
		proto = "(http)"
	}
	logger.Info(("Starting Baby-server " + proto), "address", addr,
		"access-key", cred[0], "version", Bb_version)

	var err2 error
	if cert[0] != "" {
		err2 = bbs.server.ListenAndServeTLS(cert[0], cert[1])
	} else {
		err2 = bbs.server.ListenAndServe()
	}
	if err2 != nil {
		bbs.logger.Warn("http.ListenAndServe() returns", "error", err2)
	}
}

func get_action_name(ctx context.Context) (string, uint64) {
	var action = ctx.Value("action-name").(*string)
	if action == nil {
		log.Fatal("BAD-IMPL: action-name not set")
		return "", 0
	}
	var frame = ctx.Value("handler-data").(*Handler_data)
	if frame == nil {
		log.Fatal("BAD-IMPL: handler-data not set")
		return "", 0
	}
	return *action, frame.Request_id
}

func get_handler_arguments(ctx context.Context) (http.ResponseWriter, *http.Request) {
	var frame = ctx.Value("handler-data").(*Handler_data)
	if frame == nil {
		log.Fatal("BAD-IMPL: handler-data not set")
		return nil, nil
	}
	return frame.ResponseWriter, frame.Request
}

func (bbs *Bb_server) attest_authorization(w http.ResponseWriter, r *http.Request) (string, error) {
	var timewindow = 20 * time.Second
	var key, reason = awss3aide.Check_credential(r, bbs.cred_pair, timewindow)
	if reason != nil {
		bbs.logger.Info("Bad authorization", "key", key, "reason", reason)
		time.Sleep(1 * time.Second)
		http.Error(w, "Bad authorization", 401)
	}
	return key, reason
}

// SERVER_CONTROL handles requests to control.  It is hooked on
// "POST_/bbs.ctl/{command}".  While starting a shutdown, it will send
// an empty OK return.
func (bbs *Bb_server) server_control(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	//var q = r.URL.Query()
	var q = r.PathValue("command")
	switch q {
	case "quit":
		go shutdown_server(bbs, ctx)
	case "stat":
		dump_memory_statistics(bbs.logger, false)
	}
	// Send an empty return.
	var status int = 200
	w.WriteHeader(status)
}

func shutdown_server(bbs *Bb_server, ctx context.Context) {
	bbs.logger.Warn("Shutdown requested")
	var err1 = bbs.server.Shutdown(ctx)
	if err1 != nil {
		bbs.logger.Error("Shutdown failed", "error", err1)
		time.Sleep(3 * time.Second)
		log.Fatal("SHUTDOWN FORCED QUIT")
	}
}

func dump_memory_statistics(logger *slog.Logger, details bool) {
	//runtime.MemProfile()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if !details {
		var ms = struct {
			HeapAlloc   uint64
			HeapSys     uint64
			HeapObjects uint64
			HeapInuse   uint64
			StackInuse  uint64
			OtherSys    uint64
			NumGC       uint32
			NumForcedGC uint32
		}{
			HeapAlloc:   m.HeapAlloc,
			HeapInuse:   m.HeapInuse,
			HeapSys:     m.HeapSys,
			StackInuse:  m.StackInuse,
			OtherSys:    m.OtherSys,
			HeapObjects: m.HeapObjects,
			NumGC:       m.NumGC,
			NumForcedGC: m.NumForcedGC,
		}
		logger.Info("MemStats", "Summary", ms)
	} else {
		logger.Info("MemStats", "MemStats", m)
	}
	if details {
		var g debug.GCStats
		debug.ReadGCStats(&g)
		logger.Info("GCStats", "GCStats", g)
	}
}

// SERVICE_PROFILER starts the http server for "go tool pprof".  Note
// importing "net/http/pprof" initializes profiler in DefaultServeMux.
func service_profiler(logger *slog.Logger, port int) {
	var ep = net.JoinHostPort("", strconv.Itoa(port))
	var router = http.DefaultServeMux
	logger.Info("Enabling pprof", "port", port)
	var err1 = http.ListenAndServe(ep, router)
	logger.Error("Enabling pprof failed", "error", err1)
}
