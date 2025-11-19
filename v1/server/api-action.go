// api-action.go (2025-10-01)

// Copyright 2025-2025 RIKEN R-CCS.
// SPDX-License-Identifier: BSD-2-Clause

// API-STUB.  Handler templates. They should be replaced by
// actual implementations.

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
	"os"
	"path"
	"time"
	"github.com/riken-rccs/s3-baby-server/pkg/httpaide"
	"bytes"
	"encoding/base64"
	//"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
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

// make_meta_info makes a meta-info from i.Metadata and i.Tagging.
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
		return &Meta_info{Headers: &headers, Tags: tags}, nil
	} else {
		return nil, nil
	}
}

// AHO: I cannot find about nested tagging, while v1.1.1 code
// allowed nested tagging in values in the format
// 'TagSet=[{Key=<key>,Value=<value>}]'.

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

func (bbs *Bb_server) AbortMultipartUpload(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
	var o = s3.AbortMultipartUploadOutput{}

	// i.Bucket *string
	// i.Key *string
	// i.UploadId *string
	// i.ExpectedBucketOwner *string
	// i.IfMatchInitiatedTime *time.Time
	// i.RequestPayer types.RequestPayer

	// o.RequestCharged types.RequestCharged
	// o.ResultMetadata middleware.Metadata

	return &o, nil
}

func (bbs *Bb_server) CompleteMultipartUpload(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	var o = s3.CompleteMultipartUploadOutput{}

	// i.Bucket *string
	// i.Key *string
	// i.UploadId *string
	// i.ChecksumCRC32 *string
	// i.ChecksumCRC32C *string
	// i.ChecksumCRC64NVME *string
	// i.ChecksumSHA1 *string
	// i.ChecksumSHA256 *string
	// i.ChecksumType types.ChecksumType
	// i.ExpectedBucketOwner *string
	// i.IfMatch *string
	// i.IfNoneMatch *string
	// i.MpuObjectSize *int64
	// i.MultipartUpload *types.CompletedMultipartUpload
	// i.RequestPayer types.RequestPayer
	// i.SSECustomerAlgorithm *string
	// i.SSECustomerKey *string
	// i.SSECustomerKeyMD5 *string

	// o.Bucket *string
	// o.BucketKeyEnabled *bool
	// o.ChecksumCRC32 *string
	// o.ChecksumCRC32C *string
	// o.ChecksumCRC64NVME *string
	// o.ChecksumSHA1 *string
	// o.ChecksumSHA256 *string
	// o.ChecksumType types.ChecksumType
	// o.ETag *string
	// o.Expiration *string
	// o.Key *string
	// o.Location *string
	// o.RequestCharged types.RequestCharged
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption
	// o.VersionId *string
	// o.ResultMetadata middleware.Metadata

	return &o, nil
}

func (bbs *Bb_server) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	var o = s3.CopyObjectOutput{}

	// i.Bucket *string
	// i.CopySource *string
	// i.Key *string
	// i.ACL types.ObjectCannedACL
	// i.BucketKeyEnabled *bool
	// i.CacheControl *string
	// i.ChecksumAlgorithm types.ChecksumAlgorithm
	// i.ContentDisposition *string
	// i.ContentEncoding *string
	// i.ContentLanguage *string
	// i.ContentType *string
	// i.CopySourceIfMatch *string
	// i.CopySourceIfModifiedSince *time.Time
	// i.CopySourceIfNoneMatch *string
	// i.CopySourceIfUnmodifiedSince *time.Time
	// i.CopySourceSSECustomerAlgorithm *string
	// i.CopySourceSSECustomerKey *string
	// i.CopySourceSSECustomerKeyMD5 *string
	// i.ExpectedBucketOwner *string
	// i.ExpectedSourceBucketOwner *string
	// i.Expires *time.Time
	// i.GrantFullControl *string
	// i.GrantRead *string
	// i.GrantReadACP *string
	// i.GrantWriteACP *string
	// i.Metadata map[string]string
	// i.MetadataDirective types.MetadataDirective
	// i.ObjectLockLegalHoldStatus types.ObjectLockLegalHoldStatus
	// i.ObjectLockMode types.ObjectLockMode
	// i.ObjectLockRetainUntilDate *time.Time
	// i.RequestPayer types.RequestPayer
	// i.SSECustomerAlgorithm *string
	// i.SSECustomerKey *string
	// i.SSECustomerKeyMD5 *string
	// i.SSEKMSEncryptionContext *string
	// i.SSEKMSKeyId *string
	// i.ServerSideEncryption types.ServerSideEncryption
	// i.StorageClass types.StorageClass
	// i.Tagging *string
	// i.TaggingDirective types.TaggingDirective
	// i.WebsiteRedirectLocation *string

	// o.BucketKeyEnabled *bool
	// o.CopyObjectResult *types.CopyObjectResult
	// o.CopySourceVersionId *string
	// o.Expiration *string
	// o.RequestCharged types.RequestCharged
	// o.SSECustomerAlgorithm *string
	// o.SSECustomerKeyMD5 *string
	// o.SSEKMSEncryptionContext *string
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption
	// o.VersionId *string
	// o.ResultMetadata middleware.Metadata

	return &o, nil
}
func (bbs *Bb_server) CreateBucket(ctx context.Context, i *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	fmt.Printf("*CreateBucket*\n")
	var o = s3.CreateBucketOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.ACL types.BucketCannedACL
	// i.CreateBucketConfiguration *types.CreateBucketConfiguration
	// i.GrantFullControl *string
	// i.GrantRead *string
	// i.GrantReadACP *string
	// i.GrantWrite *string
	// i.GrantWriteACP *string
	// i.ObjectLockEnabledForBucket *bool
	// i.ObjectOwnership types.ObjectOwnership

	if i.Bucket == nil {
		log.Fatalf("BAD-IMPL: Bucket parameter missing")
	}
	var bucket = *i.Bucket
	if !check_bucket_naming(bucket) {
		var err5 = &Aws_s3_error{Code: InvalidBucketName}
		return nil, err5
	}

	var location = "/" + bucket

	var path = bbs.make_path_of_bucket(bucket)
	var err2 = os.Mkdir(path, 0755)
	if err2 != nil {
		// Note the error on existing path is fs.PathError and not
		// fs.ErrExist.

		/*if errors.As(err2, &err3) {*/
		/*if !errors.Is(err2, fs.ErrExist) {*/
		/*var err4, ok = err3.Err.(syscall.Errno)*/

		bbs.Logger.Info("os.Mkdir() failed", "error", err2)
		var m = map[error]Aws_s3_error_code{fs.ErrExist: BucketAlreadyOwnedByYou}
		var err5 = map_os_error(location, err2, m)
		return nil, err5
	}

	o.Location = &location
	return &o, nil
}

