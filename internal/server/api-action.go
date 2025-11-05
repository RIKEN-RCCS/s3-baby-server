// api-action.go (2025-10-01)

// Copyright 2025-2025 RIKEN R-CCS.
// SPDX-License-Identifier: BSD-2-Clause

// API-STUB.  Handler templates. They should be replaced by
// actual implementations.

package server

import (
	"context"
	"fmt"
	"os"
	"io/fs"
	"time"
	"errors"
	//"s3-baby-server/internal/api"
	"s3-baby-server/internal/service"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	//"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"encoding/xml"
	"log"
	"log/slog"
	"net/http"
	//"strconv"
	"sync"
	//"syscall"
)

type BB_configuration struct {
	Access_logging            bool
	Anonymize_ower            bool
	Verify_fs_write           bool
	Pending_upload_expiration time.Duration
	Server_control_path       string

	File_follow_link       bool
	File_creation_mode       fs.FileMode
}

type Bb_server struct {
	S3     *service.S3Service
	Logger *slog.Logger
	AuthKey string

	BB_config BB_configuration

	rid int64
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

// MAKE_REQUEST_ID makes a new request-id.  It uses timer, or when
// timer does not advance, uses the last value plus one.
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
	var e1, ok = e.(*Error)
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

// RESPOND_ON_INPUT_ERROR is an error on interning
// enumerations and makes a response for it.
func (bbs *Bb_server) respond_on_input_error(ctx context.Context, w http.ResponseWriter, r *http.Request, name string) {panic(fmt.Errorf("Bad parameter %s", name))}

// RESPOND_ON_MISSING_INPUT is an internal error and makes a
// response for it.
func (bbs *Bb_server) respond_on_missing_input(ctx context.Context, w http.ResponseWriter, r *http.Request, name string) {panic(fmt.Errorf("Missing path: %s", name))}

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
		return "ErrUnknown"
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

func (bbs *Bb_server) CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	fmt.Printf("bbs.CreateBucket\n")
	var o = s3.CreateBucketOutput{}

	var bucket = params.Bucket
	if bucket == nil {
		log.Fatalf("Bad-impl: Bucket parameter missing")
	}
	if !check_bucket_naming(*bucket) {
		var err5 = Error{Code: InvalidBucketName}
		return &o, &err5
	}

	// (Many parameters are ignored).

	var path = bbs.make_path(*bucket)
	//var _, err1 = os.Lstat(path)
	//var _, err1 = os.Stat(path)
	var err2 = os.Mkdir(path, 0755)
	if err2 != nil {
		// Note the error on existing path is fs.PathError and not
		// fs.ErrExist.
		var name = "/" + *bucket

		if errors.Is(err2, fs.ErrInvalid) {
			var err5 = Error{Code: InvalidArgument, Resource: name}
			return &o, &err5
		} else if errors.Is(err2, fs.ErrPermission) {
			var err5 = Error{Code: AccessDenied, Resource: name}
			return &o, &err5
		} else if errors.Is(err2, fs.ErrExist) {
			var err5 = Error{Code: BucketAlreadyOwnedByYou, Resource: name}
			return &o, &err5
		} else {
			/*var err3 *fs.PathError*/
			/*if errors.As(err2, &err3) {*/
			/*if !errors.Is(err2, fs.ErrExist) {*/
			/*var err4, ok = err3.Err.(syscall.Errno)*/
			bbs.Logger.Info("os.Mkdir() failed", "error", err2,
				"fs-error", fs_error_name(err2))
			var err5 = Error{Code: InternalError, Resource: name}
			return &o, &err5
		}
	}

	var location = ("/" + *bucket)
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
func (bbs *Bb_server) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	var o = s3.ListBucketsOutput{}
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
func (bbs *Bb_server) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	var o = s3.PutObjectOutput{}
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
