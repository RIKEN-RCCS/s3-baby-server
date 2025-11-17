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
	//"s3-baby-server/internal/api"
	"github.com/riken-rccs/s3-baby-server/pkg/httpaide"
	//"s3-baby-server/pkg/httpaide"
	//"s3-baby-server/service"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"encoding/xml"
	"encoding/base64"
	"encoding/hex"
	"log"
	"log/slog"
	"net/http"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"bytes"
	//"syscall"
)

type BB_configuration struct {
	Access_logging            bool
	Anonymize_ower            bool
	Verify_fs_write           bool
	Pending_upload_expiration time.Duration
	Server_controler_path       string

	request_processing_timeout       time.Duration

	File_follow_link   bool
	File_creation_mode fs.FileMode
}

type Bb_server struct {
	//S3      *service.S3Service
	pool_path string
	Logger  *slog.Logger
	AuthKey string

	BB_config BB_configuration

	rid   int64
	suffixes map[string]suffix_record
	monitor1 *monitor
	mutex sync.Mutex

	server_quit chan struct{}

	// FileSystem is in S3.FileSystem *FileSystem

	AbortMultipartUploadHandler    http.HandlerFunc
	CompleteMultipartUploadHandler http.HandlerFunc
	CopyObjectHandler              http.HandlerFunc
	CreateBucketHandler            http.HandlerFunc
	CreateMultipartUploadHandler   http.HandlerFunc
	DeleteBucketHandler            http.HandlerFunc
	DeleteObjectHandler            http.HandlerFunc
	DeleteObjectsHandler           http.HandlerFunc
	DeleteObjectTaggingHandler     http.HandlerFunc
	GetObjectAttributesHandler     http.HandlerFunc
	GetObjectHandler               http.HandlerFunc
	GetObjectTaggingHandler        http.HandlerFunc
	HeadBucketHandler              http.HandlerFunc
	HeadObjectHandler              http.HandlerFunc
	ListBucketsHandler             http.HandlerFunc
	ListMultipartUploadsHandler    http.HandlerFunc
	ListObjectsHandler             http.HandlerFunc
	ListObjectsV2Handler           http.HandlerFunc
	ListPartsHandler               http.HandlerFunc
	PutObjectHandler               http.HandlerFunc
	PutObjectTaggingHandler        http.HandlerFunc
	UploadPartCopyHandler          http.HandlerFunc
	UploadPartHandler              http.HandlerFunc
}

type suffix_record struct {
	rid int64
	timestamp time.Time
}

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

// MAKE_FILE_SUFFIX makes a short key string to make a (temporary)
// scratch file unique.  It takes request-id.  It returns a string of
// 6 hex-digits.
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