func (bbs *Bb_server) CreateMultipartUpload(ctx context.Context, i *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	var o = s3.CreateMultipartUploadOutput{}

	// i.Bucket *string
	// i.Key *string
	// i.ACL types.ObjectCannedACL
	// i.BucketKeyEnabled *bool
	// i.CacheControl *string
	// i.ChecksumAlgorithm types.ChecksumAlgorithm
	// i.ChecksumType types.ChecksumType
	// i.ContentDisposition *string
	// i.ContentEncoding *string
	// i.ContentLanguage *string
	// i.ContentType *string
	// i.ExpectedBucketOwner *string
	// i.Expires *time.Time
	// i.GrantFullControl *string
	// i.GrantRead *string
	// i.GrantReadACP *string
	// i.GrantWriteACP *string
	// i.Metadata map[string]string
	// i.ObjectLockLegalHoldStatus types.ObjectLockLegalHoldStatus
	// i.ObjectLockMode types.ObjectLockMode
	// i.ObjectLockRetainUntilDate *time.Time
	// i.RequestPayer types.RequestPayer
	// i.SSECustomerAlgorithm *string
	// i.SSECustomerKey *string
	// i.SSECustomerKeyMD5 *string
	// i.SSEKMSEncryptionContext *string
	// i.SSEKMSKeyId *string
	// i.ServerSideEncryption types.ServerSideEncryption
	// i.StorageClass types.StorageClass
	// i.Tagging *string
	// i.WebsiteRedirectLocation *string

	// (The tag-set must be encoded as URL Query parameters.)

	var object, err1 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err1 != nil {
		return nil, err1
	}
	var location = "/" + object
	var _ = location

	var err2 = check_unsupported_options(object, i.StorageClass)
	if err2 != nil {
		return nil, err2
	}
	var info, err3 = make_meta_info(i.Metadata, i.Tagging, location)
	if err3 != nil {
		return nil, err3
	}
	var _ = info

	var rid int64 = get_request_id(ctx)
	var scratchkey = bbs.make_file_suffix(rid)
	var cleanup_needed = true
	defer func() {
		bbs.discharge_file_suffix(rid)
		if cleanup_needed {
			bbs.discharge_scratch_file(ctx, object, scratchkey)
		}
	}()

	var _ = bbs.serialize_access(ctx, object, rid)
	defer bbs.release_access(ctx, object, rid)

	var err6 = bbs.create_upload_directory(ctx, object, scratchkey)
	if err6 != nil {
		return nil, err6
	}

	var uploadid = scratchkey
	var _ = uploadid

	{
		o.Bucket = i.Bucket
		o.Key = i.Key
		o.ChecksumAlgorithm = i.ChecksumAlgorithm
		o.ChecksumType = i.ChecksumType
		var uploadid = "0"
		o.UploadId = &uploadid
	}

	// o.AbortDate *time.Time
	// o.AbortRuleId *string
	// o.Bucket *string
	// o.BucketKeyEnabled *bool
	// o.ChecksumAlgorithm types.ChecksumAlgorithm
	// o.ChecksumType types.ChecksumType
	// o.Key *string
	// o.RequestCharged types.RequestCharged
	// o.SSECustomerAlgorithm *string
	// o.SSECustomerKeyMD5 *string
	// o.SSEKMSEncryptionContext *string
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption
	// o.UploadId *string

/*
	s := model.CreateMultipartUploadState{}
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	if s.Bucket = option.GetBucket(); !s3.FileSystem.isFileExists(s.Bucket) {
		return nil, NoSuchBucket()
	}
	s.Key = option.GetKey()
	if !s3.FileSystem.checkKeyName(s.Key) {
		return nil, KeyTooLongError()
	}
	var s3err *S3Error
	if s3err = s3.validateOptions(option); s3err != nil {
		return nil, s3err
	}
	if !s3.FileSystem.validateChecksumAlgorithm(option.GetOption("x-amz-checksum-algorithm")) {
		return nil, InvalidArgument()
	}
	if s.UploadID = s3.MultiPart.makeUploadID(); s.UploadID == -1 {
		return nil, InternalError()
	}
	result := s.MakeCreateMultipartUploadResult()
	if !s3.MultiPart.createMpUploadMeta(*result) {
		return nil, InternalError()
	}
	if s3err = s3.Tag.putTagging(option, s3.FileSystem.getPartNumberPath(utils.ToString(s.UploadID), s.Key)); s3err != nil {
		return nil, s3err
	}
*/

	return &o, nil
}

func (bbs *Bb_server) DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
	var o = s3.DeleteBucketOutput{}

	// i.Bucket *string
	// i.ExpectedBucketOwner *string

	// o.ResultMetadata middleware.Metadata

	return &o, nil
}

func (bbs *Bb_server) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	var o = s3.DeleteObjectOutput{}

	// i.Bucket *string
	// i.Key *string
	// i.BypassGovernanceRetention *bool
	// i.ExpectedBucketOwner *string
	// i.IfMatch *string
	// i.IfMatchLastModifiedTime *time.Time
	// i.IfMatchSize *int64
	// i.MFA *string
	// i.RequestPayer types.RequestPayer
	// i.VersionId *string

	// o.DeleteMarker *bool
	// o.RequestCharged types.RequestCharged
	// o.VersionId *string
	// o.ResultMetadata middleware.Metadata

	return &o, nil
}

func (bbs *Bb_server) DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	var o = s3.DeleteObjectsOutput{}

	// i.Bucket *string
	// i.Delete *types.Delete
	// i.BypassGovernanceRetention *bool
	// i.ChecksumAlgorithm types.ChecksumAlgorithm
	// i.ExpectedBucketOwner *string
	// i.MFA *string
	// i.RequestPayer types.RequestPayer

	// o.Deleted []types.DeletedObject
	// o.Errors []types.Error
	// o.RequestCharged types.RequestCharged
	// o.ResultMetadata middleware.Metadata

	return &o, nil
}

