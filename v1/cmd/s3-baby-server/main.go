// main.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Command line is: ./s3-baby-server serve addr path options...

package main

import (
	"flag"
	"fmt"
	"os"
	//"time"
	"s3-baby-server/server"
)

func main() {
	var options = flag.NewFlagSet("", flag.ExitOnError)
	options.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s serve addr path options...\n",
			os.Args[0])
		fmt.Fprintf(os.Stderr, ("\tArgument ADDR is host:port" +
			" and PATH is a pool-directory.\n"))
		fmt.Fprintf(os.Stderr, "Commands other than serve:\n")
		fmt.Fprintf(os.Stderr, "\thelp: Print help.\n")
		fmt.Fprintf(os.Stderr, "\tversion: Print version.\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		options.PrintDefaults()
	}
	var print_help = options.Bool("help", false,
		"Print help.")
	var print_version = options.Bool("version", false,
		"Print version.")
	var flag_cred = options.String("cred", "",
		"Credential access-key and secret-key pair"+
			" separated by a comma.")
	var flag_cert = options.String("cert", "",
		("Certificate and key for https, paths to files" +
			" separated by a comma."))
	var flag_logs = options.String("log", "",
		"Log-level, one of debug/info/warn.")
	var flag_conf = options.String("conf", "",
		"Configuration file in json.")

	var args = os.Args
	var url string = ""
	var path string = ""

	if len(args) <= 1 {
		options.Usage()
		os.Exit(2)
	}
	switch args[1] {
	case "serve":
		if len(args) < 4 {
			options.Usage()
			os.Exit(2)
		}
		url = args[2]
		path = args[3]
	case "version":
		fmt.Fprintf(os.Stdout, "%s\n", server.Bb_version)
		os.Exit(0)
	case "help":
		fallthrough
	default:
		options.Usage()
		os.Exit(2)
	}

	var err1 = options.Parse(args[4:])
	if err1 != nil {
		fmt.Printf("error: %s", err1)
		return
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
		fmt.Fprintf(os.Stdout, "Unrecognized options exit.\n")
		options.Usage()
		os.Exit(2)
	}

	var cred = os.Getenv("S3BBS_CRED")
	if len(cred) == 0 {
		cred = *flag_cred
	}
	if len(cred) == 0 {
		fmt.Fprintf(os.Stderr, "Credential not specified, use --cred.\n")
		os.Exit(2)
	}

	var cert = os.Getenv("S3BBS_CERT")
	if len(cert) == 0 {
		cert = *flag_cert
	}

	var conf = *flag_conf
	var logs = *flag_logs

	server.Start_server(path, url, cred, cert, conf, logs)
}
