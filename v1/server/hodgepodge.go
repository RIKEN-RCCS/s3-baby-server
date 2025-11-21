// hodgepodge.go
// Copyright 2025-2025 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// SPECIAL CONDITIONS OF HANDLING RFC7232.
//
// The rule described in the AWS-S3 API document:
// If-Match ∧ ¬If-Unmodified-Since → 200 OK
// ¬If-None-Match ∧ If-Modified-Since → 304 Not Modified
//
// https://datatracker.ietf.org/doc/html/rfc7232
//
// The order of condition evaluation:
// If-Match < If-Unmodified-Since (skip if If-Match exists)
// < If-None-Match < If-Modified-Since (skip if If-None-Match exists)
//
// Status code to be returned on failure:
// ¬If-Match → 412 Precondition Failed
// ¬If-Unmodified-Since → 412 Precondition Failed
// ¬If-None-Match (GET/HEAD) → 304 Not Modified
// ¬If-None-Match (other) → 412 Precondition Failed
// ¬If-Modified-Since → 304 Not Modified

package server

import (
	"context"
	//"errors"
	"fmt"
	"io/fs"
	//"os"
	"path"
	"time"
	"github.com/riken-rccs/s3-baby-server/pkg/httpaide"
	//"bytes"
	//"encoding/base64"
	//"encoding/binary"
	//"encoding/hex"
	"encoding/xml"
	//"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	//"strconv"
	"strings"
	"sync"
)

type Bb_configuration struct {
	Access_logging            bool
	Anonymize_ower            bool
	Verify_fs_write           bool
	Pending_upload_expiration time.Duration
	Server_controler_path     string

	request_processing_timeout time.Duration

	File_follow_link   bool
	File_creation_mode fs.FileMode
}

type Bb_server struct {
	pool_path string
	Logger    *slog.Logger
	AuthKey   string

	Bb_config Bb_configuration

	rid      int64
	suffixes map[string]suffix_record
	monitor1 *monitor
	mutex    sync.Mutex

	server_quit chan struct{}
}

type suffix_record struct {
	rid       int64
	timestamp time.Time
}

type Bb_mpul_record struct {
	upload_id       int64
	//o.AbortDate *time.Time
	//o.AbortRuleId *string
	timestamp time.Time
}

const alwasy_use_flat_lister = true

// MAKE_REQUEST_ID makes a new request-id.  It uses time, or when time
// does not advance, uses the last value plus one.  It is strictly
// increasing.
func (bbs *Bb_server) make_request_id() *int64 {
	var t int64 = time.Now().UnixMicro()
	bbs.mutex.Lock()
	defer bbs.mutex.Unlock()
	if bbs.rid < t {
		bbs.rid = t
	} else {
		t = bbs.rid + 1
		bbs.rid = t
	}
	//return strconv.FormatInt(t, 16)
	//return fmt.Sprintf("%016x", t)
	return &t
}

func get_request_id(ctx context.Context) int64 {
	var ridx = ctx.Value("request-id").(*int64)
	if ridx == nil {
		log.Fatal("BAD-IMPL: request-id not assigned")
		return 0
	} else {
		return *ridx
	}
}

// MAKE_FILE_SUFFIX makes a key string for a scratch file.  It takes
// request-id or zero.  A key is valid during request processing when
// a request-id is given.  Otherwise, when zero is given, a key is for
// multipart upload and it is active until completion.  It returns a
// string of 6 hex-digits.
func (bbs *Bb_server) make_file_suffix(rid int64) string {
	bbs.mutex.Lock()
	defer bbs.mutex.Unlock()
	var loops int = 0
	for true {
		var r = rand.Int63()
		var s = fmt.Sprintf("%06x", r)[:6]
		var _, ok = bbs.suffixes[s]
		if !ok {
			bbs.suffixes[s] = suffix_record{rid, time.Now()}
			return s
		}
		loops++
		if loops >= 10 {
			log.Fatal("BAD-IMPL: unique key generation loops forever")
		}
	}
	// NEVER.
	panic("NEVER")
}

func (bbs *Bb_server) discharge_file_suffix(rid int64) {
	bbs.mutex.Lock()
	defer bbs.mutex.Unlock()
	for k, v := range bbs.suffixes {
		if v.rid == rid {
			delete(bbs.suffixes, k)
		}
	}
}

func (bbs *Bb_server) serialize_access(ctx context.Context, object string, rid int64) error {
	var ok = bbs.monitor1.enter(object, rid, (10 * time.Millisecond))
	if !ok {
		return &Aws_s3_error{Code: RequestTimeout}
	}
	return nil
}

func (bbs *Bb_server) release_access(ctx context.Context, object string, rid int64) error {
	bbs.monitor1.exit(object, rid)
	return nil
}