func (bbs *Bb_server) DeleteObjectTagging(ctx context.Context, params *s3.DeleteObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectTaggingOutput, error) {
	var o = s3.DeleteObjectTaggingOutput{}

	// i.Bucket *string
	// i.Key *string
	// i.ExpectedBucketOwner *string
	// i.VersionId *string
	// i.noSmithyDocumentSerde

	// o.VersionId *string
	// o.ResultMetadata middleware.Metadata
	// o.noSmithyDocumentSerde

	return &o, nil
}

func (bbs *Bb_server) GetObject(ctx context.Context, i *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	fmt.Printf("*GetObject*\n")
	var o = s3.GetObjectOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.Key *string
	// i.ChecksumMode types.ChecksumMode
	// i.ExpectedBucketOwner *string
	// i.IfMatch *string
	// i.IfModifiedSince *time.Time
	// i.IfNoneMatch *string
	// i.IfUnmodifiedSince *time.Time
	// i.PartNumber *int32
	// i.Range *string
	// i.RequestPayer types.RequestPayer
	// i.ResponseCacheControl *string
	// i.ResponseContentDisposition *string
	// i.ResponseContentEncoding *string
	// i.ResponseContentLanguage *string
	// i.ResponseContentType *string
	// i.ResponseExpires *time.Time
	// i.SSECustomerAlgorithm *string
	// i.SSECustomerKey *string
	// i.SSECustomerKeyMD5 *string
	// i.VersionId *string

	var object, err1 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err1 != nil {
		return nil, err1
	}
	var location = "/" + object

	var stat, err2 = bbs.fetch_file_stat(object)
	if err2 != nil {
		return nil, err2
	}

	var size = stat.Size()
	var extent, err3 = scan_range(i.Range, size, location)
	if err3 != nil {
		return nil, err3
	}
	if extent != nil {
		var length = extent[1] - extent[0]
		o.ContentLength = &length
		var rangei = fmt.Sprintf("bytes %d-%d/%d", extent[0], extent[1], size)
		o.ContentRange = &rangei
	} else {
		o.ContentLength = &size
	}
	var mtime = stat.ModTime()
	o.LastModified = &mtime

	var checksum = types.ChecksumAlgorithmCrc64nvme
	var md5, csum, err4 = bbs.calculate_csum2(checksum, object, "")
	if err4 != nil {
		return nil, err4
	}
	if i.ChecksumMode == types.ChecksumModeEnabled {
		o.ChecksumType = types.ChecksumTypeFullObject
		var crc = base64.StdEncoding.EncodeToString(csum)
		o.ChecksumCRC64NVME = &crc
	}
	o.ETag = make_etag_from_md5(md5)

	var _, err5 = bbs.check_conditions(ctx, i.IfMatch, i.IfNoneMatch,
		i.IfModifiedSince, i.IfUnmodifiedSince)
	if err5 != nil {
		return nil, err5
	}

	var info, err6 = bbs.fetch_metainfo(ctx, object)
	if err6 != nil {
		return nil, err6
	}
	if info != nil {
		// Always leave "MissingMeta" nil for zero.
		o.Metadata = *info.Headers
		o.MissingMeta = nil
	}
	if info != nil && info.Tags != nil {
		var count = int32(len(info.Tags.TagSet))
		if count > 0 {
			o.TagCount = &count
		}
	}

	var f1, err7 = bbs.make_file_stream(ctx, object, nil)
	if err7 != nil {
		return nil, err7
	}
	o.Body = f1

	{
		o.StorageClass = types.StorageClassStandard
		o.AcceptRanges = i.Range
		o.CacheControl = i.ResponseCacheControl
		o.ContentDisposition = i.ResponseContentDisposition
		o.ContentEncoding = i.ResponseContentEncoding
		o.ContentLanguage = i.ResponseContentLanguage
		o.ContentType = i.ResponseContentType
		if i.ResponseExpires != nil {
			var expires = i.ResponseExpires.Format(time.RFC3339)
			o.ExpiresString = &expires
		}
	}

	/*
		s := model.GetObjectState{}
		if err := s3.validateGetObjectOptions(option); err != nil {
			return nil, err
		}
		if !s3.FileSystem.checkBucketName(option.GetBucket()) {
			return nil, InvalidBucketName()
		}
		if err := s3.isBucketAndKeyExists(option.GetBucket(), option.GetPath()); err != nil {
			return nil, err
		}
		if !s3.FileSystem.checkKeyName(option.GetKey()) {
			return nil, KeyTooLongError()
		}
		if s.Content = s3.FileSystem.readFile(option.GetPath()); s.Content == nil {
			return nil, InternalError()
		}
		var s3err *S3Error
		if s.Info, s.ETag, s3err = s3.getFileInfoAndETag(option.GetPath()); s3err != nil {
			return nil, s3err
		}
		if s3err = s3.validateETagAndTime(option); s3err != nil {
			return nil, s3err
		}
		if s.ContentRange, s.Content, s3err = s3.getRangeContent(option, s.Content); s3err != nil {
			return nil, s3err
		}
		if !s3.getPartNumberContent(option) {
			return nil, InvalidArgument()
		}
		if s.ResponseCrc64nvme, s3err = s3.checkChecksumMode(option, s.Content); s3err != nil {
			return nil, s3err
		}
		s.TagCount, s.MissingMeta = s3.Tag.getTagCount(option.GetPath())
		result := s.MakeGetObjectResult()
		result.ContentDisposition = option.GetOption("response-content-disposition")
		result.ContentEncoding = option.GetOption("response-content-encoding")
		result.ContentLanguage = option.GetOption("response-content-language")
		result.ContentType = option.GetOption("response-content-type")
		return result, nil
	*/

	return &o, nil
}

func (bbs *Bb_server) GetObjectAttributes(ctx context.Context, params *s3.GetObjectAttributesInput, optFns ...func(*s3.Options)) (*s3.GetObjectAttributesOutput, error) {
	var o = s3.GetObjectAttributesOutput{}
	return &o, nil
}
func (bbs *Bb_server) GetObjectTagging(ctx context.Context, params *s3.GetObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.GetObjectTaggingOutput, error) {
	var o = s3.GetObjectTaggingOutput{}

	// i.Bucket *string
	// i.Key *string
	// i.ExpectedBucketOwner *string
	// i.RequestPayer types.RequestPayer
	// i.VersionId *string

	// o.TagSet []types.Tag
	// o.VersionId *string
	// o.ResultMetadata middleware.Metadata

	return &o, nil
}

