// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Small Functions

package server

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/riken-rccs/s3-baby-server/pkg/httpaide"
)

func bbs_assert(c bool) {
	if !c {
		panic("assertion")
	}
}

type scratch_suffix struct {
	rid       uint64
	timestamp time.Time
}

// PRINTF/FATALF for debug printing.
var Printf = fmt.Printf
var Fatalf = log.Fatalf

// MAKE_REQUEST_ID makes a new request-id.  It uses time, or when time
// does not advance, uses the last value plus one.  It is strictly
// increasing.
func (bbs *Bbs_server) make_request_id() uint64 {
	var t uint64 = uint64(time.Now().UnixMicro())
	bbs.mutex.Lock()
	defer bbs.mutex.Unlock()
	if bbs.rid_past < t {
		bbs.rid_past = t
	} else {
		t = bbs.rid_past + 1
		bbs.rid_past = t
	}
	//return strconv.FormatInt(t, 16)
	//return fmt.Sprintf("%016x", t)
	return t
}

// RESPOND_ON_ACTION_ERROR is called on an action error and makes a
// response for it.  Note on a status=304 error, it cannot have a
// response body.  In addition, such an error is required to return
// some headers ("ETag" for instance).  An error record may contain
// values for the headers.
func (bbs *Bbs_server) respond_on_action_error(ctx context.Context, w http.ResponseWriter, r *http.Request, e error) {
	var e1, ok = e.(*Aws_s3_error)
	if !ok {
		log.Fatalf("Bad error from action: %#v", e)
	}
	var action, rid, _ = get_action_name(ctx)
	bbs.logger.Info("Error in action",
		"action", action, "rid", rid, "code", string(e1.Code), "error", e1)

	e1.RequestId = fmt.Sprintf("%016x", rid)
	var m = Aws_s3_error_to_message[e1.Code]
	if len(e1.Message) == 0 {
		e1.Message = m.Message
	}

	if e1.headers != nil {
		for k, vv := range e1.headers {
			for _, v := range vv {
				w.Header().Set(k, v)
			}
		}
	}

	switch m.Status {
	case 304:
		w.WriteHeader(m.Status)

	default:
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(m.Status)
		var w1 = http.NewResponseController(w)
		var err1 = xml.NewEncoder(w).Encode(e1)
		if err1 != nil {
			bbs.logger.Error("xml.Encode() in writing a response failed",
				"action", action, "rid", rid, "error", err1)
			/*panic(fmt.Errorf("xml-encode failed: %w", err1))*/
		}
		w1.Flush()
	}
}

