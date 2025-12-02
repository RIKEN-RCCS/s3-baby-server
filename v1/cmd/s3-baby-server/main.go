// main.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Command line is: ./s3-baby-server serve url path options...

package main

import (
	"flag"
	"fmt"
	"os"
	//"time"
	"s3-baby-server/server"
)

func main() {
	var fs = flag.NewFlagSet("", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s serve url path options...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Other commands:\n")
		fmt.Fprintf(os.Stderr, "\tversion:\tPrint version.\n")
		fmt.Fprintf(os.Stderr, "\thelp:\tPrint help.\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}
	var print_help = fs.Bool("help", false,
		"Print help.")
	var print_version = fs.Bool("version", false,
		"Print version.")
	var flag_cred = fs.String("cred", "",
		"Credential, a pair of key and secret separated by a comma.")
	var log_file = fs.String("log-file", "",
		"Log file.")

	var args = os.Args
	if len(args) <= 2 {
		fs.Usage()
		os.Exit(2)
	}
	var url string = ""
	var path string = ""
	switch args[1] {
	case "serve":
		if len(args) < 4 {
			fs.Usage()
			os.Exit(2)
		}
		url = args[2]
		path = args[3]
	case "help":
		fallthrough
	default:
		fs.Usage()
		os.Exit(2)
	}

	var err1 = fs.Parse(args[4:])
	if err1 != nil {
		fmt.Printf("error: %s", err1)
		return
	}

	if *print_help {
		fs.Usage()
		os.Exit(2)
	}
	if *print_version {
		fmt.Fprintf(os.Stdout, "s3-baby-server %s:\n", server.Bb_version)
		os.Exit(0)
	}

	if fs.NArg() != 0 {
		fmt.Fprintf(os.Stdout, "Unrecognized options exit.\n")
		fs.Usage()
		os.Exit(2)
	}

	var cred = *flag_cred
	if len(cred) == 0 {
		cred = os.Getenv("S3BBS_CRED")
	}
	if len(cred) == 0 {
		fmt.Fprintf(os.Stderr, "Credential not specified, use --cred.\n")
		os.Exit(2)
	}

	server.Start_server(path, url, *log_file, cred)
}