func (bbs *Bb_server) HeadBucket(ctx context.Context, i *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	fmt.Printf("*HeadBucket*\n")
	var o = s3.HeadBucketOutput{}

	/*NOTYET*/

	// List of parameters.
	// i.Bucket *string
	// i.ExpectedBucketOwner *string

	var bucket, err1 = check_usual_bucket_setup(ctx, bbs, i.Bucket)
	if err1 != nil {
		return nil, err1
	}
	var _ = bucket

	// o.AccessPointAlias *bool
	// o.BucketArn *string
	// o.BucketLocationName *string
	// o.BucketLocationType types.LocationType
	// o.BucketRegion *string

	return &o, nil
}

func (bbs *Bb_server) HeadObject(ctx context.Context, i *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	fmt.Printf("*HeadObject*\n")
	var o = s3.HeadObjectOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.Key *string
	// i.ChecksumMode types.ChecksumMode
	// i.ExpectedBucketOwner *string
	// i.IfMatch *string
	// i.IfModifiedSince *time.Time
	// i.IfNoneMatch *string
	// i.IfUnmodifiedSince *time.Time
	// i.PartNumber *int32
	// i.Range *string
	// i.RequestPayer types.RequestPayer
	// i.ResponseCacheControl *string
	// i.ResponseContentDisposition *string
	// i.ResponseContentEncoding *string
	// i.ResponseContentLanguage *string
	// i.ResponseContentType *string
	// i.ResponseExpires *time.Time
	// i.SSECustomerAlgorithm *string
	// i.SSECustomerKey *string
	// i.SSECustomerKeyMD5 *string
	// i.VersionId *string

	var object, err1 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err1 != nil {
		return nil, err1
	}
	var location = "/" + object

	if i.IfMatch != nil || i.IfNoneMatch != nil {
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message: "if-match and if-none-match are unsupported"}
		return nil, errz
	}
	if i.IfModifiedSince != nil || i.IfUnmodifiedSince != nil {
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message: "if-modified-since and if-unmodified-since are unsupported"}
		return nil, errz
	}

	if i.PartNumber != nil && *i.PartNumber != 1 {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message: "PartNumber should be one always."}
		return nil, errz
	}
	var one int32 = 1
	o.PartsCount = &one

	var stat, err2 = bbs.fetch_file_stat(object)
	if err2 != nil {
		return nil, err2
	}

	var size = stat.Size()
	var extent, err3 = scan_range(i.Range, size, location)
	if err3 != nil {
		return nil, err3
	}
	if extent != nil {
		var length = extent[1] - extent[0]
		o.ContentLength = &length
		var srange = fmt.Sprintf("bytes %d-%d/%d", extent[0], extent[1], size)
		o.ContentRange = &srange
	} else {
		o.ContentLength = &size
	}
	var mtime = stat.ModTime()
	o.LastModified = &mtime

	var checksum = types.ChecksumAlgorithmCrc64nvme
	var md5, csum, err4 = bbs.calculate_csum2(checksum, object, "")
	if err4 != nil {
		return nil, err4
	}
	if i.ChecksumMode == types.ChecksumModeEnabled {
		o.ChecksumType = types.ChecksumTypeFullObject
		var crc = base64.StdEncoding.EncodeToString(csum)
		o.ChecksumCRC64NVME = &crc
	}
	o.ETag = make_etag_from_md5(md5)

	var info, err5 = bbs.fetch_metainfo(ctx, object)
	if err5 != nil {
		return nil, err5
	}
	if info != nil {
		// Always leave "MissingMeta" nil for zero.
		o.Metadata = *info.Headers
		o.MissingMeta = nil
	}
	if info != nil && info.Tags != nil {
		var count = int32(len(info.Tags.TagSet))
		if count > 0 {
			o.TagCount = &count
		}
	}

	{
		o.StorageClass = types.StorageClassStandard
		o.AcceptRanges = i.Range
		o.CacheControl = i.ResponseCacheControl
		o.ContentDisposition = i.ResponseContentDisposition
		o.ContentEncoding = i.ResponseContentEncoding
		o.ContentLanguage = i.ResponseContentLanguage
		o.ContentType = i.ResponseContentType
		if i.ResponseExpires != nil {
			var expires = i.ResponseExpires.Format(time.RFC3339)
			o.ExpiresString = &expires
		}
	}

	// o.ArchiveStatus types.ArchiveStatus
	// o.BucketKeyEnabled *bool
	// o.DeleteMarker *bool
	// o.Expiration *string
	// o.Expires *time.Time
	// o.ExpiresString *string
	// o.ObjectLockLegalHoldStatus types.ObjectLockLegalHoldStatus
	// o.ObjectLockMode types.ObjectLockMode
	// o.ObjectLockRetainUntilDate *time.Time
	// o.ReplicationStatus types.ReplicationStatus
	// o.RequestCharged types.RequestCharged
	// o.Restore *string
	// o.SSECustomerAlgorithm *string
	// o.SSECustomerKeyMD5 *string
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption
	// o.VersionId *string
	// o.WebsiteRedirectLocation *string

	/*
		s := model.GetObjectState{}
		if err := s3.validateGetObjectOptions(option); err != nil {
			return nil, err
		}
		if !s3.FileSystem.checkBucketName(option.GetBucket()) {
			return nil, InvalidBucketName()
		}
		path := option.GetPath()
		if err := s3.isBucketAndKeyExists(option.GetBucket(), path); err != nil {
			return nil, err
		}
		if !s3.FileSystem.checkKeyName(option.GetKey()) {
			return nil, KeyTooLongError()
		}
		if s.Content = s3.FileSystem.readFile(path); s.Content == nil {
			return nil, InternalError()
		}
		var s3err *S3Error
		if s.Info, s.ETag, s3err = s3.getFileInfoAndETag(path); s3err != nil {
			return nil, s3err
		}
		if s3err = s3.validateETagAndTime(option); s3err != nil {
			return nil, s3err
		}
		if s.ContentRange, s.Content, s3err = s3.getRangeContent(option, s.Content); s3err != nil {
			return nil, s3err
		}
		if !s3.getPartNumberContent(option) {
			return nil, InvalidArgument()
		}
		if s.ResponseCrc64nvme, s3err = s3.checkChecksumMode(option, s.Content); s3err != nil {
			return nil, s3err
		}
		result := s.MakeHeadObjectResult()
		result.ContentDisposition = option.GetOption("response-content-disposition")
		result.ContentEncoding = option.GetOption("response-content-encoding")
		result.ContentLanguage = option.GetOption("response-content-language")
		result.ContentType = option.GetOption("response-content-type")
	*/

	return &o, nil
}