// RESPOND_ON_INPUT_ERROR is an action error and makes a
// response for it.
func (bbs *Bbs_server) respond_on_input_error(ctx context.Context, w http.ResponseWriter, r *http.Request, m map[string]error) {
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

// COPE_WITH_WRITE_ERROR is called on a write error of response
// payload and makes a response for it.
func (bbs *Bbs_server) cope_with_write_error(ctx context.Context, w http.ResponseWriter, r *http.Request, e error) {
	var action, rid, _ = get_action_name(ctx)
	bbs.logger.Info("Writing response failed",
		"action", action, "rid", rid, "error", e)
}

// MAKE_SCRATCH_SUFFIX makes a key string for a scratch file.  It
// returns a string of "@" plus 6 hex-digits.  A key is valid during
// request processing with a request-id.  It takes a request-id.
func (bbs *Bbs_server) make_scratch_suffix(rid uint64) string {
	bbs.mutex.Lock()
	defer bbs.mutex.Unlock()
	var loops int = 0
	for true {
		var r = rand.Int63()
		var s = "@" + fmt.Sprintf("%06x", r)[:6]
		var _, ok = bbs.suffixes[s]
		if !ok {
			bbs.suffixes[s] = scratch_suffix{rid, time.Now()}
			return s
		}
		loops++
		if loops >= 10 {
			log.Fatal("BAD-IMPL: unique key generation loops forever")
		}
	}
	panic("NEVER")
}

func (bbs *Bbs_server) discharge_scratch_suffix(rid uint64, suffix string) {
	bbs.mutex.Lock()
	defer bbs.mutex.Unlock()
	for k, v := range bbs.suffixes {
		if v.rid == rid {
			delete(bbs.suffixes, k)
		}
	}
}

// MAKE_NEW_UPLOAD_ID makes a key string for a upload-id.  Its
// uniqueness is NOT guaranteed.  It is only by probability.
func (bbs *Bbs_server) make_new_upload_id() string {
	var r = rand.Uint32()
	var s = fmt.Sprintf("%08x", r)
	return s
}

func (bbs *Bbs_server) serialize_access(ctx context.Context, object string, rid uint64) *Aws_s3_error {
	var duration = time_msec_duration(bbs.config.Exclusion_wait)
	var ok, elapse = bbs.monitor1.Enter(object, rid, duration)
	if !ok {
		bbs.logger.Warn("Timeout in entering monitor",
			"rid", rid, "elapse", elapse)
		return &Aws_s3_error{Code: RequestTimeout}
	}
	if bbs.config.Log_monitor_timing {
		bbs.logger.Debug("Time to enter monitor",
			"rid", rid, "elapse", elapse)
	}
	return nil
}

func (bbs *Bbs_server) release_access(ctx context.Context, object string, rid uint64) *Aws_s3_error {
	var elapse = bbs.monitor1.Exit(object, rid)
	if bbs.config.Log_monitor_timing {
		bbs.logger.Debug("Time grabbed in exclusion",
			"rid", rid, "elapse", elapse)
	}
	return nil
}

func (bbs *Bbs_server) test_access_serialized(ctx context.Context, object string, rid uint64) bool {
	var ok = bbs.monitor1.Attest(object, rid)
	return ok
}

func make_parameter_error(name string, err error) error {
	return fmt.Errorf(("Parameter \"" + name + "\" error: %w"), err)
}

func (bbs *Bbs_server) check_usual_object_setup(rid uint64, bucket1 *string, key1 *string) (string, *Aws_s3_error) {
	if bucket1 == nil {
		bbs.logger.Debug("Bucket name missing in request",
			"rid", rid)
		var errz = &Aws_s3_error{Code: InvalidBucketName,
			Message: "Bucket name missing in request."}
		return "", errz
	}
	if key1 == nil {
		bbs.logger.Debug("Object key name missing in request",
			"rid", rid)
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message: "Object key name missing in request."}
		return "", errz
	}
	var bucket = *bucket1
	var key = *key1

	if !check_bucket_naming(bucket) {
		bbs.logger.Debug("Invalid bucket naming",
			"rid", rid, "bucket", bucket, "key", key)
		var errz = &Aws_s3_error{Code: InvalidBucketName}
		return "", errz
	}

	if strings.HasPrefix(key, "..") {
		log.Fatalf("BAD-IMPL: Key parameter not clean")
	}
	if !check_object_naming(key) {
		bbs.logger.Debug("Invalid object naming",
			"rid", rid, "bucket", bucket, "key", key)
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message: "Invalid object naming."}
		return "", errz
	}

	var err2 = bbs.check_bucket_directory_exists(rid, bucket)
	if err2 != nil {
		return "", err2
	}

	var object = path.Join(bucket, key)
	return object, nil
}

func (bbs *Bbs_server) check_usual_bucket_setup(rid uint64, bucket1 *string) (string, *Aws_s3_error) {
	if bucket1 == nil {
		log.Fatalf("BAD-IMPL: Bucket parameter missing")
	}
	var bucket = *bucket1
	if !check_bucket_naming(bucket) {
		var errz = &Aws_s3_error{Code: InvalidBucketName}
		return "", errz
	}

	var err2 = bbs.check_bucket_directory_exists(rid, bucket)
	if err2 != nil {
		return "", err2
	}

	return bucket, nil
}

