// control.go

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/riken-rccs/s3-baby-server/pkg/awss3aide"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

var empty_payload_hash_sha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func main() {
	var options = flag.NewFlagSet("", flag.ExitOnError)
	options.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage: %s command host port options ...\n",
			os.Args[0])
		fmt.Fprintf(os.Stdout, "\tcommand is one of {quit, stat}.\n")
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

	var command_list = []string{"quit", "stat"}
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

	access_server(cmd, host, port, cred, https)
}

func access_server(command string, host string, port int, keypair [2]string, https bool) error {
	var timeout = time.Duration(60000 * time.Millisecond)
	var ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var ep = net.JoinHostPort(host, strconv.Itoa(port))
	var url1 string
	if !https {
		url1 = ("http://" + ep + "/bbs.ctl/" + command)
	} else {
		url1 = ("https://" + ep + "/bbs.ctl/" + command)
	}
	var body io.Reader = nil

	var r, err4 = http.NewRequestWithContext(ctx, http.MethodPost, url1, body)
	if err4 != nil {
		slog.Error("http.NewRequestWithContext failed",
			"error", err4)
		return err4
	}

	//r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var hash = empty_payload_hash_sha256
	r.Header.Set("X-Amz-Content-Sha256", hash)

	var err5 = awss3aide.Sign_by_credential(r, host, keypair)
	if err5 != nil {
		slog.Error("S3-Signing failed",
			"error", err5)
		return err5
	}

	// Set to skip https server certificate verification.

	var xport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	var c = &http.Client{
		Transport: xport,
		Timeout:   timeout,
	}
	var rspn, err6 = c.Do(r)
	if err6 != nil {
		slog.Error("http/Client.Do() failed",
			"error", err6)
		return err6
	}
	defer rspn.Body.Close()

	if rspn.StatusCode == http.StatusOK {
		return nil
	} else {
		var err8 = fmt.Errorf("http/Client.Do() returns not OK",
			"status", rspn.StatusCode)
		slog.Debug("http/Client.Do() not good",
			"error", err8)
		return err8
	}

	return nil
}