func (bbs *Bb_server) ListBuckets(ctx context.Context, i *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	fmt.Printf("*ListBuckets*\n")
	var o = s3.ListBucketsOutput{}

	// List of parameters.
	// i.BucketRegion *string
	// i.ContinuationToken *string
	// i.MaxBuckets *int32
	// i.Prefix *string

	var start int
	if i.ContinuationToken != nil {
		var n, err1 = strconv.ParseInt(*i.ContinuationToken, 10, 32)
		if err1 != nil {
			var err2 = Bb_input_error{"continuation-token", err1}
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message: err2.Error()}
			return nil, errz
		}
		start = int(n)
	} else {
		start = 0
	}

	var max_buckets int
	if i.MaxBuckets != nil {
		max_buckets = int(*i.MaxBuckets)
		if max_buckets > list_buckets_limit {
			var err2 = fmt.Errorf("Value too large: %d", max_buckets)
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message: err2.Error()}
			return nil, errz
		}
	} else {
		max_buckets = list_buckets_limit
	}

	var pool_path = bbs.pool_path
	var entries1, err3 = os.ReadDir(pool_path)
	if err3 != nil {
		bbs.Logger.Info("os.ReadDir() failed in ListBuckets", "error", err3)
		var m = map[error]Aws_s3_error_code{}
		var err5 = map_os_error("/", err3, m)
		return nil, err5
	}

	if i.Prefix != nil {
		var prefix = *i.Prefix
		var dirs2 = []fs.DirEntry{}
		for _, e := range entries1 {
			if strings.HasPrefix(e.Name(), prefix) {
				dirs2 = append(dirs2, e)
			}
		}
	}

	// Filter only directories that satisfies bucket naming.
	// check_bucket_naming implies !strings.HasPrefix(name, ".").

	var dirs2 = []os.DirEntry{}
	for _, e := range entries1 {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") &&
			check_bucket_naming(e.Name()) {
			dirs2 = append(dirs2, e)
		}
	}

	var dirs3 []os.DirEntry
	var continuation int
	if start < len(dirs2) {
		var end = min(start+max_buckets, len(dirs2))
		dirs3 = dirs2[start:end]
		if end < len(dirs2) {
			continuation = end
		} else {
			continuation = 0
		}
	} else {
		dirs3 = []os.DirEntry{}
		continuation = 0
	}

	var buckets = []types.Bucket{}
	for _, e := range dirs3 {
		var stat, err4 = e.Info()
		if err4 != nil {
			// Skip the entry because it may be removed after scanning
			// directory.  SHOULD CHECK errors.Is(err, ErrNotExist).
			continue
		}
		var times, ok = file_time(stat)
		if !ok {
			var t0 = stat.ModTime()
			times = [3]time.Time{t0, t0, t0}
		}
		var name = e.Name()
		var b = types.Bucket{
			// BucketArn:,
			// BucketRegion:,
			CreationDate: &times[1],
			Name:         &name,
		}
		buckets = append(buckets, b)
	}

	o.Buckets = buckets
	if continuation != 0 {
		var scontinuation = strconv.FormatInt(int64(continuation), 10)
		o.ContinuationToken = &scontinuation
	}
	// o.Owner = &s3.Owner{DisplayName:, ID:}
	o.Prefix = i.Prefix

	return &o, nil
}

func (bbs *Bb_server) ListMultipartUploads(ctx context.Context, params *s3.ListMultipartUploadsInput, optFns ...func(*s3.Options)) (*s3.ListMultipartUploadsOutput, error) {
	var o = s3.ListMultipartUploadsOutput{}

	// i.Bucket *string
	// i.Delimiter *string
	// i.EncodingType types.EncodingType
	// i.ExpectedBucketOwner *string
	// i.KeyMarker *string
	// i.MaxUploads *int32
	// i.Prefix *string
	// i.RequestPayer types.RequestPayer
	// i.UploadIdMarker *string


	// o.Bucket *string
	// o.CommonPrefixes []types.CommonPrefix
	// o.Delimiter *string
	// o.EncodingType types.EncodingType
	// o.IsTruncated *bool
	// o.KeyMarker *string
	// o.MaxUploads *int32
	// o.NextKeyMarker *string
	// o.NextUploadIdMarker *string
	// o.Prefix *string
	// o.RequestCharged types.RequestCharged
	// o.UploadIdMarker *string
	// o.Uploads []types.MultipartUpload
	// o.ResultMetadata middleware.Metadata

	return &o, nil
}

func (bbs *Bb_server) ListObjects(ctx context.Context, i *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, error) {
	fmt.Printf("*ListObjects*\n")
	var o = s3.ListObjectsOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.Delimiter *string
	// i.EncodingType types.EncodingType
	// i.ExpectedBucketOwner *string
	// i.Marker *string
	// i.MaxKeys *int32
	// i.OptionalObjectAttributes []types.OptionalObjectAttributes
	// i.Prefix *string
	// i.RequestPayer types.RequestPayer

	var bucket, err1 = check_usual_bucket_setup(ctx, bbs, i.Bucket)
	if err1 != nil {
		return nil, err1
	}

	var index = 0
	var marker string
	var maxkeys int
	var delimiter string
	var prefix string

	if i.Marker != nil {
		marker = *i.Marker
	}
	if i.MaxKeys != nil {
		maxkeys = int(min(list_objects_limit, *i.MaxKeys))
	} else {
		maxkeys = list_objects_limit
	}
	if i.Delimiter != nil {
		delimiter = *i.Delimiter
	}
	if i.Prefix != nil {
		prefix = *i.Prefix
	}

	var entries []object_list_entry
	var nextindex int
	var nextmarker string
	var err2 error
	if !alwasy_use_flat_lister && delimiter == "/" {
		entries, nextindex, nextmarker, err2 = bbs.list_objects_delimited(
			bucket, index, marker, maxkeys, delimiter, prefix)
	} else {
		entries, nextindex, nextmarker, err2 = bbs.list_objects_flat(
			bucket, index, marker, maxkeys, delimiter, prefix)
	}
	if err2 != nil {
		return nil, err2
	}

	var contents, commonprefixes, err3 = bbs.make_list_objects_entries(
		entries, bucket, delimiter, prefix, false)
	var _ = err3
	var istruncated = (nextindex != 0)

	o.Contents = contents
	o.CommonPrefixes = commonprefixes
	o.IsTruncated = &istruncated
	o.NextMarker = &nextmarker

	// o.RequestCharged types.RequestCharged

	{
		// var maxkeys int32 = list_objects_limit
		o.Delimiter = i.Delimiter
		o.EncodingType = i.EncodingType
		o.Marker = i.Marker
		o.MaxKeys = i.MaxKeys
		o.Name = &bucket
		o.Prefix = i.Prefix
	}

	/*
		s := model.ListObjectsState{MaxKeys: 1000}
		if !s3.FileSystem.checkBucketName(option.GetBucket()) {
			return nil, InvalidBucketName()
		}
		s.Bucket = option.GetBucket()
		if !s3.FileSystem.isFileExists(s.Bucket) {
			return nil, NoSuchBucket()
		}
		s.BucketAPath = s3.FileSystem.getFullPath(s.Bucket)
		if v := option.GetOption("max-keys"); v != "" {
			if s.MaxKeys = utils.ToInt(v); s.MaxKeys > 1000 { // max-keysの上限は1000
				s.MaxKeys = 1000
			}
		}
		if v := option.GetOption("prefix"); v != "" {
			s.Prefix = v
		}
		if v := option.GetOption("marker"); v != "" {
			s.Marker = strings.ReplaceAll(v, "/", "\\")
		}
		if v := option.GetOption("delimiter"); v == "/" {
			s.Delimiter = filepath.FromSlash(v)
		}
		var s3err *S3Error
		if s.URLFlag, s3err = s3.checkEncodingType(option); s3err != nil {
			return nil, s3err
		}
		res, responseRes := s3.listObjects(s)
		if res == nil {
			return nil, NotImplemented()
		}
		result := s.MakeListObjectsResult(*responseRes)
		result.Contents = *res
	*/

	return &o, nil
}