type option_check_list struct {
	ACL_bucket_canned              types.BucketCannedACL
	ACL_object_canned              types.ObjectCannedACL
	BucketKeyEnabled               *bool
	BucketRegion                   *string
	BypassGovernanceRetention      *bool
	CacheControl                   *string
	ChecksumAlgorithm              types.ChecksumAlgorithm
	ChecksumCRC32                  *string
	ChecksumCRC32C                 *string
	ChecksumCRC64NVME              *string
	ChecksumMode                   types.ChecksumMode
	ChecksumSHA1                   *string
	ChecksumSHA256                 *string
	ChecksumType                   types.ChecksumType
	ContentDisposition             *string
	ContentEncoding                *string
	ContentLanguage                *string
	ContentLength                  *int64
	ContentMD5                     *string
	ContentType                    *string
	ContinuationToken              *string
	CopySourceIfMatch              *string
	CopySourceIfModifiedSince      *time.Time
	CopySourceIfNoneMatch          *string
	CopySourceIfUnmodifiedSince    *time.Time
	CopySourceRange                *string
	CopySourceSSECustomerAlgorithm *string
	CopySourceSSECustomerKey       *string
	CopySourceSSECustomerKeyMD5    *string
	CreateBucketConfiguration      *types.CreateBucketConfiguration
	Delete                         *types.Delete
	Delimiter                      *string
	EncodingType                   types.EncodingType
	ExpectedBucketOwner            *string
	ExpectedSourceBucketOwner      *string
	Expires                        *time.Time
	FetchOwner                     *bool
	GrantFullControl               *string
	GrantRead                      *string
	GrantReadACP                   *string
	GrantWrite                     *string
	GrantWriteACP                  *string
	IfMatch                        *string
	IfMatchInitiatedTime           *time.Time
	IfMatchLastModifiedTime        *time.Time
	IfMatchSize                    *int64
	IfModifiedSince                *time.Time
	IfNoneMatch                    *string
	IfUnmodifiedSince              *time.Time
	KeyMarker                      *string
	MFA                            *string
	Marker                         *string
	MaxBuckets                     *int32
	MaxKeys                        *int32
	MaxParts                       *int32
	MaxUploads                     *int32
	Metadata                       map[string]string
	MetadataDirective              types.MetadataDirective
	MpuObjectSize                  *int64
	MultipartUpload                *types.CompletedMultipartUpload
	ObjectAttributes               []types.ObjectAttributes
	ObjectLockEnabledForBucket     *bool
	ObjectLockLegalHoldStatus      types.ObjectLockLegalHoldStatus
	ObjectLockMode                 types.ObjectLockMode
	ObjectLockRetainUntilDate      *time.Time
	ObjectOwnership                types.ObjectOwnership
	OptionalObjectAttributes       []types.OptionalObjectAttributes
	PartNumber                     *int32
	PartNumberMarker               *string
	Prefix                         *string
	Range                          *string
	RequestPayer                   types.RequestPayer
	ResponseCacheControl           *string
	ResponseContentDisposition     *string
	ResponseContentEncoding        *string
	ResponseContentLanguage        *string
	ResponseContentType            *string
	ResponseExpires                *time.Time
	SSECustomerAlgorithm           *string
	SSECustomerKey                 *string
	SSECustomerKeyMD5              *string
	SSEKMSEncryptionContext        *string
	SSEKMSKeyId                    *string
	ServerSideEncryption           types.ServerSideEncryption
	StartAfter                     *string
	StorageClass                   types.StorageClass
	Tagging_string                 *string
	Tagging_tagging                *types.Tagging
	TaggingDirective               types.TaggingDirective
	UploadId                       *string
	UploadIdMarker                 *string
	VersionId                      *string
	WebsiteRedirectLocation        *string
	WriteOffsetBytes               *int64
}

func check_options_unsupported(bbs *Bbs_server, action string, i *option_check_list) *Aws_s3_error {
	if i.ExpectedBucketOwner != nil {
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message: "x-amz-expected-bucket-owner is not supported."}
		return errz
	}
	if i.MFA != nil {
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message: "x-amz-mfa is not supported."}
		return errz
	}
	if i.PartNumber != nil {
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message: "partNumber is not supported."}
		return errz
	}
	if i.VersionId != nil {
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message: "versionId is not supported."}
		return errz
	}

	if i.ExpectedBucketOwner != nil {
		return &Aws_s3_error{Code: AccessDenied,
			Message: "x-amz-expected-bucket-owner is not supported."}
	}

	// Query "fetch-owner" is not supported but usually it is ignored.
	// Enabling accept_fetch_owner causes an error.

	if bbs.config.Accept_fetch_owner {
		if i.FetchOwner != nil && *i.FetchOwner == true {
			return &Aws_s3_error{Code: AccessDenied,
				Message: "fetch-owner is not allowed."}
		}
	}

	// Options that support only the restricted set.

	if i.StorageClass != "" {
		if i.StorageClass != types.StorageClassStandard {
			var errz = &Aws_s3_error{Code: InvalidStorageClass,
				Message: "Bad x-amz-storage-class."}
			return errz
		}
	}

	return nil
}

