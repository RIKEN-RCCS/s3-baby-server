// main.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Command line: ./s3-baby-server "serve" host-port pool-path options...

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"s3-baby-server/server"
)

func main() {
	// Use a plain structured-logger until a logger is initialized.

	var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	var o = os.Stdout
	var options = flag.NewFlagSet("", flag.ExitOnError)
	options.Usage = func() {
		fmt.Fprintf(o, "Usage: %s serve ADDR PATH options ...\n",
			os.Args[0])
		fmt.Fprintf(o, ("  Argument ADDR is host:port" +
			" and PATH is a directory where buckets inhabit.\n"))
		fmt.Fprintf(o, "Commands other than serve:\n")
		fmt.Fprintf(o, "  help: Print help.\n")
		fmt.Fprintf(o, "  version: Print version.\n")
		fmt.Fprintf(o, "  dump-conf: Dump configuaration and exit.\n")
		fmt.Fprintf(o, "Options:\n")
		options.PrintDefaults()
	}
	var print_help = options.Bool("help", false,
		"Print help.")
	var print_version = options.Bool("version", false,
		"Print version.")
	var flag_cred = options.String("cred", "",
		("Credential access+secret keys, separated by a comma." +
			" It is required.\n" +
			"It can be passed by an environment variable S3BBS_CRED."))
	var flag_https_crt = options.String("https-crt", "",
		("Certificate for https, a path to a certificate file."))
	var flag_https_key = options.String("https-key", "",
		("Key for the certificate, a path to a key file."))
	var flag_log_level = options.String("log", "",
		"Log-level, one of debug/info/warn.")
	var flag_log_access = options.Bool("log-access", false,
		"Logging access logs to stdout, unless logging directed to a file.")

	var flag_conf = options.String("conf", "",
		"Configuration file in json.")

	var flag_prof = options.Int("prof", 0,
		"Port to enable pprof service for 'go tool pprof'.")

	var args = os.Args

	var addr string = ""
	var pool string = ""
	if len(args) <= 1 {
		options.Usage()
		os.Exit(2)
	}

	var dump_conf = false
	var cred_pair = ""
	var args_optional []string

	switch args[1] {
	case "serve":
		if len(args) < 4 {
			options.Usage()
			os.Exit(2)
		}
		addr = args[2]
		pool = args[3]
		args_optional = args[4:]
	case "dump-conf":
		dump_conf = true
		if len(args) < 2 {
			options.Usage()
			os.Exit(2)
		}
		addr = "--"
		pool = "--"
		cred_pair = "--,--"
		args_optional = args[2:]
	case "version":
		fmt.Fprintf(os.Stdout, "%s\n", server.Bb_version)
		args_optional = []string{}
		os.Exit(0)
	case "help":
		fallthrough
	default:
		options.Usage()
		args_optional = []string{}
		os.Exit(2)
	}

	var err1 = options.Parse(args_optional)
	if err1 != nil {
		fmt.Printf("error: %s", err1)
		os.Exit(2)
	}

	if *print_help {
		options.Usage()
		os.Exit(2)
	}
	if *print_version {
		fmt.Fprintf(os.Stdout, "%s\n", server.Bb_version)
		os.Exit(0)
	}

	if options.NArg() != 0 {
		fmt.Fprintf(o, "Unrecognized options.\n")
		options.Usage()
		os.Exit(2)
	}

	var cred [2]string

	{
		if len(cred_pair) == 0 {
			cred_pair = os.Getenv("S3BBS_CRED")
		}
		if len(cred_pair) == 0 {
			cred_pair = *flag_cred
		}
		if len(cred_pair) == 0 {
			logger.Error("Credential not specified, it is required.\n")
			os.Exit(2)
		}

		var access, secret, ok = strings.Cut(cred_pair, ",")
		if !ok || len(access) == 0 || len(secret) == 0 {
			logger.Error("Bad credential key pair", "pair", cred_pair)
			os.Exit(2)
		}
		cred = [2]string{access, secret}
	}

	var cert [2]string

	{
		if *flag_https_crt != "" || *flag_https_key != "" {
			var crt1 = *flag_https_crt
			var key1 = *flag_https_key
			if len(crt1) == 0 || len(key1) == 0 {
				logger.Error("Both certificate and key needed for https",
					"crt", crt1, "key", key1)
				os.Exit(2)
			}

			// Make paths absolute, because server runs after changing
			// the working directory.

			var crt2, err1 = filepath.Abs(crt1)
			if err1 != nil {
				logger.Error("filepath.Abs() on certificate/key failed",
					"crt", crt1, "key", key1, "error", err1)
				os.Exit(2)
			}
			var key2, err2 = filepath.Abs(key1)
			if err2 != nil {
				logger.Error("filepath.Abs() on certificate/key failed",
					"crt", crt1, "key", key1, "error", err2)
				os.Exit(2)
			}
			cert = [2]string{crt2, key2}
		}
	}

	var conf = *flag_conf
	var logs = *flag_log_level
	var loga = *flag_log_access
	var prof = *flag_prof

	server.Start_server(dump_conf, cred, cert, pool, addr, conf, logs, loga, prof)
}