func (bbs *Bb_server) ListObjectsV2(ctx context.Context, i *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	fmt.Printf("*ListObjectsV2*\n")
	var o = s3.ListObjectsV2Output{}

	// List of parameters.
	// i.Bucket *string
	// i.ContinuationToken *string
	// i.Delimiter *string
	// i.EncodingType types.EncodingType
	// i.ExpectedBucketOwner *string
	// i.FetchOwner *bool
	// i.MaxKeys *int32
	// i.OptionalObjectAttributes []types.OptionalObjectAttributes
	// i.Prefix *string
	// i.RequestPayer types.RequestPayer
	// i.StartAfter *string

	var bucket, err1 = check_usual_bucket_setup(ctx, bbs, i.Bucket)
	if err1 != nil {
		return nil, err1
	}

	if i.ExpectedBucketOwner != nil {
		return nil, &Aws_s3_error{Code: AccessDenied}
	}
	if i.FetchOwner != nil && *i.FetchOwner == true {
		return nil, &Aws_s3_error{Code: AccessDenied}
	}

	var index int
	var marker string
	var maxkeys int
	var delimiter string
	var prefix string
	var urlencode bool

	if i.ContinuationToken != nil {
		var n, err1 = strconv.ParseInt(*i.ContinuationToken, 10, 32)
		if err1 != nil {
			var err2 = Bb_input_error{"continuation-token", err1}
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message: err2.Error()}
			return nil, errz
		}
		index = int(n)
	}
	if i.StartAfter != nil {
		marker = *i.StartAfter
	}
	if i.MaxKeys != nil {
		maxkeys = int(min(list_objects_limit, *i.MaxKeys))
	} else {
		maxkeys = list_objects_limit
	}
	if i.Delimiter != nil {
		delimiter = *i.Delimiter
	}
	if i.Prefix != nil {
		prefix = *i.Prefix
	}
	if i.EncodingType == types.EncodingTypeUrl {
		urlencode = true
	}

	var entries []object_list_entry
	var nextindex int
	var nextmarker string
	var err2 error
	if !alwasy_use_flat_lister && delimiter == "/" {
		entries, nextindex, nextmarker, err2 = bbs.list_objects_delimited(
			bucket, index, marker, maxkeys, delimiter, prefix)
	} else {
		entries, nextindex, nextmarker, err2 = bbs.list_objects_flat(
			bucket, index, marker, maxkeys, delimiter, prefix)
	}
	if err2 != nil {
		return nil, err2
	}
	var _ = nextmarker

	var contents, commonprefixes, err3 = bbs.make_list_objects_entries(
		entries, bucket, delimiter, prefix, urlencode)
	var _ = err3
	var istruncated = (nextindex != 0)

	var keys = int32(len(contents) + len(commonprefixes))
	o.Contents = contents
	o.CommonPrefixes = commonprefixes
	o.KeyCount = &keys
	o.IsTruncated = &istruncated
	if nextindex != 0 {
		var scontinuation = strconv.FormatInt(int64(nextindex), 10)
		o.NextContinuationToken = &scontinuation
	}

	// o.RequestCharged types.RequestCharged

	{
		// var maxkeys int32 = 1000
		o.ContinuationToken = i.ContinuationToken
		o.Delimiter = i.Delimiter
		o.EncodingType = i.EncodingType
		o.MaxKeys = i.MaxKeys
		o.Name = &bucket
		o.Prefix = i.Prefix
		o.StartAfter = i.StartAfter
	}

	/*
		s := model.ListObjectsState{MaxKeys: 1000, V2Flg: true}
		s.Bucket = option.GetBucket()
		if !s3.FileSystem.checkBucketName(s.Bucket) {
			return nil, InvalidBucketName()
		}
		if !s3.FileSystem.isFileExists(s.Bucket) {
			return nil, NoSuchBucket()
		}
		s.BucketAPath = s3.FileSystem.getFullPath(s.Bucket)
		if v := option.GetOption("max-keys"); v != "" {
			if s.MaxKeys = utils.ToInt(v); s.MaxKeys > 1000 { // max-keysの上限は1000
				s.MaxKeys = 1000
			}
		}
		if v := option.GetOption("prefix"); v != "" {
			s.Prefix = v
		}
		if v := option.GetOption("start-after"); v != "" {
			v = strings.ReplaceAll(v, "/", "\\")
			s.StartAfter = v
		}
		if v := option.GetOption("delimiter"); v == "/" {
			s.Delimiter = filepath.FromSlash(v)
		}
		var s3err *S3Error
		if s.URLFlag, s3err = s3.checkEncodingType(option); s3err != nil {
			return nil, s3err
		}
		if v := option.GetOption("continuation-token"); v != "" {
			if s.ContinuationToken, s.Target = s3.FileSystem.decodeContinuationToken(v); s.Target == 0 {
				return nil, InternalError()
			}
		}
		res, responseRes := s3.listObjects(s)
		if res == nil {
			return nil, NotImplemented()
		}
		result := s.MakeListObjectsV2Result(*responseRes)
		result.Contents = *res
	*/

	return &o, nil
}