func (bbs *Bbs_server) check_options_ignored(action string, rid uint64, resource string, i *option_check_list) *Aws_s3_error {
	if false {
		if i.ACL_bucket_canned != "" {
			bbs.logger.Debug("x-amz-acl ignored",
				"action", action, "rid", rid, "resource", resource)
		}
		if i.CreateBucketConfiguration != nil {
			bbs.logger.Debug("CreateBucketConfiguration ignored",
				"action", action, "rid", rid, "resource", resource)
		}
		if i.ObjectOwnership != types.ObjectOwnershipBucketOwnerEnforced {
			bbs.logger.Debug("x-amz-object-ownership ignored",
				"action", action, "rid", rid, "resource", resource)
		}
	}

	// Ignore "Cache-Control" totally.

	if i.CacheControl != nil {
		if !strings.EqualFold(*i.CacheControl, "no-cache") {
			bbs.logger.Info("Cache-Control header ignored",
				"action", action, "rid", rid, "resource", resource,
				"directive", *i.CacheControl)
		}
	}

	return nil
}

// SCAN_RANGE parses ranges in rfc-9110.  It returns a single range,
// or it returns nil when no range is specified.  A returned range is
// bounded by the file size.  A range exceeding the file size are
// rejected.  Multiple ranges are rejected, too.  Note it fixes the
// upper bound of rfc-9110 which is inclusive.
func scan_range(object string, rangestring *string, size int64) (*[2]int64, *Aws_s3_error) {
	var location = "/" + object
	if rangestring == nil {
		return nil, nil
	} else {
		var r, err3 = httpaide.Scan_rfc9110_ranges(*rangestring)
		if err3 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message:  "Range format is illegal.",
				Resource: location}
			return nil, errz
		}
		if len(r) != 1 {
			var errz = &Aws_s3_error{Code: InvalidRange,
				Message:  "Range is not more than one.",
				Resource: location}
			return nil, errz
		}

		if r[0][1] > size {
			var errz = &Aws_s3_error{Code: InvalidRange,
				Resource: location}
			return nil, errz
		}

		// Fix an unspecified upper bound.

		var extent *[2]int64
		if r[0][1] == -1 {
			extent = &[2]int64{r[0][0], size}
		} else {
			extent = &[2]int64{r[0][0], (r[0][1] + 1)}
		}
		return extent, nil
	}
}

// Request Condition Handling -- It is described in RFC-7232.
//
// https://datatracker.ietf.org/doc/html/rfc7232
//
// The order of evaluating conditions:
// If-Match < If-Unmodified-Since (skip if If-Match exists)
// < If-None-Match < If-Modified-Since (skip if If-None-Match exists)
//
// Status code to be returned on failure:
//   - ¬If-Match → 412 Precondition Failed
//   - ¬If-Unmodified-Since → 412 Precondition Failed
//   - ¬If-None-Match (GET/HEAD) → 304 Not Modified
//   - ¬If-None-Match (other) → 412 Precondition Failed
//   - ¬If-Modified-Since → 304 Not Modified
//
// The simplified rule described in the AWS-S3 API document is below.
// It somewhat differs from RFC-7232:
//   - If-Match ∧ ¬If-Unmodified-Since → 200 OK
//   - ¬If-None-Match ∧ If-Modified-Since → 304 Not Modified

// MEMO: It should ignore a bad format http-date.  However, the header
// scanner generated by the stub-generator signals an error.

