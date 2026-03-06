// aws-s3-signing.go

// Copyright 2022-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// This verifies a sign by AWS-S3 signing.  This checks the headers in
// a server-side request.  DEFICIENCY: It does not calculate the
// message digest and uses a passed one.

// MEMO: An AWS-S3 V4 authorization-header looks like:
//
//	 Authorization=AWS4-HMAC-SHA256
//     Credential={key}/20240511/us-east-1/s3/aws4_request,
//	   SignedHeaders=host;x-amz-content-sha256;x-amz-date,
//	   Signature={signature}
//
// "Authorization=" starts with a keyword "AWS4-HMAC-SHA256" and three
// subentries separated by "," with zero or more whitespaces.  A
// "Credential=" subentry is a five fields separated by "/" as
// KEY/DATE/REGION/SERVICE/USAGE, with KEY as an access key, DATE as
// "yyyymmdd", REGION as a S3 region, SERVICE="s3", and
// USAGE="aws4_request".  A "SignedHeaders=" subentry is a list of
// header keys separated by ";" as
// "host;x-amz-content-sha256;x-amz-date".  A "Signature=" subentry is
// a string.

// Some reference documents are:
//  - https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-header-based-auth.html
//  - https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-auth-using-authorization-header.html
//  - https://docs.aws.amazon.com/AmazonS3/latest/API/RESTCommonRequestHeaders.html

package awss3aide

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// AUTHORIZATION_S3V4 lists entries of an authorization-header.  That
// is the slots of "Credential=", "SignedHeaders=", and Signature=".
// Keys in signed headers are canonicalized.
type Authorization_s3v4 struct {
	Credential    [5]string
	SignedHeaders []string
	Signature     string
}

// REQUIRED_HEADERS is a list that are checked their existence in
// Authorization.Signedheaders.  They are canonicalized although they
// appear in lowercase in Authorization.Signedheaders.  Other required
// headers are (in the chunked case): "Content-Encoding",
// "X-Amz-Decoded-Content-Length", "Content-Length".  Additionally,
// AWS-CLI also sends "Content-Md5".
var required_headers = [3]string{
	"Host", "X-Amz-Content-Sha256", "X-Amz-Date",
}

const aws_s3v4_authorization = "AWS4-HMAC-SHA256"
const aws_s3_region_default = "us-east-1"
const amz_date_layout = "20060102T150405Z"
const empty_payload_hash_sha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

var check_all_digits_re = regexp.MustCompile(`^[0-9]+$`)

// PROXY_ATTACHED_HEADERS lists headers dropped in signing, which a
// proxy may change.  See the section "ProxyPass" in the Apache-HTTPD
// document.  It includes other often-used headers: "Connection",
// "X-Forwarded-Proto", "X-Real-Ip".
var proxy_attached_headers = []string{
	"Accept-Encoding",
	"Amz-Sdk-Invocation-Id",
	"Amz-Sdk-Request",
	"X-Forwarded-For",
	"X-Forwarded-Host",
	"X-Forwarded-Server",
	"Connection",
	"X-Forwarded-Proto",
	"X-Real-Ip",
}

var Verbose = false

func signing_verbose(msg ...any) {
	if Verbose {
		fmt.Println(msg...)
	}
}

var E_no_auth = errors.New("no-auth")
var E_bad_auth = errors.New("bad-auth")
var E_bad_date = errors.New("bad-date")
var E_outdated_date = errors.New("outdated-date")
var E_wrong_key = errors.New("wrong-key")
var E_cannot_sign = errors.New("cannot-sign")
var E_bad_sign = errors.New("bad-sign")

