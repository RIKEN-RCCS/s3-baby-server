// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Baby-server Toplevel

// The entry Start_server() is called from the main in
// cmd/s3-baby-server/main.go.  A start-up message will be printed
// when listening on a port is successful.  It assumes no further
// errors occur.

package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"maps"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
	//runtimepprof "runtime/pprof"

	"github.com/riken-rccs/s3-baby-server/pkg/awss3aide"
	"github.com/riken-rccs/s3-baby-server/pkg/httpaide"
)

const Bbs_version = "v1.3.1-a"
const Bbs_metafile_format = "v1.3"

type Bbs_server struct {
	server    *http.Server
	cert_pair [2]string
	cred_pair [2]string
	config    Bbs_configuration
	logger    *slog.Logger

	access_logging io.Writer

	rid_past uint64
	suffixes map[string]scratch_suffix
	monitor1 *Monitor
	mutex    sync.Mutex

	server_quit chan struct{}

	// POOL_DIRECTORY is a path where buckets reside.  Baby-server
	// changes the working directory to that path, and this is only a
	// record.
	pool_directory string
}

type msec_duration int64
type sec_duration int64
type mbyte_size int64

func time_msec_duration(v msec_duration) time.Duration {
	return time.Duration(v) * time.Millisecond
}

func time_sec_duration(v sec_duration) time.Duration {
	return time.Duration(v) * time.Second
}

func byte_size(v mbyte_size) int64 {
	return int64(v) * 1024 * 1024
}

// BBS_CONFIGURATION is the configuration.  It may be loaded from a
// specified file.  Parameters from "ReadTimeout" to "MaxHeaderBytes"
// are set to Golang's http.Server.  Time values are in msec duration,
// because they get large numbers in time.Duration that are not an
// appropriate representation in a configuration file.
type Bbs_configuration struct {
	Server_control_name      string        `json:"server_control_name"`
	Site_base_url            *string       `json:"site_base_url"`
	Exclusion_wait           msec_duration `json:"exclusion_wait"`
	Sign_valid_window        sec_duration  `json:"sign_valid_window"`
	Etag_save_threshold      mbyte_size    `json:"etag_save_threshold"`
	Xml_parameter_size_limit mbyte_size    `json:"xml_parameter_size_limit"`
	Verify_fs_write          bool          `json:"verify_fs_write"`
	Keep_trailing_slash      bool          `json:"keep_trailing_slash"`
	Accept_fetch_owner       bool          `json:"accept_fetch_owner"`
	Strict_etag_quoting      bool          `json:"strict_etag_quoting"`
	Forbid_last_chunk_crlf   bool          `json:"forbid_last_chunk_crlf"`

	// Anonymize_ower bool
	// File_creation_mode fs.FileMode

	ReadTimeout       msec_duration `json:"ReadTimeout"`
	ReadHeaderTimeout msec_duration `json:"ReadHeaderTimeout"`
	WriteTimeout      msec_duration `json:"WriteTimeout"`
	IdleTimeout       msec_duration `json:"IdleTimeout"`
	MaxHeaderBytes    int           `json:"MaxHeaderBytes"`

	Pretty_xml_response bool `json:"pretty_xml_response"`
	Log_monitor_timing  bool `json:"log_monitor_timing"`
	Skip_trace_logging  bool `json:"skip_trace_logging"`
	Dump_request_header bool `json:"dump_request_header"`
}

// HANDLER_FRAME is a request-specific record stored in the context
// under the key "hander-frame".
type Handler_frame struct {
	Request_id     uint64
	Scratch_suffix string
	ResponseWriter http.ResponseWriter
	Request        *http.Request
}

const action_name_key string = "action-name"
const handler_frame_key string = "hander-frame"

// H_LIMIT_OF_XML_PARAMETERS limits the size of XML parameters
// in a request body.
var h_limit_of_xml_parameters int64 = (2 * 1024 * 1024)

// H_PRETTY_XML_RESPONSE=true sets xml.NewEncoder.Indent() in response
// generation.
var h_pretty_xml_response bool = false

// Log Trace level is more verbose than Debug.
const LevelTrace = slog.Level(-8)

// PRIOR_HANDLER is an http.Handler and it checks an authorization
// header in a request before passing it to actual handlers.  It also
// prints access logs.  See its ServeHTTP() method.
type prior_handler struct {
	bbs *Bbs_server
	sx  *http.ServeMux
}