// CHECK_CONDITIONS checks conditions of "if-match", "if-none-match",
// "if-modified-since", "if-unmodified-since", and
// "x-amz-if-match-size".  Conditionals are classified by a mode:
// "read" (GET/HEAD), "write" (PUT/POST), and "delete" (DELETE).  The
// mode may disagree with the method, when the object is a copy
// source.  It considers an equal time as included.
func (bbs *Bbs_server) check_conditions(rid uint64, object string, etag string, mtime time.Time, size int64, mode string, conditions copy_conditions) *Aws_s3_error {
	bbs_assert(slices.Contains([]string{"read", "write", "delete"}, mode))

	// Empty conditions are unconditionally Okay.

	if conditions == (copy_conditions{}) {
		return nil
	}

	// Scan list of etags in the headers.

	var etags_include []string
	var etags_exclude []string
	if conditions.some_match != nil {
		var m1, err1 = httpaide.Scan_rfc7232_etags(*conditions.some_match)
		if err1 != nil {
			bbs.logger.Info("Bad condition format (if-match)",
				"rid", rid, "error", err1)
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message: "Bad if-match."}
			return errz
		}
		etags_include = m1
	}
	if conditions.none_match != nil {
		var m2, err2 = httpaide.Scan_rfc7232_etags(*conditions.none_match)
		if err2 != nil {
			bbs.logger.Info("Bad condition format (if-none-match)",
				"rid", rid, "error", err2)
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message: "Bad if-none-match."}
			return errz
		}
		etags_exclude = m2
	}

	// Fetch status of an object.  It accepts non-existing case.

	/*
		var _, stat, err1 = bbs.fetch_object_status(rid, object, false)
		if err1 != nil {
			bbs.logger.Info("Bad condition, object missing",
				"rid", rid, "error", err1)
			return err1
		}
	*/

	/*
		var mtime time.Time
		var size int64
		if stat != nil {
			mtime = stat.ModTime()
			size = stat.Size()
		} else {
			mtime = time.Time{}
			size = -1
		}
	*/

	var nonexist = (etag == "")

	if nonexist && mode == "delete" {
		// Okay.
		return nil
	}

	// Header values returned on an error.

	var mtimes = mtime.UTC().Format(time.RFC1123)
	var headers = map[string][]string{
		"ETag":          []string{etag},
		"Last-Modified": []string{mtimes}}

	// Evaluate conditions in the order specified in RFC-7232.

	if etags_include != nil {
		// "if-match" and "e.ETag" in DeleteObjects.
		if !nonexist && match_etags_is_star(etags_include) {
			// Always matches.
		} else if nonexist || !slices.Contains(etags_include, etag) {
			bbs.logger.Info("Conditional fails (if-match)",
				"rid", rid, "etag", etag, "etags_include", etags_include)
			var errz = &Aws_s3_error{Code: PreconditionFailed,
				Message: "Condition if-match fails.",
				headers: headers}
			return errz
		}
	} else if conditions.modified_before != nil {
		// "if-unmodified-since"
		if nonexist || !(mtime.Compare(*conditions.modified_before) <= 0) {
			bbs.logger.Info("Conditional fails (if-unmodified-since)",
				"rid", rid, "mtime", mtime,
				"modified_before", *conditions.modified_before)
			var errz = &Aws_s3_error{Code: PreconditionFailed,
				Message: "Condition if-unmodified-since fails.",
				headers: headers}
			return errz
		}
	}

	if etags_exclude != nil {
		// "if-none-match",
		var errorcode string
		if mode == "read" {
			errorcode = NotModified
		} else {
			errorcode = PreconditionFailed
		}
		if nonexist {
			// Okay.
		} else if match_etags_is_star(etags_exclude) {
			bbs.logger.Info("Conditional fails (if-none-match)",
				"rid", rid, "etag", etag, "etags_exclude", etags_exclude)
			var errz = &Aws_s3_error{Code: errorcode,
				Message: "Condition if-none-match fails.",
				headers: headers}
			return errz
		} else if slices.Contains(etags_exclude, etag) {
			bbs.logger.Info("Conditional fails (if-none-match)",
				"rid", rid, "etag", etag, "etags_exclude", etags_exclude)
			var errz = &Aws_s3_error{Code: errorcode,
				Message: "Condition if-none-match fails.",
				headers: headers}
			return errz
		}
	} else if conditions.modified_after != nil {
		// "if-modified-since"
		if nonexist || !(conditions.modified_after.Compare(mtime) <= 0) {
			bbs.logger.Info("Conditional fails (if-modified-since)",
				"rid", rid, "mtime", mtime,
				"modified_after", *conditions.modified_after)
			var errz = &Aws_s3_error{Code: NotModified,
				Message: "Condition if-modified-since fails.",
				headers: headers}
			return errz
		}
	}

	if conditions.modified_time != nil {
		// "x-amz-if-match-last-modified-time" and
		// "e.LastModifiedTime" in DeleteObjects.
		if nonexist {
			// Okay.
		} else if !conditions.modified_time.Equal(mtime) {
			bbs.logger.Info("Conditional fails (if-match-last-modified-time)",
				"rid", rid, "mtime", mtime,
				"modified_time", *conditions.modified_time)
			var errz = &Aws_s3_error{Code: PreconditionFailed,
				Message: "Condition x-amz-if-match-last-modified-time fails.",
				headers: headers}
			return errz
		}
	}

	if conditions.size != nil {
		// "x-amz-if-match-size" and "e.Size" in DeleteObjects.
		if nonexist {
			// Okay.
		} else if !(*conditions.size == size) {
			bbs.logger.Info("Conditional fails (if-match-size)",
				"rid", rid, "size", size,
				"specified_size", *conditions.size)
			var errz = &Aws_s3_error{Code: PreconditionFailed,
				Message: "Condition x-amz-if-match-size fails."}
			return errz
		}
	}

	return nil
}

