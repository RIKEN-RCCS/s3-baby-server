// hodgepodge.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"context"
	"encoding/base64"
	//"errors"
	"fmt"
	//"io/fs"
	//"os"
	"github.com/riken-rccs/s3-baby-server/pkg/httpaide"
	"path"
	"time"
	//"bytes"
	//"encoding/base64"
	//"encoding/binary"
	//"encoding/hex"
	"encoding/xml"
	//"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"log"
	//"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"slices"
	//"strconv"
	"strings"
	//"sync"
)

func bb_assert(c bool) {
	if !c {
		panic("assertion")
	}
}

type suffix_record struct {
	rid       int64
	timestamp time.Time
}

// MAKE_REQUEST_ID makes a new request-id.  It uses time, or when time
// does not advance, uses the last value plus one.  It is strictly
// increasing.
func (bbs *Bb_server) make_request_id() *int64 {
	var t int64 = time.Now().UnixMicro()
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

func get_request_action(ctx context.Context) string {
	var action = ctx.Value("action-name").(string)
	return action
}

// MAKE_SCRATCH_SUFFIX makes a key string for a scratch file.  It
// takes request-id or zero.  A key is valid during request processing
// when a request-id is given.  Otherwise, when zero is given, a key
// is for multipart upload and it is active until completion.  It
// returns a string of 6 hex-digits.
func (bbs *Bb_server) make_scratch_suffix(rid int64) string {
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

func (bbs *Bb_server) discharge_scratch_suffix(rid int64) {
	bbs.mutex.Lock()
	defer bbs.mutex.Unlock()
	for k, v := range bbs.suffixes {
		if v.rid == rid {
			delete(bbs.suffixes, k)
		}
	}
}

func (bbs *Bb_server) serialize_access(ctx context.Context, object string, rid int64) *Aws_s3_error {
	var ok = bbs.monitor1.enter(object, rid, (10 * time.Millisecond))
	if !ok {
		return &Aws_s3_error{Code: RequestTimeout}
	}
	return nil
}

func (bbs *Bb_server) release_access(ctx context.Context, object string, rid int64) *Aws_s3_error {
	bbs.monitor1.exit(object, rid)
	return nil
}

func make_parameter_error(name string, err error) error {
	return fmt.Errorf(("Parameter \"" + name + "\" error: %w"), err)
}

// RESPOND_ON_ACTION_ERROR is an action error and makes a
// response for it.
func (bbs *Bb_server) respond_on_action_error(ctx context.Context, w http.ResponseWriter, r *http.Request, e error) {
	var e1, ok = e.(*Aws_s3_error)
	if !ok {
		log.Fatalf("Bad error from action: %#v", e)
	}
	bbs.logger.Info("Error in action", "code", string(e1.Code), "error", e1)

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
		bbs.logger.Error("xml-encoder failure", "error", err1)
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

// XML_MARSHAL_ERROR is called on unmarshal failure
// in import functions for tag-affix.
func xml_marshal_error(ty string, e error) error {
	var err1 = fmt.Errorf("Marshaling for type %s with %w", ty, e)
	var errz = &Aws_s3_error{Code: MalformedXML,
		Message: err1.Error()}
	return errz
}

func check_usual_object_setup(ctx context.Context, bbs *Bb_server, bucket1 *string, key1 *string) (string, *Aws_s3_error) {
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

func check_usual_bucket_setup(ctx context.Context, bbs *Bb_server, bucket1 *string) (string, *Aws_s3_error) {
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

func check_options_unsupported(action string, i *option_check_list) *Aws_s3_error {
	if i.ExpectedBucketOwner != nil {
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message: "expected-bucket-owner is not supported."}
		return errz
	}
	if i.MFA != nil {
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message: "MFA is not supported."}
		return errz
	}
	if i.PartNumber != nil {
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message: "PartNumber is not supported."}
		return errz
	}
	if i.VersionId != nil {
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message: "Version-ID is not supported."}
		return errz
	}

	if i.ExpectedBucketOwner != nil {
		return &Aws_s3_error{Code: AccessDenied}
	}

	if i.FetchOwner != nil && *i.FetchOwner == true {
		return &Aws_s3_error{Code: AccessDenied}
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

func (bbs *Bb_server) check_options_ignored(action, resource string, i *option_check_list) *Aws_s3_error {
	if i.ACL_bucket_canned != "" {
		bbs.logger.Debug("x-amz-acl ignored",
			"action", action, "resource", resource)
	}
	if i.CreateBucketConfiguration != nil {
		bbs.logger.Debug("CreateBucketConfiguration ignored",
			"action", action, "resource", resource)
	}
	if i.ObjectOwnership != types.ObjectOwnershipBucketOwnerEnforced {
		bbs.logger.Debug("x-amz-object-ownership ignored",
			"action", action, "resource", resource)
	}

	// Ignore "Cache-Control" totally.

	if i.CacheControl != nil {
		if !strings.EqualFold(*i.CacheControl, "no-cache") {
			bbs.logger.Info("Cache-Control header ignored",
				"action", action, "resource", resource,
				"directive", *i.CacheControl)
		}
	}

	return nil
}

// SCAN_RANGE parses ranges in rfc9110.  It returns a single range.
// Ranges exceeding the file size are rejected.
func scan_range(rangestring *string, size int64, location string) (*[2]int64, *Aws_s3_error) {
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
			extent = &r[0]
		}
		return extent, nil
	}
}

// Request condition handling described in RFC-7232.
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

// CHECK_REQUEST_CONDITIONALS checks conditions of "if-match",
// "if-none-match", "if-modified-since", and "if-unmodified-since".
// Conditionals are classified by a mode: "read" (GET/HEAD), "write"
// (PUT/POST), and "delete" (DELETE).  A mode may disagree with the
// method, when the object is a copy source.  It considers the equal
// time as included.
func (bbs *Bb_server) check_request_conditionals(object string, mode string, conditionals copy_conditionals) *Aws_s3_error {
	bb_assert(slices.Contains([]string{"read", "write", "delete"}, mode))

	// No conditions are unconditionally Okay.

	if conditionals == (copy_conditionals{}) {
		return nil
	}

	// Scan list of etags in the headers.

	var etags_include []string
	var etags_exclude []string
	if conditionals.some_match != nil {
		var m1, err1 = httpaide.Scan_rfc7232_etags(*conditionals.some_match)
		if err1 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message: "Bad if-match."}
			return errz
		}
		etags_include = m1
	}
	if conditionals.none_match != nil {
		var m2, err2 = httpaide.Scan_rfc7232_etags(*conditionals.none_match)
		if err2 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message: "Bad if-none-match."}
			return errz
		}
		etags_exclude = m2
	}

	// Fetch status of an object.  It accepts non-existing case.

	var stat, etag, err1 = bbs.fetch_object_status(object)
	if err1 != nil {
		return err1
	}
	var nonexist = (stat == nil)

	var mtime time.Time
	var size int64
	if stat != nil {
		mtime = stat.ModTime()
		size = stat.Size()
	} else {
		mtime = time.Time{}
		size = -1
	}

	if nonexist && mode == "delete" {
		// OK.
		return nil
	}

	// Evaluate conditions in the order specified in RFC-7232.

	if etags_include != nil {
		// "if-match"
		if !nonexist && match_etags_is_star(etags_include) {
			// Always matches.
		} else if nonexist || !slices.Contains(etags_include, etag) {
			var errz = &Aws_s3_error{Code: PreconditionFailed,
				Message: "Condition if-match fails."}
			return errz
		}
	} else if conditionals.modified_before != nil {
		// "if-unmodified-since"
		if nonexist || !(mtime.Compare(*conditionals.modified_before) <= 0) {
			var errz = &Aws_s3_error{Code: PreconditionFailed,
				Message: "Condition if-unmodified-since fails."}
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
			// OK.
		} else if match_etags_is_star(etags_include) {
			var errz = &Aws_s3_error{Code: errorcode,
				Message: "Condition if-none-match fails."}
			return errz
		} else if slices.Contains(etags_exclude, etag) {
			var errz = &Aws_s3_error{Code: errorcode,
				Message: "Condition if-none-match fails."}
			return errz
		}
	} else if conditionals.modified_after != nil {
		// "if-modified-since"
		if nonexist || !(conditionals.modified_after.Compare(mtime) <= 0) {
			var errz = &Aws_s3_error{Code: NotModified,
				Message: "Condition if-modified-since fails."}
			return errz
		}
	}

	if conditionals.modified_time != nil {
		// "x-amz-if-match-last-modified-time"
		if nonexist {
			// OK.
		} else if !conditionals.modified_time.Equal(mtime) {
			var errz = &Aws_s3_error{Code: PreconditionFailed,
				Message: "Condition x-amz-if-match-last-modified-time fails."}
			return errz
		}
	}

	if conditionals.size != nil {
		// "x-amz-if-match-size"
		if nonexist {
			// OK.
		} else if !(*conditionals.size == size) {
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

// MAKE_METAINFO makes a metainfo from i.Metadata and i.Tagging.
func (bbs *Bb_server) make_metainfo(headers map[string]string, tagging *string, location string) (*Meta_info, *Aws_s3_error) {
	var tags, err1 = bbs.parse_tags(tagging, location)
	if err1 != nil {
		return nil, err1
	}
	if tags != nil || headers != nil {
		return &Meta_info{Headers: headers, Tags: tags}, nil
	} else {
		return nil, nil
	}
}

// PARSE_TAGS scans tags in a request.  Note a tag-set is encoded as
// URL query parameters.
func (bbs *Bb_server) parse_tags(s *string, location string) (*types.Tagging, *Aws_s3_error) {
	if s == nil {
		return nil, nil
	}
	var m, err1 = url.ParseQuery(*s)
	if err1 != nil {
		bbs.logger.Info("Parse_tags: .ParseQuery() failed",
			"error", err1)
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "Tag format error.",
			Resource: location}
		return nil, errz
	}
	var tags = []types.Tag{}
	for k, v := range m {
		if len(v) != 1 {
			bbs.logger.Info("Parse_tags: Multiple values in tags, ignored")
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

func (bbs *Bb_server) lookat_part_number(object string, partnumber *int32) (int32, *Aws_s3_error) {
	var location = "/" + object
	if partnumber == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "PartNumber missing.",
			Resource: location}
		return 0, errz
	} else {
		var part = *partnumber
		if part < 1 || max_part_number < part {
			var errz = &Aws_s3_error{Code: InvalidPart,
				Resource: location}
			return 0, errz
		}
		return part, nil
	}
}

func (bbs *Bb_server) lookat_copy_source(object string, copysource *string) (string, *Aws_s3_error) {
	if copysource == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message: "No x-amz-copy-source supplied."}
		return "", errz
	}
	var u, err3 = url.Parse(*copysource)
	if err3 != nil {
		bbs.logger.Debug("url.Parse() fail",
			"string", *copysource, "error", err3)
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

func decode_checksum_value(object string, checksum types.ChecksumAlgorithm, csumset *types.Checksum) ([]byte, *Aws_s3_error) {
	var location = "/" + object
	if checksum == "" {
		return nil, nil
	}
	var csum1 *string
	switch checksum {
	case types.ChecksumAlgorithmCrc32:
		csum1 = csumset.ChecksumCRC32
	case types.ChecksumAlgorithmCrc32c:
		csum1 = csumset.ChecksumCRC32C
	case types.ChecksumAlgorithmSha1:
		csum1 = csumset.ChecksumSHA1
	case types.ChecksumAlgorithmSha256:
		csum1 = csumset.ChecksumSHA256
	case types.ChecksumAlgorithmCrc64nvme:
		csum1 = csumset.ChecksumCRC64NVME
	default:
		log.Fatalf("BAD-IMPL: Bad s3/types.ChecksumAlgorithm: %s", checksum)
	}
	if csum1 == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "Checksum value is missing.",
			Resource: location}
		return nil, errz
	}
	var csum2, err5 = base64.StdEncoding.DecodeString(*csum1)
	if err5 != nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "Bad checksum encoding.",
			Resource: location}
		return nil, errz
	}
	return csum2, nil
}

func metainfo_zero(m *Meta_info) bool {
	if m == nil {
		return true
	}
	//m.ContentDisposition == nil &&
	//m.ContentEncoding == nil &&
	//m.ContentLanguage == nil &&
	//m.ContentType == nil &&
	//m.Expires == nil &&
	return (m.Headers == nil &&
		m.Tags == nil)
}