// DISCHARGE_FILE_SUFFIXES removes recorded file suffixes for
// temporary files associated to a request-id.
func (bbs *Bb_server) discharge_file_suffixes(rid int64) {
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

func scan_range(rangestring *string, size int64, location string) (*[2]int64, error) {
	var extent *[2]int64
	if rangestring != nil {
		var r, err3 = httpaide.Scan_rfc9110_range(*rangestring)
		if err3 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Resource: location,
				Message: "Range format is illegal."}
			return nil, errz
		}
		if len(r) != 1 {
			var errz = &Aws_s3_error{Code: InvalidRange,
				Resource: location,
				Message: "Range is not more than one."}
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

func (bbs *Bb_server) AbortMultipartUpload(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
	var o = s3.AbortMultipartUploadOutput{}
	return &o, nil
}
func (bbs *Bb_server) CompleteMultipartUpload(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	var o = s3.CompleteMultipartUploadOutput{}
	return &o, nil
}
func (bbs *Bb_server) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	var o = s3.CopyObjectOutput{}
	return &o, nil
}

func (bbs *Bb_server) CreateBucket(ctx context.Context, i *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	fmt.Printf("*CreateBucket*\n")
	var o = s3.CreateBucketOutput{}

	// List of parameters.
	// + Bucket *string
	// - ACL types.BucketCannedACL
	// - CreateBucketConfiguration *types.CreateBucketConfiguration
	// - GrantFullControl *string
	// - GrantRead *string
	// - GrantReadACP *string
	// - GrantWrite *string
	// - GrantWriteACP *string
	// - ObjectLockEnabledForBucket *bool
	// - ObjectOwnership types.ObjectOwnership

	if i.Bucket == nil {
		log.Fatalf("BAD-IMPL: Bucket parameter missing")
	}
	var bucket = *i.Bucket
	if !check_bucket_naming(bucket) {
		var err5 = &Aws_s3_error{Code: InvalidBucketName}
		return nil, err5
	}

	var location = "/" + bucket

	var path = bbs.make_path(bucket)
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

func (bbs *Bb_server) CreateMultipartUpload(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	var o = s3.CreateMultipartUploadOutput{}
	return &o, nil
}
func (bbs *Bb_server) DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
	var o = s3.DeleteBucketOutput{}
	return &o, nil
}
func (bbs *Bb_server) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	var o = s3.DeleteObjectOutput{}
	return &o, nil
}
func (bbs *Bb_server) DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	var o = s3.DeleteObjectsOutput{}
	return &o, nil
}
func (bbs *Bb_server) DeleteObjectTagging(ctx context.Context, params *s3.DeleteObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectTaggingOutput, error) {
	var o = s3.DeleteObjectTaggingOutput{}
	return &o, nil
}

func (bbs *Bb_server) GetObject(ctx context.Context, i *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	fmt.Printf("*GetObject*\n")
	var o = s3.GetObjectOutput{}

	// List of parameters.
	// + Bucket *string
	// + Key *string
	// - ChecksumMode types.ChecksumMode
	// - ExpectedBucketOwner *string
	// - IfMatch *string
	// - IfModifiedSince *time.Time
	// - IfNoneMatch *string
	// - IfUnmodifiedSince *time.Time
	// - PartNumber *int32
	// - Range *string
	// - RequestPayer types.RequestPayer
	// - ResponseCacheControl *string
	// - ResponseContentDisposition *string
	// - ResponseContentEncoding *string
	// - ResponseContentLanguage *string
	// - ResponseContentType *string
	// - ResponseExpires *time.Time
	// - SSECustomerAlgorithm *string
	// - SSECustomerKey *string
	// - SSECustomerKeyMD5 *string
	// - VersionId *string

	var object, err1 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err1 != nil {
		return nil, err1
	}
	var location = "/" + object
	var _ = location

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
	if err3 != nil {
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
		o.Metadata = info.Headers
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

	var f1, err6 = bbs.make_file_stream(ctx, object, nil)
	if err6 != nil {
		return nil, err6
	}
	o.Body = f1

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
	return &o, nil
}

func (bbs *Bb_server) HeadBucket(ctx context.Context, i *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	fmt.Printf("*HeadBucket*\n")
	var o = s3.HeadBucketOutput{}

	// List of parameters.
	// - Bucket *string
	// - ExpectedBucketOwner *string

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
	// - Bucket *string
	// - Key *string
	// - ChecksumMode types.ChecksumMode
	// - ExpectedBucketOwner *string
	// - IfMatch *string
	// - IfModifiedSince *time.Time
	// - IfNoneMatch *string
	// - IfUnmodifiedSince *time.Time
	// - PartNumber *int32
	// - Range *string
	// - RequestPayer types.RequestPayer
	// - ResponseCacheControl *string
	// - ResponseContentDisposition *string
	// - ResponseContentEncoding *string
	// - ResponseContentLanguage *string
	// - ResponseContentType *string
	// - ResponseExpires *time.Time
	// - SSECustomerAlgorithm *string
	// - SSECustomerKey *string
	// - SSECustomerKeyMD5 *string
	// - VersionId *string

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
		o.Metadata = info.Headers
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
	// - BucketRegion *string
	// + ContinuationToken *string
	// + MaxBuckets *int32
	// + Prefix *string

	var start int
	if i.ContinuationToken != nil {
		var n, err1 = strconv.ParseInt(*i.ContinuationToken, 10, 32)
		if err1 != nil {
			/*
			var m = map[string]error{"continuation-token":
				&Bb_input_error{"continuation-token", err1}}
			bbs.respond_on_input_error(m)
			*/
			var err2 = Bb_input_error{"continuation-token", err1}
			var err3 = &Aws_s3_error{Code: InvalidArgument, Message: err2.Error()}
			return nil, err3
		}
		start = int(n)
	} else {
		start = 0
	}

	var max_buckets int
	if i.MaxBuckets != nil {
		max_buckets = int(*i.MaxBuckets)
		if max_buckets > 10000 {
			var err2 = fmt.Errorf("Value too large: %d", max_buckets)
			/*
			var m = map[string]error{"max-buckets":
				&Bb_input_error{"max-buckets", err2}}
			bbs.respond_on_input_error(m)
			*/
			var err3 = &Aws_s3_error{Code: InvalidArgument, Message: err2.Error()}
			return nil, err3
		}
	} else {
		max_buckets = 10000
	}

	//var pool_path = bbs.S3.FileSystem.RootPath
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
		var end = min(start + max_buckets, len(dirs2))
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
		var info, err4 = e.Info()
		if err4 != nil {
			// Skip the entry because it may be removed after scanning
			// directory.  SHOULD CHECK errors.Is(err, ErrNotExist).
			continue
		}
		var times, ok = file_time(info)
		if !ok {
			var t0 = info.ModTime()
			times = [3]time.Time{t0, t0, t0}
		}
		var name = e.Name()
		var b = types.Bucket{
			// BucketArn:,
			// BucketRegion:,
			CreationDate: &times[1],
			Name: &name,
		}
		buckets = append(buckets, b)
	}

	o.Buckets = buckets
	if continuation != 0 {
		var scontinuation = strconv.FormatInt(int64(continuation), 10)
		o.ContinuationToken = &scontinuation
	}
	// o.Owner = &s3.Owner{DisplayName: , ID:}
	o.Prefix = i.Prefix

	return &o, nil
}