func (bbs *Bb_server) ListParts(ctx context.Context, params *s3.ListPartsInput, optFns ...func(*s3.Options)) (*s3.ListPartsOutput, error) {
	var o = s3.ListPartsOutput{}

	// i.Bucket *string
	// i.Key *string
	// i.UploadId *string
	// i.ExpectedBucketOwner *string
	// i.MaxParts *int32
	// i.PartNumberMarker *string
	// i.RequestPayer types.RequestPayer
	// i.SSECustomerAlgorithm *string
	// i.SSECustomerKey *string
	// i.SSECustomerKeyMD5 *string

	// o.AbortDate *time.Time
	// o.AbortRuleId *string
	// o.Bucket *string
	// o.ChecksumAlgorithm types.ChecksumAlgorithm
	// o.ChecksumType types.ChecksumType
	// o.Initiator *types.Initiator
	// o.IsTruncated *bool
	// o.Key *string
	// o.MaxParts *int32
	// o.NextPartNumberMarker *string
	// o.Owner *types.Owner
	// o.PartNumberMarker *string
	// o.Parts []types.Part
	// o.RequestCharged types.RequestCharged
	// o.StorageClass types.StorageClass
	// o.UploadId *string
	// o.ResultMetadata middleware.Metadata

	return &o, nil
}

func (bbs *Bb_server) PutObject(ctx context.Context, i *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	fmt.Printf("*PutObject*\n")
	var o = s3.PutObjectOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.Key *string
	// i.ACL types.ObjectCannedACL
	// i.Body io.Reader
	// i.BucketKeyEnabled *bool
	// i.CacheControl *string
	// i.ChecksumAlgorithm types.ChecksumAlgorithm
	// i.ChecksumCRC32 *string
	// i.ChecksumCRC32C *string
	// i.ChecksumCRC64NVME *string
	// i.ChecksumSHA1 *string
	// i.ChecksumSHA256 *string
	// i.ContentDisposition *string
	// i.ContentEncoding *string
	// i.ContentLanguage *string
	// i.ContentLength *int64
	// i.ContentMD5 *string
	// i.ContentType *string
	// i.ExpectedBucketOwner *string
	// i.Expires *time.Time
	// i.GrantFullControl *string
	// i.GrantRead *string
	// i.GrantReadACP *string
	// i.GrantWriteACP *string
	// i.IfMatch *string
	// i.IfNoneMatch *string
	// i.Metadata map[string]string
	// i.ObjectLockLegalHoldStatus types.ObjectLockLegalHoldStatus
	// i.ObjectLockMode types.ObjectLockMode
	// i.ObjectLockRetainUntilDate *time.Time
	// i.RequestPayer types.RequestPayer
	// i.SSECustomerAlgorithm *string
	// i.SSECustomerKey *string
	// i.SSECustomerKeyMD5 *string
	// i.SSEKMSEncryptionContext *string
	// i.SSEKMSKeyId *string
	// i.ServerSideEncryption types.ServerSideEncryption
	// i.StorageClass types.StorageClass
	// i.Tagging *string
	// i.WebsiteRedirectLocation *string
	// i.WriteOffsetBytes *int64

	var object, err1 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err1 != nil {
		return nil, err1
	}
	var location = "/" + object

	// AHO ?? Check "Cache-Control" which only accepts "no-cache".

	if i.CacheControl != nil {
		if !strings.EqualFold(*i.CacheControl, "no-cache") {
			var errz = &Aws_s3_error{Code: InvalidStorageClass,
				Message:  "Bad Cache-Control",
				Resource: location}
			return nil, errz
		}
	}

	var err2 = check_unsupported_options(object, i.StorageClass)
	if err2 != nil {
		return nil, err2
	}
	var info, err3 = make_meta_info(i.Metadata, i.Tagging, location)
	if err3 != nil {
		return nil, err3
	}

	var size int64
	if i.ContentLength != nil {
		size = *i.ContentLength
	} else {
		size = -1
	}
	var md5_to_check []byte
	if i.ContentMD5 != nil {
		var bs, err4 = base64.StdEncoding.DecodeString(*i.ContentMD5)
		if err4 != nil {
			md5_to_check = bs
		} else {
			md5_to_check = []byte{}
		}
	} else {
		md5_to_check = []byte{}
	}

	var rid int64 = get_request_id(ctx)
	var scratchkey = bbs.make_file_suffix(rid)
	var cleanup_needed = true
	defer func() {
		bbs.discharge_file_suffix(rid)
		if cleanup_needed {
			bbs.discharge_scratch_file(ctx, object, scratchkey)
		}
	}()

	var err6 = bbs.upload_file(ctx, object, scratchkey, size, i.Body)
	if err6 != nil {
		return nil, err6
	}
	//cleanup_needed = true

	var checksum types.ChecksumAlgorithm = i.ChecksumAlgorithm
	var md5, csum, errx = bbs.calculate_csum2(checksum, object, scratchkey)
	if errx != nil {
		return nil, errx
	}
	o.ETag = make_etag_from_md5(md5)

	if len(md5_to_check) != 0 && bytes.Compare(md5_to_check, md5) != 0 {
		bbs.Logger.Info("Digests mismatch",
			"algorithm", "MD5",
			"passed", hex.EncodeToString(md5_to_check),
			"calculated", hex.EncodeToString(md5))
		var errz = &Aws_s3_error{Code: BadDigest,
			Resource: location}
		return nil, errz
	}

	if checksum != "" {
		var csum1 *string
		switch checksum {
		case types.ChecksumAlgorithmCrc32:
			csum1 = i.ChecksumCRC32
		case types.ChecksumAlgorithmCrc32c:
			csum1 = i.ChecksumCRC32C
		case types.ChecksumAlgorithmSha1:
			csum1 = i.ChecksumSHA1
		case types.ChecksumAlgorithmSha256:
			csum1 = i.ChecksumSHA256
		case types.ChecksumAlgorithmCrc64nvme:
			csum1 = i.ChecksumCRC64NVME
		default:
			log.Fatalf("BAD-IMPL: Bad s3/types.ChecksumAlgorithm: %s",
				checksum)
		}
		if csum1 == nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message:  "Checksum value is missing.",
				Resource: location}
			return nil, errz
		}
		var csum_to_check, err9 = base64.StdEncoding.DecodeString(*csum1)
		if err9 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Resource: location,
				Message:  "Checksum value is illegal."}
			return nil, errz
		}
		if bytes.Compare(csum_to_check, csum) != 0 {
			bbs.Logger.Info("Checksums mismatch",
				"algorithm", checksum,
				"passed", hex.EncodeToString(csum_to_check),
				"calculated", hex.EncodeToString(csum))
			var errz = &Aws_s3_error{Code: BadDigest,
				Resource: location,
				Message:  "The checksum did not match what we received."}
			return nil, errz
		}
	}

	if checksum != "" {
		var csum1 = base64.StdEncoding.EncodeToString(csum)
		switch i.ChecksumAlgorithm {
		case types.ChecksumAlgorithmCrc32:
			o.ChecksumCRC32 = &csum1
		case types.ChecksumAlgorithmCrc32c:
			o.ChecksumCRC32C = &csum1
		case types.ChecksumAlgorithmSha1:
			o.ChecksumSHA1 = &csum1
		case types.ChecksumAlgorithmSha256:
			o.ChecksumSHA256 = &csum1
		case types.ChecksumAlgorithmCrc64nvme:
			o.ChecksumCRC64NVME = &csum1
		}
		o.ChecksumType = types.ChecksumTypeFullObject
	}

	// It should be atomic on placing an uploaded file and saving a
	// meta-info file.  Failing to place an uploaded file may lose
	// meta-info.

	var _ = bbs.serialize_access(ctx, object, rid)
	defer bbs.release_access(ctx, object, rid)

	{
		var err1 = bbs.store_metainfo(ctx, object, info)
		if err1 != nil {
			return nil, err1
		}
		var err2 = bbs.place_uploaded(ctx, object, scratchkey)
		if err2 != nil && info != nil {
			var _ = bbs.store_metainfo(ctx, object, nil)
		}
		if err2 != nil {
			return nil, err2
		}
	}

	cleanup_needed = false

	// o.BucketKeyEnabled *bool
	// o.Expiration *string
	// o.RequestCharged types.RequestCharged
	// o.Size *int64
	// o.SSECustomerAlgorithm *string
	// o.SSECustomerKeyMD5 *string
	// o.SSEKMSEncryptionContext *string
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption
	// o.VersionId *string

	/*
		if f := s3.FileSystem.uploadFile(option.GetBody(), option.GetPath()); !f {
			return nil, InternalError()
		}

		if err := s3.Tag.putTagging(option, option.GetPath()); err != nil {
			return nil, err
		}

		var s3err *S3Error
		if s.ETag, s3err = s3.getETag(option.GetPath()); s3err != nil {
			return nil, InternalError()
		}
		if s3err = s3.compareMd5(option, nil, s.ETag); s3err != nil {
			return nil, s3err
		}
		if s.ChecksumAlgorithm, s.ChecksumValue, s3err = s3.getChecksumMode(option, ""); s3err != nil {
			return nil, s3err
		}
		result := s.MakePutObjectResult()
	*/

	return &o, nil
}