func match_etags_is_star(etags []string) bool {
	return len(etags) == 1 && etags[0] == "*"
}

// PARSE_TAGS scans tags in a request.  Note a tag-set is encoded as
// URL query parameters.
func (bbs *Bbs_server) parse_tags(rid uint64, object string, s *string) (*types.Tagging, *Aws_s3_error) {
	var location = "/" + object
	if s == nil {
		return nil, nil
	}
	var m, err1 = url.ParseQuery(*s)
	if err1 != nil {
		bbs.logger.Info("url.ParseQuery() in parsing tags failed",
			"rid", rid, "error", err1)
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "Tag format error.",
			Resource: location}
		return nil, errz
	}
	var tags = []types.Tag{}
	for k, v := range m {
		if len(v) != 1 {
			bbs.logger.Info("Multiple values in tags, ignored",
				"rid", rid)
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

// LOOKAT_PART_NUMBER checks the part number.  Baby-server does not
// support by-part downloading, and specifying a part-number except
// for MPUL actions is an error.  required=true indicates by-part
// downloading.  As an exception, the part=1 is treated as the whole
// object.
func (bbs *Bbs_server) lookat_part_number(object string, partnumber *int32, required bool) (int32, *Aws_s3_error) {
	var location = "/" + object
	if partnumber == nil {
		if required {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message:  "PartNumber missing.",
				Resource: location}
			return 0, errz
		} else {
			return 0, nil
		}
	}
	var part = *partnumber
	if part < 1 || max_part_number < part {
		var errz = &Aws_s3_error{Code: InvalidPart,
			Resource: location}
		return 0, errz
	}
	if !required && part >= 2 {
		var errz = &Aws_s3_error{
			Code:     NoSuchUpload,
			Message:  "Object part unsupported.",
			Resource: location}
		return 0, errz
	}
	return part, nil
}

func (bbs *Bbs_server) lookat_copy_source(rid uint64, object string, copysource *string) (string, *Aws_s3_error) {
	if copysource == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message: "No x-amz-copy-source supplied."}
		return "", errz
	}
	var u, err3 = url.Parse(*copysource)
	if err3 != nil {
		bbs.logger.Debug("url.Parse() fail",
			"rid", rid, "string", *copysource, "error", err3)
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message: "Bad x-amz-copy-source."}
		return "", errz
	}
	var source = u.Path
	if !check_object_naming(source) {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message: "Bad x-amz-copy-source, bad naming."}
		return "", errz
	}
	var d1 = strings.Split(object, "/")
	var s1 = strings.Split(source, "/")
	if !(len(d1) >= 2 && len(s1) >= 2 && d1[0] == s1[0]) {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message: "x-amz-copy-source must be in the same bucket."}
		return "", errz
	}
	return source, nil
}

func decode_base64(object string, csum *string) ([]byte, *Aws_s3_error) {
	if csum == nil {
		return nil, nil
	} else {
		var location = "/" + object
		var csum2, err5 = base64.StdEncoding.DecodeString(*csum)
		if err5 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message:  "Bad base64 (MD5) encoding.",
				Resource: location}
			return nil, errz
		}
		return csum2, nil
	}
}