// CHECK_CREDENTIAL checks a signing in an http request.  It returns
// an access-key and a simple failure reason.  It once signs a request
// by using AWS-SDK, and compares it with the one in the request.
// Note it does not calculate the message digest and uses a given one.
// Date in "X-Amz-Date" should be around the present time within a
// timewindow tolerance.  It returns "anon" as an access-key when no
// key is found.  Failure reasons are one of {"no-auth", "bad-auth",
// "bad-date", "outdated-date", "wrong-key", "cannot-sign",
// "bad-sign"}.  It substitutes "Host" by "X-Forwarded-Host" if it is
// missing.  It copies a request before modifying it.  It prints
// important logs to the default "logger" by slog.Debug().
func Check_credential(rqst1 *http.Request, keypair [2]string, timewindow time.Duration) (string, error) {
	var header1 = rqst1.Header.Get("Authorization")
	signing_verbose("*** authorization=", header1)
	if header1 == "" {
		signing_verbose("*** empty authorization=", header1)
		return "anon", E_no_auth
	}
	var auth_passed = Scan_aws_authorization(header1)
	if auth_passed == nil {
		signing_verbose("*** bad-auth=", header1)
		return "anon", E_bad_auth
	}

	var access_key = auth_passed.Credential[0]
	if access_key != keypair[0] {
		signing_verbose("*** wrong-key=", access_key)
		return access_key, E_wrong_key
	}

	// Copy the request.  Note Golang's copy is shallow.

	var rqst2 = *rqst1
	rqst2.Header = maps.Clone(rqst1.Header)

	// Filter out except the specified headers for signing.

	maps.DeleteFunc(rqst2.Header, func(k string, v []string) bool {
		return slices.Index(auth_passed.SignedHeaders, k) == -1
	})
	if slices.Index(auth_passed.SignedHeaders, "Content-Length") == -1 {
		rqst2.ContentLength = -1
	}
	if rqst2.Host == "" {
		rqst2.Host = rqst1.Header.Get("X-Forwarded-Host")
	}

	var service = auth_passed.Credential[3]
	var region = auth_passed.Credential[2]
	//var datestring = adjust_x_amz_date(rqst1.Header.Get("X-Amz-Date"))
	//var date, err1 = time.Parse(time.RFC3339, datestring)
	var datestring = rqst1.Header.Get("X-Amz-Date")
	var date, err1 = time.Parse(amz_date_layout, datestring)
	if err1 != nil {
		signing_verbose("*** bad-date=", auth_passed)
		return access_key, E_bad_date
	}
	var now = time.Now()
	if !(now.Sub(date).Abs() <= timewindow) {
		signing_verbose("*** outdated-date=", auth_passed)
		return access_key, E_outdated_date
	}

	var credentials = aws.Credentials{
		AccessKeyID:     keypair[0],
		SecretAccessKey: keypair[1],
		//SessionToken string
		//Source string
		//CanExpire bool
		//Expires time.Time
	}
	var hash = rqst2.Header.Get("X-Amz-Content-Sha256")
	if hash == "" {
		// It is a bad idea to use a hash for an empty payload.
		hash = empty_payload_hash_sha256
	}
	var sign1 = signer.NewSigner(func(s *signer.SignerOptions) {
		s.DisableHeaderHoisting = true
		s.DisableURIPathEscaping = true
	})
	var timeout = time.Duration(10 * time.Second)
	var ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var err2 = sign1.SignHTTP(ctx, credentials, &rqst2,
		hash, service, region, date)
	if err2 != nil {
		slog.Debug("signer/SignHTTP() failed", "error", err2)
		return access_key, E_cannot_sign
	}

	var header2 = rqst2.Header.Get("Authorization")
	var auth_forged = Scan_aws_authorization(header2)
	if auth_forged == nil {
		signing_verbose("*** bad-auth=", header2)
		return access_key, E_bad_auth
	}

	var ok = auth_passed.Signature == auth_forged.Signature
	if !ok {
		slog.Debug("Bad authorization, signs unmatch",
			"access-key", auth_passed.Credential[0],
			"auth-passed", auth_passed, "auth-signed", auth_forged)
		return access_key, E_bad_sign
	}
	return access_key, nil
}

