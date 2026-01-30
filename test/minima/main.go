// main.go

// Command "bbs-ctl".  It accepts server-control "quit" or "stat",
// and some test commands.

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
)

var command_list = []string{"quit", "stat", "test-buckets"}

func main() {
	var options = flag.NewFlagSet("", flag.ExitOnError)
	options.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage: %s command host port options ...\n",
			os.Args[0])
		fmt.Fprintf(os.Stdout, "\tcommand is one of %v.\n", command_list)
		fmt.Fprintf(os.Stdout, "Options:\n")
		options.PrintDefaults()
	}
	var print_help = options.Bool("help", false,
		"Print help.")
	var flag_cred = options.String("cred", "",
		"Pair of access-key and secret-key, separated by a comma.")
	var flag_https = options.Bool("https", false,
		"Use https to talk to the server.")

	var args = os.Args
	if len(args) < 4 {
		options.Usage()
		os.Exit(2)
	}

	var cmd = args[1]
	if !slices.Contains(command_list, cmd) {
		slog.Error("cmd is one of quit or stat")
		os.Exit(2)
	}

	var host = args[2]
	var v, err1 = strconv.ParseInt(args[3], 10, 32)
	if err1 != nil {
		slog.Error("strconv.ParseInt() failed",
			"error", err1)
		os.Exit(2)
	}
	var port = int(v)

	var err2 = options.Parse(args[4:])
	if err2 != nil {
		slog.Error("options.Parse() failed",
			"error", err2)
		os.Exit(2)
	}

	if *print_help {
		options.Usage()
		os.Exit(2)
	}

	var cred [2]string
	{
		var credpair = *flag_cred
		if len(credpair) == 0 {
			slog.Error("Option --cred is required.\n")
			os.Exit(2)
		}
		var access, secret, ok = strings.Cut(credpair, ",")
		if !ok || len(access) == 0 || len(secret) == 0 {
			slog.Error("Bad authorization key pair", "pair", credpair)
			os.Exit(2)
		}
		cred = [2]string{access, secret}
	}

	var https = *flag_https

	switch cmd {
	case "quit", "stat":
		control_server(cmd, host, port, cred, https)
	case "test-buckets":
		test_with_many_buckets(10)
	}
}
