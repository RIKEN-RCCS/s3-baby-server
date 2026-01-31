// main.go

// Command top-level of "bbs-ctl".  It accepts server-control "quit"
// or "stat", and some test commands.

// It assumes the default configuration "~/.aws/config" contains
// definitions at least: endpoint_url, aws_access_key_id, and
// aws_secret_access_key.

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"time"

	//awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

var command_name = "bbs-ctl"
var command_list = []string{"quit", "stat", "test-buckets"}

func main() {
	var options = flag.NewFlagSet("", flag.ExitOnError)
	options.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage: %s command options ...\n",
			command_name)
		fmt.Fprintf(os.Stdout, "  command is one of %v.\n", command_list)
		fmt.Fprintf(os.Stdout, "Options:\n")
		options.PrintDefaults()
		fmt.Fprintf(os.Stdout,
			"Note %s reads '~/.aws/config', and it should have the entries:\n",
			command_name)
		fmt.Fprintf(os.Stdout,
			("" +
				"  endpoint_url = https://127.0.0.1:9000\n" +
				"  aws_access_key_id = abcdefghijklmnopqrstuvwxyz\n" +
				"  aws_secret_access_key = abcdefghijklmnopqrstuvwxyz\n"))
	}
	var flag_help = options.Bool("help", false,
		"Print help.")
	var flag_verbose = options.Bool("v", false,
		"Verbose.")

	var args = os.Args
	if len(args) < 2 {
		options.Usage()
		os.Exit(2)
	}
	if args[1][0] == '-' {
		options.Usage()
		os.Exit(2)
	}

	var err2 = options.Parse(args[2:])
	if err2 != nil {
		slog.Error("options.Parse() failed",
			"error", err2)
		os.Exit(2)
	}
	if *flag_help {
		options.Usage()
		os.Exit(2)
	}
	var verbose = *flag_verbose

	var cmd = args[1]
	if !slices.Contains(command_list, cmd) {
		slog.Error(fmt.Sprintf("command is one of %v", command_list))
		os.Exit(2)
	}

	var cfg, err3 = load_aws_config(verbose)
	if err3 != nil {
		options.Usage()
		os.Exit(2)
	}

	switch cmd {
	case "quit", "stat":
		control_server(cmd, cfg)
	case "test-buckets":
		test_with_many_buckets(cfg, 10)
	}
}

func load_aws_config(verbose bool) (*aws.Config, error) {
	/* var c1 = awshttp.NewBuildableClient().WithTransportOptions(...) */
	var timeout = time.Duration(60000 * time.Millisecond)
	var xport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:    60,
		IdleConnTimeout: 30 * time.Second,
	}
	var c = &http.Client{
		Transport: xport,
		Timeout:   timeout,
	}

	var cfg, err1 = config.LoadDefaultConfig(context.TODO(),
		config.WithHTTPClient(c),
		config.WithSharedConfigProfile("default"),
		config.WithDefaultRegion("us-east-1"))
	if err1 != nil {
		slog.Error("Loading config failed", "error", err1)
		return nil, err1
	}

	if cfg.BaseEndpoint == nil {
		slog.Error("No BaseEndpoint in config")
		var err2 = fmt.Errorf("No BaseEndpoint in config.")
		return nil, err2
	}
	if cfg.Region == "" {
		slog.Error("No Region in config")
		var err3 = fmt.Errorf("No Region in config.")
		return nil, err3
	}

	/* (cfg : aws.Config) */
	/* (cfg.Credentials : aws.CredentialsProvider) */
	/* (credentials : aws.Credentials) */

	var credentials, err4 = cfg.Credentials.Retrieve(context.TODO())
	if err4 != nil {
		slog.Error("No credentials in config", "error", err4)
		return nil, err4
	}

	if verbose {
		fmt.Printf("- BaseEndpoint=%#v\n", *cfg.BaseEndpoint)
		fmt.Printf("- Region=%#v\n", cfg.Region)
		fmt.Printf("- AccessKeyID=%#v\n", credentials.AccessKeyID)
	}

	return &cfg, nil
}
