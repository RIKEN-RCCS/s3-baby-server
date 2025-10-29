// api-template.go (2025-10-01)
// API-STUB.  Handler templates. They should be replaced by
// actual implementations.
package server
import (
"context"
"net/http"
"s3-baby-server/internal/service"
"github.com/aws/aws-sdk-go-v2/service/s3"
)

type BB_server struct {
	S3 *service.S3Service
	AbortMultipartUploadHandler http.HandlerFunc
	CompleteMultipartUploadHandler http.HandlerFunc
	CopyObjectHandler http.HandlerFunc
	CreateBucketHandler http.HandlerFunc
	CreateMultipartUploadHandler http.HandlerFunc
	DeleteBucketHandler http.HandlerFunc
	DeleteObjectHandler http.HandlerFunc
	DeleteObjectsHandler http.HandlerFunc
	DeleteObjectTaggingHandler http.HandlerFunc
	GetObjectAttributesHandler http.HandlerFunc
	GetObjectHandler http.HandlerFunc
	GetObjectTaggingHandler http.HandlerFunc
	HeadBucketHandler http.HandlerFunc
	HeadObjectHandler http.HandlerFunc
	ListBucketsHandler http.HandlerFunc
	ListMultipartUploadsHandler http.HandlerFunc
	ListObjectsHandler http.HandlerFunc
	ListObjectsV2Handler http.HandlerFunc
	ListPartsHandler http.HandlerFunc
	PutObjectHandler http.HandlerFunc
	PutObjectTaggingHandler http.HandlerFunc
	UploadPartCopyHandler http.HandlerFunc
	UploadPartHandler http.HandlerFunc
}

func (bbs *BB_server) AbortMultipartUpload(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
var o = s3.AbortMultipartUploadOutput{}
return &o, nil}
func (bbs *BB_server) CompleteMultipartUpload(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
var o = s3.CompleteMultipartUploadOutput{}
return &o, nil}
func (bbs *BB_server) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
var o = s3.CopyObjectOutput{}
return &o, nil}
func (bbs *BB_server) CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
var o = s3.CreateBucketOutput{}
return &o, nil}
func (bbs *BB_server) CreateMultipartUpload(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
var o = s3.CreateMultipartUploadOutput{}
return &o, nil}
func (bbs *BB_server) DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
var o = s3.DeleteBucketOutput{}
return &o, nil}
func (bbs *BB_server) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
var o = s3.DeleteObjectOutput{}
return &o, nil}
func (bbs *BB_server) DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
var o = s3.DeleteObjectsOutput{}
return &o, nil}
func (bbs *BB_server) DeleteObjectTagging(ctx context.Context, params *s3.DeleteObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectTaggingOutput, error) {
var o = s3.DeleteObjectTaggingOutput{}
return &o, nil}
func (bbs *BB_server) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
var o = s3.GetObjectOutput{}
return &o, nil}
func (bbs *BB_server) GetObjectAttributes(ctx context.Context, params *s3.GetObjectAttributesInput, optFns ...func(*s3.Options)) (*s3.GetObjectAttributesOutput, error) {
var o = s3.GetObjectAttributesOutput{}
return &o, nil}
func (bbs *BB_server) GetObjectTagging(ctx context.Context, params *s3.GetObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.GetObjectTaggingOutput, error) {
var o = s3.GetObjectTaggingOutput{}
return &o, nil}
func (bbs *BB_server) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
var o = s3.HeadBucketOutput{}
return &o, nil}
func (bbs *BB_server) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
var o = s3.HeadObjectOutput{}
return &o, nil}
func (bbs *BB_server) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
var o = s3.ListBucketsOutput{}
return &o, nil}
func (bbs *BB_server) ListMultipartUploads(ctx context.Context, params *s3.ListMultipartUploadsInput, optFns ...func(*s3.Options)) (*s3.ListMultipartUploadsOutput, error) {
var o = s3.ListMultipartUploadsOutput{}
return &o, nil}
func (bbs *BB_server) ListObjects(ctx context.Context, params *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, error) {
var o = s3.ListObjectsOutput{}
return &o, nil}
func (bbs *BB_server) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
var o = s3.ListObjectsV2Output{}
return &o, nil}
func (bbs *BB_server) ListParts(ctx context.Context, params *s3.ListPartsInput, optFns ...func(*s3.Options)) (*s3.ListPartsOutput, error) {
var o = s3.ListPartsOutput{}
return &o, nil}
func (bbs *BB_server) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
var o = s3.PutObjectOutput{}
return &o, nil}
func (bbs *BB_server) PutObjectTagging(ctx context.Context, params *s3.PutObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.PutObjectTaggingOutput, error) {
var o = s3.PutObjectTaggingOutput{}
return &o, nil}
func (bbs *BB_server) UploadPart(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
var o = s3.UploadPartOutput{}
return &o, nil}
func (bbs *BB_server) UploadPartCopy(ctx context.Context, params *s3.UploadPartCopyInput, optFns ...func(*s3.Options)) (*s3.UploadPartCopyOutput, error) {
var o = s3.UploadPartCopyOutput{}
return &o, nil}