// SCAN_AWS_AUTHORIZATION extracts slots in an "Authorization" header.
// It does not check the semantics but only extracts slots with regard
// to the format.  On failure, it returns nil.  It accepts an empty
// string and returns nil.
func Scan_aws_authorization(auth string) *Authorization_s3v4 {
	if !strings.HasPrefix(auth, aws_s3v4_authorization) {
		signing_verbose("*** bad auth method", auth)
		return nil
	}
	var auth2 = strings.TrimPrefix(auth, aws_s3v4_authorization)
	var auth3 = strings.TrimSpace(auth2)
	var entries [][2]string
	for _, s1 := range strings.Split(auth3, ",") {
		var k, v, ok = strings.Cut(strings.TrimSpace(s1), "=")
		if ok && len(k) > 0 && len(v) > 0 {
			entries = append(entries, [2]string{k, v})
		}
	}
	if len(entries) != 3 {
		signing_verbose("*** bad auth entries", auth)
		return nil
	}
	var credential []string
	var signedheaders []string
	var signature string
	for _, kv := range entries {
		switch kv[0] {
		case "Credential":
			// "Credential={key}/20240511/us-east-1/s3/aws4_request"
			var cred = strings.Split(kv[1], "/")
			if len(cred) != 5 {
				signing_verbose("*** bad credential slot", auth)
				return nil
			}
			if !(len(cred[1]) == 8 && check_all_digits(cred[1])) {
				signing_verbose("*** bad credential-date slot", auth)
				return nil
			}
			if cred[3] != "s3" {
				signing_verbose("*** bad credential-service slot", auth)
				return nil
			}
			if cred[4] != "aws4_request" {
				signing_verbose("*** bad credential-usage slot", auth)
				return nil
			}
			credential = cred
		case "SignedHeaders":
			// SignedHeaders=host;x-amz-content-sha256;x-amz-date
			var headers []string
			for _, h1 := range strings.Split(kv[1], ";") {
				headers = append(headers, http.CanonicalHeaderKey(h1))
			}
			for _, h2 := range required_headers {
				if slices.Index(headers, h2) == -1 {
					signing_verbose("*** bad signedheaders", h2, headers)
					return nil
				}
			}
			signedheaders = headers
		case "Signature":
			signature = kv[1]
		default:
			signing_verbose("*** bad entry", kv)
			return nil
		}
	}
	if credential == nil || signedheaders == nil || signature == "" {
		signing_verbose("*** bad missing slots", auth)
		return nil
	}
	return &Authorization_s3v4{
		Credential:    [5]string(credential),
		SignedHeaders: signedheaders,
		Signature:     signature}
}

func check_all_digits(s string) bool {
	return check_all_digits_re.MatchString(s)
}

// ADJUST_X_AMZ_DATE converts an X-Amz-Date string to be parsable in
// RFC3339.  It returns "" if a string is ill formed.  DO NOT USE
// THIS.  Instead, it should use the date format for "X-Amz-Date".  It
// is amz_date_layout="20060102T150405Z".  (* Note X-Amz-Date is an
// acceptable string by ISO-8601 *).
func adjust_x_amz_date(d string) string {
	if len(d) != 16 {
		return ""
	}
	return (d[0:4] + "-" +
		d[4:6] + "-" +
		d[6:11] + ":" +
		d[11:13] + ":" +
		d[13:])
}

/*
type Signing_credential struct {
	Host       string
	Access_key string
	Secret_key string
}
*/

// SIGN_BY_CREDENTIAL replaces an authorization header in a request
// for a given key-pair.  HOST is copied to the Host header and it is
// an endpoint "host:port".  KEYPAIR[0] is an access-key and
// KEYPAIR[1] is a secret-key.  It returns an error from
// Signer.SignHTTP().  Note that it drops the headers attached by a
// proxy, which would confuse the signer.
func Sign_by_credential(r *http.Request, host string, keypair [2]string) error {
	if false {
		fmt.Printf("*** r.Host(1)=%v\n", r.Host)
		fmt.Printf("*** Authorization(1)=%v\n", r.Header.Get("Authorization"))
		fmt.Printf("*** r.Header(1)=%v\n", r.Header)
	}

	signing_verbose("*** new host=", host)

	for _, h := range proxy_attached_headers {
		r.Header.Del(h)
	}

	r.Host = host
	var credentials = aws.Credentials{
		AccessKeyID:     keypair[0],
		SecretAccessKey: keypair[1],
		//SessionToken string
		//Source string
		//CanExpire bool
		//Expires time.Time
	}
	var hash = r.Header.Get("X-Amz-Content-Sha256")
	if hash == "" {
		// It is a bad idea to use a hash for an empty payload.
		hash = empty_payload_hash_sha256
	}
	var service = "s3"
	var region = aws_s3_region_default
	var date = time.Now()
	var sign1 = signer.NewSigner(func(s *signer.SignerOptions) {
		s.DisableHeaderHoisting = true
		s.DisableURIPathEscaping = true
	})
	var timeout = time.Duration(10 * time.Second)
	var ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var err1 = sign1.SignHTTP(ctx, credentials, r,
		hash, service, region, date)

	if false {
		fmt.Printf("*** r.Host(2)=%v\n", r.Host)
		fmt.Printf("*** Authorization(2)=%#v\n", r.Header.Get("Authorization"))
		fmt.Printf("*** r.Header(2)=%v\n", r.Header)
	}

	return err1
}
