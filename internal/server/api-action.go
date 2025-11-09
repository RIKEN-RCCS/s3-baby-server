// api-action.go (2025-10-01)

// Copyright 2025-2025 RIKEN R-CCS.
// SPDX-License-Identifier: BSD-2-Clause

// API-STUB.  Handler templates. They should be replaced by
// actual implementations.

package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
	//"s3-baby-server/internal/api"
	"s3-baby-server/internal/service"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"encoding/xml"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	//"syscall"
)

type BB_configuration struct {
	Access_logging            bool
	Anonymize_ower            bool
	Verify_fs_write           bool
	Pending_upload_expiration time.Duration
	Server_control_path       string

	File_follow_link   bool
	File_creation_mode fs.FileMode
}

type Bb_server struct {
	S3      *service.S3Service
	Logger  *slog.Logger
	AuthKey string

	BB_config BB_configuration

	rid   int64
	mutex sync.Mutex

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

// MAKE_REQUEST_ID makes a new request-id.  It uses time, or when time
// does not advance, uses the last value plus one.
func (bbs *Bb_server) make_request_id() string {
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
	return fmt.Sprintf("%016x", t)
}

// RESPOND_ON_ACTION_ERROR is an action error and makes a
// response for it.
func (bbs *Bb_server) respond_on_action_error(ctx context.Context, w http.ResponseWriter, r *http.Request, e error) {
	var e1, ok = e.(*Aws_s3_Error)
	if !ok {
		log.Fatalf("Bad error from action: %#v", e)
	}
	bbs.Logger.Info(string(e1.Code), "error", e1)

	e1.RequestId = ctx.Value("request-id").(string)
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
	var err1 = Aws_s3_Error{Code: InvalidArgument, Message: e.Error()}
	bbs.respond_on_action_error(ctx, w, r, err1)
}

func fs_error_name(err error) string {
	if errors.Is(err, fs.ErrInvalid) {
		return "ErrInvalid"
	} else if errors.Is(err, fs.ErrPermission) {
		return "ErrPermission"
	} else if errors.Is(err, fs.ErrExist) {
		return "ErrExist"
	} else if errors.Is(err, fs.ErrNotExist) {
		return "ErrNotExist"
	} else if errors.Is(err, fs.ErrClosed) {
		return "ErrClosed"
	} else {
		// os.ErrNoDeadline
		// os.ErrDeadlineExceeded
		return "ErrUnknown"
	}
}

// Makes an AWS-S3 error from a given OS error.  Error codes can be
// mapped to something like, fs.ErrExist to "BucketAlreadyOwnedByYou".
func map_os_error(ctx context.Context, location string, err1 error, m map[error]Aws_s3_error_code) error {
	var kind error
	var code1 Aws_s3_error_code
	if errors.Is(err1, fs.ErrInvalid) {
		kind = fs.ErrInvalid
		code1 = InvalidArgument
	} else if errors.Is(err1, fs.ErrPermission) {
		kind = fs.ErrPermission
		code1 = AccessDenied
	} else if errors.Is(err1, fs.ErrExist) {
		kind = fs.ErrExist
		code1 = InternalError
	} else if errors.Is(err1, fs.ErrNotExist) {
		kind = fs.ErrNotExist
		code1 = InternalError
	} else if errors.Is(err1, fs.ErrClosed) {
		kind = fs.ErrClosed
		code1 = InternalError
	} else {
		kind = nil
		code1 = InternalError
	}

	var code2, ok1 = m[kind]
	if ok1 {
		var err5 = Aws_s3_Error{Code: code2, Resource: location}
		return &err5
	} else {
		var err5 = Aws_s3_Error{Code: code1, Resource: location,
			Message: fs_error_name(kind)}
		return &err5
	}
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
	fmt.Printf("bbs.CreateBucket\n")
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

	var bucket = i.Bucket
	if bucket == nil {
		log.Fatalf("BAD-IMPL: Bucket parameter missing")
	}
	if !check_bucket_naming(*bucket) {
		var err5 = Aws_s3_Error{Code: InvalidBucketName}
		return &o, &err5
	}

	var location = "/" + *bucket

	var path = bbs.make_path(*bucket)
	var err2 = os.Mkdir(path, 0755)
	if err2 != nil {
		// Note the error on existing path is fs.PathError and not
		// fs.ErrExist.

		/*if errors.As(err2, &err3) {*/
		/*if !errors.Is(err2, fs.ErrExist) {*/
		/*var err4, ok = err3.Err.(syscall.Errno)*/

		bbs.Logger.Info("os.Mkdir() failed", "error", err2)
		var m = map[error]Aws_s3_error_code{fs.ErrExist: BucketAlreadyOwnedByYou}
		var err5 = map_os_error(ctx, location, err2, m)
		return &o, err5
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
func (bbs *Bb_server) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	var o = s3.GetObjectOutput{}
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
func (bbs *Bb_server) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	var o = s3.HeadBucketOutput{}
	return &o, nil
}
func (bbs *Bb_server) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	var o = s3.HeadObjectOutput{}
	return &o, nil
}

func (bbs *Bb_server) ListBuckets(ctx context.Context, i *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	fmt.Printf("bbs.ListBuckets\n")
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
			var err3 = Aws_s3_Error{Code: InvalidArgument, Message: err2.Error()}
			return &o, err3
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
			var err3 = Aws_s3_Error{Code: InvalidArgument, Message: err2.Error()}
			return &o, err3
		}
	} else {
		max_buckets = 10000
	}

	var pool_path = bbs.S3.FileSystem.RootPath
	var entries1, err3 = os.ReadDir(pool_path)
	if err3 != nil {
		bbs.Logger.Info("os.ReadDir() failed in ListBuckets", "error", err3)
		var m = map[error]Aws_s3_error_code{}
		var err5 = map_os_error(ctx, "/", err3, m)
		return &o, err5
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
	var o = s3.PutObjectOutput{}

	// List of parameters.
	// - ACL types.ObjectCannedACL
	// x Body io.Reader
	// - BucketKeyEnabled *bool
	// x CacheControl *string
	// - ChecksumAlgorithm types.ChecksumAlgorithm
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
	// x StorageClass types.StorageClass
	// x Tagging *string
	// - WebsiteRedirectLocation *string
	// - WriteOffsetBytes *int64

	var bucket = i.Bucket
	if bucket == nil {
		log.Fatalf("BAD-IMPL: Bucket parameter missing")
	}
	if !check_bucket_naming(*bucket) {
		var err5 = Aws_s3_Error{Code: InvalidBucketName}
		return &o, &err5
	}

	var location = "/" + *bucket
	var path = bbs.make_path(*bucket)
	var info, err2 = os.Lstat(path)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			var err5 = Aws_s3_Error{Code: NoSuchBucket,
				Resource: location}
			return &o, err5
		} else {
			var m = map[error]Aws_s3_error_code{}
			var err5 = map_os_error(ctx, location, err2, m)
			return &o, err5
		}
	}
	if !info.IsDir() {
		var err5 = Aws_s3_Error{Code: NoSuchBucket,
			Resource: location}
		return &o, err5
	}

	// ?? CHECK "Cache-Control": "no-cache".

	if i.CacheControl != nil {
		if !strings.EqualFold(*i.CacheControl, "no-cache") {
			var err5 = Aws_s3_Error{Code: InvalidStorageClass,
				Resource: location}
			return &o, err5
		}
	}
	if i.StorageClass != types.StorageClassStandard {
		var err5 = Aws_s3_Error{Code: InvalidStorageClass,
			Resource: location}
		return &o, err5
	}

	var tags types.Tagging
	if i.Tagging != nil {
		var tags1, err3 = parse_tags(*i.Tagging)
		if err3 != nil {
			var err5 = Aws_s3_Error{Code: InvalidArgument,
				Message: "Tag format error.",
				Resource: location}
			return &o, err5
		}
		tags = tags1
	}
	var _ = tags

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