func (sv *prior_handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var start_time = time.Now()

	var auth, err1 = sv.bbs.attest_authorization(w, r)
	if err1 != nil {
		return
	}

	var rid = sv.bbs.make_request_id()
	var suffix = sv.bbs.make_scratch_suffix(rid)
	defer sv.bbs.discharge_scratch_suffix(rid, suffix)

	var request = fmt.Sprintf("%s %s", r.Method, r.URL)

	var ctx1 = r.Context()
	var w2 = &httpaide.ResponseWriter2{ResponseWriter: w}
	var frame = &Handler_frame{
		Request_id:     rid,
		Scratch_suffix: suffix,
		ResponseWriter: w,
		Request:        r,
	}
	var ctx2 = context.WithValue(ctx1, handler_frame_key, frame)
	var r2 = r.WithContext(ctx2)

	// Dump the header for debugging.  It restores some headers
	// http-server swallowed.

	if sv.bbs.config.Dump_request_header {
		var h = maps.Clone(r.Header)
		var contentlength = strconv.FormatInt(r.ContentLength, 10)
		h["Content-Length"] = []string{contentlength}
		h["Transfer-Encoding"] = r.TransferEncoding
		var tailers []string
		for k, _ := range r.Trailer {
			tailers = append(tailers, k)
		}
		h["Trailer"] = tailers
		sv.bbs.logger.Log(ctx1, LevelTrace,
			"Request-Header", "header", h)
	}

	if r.Trailer != nil {
		sv.bbs.logger.Error("http trailer header is unsupported",
			"trailer", r.Trailer)
	}

	// Drop (multiple) trailing-slashes in the path by rewriting.
	// Some clients attach a slash-suffix to a bucket/object name.

	if !sv.bbs.config.Keep_trailing_slash {
		if r2.URL.Path != "/" && strings.HasSuffix(r2.URL.Path, "/") {
			sv.bbs.logger.Debug("Drop trailing-slashes in url",
				"request", request)
			var r2url url.URL = *r2.URL
			r2.URL = &r2url
			r2.URL.Path = strings.TrimRight(r2.URL.Path, "/")
		}
	}

	// Call the dispatcher.

	sv.sx.ServeHTTP(w2, r2)

	var q_length = r2.ContentLength
	var user = auth[:min(len(auth), 16)]
	var code = w2.Status_code
	var r_length = w2.Content_length

	var elapse_time = time.Since(start_time)
	sv.bbs.logger.Info("Handling time",
		"rid", rid, "request", request, "request-length", q_length,
		"code", code, "response-length", r_length, "elapse", elapse_time)

	if sv.bbs.access_logging != nil {
		var m = httpaide.Log_access(r, code, r_length, user)
		fmt.Fprintf(sv.bbs.access_logging, "%s\n", m)
	}
}

