// api-action.go (2025-10-01)

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
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"log/slog"
	"net/http"
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

type BB_server struct {
	S3     *service.S3Service
	Logger *slog.Logger
	AuthKey string

	BB_config BB_configuration

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

// RESPOND_ON_ACTION_ERROR is an action error and makes a
// response for it.
func (bbs *BB_server) respond_on_action_error(w http.ResponseWriter, r *http.Request, e error) {panic(e)}

// RESPOND_ON_INPUT_ERROR is an error on interning
// enumerations and makes a response for it.
func (bbs *BB_server) respond_on_input_error(w http.ResponseWriter, r *http.Request, name string) {panic(fmt.Errorf("Bad parameter %s", name))}

// RESPOND_ON_MISSING_INPUT is an internal error and makes a
// response for it.
func (bbs *BB_server) respond_on_missing_input(w http.ResponseWriter, r *http.Request, name string) {panic(fmt.Errorf("Missing path: %s", name))}


//func (bbs *BB_server) handle_input_error(w http.ResponseWriter, r *http.Request, e error) {panic(e)}

func (bbs *BB_server) AbortMultipartUpload(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
	var o = s3.AbortMultipartUploadOutput{}
	return &o, nil
}
func (bbs *BB_server) CompleteMultipartUpload(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	var o = s3.CompleteMultipartUploadOutput{}
	return &o, nil
}
func (bbs *BB_server) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	var o = s3.CopyObjectOutput{}
	return &o, nil
}

func (bbs *BB_server) CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	fmt.Printf("bbs.CreateBucket\n")
	var o = s3.CreateBucketOutput{}

	var bucket = params.Bucket
	if bucket == nil {
		panic(fmt.Errorf("Bad-impl: Bucket parameter missing"))
	}
	if !check_bucket_naming(*bucket) {
		return &o, &types.InvalidRequest{}
	}

	// (Many parameters are ignored).

	var path = bbs.make_path(*bucket)
	//var _, err1 = os.Lstat(path)
	//var _, err1 = os.Stat(path)
	var err2 = os.Mkdir(path, 0755)
	if err2 != nil {
		/*if !errors.Is(err2, fs.ErrExist) {*/
		var err3 *fs.PathError
		if errors.As(err2, &err3) {
			// Path entry exists.  The error is not "fs.ErrExist".
			// "BucketAlreadyExists"
			fmt.Printf("os.Mkdir() failed: %#v\n", err3)
			return &o, &types.BucketAlreadyOwnedByYou{}
		} else {
			fmt.Printf("os.Mkdir() failed: %#v\n", err2)
			return &o, fmt.Errorf("os.Mkdir(%v) failed: %w", path, err2)
		}
	}

	var location = ("/" + *bucket)
	o.Location = &location
	return &o, nil
}

func (bbs *BB_server) CreateMultipartUpload(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	var o = s3.CreateMultipartUploadOutput{}
	return &o, nil
}
func (bbs *BB_server) DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
	var o = s3.DeleteBucketOutput{}
	return &o, nil
}
func (bbs *BB_server) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	var o = s3.DeleteObjectOutput{}
	return &o, nil
}
func (bbs *BB_server) DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	var o = s3.DeleteObjectsOutput{}
	return &o, nil
}
func (bbs *BB_server) DeleteObjectTagging(ctx context.Context, params *s3.DeleteObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectTaggingOutput, error) {
	var o = s3.DeleteObjectTaggingOutput{}
	return &o, nil
}
func (bbs *BB_server) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	var o = s3.GetObjectOutput{}
	return &o, nil
}
func (bbs *BB_server) GetObjectAttributes(ctx context.Context, params *s3.GetObjectAttributesInput, optFns ...func(*s3.Options)) (*s3.GetObjectAttributesOutput, error) {
	var o = s3.GetObjectAttributesOutput{}
	return &o, nil
}
func (bbs *BB_server) GetObjectTagging(ctx context.Context, params *s3.GetObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.GetObjectTaggingOutput, error) {
	var o = s3.GetObjectTaggingOutput{}
	return &o, nil
}
func (bbs *BB_server) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	var o = s3.HeadBucketOutput{}
	return &o, nil
}
func (bbs *BB_server) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	var o = s3.HeadObjectOutput{}
	return &o, nil
}
func (bbs *BB_server) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	var o = s3.ListBucketsOutput{}
	return &o, nil
}
func (bbs *BB_server) ListMultipartUploads(ctx context.Context, params *s3.ListMultipartUploadsInput, optFns ...func(*s3.Options)) (*s3.ListMultipartUploadsOutput, error) {
	var o = s3.ListMultipartUploadsOutput{}
	return &o, nil
}
func (bbs *BB_server) ListObjects(ctx context.Context, params *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, error) {
	var o = s3.ListObjectsOutput{}
	return &o, nil
}
func (bbs *BB_server) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	var o = s3.ListObjectsV2Output{}
	return &o, nil
}
func (bbs *BB_server) ListParts(ctx context.Context, params *s3.ListPartsInput, optFns ...func(*s3.Options)) (*s3.ListPartsOutput, error) {
	var o = s3.ListPartsOutput{}
	return &o, nil
}
func (bbs *BB_server) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	var o = s3.PutObjectOutput{}
	return &o, nil
}
func (bbs *BB_server) PutObjectTagging(ctx context.Context, params *s3.PutObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.PutObjectTaggingOutput, error) {
	var o = s3.PutObjectTaggingOutput{}
	return &o, nil
}
func (bbs *BB_server) UploadPart(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
	var o = s3.UploadPartOutput{}
	return &o, nil
}
func (bbs *BB_server) UploadPartCopy(ctx context.Context, params *s3.UploadPartCopyInput, optFns ...func(*s3.Options)) (*s3.UploadPartCopyOutput, error) {
	var o = s3.UploadPartCopyOutput{}
	return &o, nil
}