// RESPOND_ON_ACTION_ERROR is an action error and makes a
// response for it.
func (bbs *Bb_server) respond_on_action_error(ctx context.Context, w http.ResponseWriter, r *http.Request, e error) {
	var e1, ok = e.(*Aws_s3_error)
	if !ok {
		log.Fatalf("Bad error from action: %#v", e)
	}
	bbs.Logger.Info(string(e1.Code), "error", e1)

	var rid int64 = get_request_id(ctx)

	e1.RequestId = fmt.Sprintf("%016x", rid)
	var m = Aws_s3_error_to_message[e1.Code]
	if len(e1.Message) == 0 {
		e1.Message = m.Message
		//fmt.Printf("e1=%#v\n", e1)
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(m.Status)
	var w1 = http.NewResponseController(w)
	var err1 = xml.NewEncoder(w).Encode(e1)
	if err1 != nil {
		bbs.Logger.Error("xml-encoder failure", "error", err1)
		panic(fmt.Errorf("xml-encoder failure: %w", err1))
	}
	w1.Flush()
}

// RESPOND_ON_INPUT_ERROR is an action error and makes a
// response for it.
func (bbs *Bb_server) respond_on_input_error(ctx context.Context, w http.ResponseWriter, r *http.Request, m map[string]error) {
	if len(m) == 0 {
		log.Fatalf("BAD-IMPL: error handler is called without errors: %#v", m)
	}
	var e error
	for _, e = range m {
		break
	}
	var err1 = &Aws_s3_error{Code: InvalidArgument, Message: e.Error()}
	bbs.respond_on_action_error(ctx, w, r, err1)
}

// COPE_WRITE_ERROR is called on a write error of response payload and
// makes a response for it.
func (bbs *Bb_server) cope_write_error(ctx context.Context, w http.ResponseWriter, r *http.Request, e error) {
	panic(e)
}

func check_usual_object_setup(ctx context.Context, bbs *Bb_server, bucket1 *string, key1 *string) (string, error) {
	if bucket1 == nil {
		log.Fatalf("BAD-IMPL: Bucket parameter missing")
	}
	var bucket = *bucket1
	if !check_bucket_naming(bucket) {
		var errz = &Aws_s3_error{Code: InvalidBucketName}
		return "", errz
	}

	if key1 == nil {
		log.Fatalf("BAD-IMPL: Key parameter missing")
	}
	var key = *key1
	if strings.HasPrefix(key, "..") {
		log.Fatalf("BAD-IMPL: Key parameter not clean")
	}
	if !check_object_naming(key) {
		var errz = &Aws_s3_error{Code: InvalidArgument}
		return "", errz
	}

	var err2 = bbs.check_bucket_directory_exists(ctx, bucket)
	if err2 != nil {
		return "", err2
	}

	var object = path.Join(bucket, key)
	return object, nil
}

func check_usual_bucket_setup(ctx context.Context, bbs *Bb_server, bucket1 *string) (string, error) {
	if bucket1 == nil {
		log.Fatalf("BAD-IMPL: Bucket parameter missing")
	}
	var bucket = *bucket1
	if !check_bucket_naming(bucket) {
		var errz = &Aws_s3_error{Code: InvalidBucketName}
		return "", errz
	}

	var err2 = bbs.check_bucket_directory_exists(ctx, bucket)
	if err2 != nil {
		return "", err2
	}

	return bucket, nil
}

func check_unsupported_options(object string, storageclass types.StorageClass) error {
	var location = "/" + object
	if storageclass != "" {
		if storageclass != types.StorageClassStandard {
			var errz = &Aws_s3_error{Code: InvalidStorageClass,
				Message:  "Bad x-amz-storage-class",
				Resource: location}
			return errz
		}
	}
	return nil
}

// SCAN_RANGE scans ranges in rfc9110.  Ranges exceeding file size are
// rejected.
func scan_range(rangestring *string, size int64, location string) (*[2]int64, error) {
	var extent *[2]int64
	if rangestring != nil {
		var r, err3 = httpaide.Scan_rfc9110_range(*rangestring)
		if err3 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Resource: location,
				Message:  "Range format is illegal."}
			return nil, errz
		}
		if len(r) != 1 {
			var errz = &Aws_s3_error{Code: InvalidRange,
				Resource: location,
				Message:  "Range is not more than one."}
			return nil, errz
		}
		if extent[1] > size {
			var errz = &Aws_s3_error{Code: InvalidRange,
				Resource: location}
			return nil, errz
		}

		// Fix an unspecified upper bound.

		if r[0][1] == -1 {
			extent = &[2]int64{r[0][0], size}
		} else {
			extent = &r[0]
		}
	}
	return extent, nil
}

func (bbs *Bb_server) check_conditions(ctx context.Context, match, none_match *string, modified_since, unmodified_since *time.Time) (bool, error) {
	if match != nil || none_match != nil {
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message: "if-match and if-none-match are unsupported"}
		return false, errz
	}
	if modified_since != nil || unmodified_since != nil {
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message: "if-modified-since and if-unmodified-since are unsupported"}
		return false, errz
	}
	return true, nil
}

// MAKE_META_INFO makes a meta-info from i.Metadata and i.Tagging.
func make_meta_info(headers map[string]string, tagging *string, location string) (*Meta_info, error) {
	var tags *types.Tagging
	if tagging != nil {
		var tags1, err1 = parse_tags(*tagging)
		if err1 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message:  "Tag format error.",
				Resource: location}
			return nil, errz
		}
		tags = tags1
	}
	if tags != nil || headers != nil {
		return &Meta_info{Headers: headers, Tags: tags}, nil
	} else {
		return nil, nil
	}
}

// PARSE_TAGS scans tags in a requst.  (Tag set must be encoded as URL
// query parameters).
func parse_tags(s string) (*types.Tagging, error) {
	var m, err1 = url.ParseQuery(s)
	if err1 != nil {
		return nil, err1
	}
	var tags = []types.Tag{}
	for k, v := range m {
		if len(v) != 1 {
			log.Printf("ignore multiple values in tags\n")
		}
		var value string
		if len(v) == 0 {
			value = ""
		} else {
			value = v[0]
		}
		tags = append(tags, types.Tag{Key: &k, Value: &value})
	}
	if len(tags) == 0 {
		return nil, nil
	} else {
		var tagging = types.Tagging{TagSet: tags}
		return &tagging, nil
	}
}