// START_SERVER is called from the main.  Baby-server first prints
// info message "Starting Baby-server".  Other logging messages may be
// printed only when it stops.
func Start_server(dump_conf bool, cred, cert [2]string, pool_directory, addr, conf, logs string, loga bool, prof int) {

	// Runs in GMT time zone instead of the local time zone.  It is to
	// avoid specifying the time zone every time.  The time format
	// RFC-1123 requires GMT, and time.UTC does not work.

	// time.Local = time.UTC
	time.Local = time.FixedZone("GMT", 0)

	// Create a logger.

	var loglevel = new(slog.LevelVar)
	var encounter_bad_log_level bool = false
	switch logs {
	case "trace":
		loglevel.Set(LevelTrace)
	case "debug":
		loglevel.Set(slog.LevelDebug)
	case "info":
		loglevel.Set(slog.LevelInfo)
	case "warn":
		loglevel.Set(slog.LevelWarn)
	case "":
		loglevel.Set(slog.LevelInfo)
	default:
		loglevel.Set(slog.LevelInfo)
		encounter_bad_log_level = true
	}

	/*loglevel.Set(LevelTrace)*/

	var h = new_log_handler_with_trace(os.Stdout,
		&slog.HandlerOptions{Level: loglevel})
	var logger = slog.New(h)

	if encounter_bad_log_level {
		logger.Error("Bad log level", "level", logs)
		os.Exit(2)
	}

	// Set default configurations, or read it from a file.

	var config = Bbs_configuration{
		Server_control_name:      "bbs.ctl",
		Site_base_url:            nil,
		Exclusion_wait:           5000,
		Sign_valid_window:        60,
		Etag_save_threshold:      1,
		Xml_parameter_size_limit: 2,
		/*Dump_request_header:      true,*/
	}

	if conf != "" {
		var confpath = filepath.Clean(conf)
		var f1, err1 = os.Open(confpath)
		if err1 != nil {
			logger.Error("os.Open() to a conf file failed",
				"path", confpath, "error", err1)
			os.Exit(2)
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
			logger.Error("json.Decode() on a conf file failed",
				"path", confpath, "error", err2)
			os.Exit(2)
		}
		var err3 = f1.Close()
		if err3 != nil {
			logger.Error("os.File.Close() on a conf file failed",
				"path", confpath, "error", err3)
			os.Exit(2)
		}
	}

	// Dump configuration and exit.

	if dump_conf {
		var e = json.NewEncoder(os.Stdout)
		e.SetIndent("", "  ")
		var err4 = e.Encode(&config)
		if err4 != nil {
			logger.Error("json.Encoder.Encode() on dumping a conf file failed",
				"error", err4)
			os.Exit(2)
		}
		os.Exit(0)
	}

	// Change the working directory to the pool-directory.  It is to
	// avoid accidentally disclosing the full path (which may include
	// a user name or a project name)

	{
		var wdpath = filepath.Clean(pool_directory)

		// Check the directory is with sufficient permission.

		var stat, err1 = os.Lstat(wdpath)
		if err1 != nil {
			logger.Error("os.Lstat() to pool directory failed",
				"directory", wdpath, "error", err1)
			os.Exit(2)
		}
		var mode = stat.Mode()
		var perm = mode.Perm()
		var rwx = (perm & 7) | ((perm >> 3) & 7) | ((perm >> 6) & 7)
		if rwx != 7 {
			logger.Error("No sufficient permission (rwx)",
				"directory", wdpath, "mode", mode.String())
			os.Exit(2)
		}

		var err2 = os.Chdir(wdpath)
		if err2 != nil {
			logger.Error("os.Chdir() to pool directory failed",
				"directory", wdpath, "error", err2)
			os.Exit(2)
		}
	}

	// Check the directory to store access logs.

	var access_logging io.Writer

	{
		var log1 *os.File = nil
		if loga {
			log1 = os.Stdout
		}

		var dir1 = ".s3bbs"
		var dir2 = "log"
		var dirpath = filepath.Join(".", dir1, dir2)
		var _, err1 = os.Lstat(dirpath)
		if err1 != nil && !errors.Is(err1, fs.ErrNotExist) {
			logger.Error("os.Lstat() in checking .s3bbs/log failed",
				"path", dirpath, "error", err1)
			os.Exit(2)
		}
		var log2 *os.File = nil
		if err1 == nil {
			var logpath = filepath.Join(".", dir1, dir2, "access-log")
			var oappend = os.O_APPEND | os.O_CREATE | os.O_WRONLY
			var f, err2 = os.OpenFile(logpath, oappend, 0644)
			if err2 != nil {
				logger.Error("os.OpenFile() to access-log failed",
					"path", logpath, "error", err2)
				os.Exit(2)
			}
			if err2 == nil {
				log2 = f
			}
		}

		var ww []io.Writer
		if log1 != nil {
			ww = append(ww, log1)
		}
		if log2 != nil {
			ww = append(ww, log2)
		}
		if len(ww) == 0 {
			access_logging = nil
		} else if len(ww) == 1 {
			access_logging = ww[0]
		} else {
			access_logging = io.MultiWriter(ww...)
		}
	}

	var bbs = &Bbs_server{
		pool_directory: pool_directory,
		cert_pair:      cert,
		cred_pair:      cred,
		config:         config,
		logger:         logger,
		access_logging: access_logging,
		server_quit:    make(chan struct{}),
		monitor1:       New_monitor(),
		suffixes:       make(map[string]scratch_suffix),
	}

	go bbs.monitor1.Guard_loop()

	h_limit_of_xml_parameters = byte_size(config.Xml_parameter_size_limit)
	h_pretty_xml_response = config.Pretty_xml_response

	var sx = http.NewServeMux()
	var control = "POST /" + config.Server_control_name + "/{command}"
	sx.HandleFunc(control, func(w http.ResponseWriter, r *http.Request) {
		bbs.server_control(w, r)
	})
	register_dispatcher(bbs, sx)
	var sv = &prior_handler{bbs, sx}

	bbs.server = &http.Server{
		Addr:     addr,
		Handler:  sv,
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),

		ReadTimeout:       time_msec_duration(bbs.config.ReadTimeout),
		ReadHeaderTimeout: time_msec_duration(bbs.config.ReadHeaderTimeout),
		WriteTimeout:      time_msec_duration(bbs.config.WriteTimeout),
		IdleTimeout:       time_msec_duration(bbs.config.IdleTimeout),
		MaxHeaderBytes:    bbs.config.MaxHeaderBytes,
	}

	if prof != 0 {
		go service_profiler(logger, prof)
	}

	var proto string

	if cert[0] != "" {
		proto = "https"
	} else {
		proto = "http"
	}

	var listen, err3 = net.Listen("tcp", addr)
	if err3 != nil {
		logger.Error("net.Listen() failed",
			"address", addr, "error", err3)
		os.Exit(2)
	}
	defer listen.Close()

	logger.Info("Starting Baby-server", "address", addr, "proto", proto,
		"access-key", cred[0], "version", Bbs_version)

	var err2 error
	if cert[0] != "" {
		err2 = bbs.server.ServeTLS(listen, cert[0], cert[1])
	} else {
		err2 = bbs.server.Serve(listen)
	}

	if !(err2 == nil || err2 == http.ErrServerClosed) {
		logger.Error("http.Serve() failed", "error", err2)
	} else {
		// Note http.ErrServerClosed is usual.
		logger.Info("Baby-server finished")
	}
}

