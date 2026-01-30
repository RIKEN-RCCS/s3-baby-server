// control.go

package main

import (
	"context"
	"crypto/tls"
	//"flag"
	"fmt"
	"github.com/riken-rccs/s3-baby-server/pkg/awss3aide"
	"io"
	"log/slog"
	"net"
	"net/http"
	//"os"
	//"slices"
	"strconv"
	//"strings"
	"time"
)

var empty_payload_hash_sha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func control_server(command string, host string, port int, keypair [2]string, https bool) error {
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