func (bbs *Bb_server) PutObjectTagging(ctx context.Context, params *s3.PutObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.PutObjectTaggingOutput, error) {
	var o = s3.PutObjectTaggingOutput{}

	// i.Bucket *string
	// i.Key *string
	// i.Tagging *types.Tagging
	// i.ChecksumAlgorithm types.ChecksumAlgorithm
	// i.ContentMD5 *string
	// i.ExpectedBucketOwner *string
	// i.RequestPayer types.RequestPayer
	// i.VersionId *string

	// o.VersionId *string
	// o.ResultMetadata middleware.Metadata

	return &o, nil
}
func (bbs *Bb_server) UploadPart(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
	var o = s3.UploadPartOutput{}

	// i.Bucket *string
	// i.Key *string
	// i.PartNumber *int32
	// i.UploadId *string
	// i.Body io.Reader
	// i.ChecksumAlgorithm types.ChecksumAlgorithm
	// i.ChecksumCRC32 *string
	// i.ChecksumCRC32C *string
	// i.ChecksumCRC64NVME *string
	// i.ChecksumSHA1 *string
	// i.ChecksumSHA256 *string
	// i.ContentLength *int64
	// i.ContentMD5 *string
	// i.ExpectedBucketOwner *string
	// i.RequestPayer types.RequestPayer
	// i.SSECustomerAlgorithm *string
	// i.SSECustomerKey *string
	// i.SSECustomerKeyMD5 *string

	// o.BucketKeyEnabled *bool
	// o.ChecksumCRC32 *string
	// o.ChecksumCRC32C *string
	// o.ChecksumCRC64NVME *string
	// o.ChecksumSHA1 *string
	// o.ChecksumSHA256 *string
	// o.ETag *string
	// o.RequestCharged types.RequestCharged
	// o.SSECustomerAlgorithm *string
	// o.SSECustomerKeyMD5 *string
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption
	// o.ResultMetadata middleware.Metadata

	return &o, nil
}

func (bbs *Bb_server) UploadPartCopy(ctx context.Context, params *s3.UploadPartCopyInput, optFns ...func(*s3.Options)) (*s3.UploadPartCopyOutput, error) {
	var o = s3.UploadPartCopyOutput{}

	// i.Bucket *string
	// i.CopySource *string
	// i.Key *string
	// i.PartNumber *int32
	// i.UploadId *string
	// i.CopySourceIfMatch *string
	// i.CopySourceIfModifiedSince *time.Time
	// i.CopySourceIfNoneMatch *string
	// i.CopySourceIfUnmodifiedSince *time.Time
	// i.CopySourceRange *string
	// i.CopySourceSSECustomerAlgorithm *string
	// i.CopySourceSSECustomerKey *string
	// i.CopySourceSSECustomerKeyMD5 *string
	// i.ExpectedBucketOwner *string
	// i.ExpectedSourceBucketOwner *string
	// i.RequestPayer types.RequestPayer
	// i.SSECustomerAlgorithm *string
	// i.SSECustomerKey *string
	// i.SSECustomerKeyMD5 *string

	// o.BucketKeyEnabled *bool
	// o.CopyPartResult *types.CopyPartResult
	// o.CopySourceVersionId *string
	// o.RequestCharged types.RequestCharged
	// o.SSECustomerAlgorithm *string
	// o.SSECustomerKeyMD5 *string
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption
	// o.ResultMetadata middleware.Metadata

	return &o, nil
}