// NEW_LOG_HANDLER_WITH_TRACE makes a new text log handler.  This is
// taken from "example_custom_levels_test.go", which can be found at
// "https://pkg.go.dev/log/slog#section-sourcefiles".
func new_log_handler_with_trace(w io.Writer, o *slog.HandlerOptions) *slog.TextHandler {
	var h = slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: o.Level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				var level = a.Value.Any().(slog.Level)
				switch {
				case level < slog.LevelDebug:
					a.Value = slog.StringValue("TRACE")
				case level < slog.LevelInfo:
					a.Value = slog.StringValue("DEBUG")
				case level < slog.LevelWarn:
					a.Value = slog.StringValue("INFO")
				case level < slog.LevelError:
					a.Value = slog.StringValue("WARN")
				default:
					a.Value = slog.StringValue("ERROR")
				}
			}
			return a
		},
	})
	return h
}

func get_action_name(ctx context.Context) (string, uint64, string) {
	var action = ctx.Value(action_name_key).(*string)
	if action == nil {
		log.Fatal("BAD-IMPL: action-name not set")
		return "", 0, ""
	}
	var frame = ctx.Value(handler_frame_key).(*Handler_frame)
	if frame == nil {
		log.Fatal("BAD-IMPL: Context hander-frame not set")
		return "", 0, ""
	}
	return *action, frame.Request_id, frame.Scratch_suffix
}

func get_handler_arguments(ctx context.Context) (http.ResponseWriter, *http.Request) {
	var frame = ctx.Value(handler_frame_key).(*Handler_frame)
	if frame == nil {
		log.Fatal("BAD-IMPL: Context hander-frame not set")
		return nil, nil
	}
	return frame.ResponseWriter, frame.Request
}

func (bbs *Bbs_server) attest_authorization(w http.ResponseWriter, r *http.Request) (string, error) {
	var timewindow = time_sec_duration(bbs.config.Sign_valid_window)
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
// an empty 200-OK return.
func (bbs *Bbs_server) server_control(w http.ResponseWriter, r *http.Request) {
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

func shutdown_server(bbs *Bbs_server, ctx context.Context) {
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
// importing "net/http/pprof" initializes the use of profiler in
// DefaultServeMux.
func service_profiler(logger *slog.Logger, port int) {
	var ep = net.JoinHostPort("", strconv.Itoa(port))
	var router = http.DefaultServeMux
	logger.Info("Enabling pprof", "port", port)
	var err1 = http.ListenAndServe(ep, router)
	logger.Error("Enabling pprof failed", "error", err1)
}