func (bbs *Bb_server) ListMultipartUploads(ctx context.Context, params *s3.ListMultipartUploadsInput, optFns ...func(*s3.Options)) (*s3.ListMultipartUploadsOutput, error) {
	var o = s3.ListMultipartUploadsOutput{}
	return &o, nil
}
func (bbs *Bb_server) ListObjects(ctx context.Context, params *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, error) {
	var o = s3.ListObjectsOutput{}
	return &o, nil
}
func (bbs *Bb_server) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	var o = s3.ListObjectsV2Output{}
	return &o, nil
}
func (bbs *Bb_server) ListParts(ctx context.Context, params *s3.ListPartsInput, optFns ...func(*s3.Options)) (*s3.ListPartsOutput, error) {
	var o = s3.ListPartsOutput{}
	return &o, nil
}

func (bbs *Bb_server) PutObject(ctx context.Context, i *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	fmt.Printf("*PutObject*\n")
	var o = s3.PutObjectOutput{}

	// List of parameters.
	// + Bucket *string
	// + Key *string
	// - ACL types.ObjectCannedACL
	// + Body io.Reader
	// - BucketKeyEnabled *bool
	// + CacheControl *string
	// + ChecksumAlgorithm types.ChecksumAlgorithm
	// - ChecksumCRC32 *string
	// - ChecksumCRC32C *string
	// - ChecksumCRC64NVME *string
	// - ChecksumSHA1 *string
	// - ChecksumSHA256 *string
	// - ContentDisposition *string
	// - ContentEncoding *string
	// - ContentLanguage *string
	// - ContentLength *int64
	// - ContentMD5 *string
	// - ContentType *string
	// - ExpectedBucketOwner *string
	// - Expires *time.Time
	// - GrantFullControl *string
	// - GrantRead *string
	// - GrantReadACP *string
	// - GrantWriteACP *string
	// - IfMatch *string
	// - IfNoneMatch *string
	// - Metadata map[string]string
	// - ObjectLockLegalHoldStatus types.ObjectLockLegalHoldStatus
	// - ObjectLockMode types.ObjectLockMode
	// - ObjectLockRetainUntilDate *time.Time
	// - RequestPayer types.RequestPayer
	// - SSECustomerAlgorithm *string
	// - SSECustomerKey *string
	// - SSECustomerKeyMD5 *string
	// - SSEKMSEncryptionContext *string
	// - SSEKMSKeyId *string
	// - ServerSideEncryption types.ServerSideEncryption
	// + StorageClass types.StorageClass
	// + Tagging *string
	// - WebsiteRedirectLocation *string
	// - WriteOffsetBytes *int64

	var object, err1 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err1 != nil {
		return &o, err1
	}
	var location = "/" + object

	// AHO ?? Check "Cache-Control" which only accepts "no-cache".

	if i.CacheControl != nil {
		if !strings.EqualFold(*i.CacheControl, "no-cache") {
			var errz = &Aws_s3_error{Code: InvalidStorageClass,
				Message: "Bad Cache-Control",
				Resource: location}
			return &o, errz
		}
	}
	if i.StorageClass != "" {
		if i.StorageClass != types.StorageClassStandard {
			var errz = &Aws_s3_error{Code: InvalidStorageClass,
				Message: "Bad x-amz-storage-class",
				Resource: location}
			return &o, errz
		}
	}

	var info *Meta_info
	if i.Tagging != nil {
		var tags1, err3 = parse_tags(*i.Tagging)
		if err3 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message: "Tag format error.",
				Resource: location}
			return &o, errz
		}
		if tags1 != nil {
			info = &Meta_info{Tags: tags1}
		} else {
			info = nil
		}
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
	var scratch = bbs.make_file_suffix(rid)
	defer bbs.discharge_file_suffixes(rid)

	var err6 = bbs.upload_file(ctx, object, scratch, size, i.Body)
	if err6 != nil {
		return nil, err6
	}

	var checksum types.ChecksumAlgorithm = i.ChecksumAlgorithm
	var md5, csum, errx = bbs.calculate_csum2(checksum, object, scratch)
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
				Message: "Checksum value is missing.",
				Resource: location}
			return nil, errz
		}
		var csum_to_check, err9 = base64.StdEncoding.DecodeString(*csum1)
		if err9 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Resource: location,
				Message: "Checksum value is illegal."}
			return nil, errz
		}
		if bytes.Compare(csum_to_check, csum) != 0 {
			bbs.Logger.Info("Checksums mismatch",
				"algorithm", checksum,
				"passed", hex.EncodeToString(csum_to_check),
				"calculated", hex.EncodeToString(csum))
			var errz = &Aws_s3_error{Code: BadDigest,
				Resource: location,
				Message: "The checksum did not match what we received."}
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
			return &o, err1
		}
		var err2 = bbs.place_uploaded(ctx, object, scratch)
		if err2 != nil && info != nil {
			var _ = bbs.store_metainfo(ctx, object, nil)
		}
		if err2 != nil {
			return &o, err2
		}
	}

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
	return &o, nil
}
func (bbs *Bb_server) UploadPart(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
	var o = s3.UploadPartOutput{}
	return &o, nil
}
func (bbs *Bb_server) UploadPartCopy(ctx context.Context, params *s3.UploadPartCopyInput, optFns ...func(*s3.Options)) (*s3.UploadPartCopyOutput, error) {
	var o = s3.UploadPartCopyOutput{}
	return &o, nil
}