// METAINFO_NULL_FOR_ZERO checks if metainfo is empty.  Metainfo is
// empty if the slots are zero except an entity-key and an Etag.
func metainfo_null_for_zero(metainfo *Meta_info) *Meta_info {
	if metainfo == nil {
		return nil
	} else if metainfo.Checksum == "" &&
		metainfo.Csum == "" &&
		metainfo.Headers == nil &&
		metainfo.Tags == nil &&
		metainfo.CacheControl == nil &&
		metainfo.ContentDisposition == nil &&
		metainfo.ContentEncoding == nil &&
		metainfo.ContentLanguage == nil &&
		metainfo.ContentType == nil &&
		metainfo.Expires == nil {
		return nil
	} else {
		return metainfo
	}
}

// MERGE_METAINFO_WITH_CONTENT_HEADERS copies the source when given
// and merges the content header part.  It returns nil when metainfo
// is empty.
func merge_metainfo_with_content_headers(source, h *Meta_info) *Meta_info {
	var metainfo *Meta_info
	if source != nil {
		var m Meta_info = *source
		metainfo = &m
	} else {
		metainfo = &Meta_info{}
	}
	if h.CacheControl != nil {
		metainfo.CacheControl = h.CacheControl
	}
	if h.ContentDisposition != nil {
		metainfo.ContentDisposition = h.ContentDisposition
	}
	if h.ContentEncoding != nil {
		metainfo.ContentEncoding = h.ContentEncoding
	}
	if h.ContentLanguage != nil {
		metainfo.ContentLanguage = h.ContentLanguage
	}
	if h.ContentType != nil {
		metainfo.ContentType = h.ContentType
	}
	if h.Expires != nil {
		metainfo.Expires = h.Expires
	}
	return metainfo_null_for_zero(metainfo)
}

// FIX_ETAG_QUOTING adds double-quotes to an ETag, when some client
// accidentally dropped it.
func (bbs *Bbs_server) fix_etag_quoting(etag *string, rid uint64) *string {
	if etag == nil {
		return nil
	} else if bbs.config.Strict_etag_quoting {
		return etag
	} else {
		var etag1 = *etag
		var prefixed = strings.HasPrefix(etag1, "\"")
		var suffixed = strings.HasSuffix(etag1, "\"")
		if !prefixed || !suffixed {
			var etag2 string = etag1
			if !prefixed {
				etag2 = "\"" + etag2
			}
			if !suffixed {
				etag2 = etag2 + "\""
			}
			bbs.logger.Debug("Attach missing double-quotes to ETag",
				"rid", rid, "fixed-etag", etag2)
			return &etag2
		} else {
			return etag
		}
	}
}

func (bbs *Bbs_server) check_trailer_checksum(ctx context.Context, rid uint64, object string) (types.ChecksumAlgorithm, *Aws_s3_error) {
	var location = "/" + object
	var _, r = get_handler_arguments(ctx)
	var h = r.Header
	var keys = h["X-Amz-Trailer"]
	if len(keys) == 0 {
		return "", nil
	}
	var acc []types.ChecksumAlgorithm
	for _, k := range keys {
		var checksum = intern_checksum_algorithm_by_header_name(k)
		if checksum != "" {
			acc = append(acc, checksum)
		}
	}
	if len(acc) == 0 {
		return "", nil
	} else if len(acc) == 1 {
		return acc[0], nil
	} else {
		bbs.logger.Info("Multiple checksum headers in trailer",
			"rid", rid, "object", object, "trailer", keys)
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "Multiple checksum headers in trailer.",
			Resource: location}
		return "", errz
	}
}

func intern_checksum_algorithm_by_header_name(s string) types.ChecksumAlgorithm {
	var k = strings.ToLower(s)
	switch k {
	case "x-amz-checksum-crc32":
		return types.ChecksumAlgorithmCrc32
	case "x-amz-checksum-crc32c":
		return types.ChecksumAlgorithmCrc32c
	case "x-amz-checksum-crc64nvme":
		return types.ChecksumAlgorithmCrc64nvme
	case "x-amz-checksum-sha1":
		return types.ChecksumAlgorithmSha1
	case "x-amz-checksum-sha256":
		return types.ChecksumAlgorithmSha256
	default:
		return ""
	}
}
