// api-action.go (2025-10-01)
// Copyright 2025-2025 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// API-STUB.  Handler templates. They should be replaced by
// actual implementations.

package server

import (
	"context"
	//"errors"
	"fmt"
	"io/fs"
	"os"
	//"path"
	"time"
	//"github.com/riken-rccs/s3-baby-server/pkg/httpaide"
	//"bytes"
	"encoding/base64"
	//"encoding/binary"
	//"encoding/hex"
	//"encoding/xml"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"log"
	//"log/slog"
	//"math/rand"
	//"net/http"
	//"net/url"
	"strconv"
	"strings"
	//"sync"
)

func (bbs *Bb_server) AbortMultipartUpload(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
	fmt.Printf("*AbortMultipartUpload*\n")
	var o = s3.AbortMultipartUploadOutput{}

	// List of parameters.
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

func (bbs *Bb_server) CompleteMultipartUpload(ctx context.Context, i *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	fmt.Printf("*CompleteMultipartUpload*\n")
	var o = s3.CompleteMultipartUploadOutput{}

	// List of parameters.
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

	// Error Code: EntityTooSmall
	// Error Code: InvalidPart
	// Error Code: InvalidPartOrder
	// Error Code: NoSuchUpload

	var object, err1 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err1 != nil {
		return nil, err1
	}
	var location = "/" + object

	var uploadid, err2 = bbs.check_upload_id(object, i.UploadId)
	if err2 != nil {
		return nil, err2
	}

	var _, err5 = bbs.check_conditions(ctx, i.IfMatch, i.IfNoneMatch,
		nil, nil)
	if err5 != nil {
		return nil, err5
	}

	if i.MpuObjectSize == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "x-amz-mp-object-size missing.",
			Resource: location}
		return nil, errz
	}
	var size = *i.MpuObjectSize
	var _ = size

	if i.MultipartUpload == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "Request body missing.",
			Resource: location}
		return nil, errz
	}

	var errx = bbs.concat_parts(ctx, object, uploadid, i.MultipartUpload)
	if errx != nil {
		return nil, errx
	}

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

	/*
	s := model.CompleteMultipartUploadState{}
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
	id := option.GetOption("uploadId")
	if !s3.MultiPart.uploadIDExists(id) {
		return nil, NoSuchUpload()
	}
	var reqBody model.CompleteMultipartUploadRequest
	if err := xml.Unmarshal(option.GetBody(), &reqBody); err != nil {
		return nil, InvalidRequest()
	}
	if !s3.checkPartSize(id, reqBody) {
		return nil, EntityTooSmallError()
	}
	s.DstPath = option.GetPath()
	if err := s3.MultiPart.completeMpUpload(s.Key, id, s.DstPath, reqBody); err != nil {
		return nil, err
	}
	if s3err = s3.checkObjectSize(option, s.DstPath); s3err != nil {
		return nil, s3err
	}
	if s.ChecksumAlgorithm, s.ChecksumValue, s3err = s3.getChecksumMode(option, ""); s3err != nil {
		return nil, s3err
	}
	if s.ETag, s3err = s3.getETag(s.DstPath); s3err != nil {
		return nil, s3err
	}
	metaAPath := s3.FileSystem.getFullPath(s3.FileSystem.getPartNumberPath(id, s.Key)) // タグが設定されている場合は指定の場所に移動
	if s3.FileSystem.isFileExists(s3.FileSystem.getMetaFileName(metaAPath)) {
		s3.FileSystem.moveFile(s3.FileSystem.getMetaFileName(metaAPath), s3.FileSystem.getMetaFileName(option.GetPath()))
	}
	result := s.MakeCompleteMultipartUploadResult()
	s3.FileSystem.forceDeleteDir(id)
	return result, nil
	*/

	return &o, nil
}

func (bbs *Bb_server) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	fmt.Printf("*CopyObject*\n")
	var o = s3.CopyObjectOutput{}

	// List of parameters.
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
		var m = map[error]Aws_s3_error_code{
			fs.ErrExist: BucketAlreadyOwnedByYou}
		var err5 = map_os_error(location, err2, m)
		return nil, err5
	}

	o.Location = &location
	return &o, nil
}

func (bbs *Bb_server) CreateMultipartUpload(ctx context.Context, i *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	fmt.Printf("*CreateMultipartUpload*\n")
	var o = s3.CreateMultipartUploadOutput{}

	// List of parameters.
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

	var checksum = i.ChecksumAlgorithm
	var checksumtype = i.ChecksumType
	if checksumtype != types.ChecksumTypeFullObject {
		bbs.Logger.Info("Change ChecksumType",
			"requested", checksumtype,
			"employed", types.ChecksumTypeFullObject)
		checksumtype = types.ChecksumTypeFullObject
	}

	var rid int64 = get_request_id(ctx)
	var scratchkey = bbs.make_file_suffix(rid)
	defer bbs.discharge_file_suffix(rid)

	var err4 = bbs.serialize_access(ctx, object, rid)
	if err4 != nil {
		return nil, err4
	}
	defer bbs.release_access(ctx, object, rid)

	var mpul = &MPUL_info{
		Upload_id: scratchkey,
		Checksum_type: checksumtype,
		Checksum_algorithm: checksum,
		Meta_info: Meta_info{Headers: info.Headers, Tags: info.Tags}}
	var err6 = bbs.create_upload_directory(ctx, object, mpul)
	if err6 != nil {
		return nil, err6
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discharge_scratch_file(object, scratchkey)
		}
	}()

	var err7 = bbs.store_metainfo(object, info)
	if err7 != nil {
		return nil, err7
	}

	var uploadid = scratchkey

	{
		o.Bucket = i.Bucket
		o.Key = i.Key
		o.ChecksumAlgorithm = i.ChecksumAlgorithm
		o.ChecksumType = checksumtype
		o.UploadId = &uploadid
	}

	// o.AbortDate *time.Time
	// o.AbortRuleId *string
	// o.BucketKeyEnabled *bool
	// o.RequestCharged types.RequestCharged
	// o.SSECustomerAlgorithm *string
	// o.SSECustomerKeyMD5 *string
	// o.SSEKMSEncryptionContext *string
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption

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
	fmt.Printf("*DeleteBucket*\n")
	var o = s3.DeleteBucketOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.ExpectedBucketOwner *string

	// o.ResultMetadata middleware.Metadata

	return &o, nil
}

func (bbs *Bb_server) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	fmt.Printf("*DeleteObject*\n")
	var o = s3.DeleteObjectOutput{}

	// List of parameters.
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
	fmt.Printf("*DeleteObjects*\n")
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
	fmt.Printf("*DeleteObjectTagging*\n")
	var o = s3.DeleteObjectTaggingOutput{}

	// List of parameters.
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
	fmt.Printf("*GetObjectAttributes*\n")
	var o = s3.GetObjectAttributesOutput{}
	return &o, nil
}

func (bbs *Bb_server) GetObjectTagging(ctx context.Context, params *s3.GetObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.GetObjectTaggingOutput, error) {
	fmt.Printf("*GetObjectTagging*\n")
	var o = s3.GetObjectTaggingOutput{}

	// List of parameters.
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
	fmt.Printf("*ListMultipartUploads*\n")
	var o = s3.ListMultipartUploadsOutput{}

	// List of parameters.
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
	fmt.Printf("*ListParts*\n")
	var o = s3.ListPartsOutput{}

	// List of parameters.
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
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Resource: location,
				Message:  "MD5 value is illegal."}
			return nil, errz
		}
		md5_to_check = bs
	}

	var checksum types.ChecksumAlgorithm = i.ChecksumAlgorithm
	var csum_to_check []byte
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
		var csum2, err5 = base64.StdEncoding.DecodeString(*csum1)
		if err5 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Resource: location,
				Message:  "Checksum value is illegal."}
			return nil, errz
		}
		csum_to_check = csum2
	}

	var rid int64 = get_request_id(ctx)
	var scratchkey = bbs.make_file_suffix(rid)
	defer bbs.discharge_file_suffix(rid)

	/*
		var err1 = bbs.make_intermediate_directories(object)
		if err1 != nil {
			return nil, err1
		}
	*/

	var check = upload_checks{
		location:      location,
		size:          size,
		checksum:      checksum,
		md5_to_check:  md5_to_check,
		csum_to_check: csum_to_check,
	}
	var md5, csum, err6 = bbs.upload_file(ctx, object, scratchkey, info,
		check, i.Body)
	if err6 != nil {
		return nil, err6
	}

	/*
		var cleanup_needed = true
		defer func() {
			if cleanup_needed {
				bbs.discharge_scratch_file(object, scratchkey)
			}
		}()

		var checksum types.ChecksumAlgorithm = i.ChecksumAlgorithm
		var md5, csum, err7 = bbs.calculate_csum2(checksum, object, scratchkey)
		if err7 != nil {
			return nil, err7
		}

		if len(md5_to_check) != 0 && bytes.Compare(md5_to_check, md5) != 0 {
			bbs.Logger.Info("Digests mismatch",
				"algorithm", "MD5",
				"passed", hex.EncodeToString(md5_to_check),
				"calculated", hex.EncodeToString(md5))
			var errz = &Aws_s3_error{Code: BadDigest,
				Resource: location}
			return nil, errz
		}

		if len(csum_to_check) != 0 && bytes.Compare(csum_to_check, csum) != 0 {
			bbs.Logger.Info("Checksums mismatch",
				"algorithm", checksum,
				"passed", hex.EncodeToString(csum_to_check),
				"calculated", hex.EncodeToString(csum))
			var errz = &Aws_s3_error{Code: BadDigest,
				Resource: location,
				Message:  "The checksum did not match what we received."}
			return nil, errz
		}

		// It should be atomic on placing an uploaded file and saving a
		// meta-info file.  Failing to place an uploaded file may lose
		// meta-info.

		var err9 = bbs.serialize_access(ctx, object, rid)
		if err9 != nil {
			return nil, err9
		}
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
	*/

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

	{
		o.ETag = make_etag_from_md5(md5)
	}

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
	fmt.Printf("*PutObjectTagging*\n")
	var o = s3.PutObjectTaggingOutput{}

	// List of parameters.
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

func (bbs *Bb_server) UploadPart(ctx context.Context, i *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
	fmt.Printf("*UploadPart*\n")
	var o = s3.UploadPartOutput{}

	// List of parameters.
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

	var object, err1 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err1 != nil {
		return nil, err1
	}
	var location = "/" + object

	/*
		if i.CacheControl != nil {
			if !strings.EqualFold(*i.CacheControl, "no-cache") {
				var errz = &Aws_s3_error{Code: InvalidStorageClass,
					Message:  "Bad Cache-Control",
					Resource: location}
				return nil, errz
			}
		}
	*/

	/*
		var err2 = check_unsupported_options(object, i.StorageClass)
		if err2 != nil {
			return nil, err2
		}
	*/

	var part int32
	if i.PartNumber == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "PartNumber missing.",
			Resource: location}
		return nil, errz
	} else {
		part = *i.PartNumber
		if part < 1 || max_part_number < part {
			var errz = &Aws_s3_error{Code: InvalidPart,
				Resource: location}
			return nil, errz
		}
	}
	var uploadid string
	if i.UploadId == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "UploadId missing.",
			Resource: location}
		return nil, errz
	} else {
		uploadid = *i.UploadId
		if part < 1 || max_part_number < part {
			var errz = &Aws_s3_error{Code: InvalidPart,
				Resource: location}
			return nil, errz
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
		var md5, err2 = base64.StdEncoding.DecodeString(*i.ContentMD5)
		if err2 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message:  "MD5 value is illegal.",
				Resource: location}
			return nil, errz
		}
		md5_to_check = md5
	}

	var checksum types.ChecksumAlgorithm = i.ChecksumAlgorithm
	var csum_to_check []byte
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
		var csum2, err3 = base64.StdEncoding.DecodeString(*csum1)
		if err3 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Resource: location,
				Message:  "Checksum value is illegal."}
			return nil, errz
		}
		csum_to_check = csum2
	}

	var rid int64 = get_request_id(ctx)
	var scratchkey = bbs.make_file_suffix(rid)
	defer bbs.discharge_file_suffix(rid)

	var mpul, err4 = bbs.fetch_mpul_info(object)
	if err4 != nil {
		// AHO
		var errz = &Aws_s3_error{Code: NoSuchUpload,
			Resource: location}
		return nil, errz
	}
	if mpul.Upload_id != uploadid {
		var errz = &Aws_s3_error{Code: InvalidPart,
			Resource: location}
		return nil, errz
	}

	var partobject = make_part_object_name(object, part)

	/*
		var err1 = bbs.make_intermediate_directories(object)
		if err1 != nil {
			return err1
		}
	*/

	var check = upload_checks{
		location:      location,
		size:          size,
		checksum:      checksum,
		md5_to_check:  md5_to_check,
		csum_to_check: csum_to_check,
	}
	var md5, csum, err6 = bbs.upload_file(ctx, partobject, scratchkey, nil,
		check, i.Body)
	if err6 != nil {
		return nil, err6
	}

	/*
		var cleanup_needed = true
		defer func() {
			if cleanup_needed {
				bbs.discharge_scratch_file(object, scratchkey)
			}
		}()

		var md5, csum, errx = bbs.calculate_csum2(checksum, object, scratchkey)
		if errx != nil {
			return nil, errx
		}

		if len(md5_to_check) != 0 && bytes.Compare(md5_to_check, md5) != 0 {
			bbs.Logger.Info("Digests mismatch",
				"algorithm", "MD5",
				"passed", hex.EncodeToString(md5_to_check),
				"calculated", hex.EncodeToString(md5))
			var errz = &Aws_s3_error{Code: BadDigest,
				Resource: location}
			return nil, errz
		}
		if len(csum_to_check) != 0 && bytes.Compare(csum_to_check, csum) != 0 {
			bbs.Logger.Info("Checksums mismatch",
				"algorithm", checksum,
				"passed", hex.EncodeToString(csum_to_check),
				"calculated", hex.EncodeToString(csum))
			var errz = &Aws_s3_error{Code: BadDigest,
				Resource: location,
				Message:  "The checksum did not match what we received."}
			return nil, errz
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
	*/

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
		//o.ChecksumType = types.ChecksumTypeFullObject
	}

	{
		o.ETag = make_etag_from_md5(md5)
	}

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

	/*
		s := model.PutObjectState{}
		partNum := option.GetOption("partNumber")
		if !utils.CheckPartNumber(partNum) {
			return nil, InvalidArgument()
		}
		if !s3.FileSystem.checkBucketName(option.GetBucket()) {
			return nil, InvalidBucketName()
		}
		if b := option.GetBucket(); !s3.FileSystem.isFileExists(b) {
			return nil, NoSuchBucket()
		}
		if !s3.FileSystem.checkKeyName(option.GetKey()) {
			return nil, KeyTooLongError()
		}
		if !s3.FileSystem.canCreateFile(s3.FileSystem.getFullPath(option.GetPath())) {
			return nil, InternalError()
		}
		id := option.GetOption("uploadId")
		if !s3.FileSystem.isFileExists(s3.FileSystem.getUploadIDPath(id)) {
			return nil, NoSuchUpload()
		}
		pNumPath := s3.FileSystem.getPartNumberPath(id, partNum)
		if f := s3.FileSystem.uploadFile(option.GetBody(), pNumPath); !f {
			return nil, InternalError()
		}
		var s3err *S3Error
		if s.ETag, s3err = s3.getETag(pNumPath); s3err != nil {
			return nil, InternalError()
		}
		if s3err = s3.compareMd5(option, nil, s.ETag); s3err != nil {
			return nil, s3err
		}
		if s.ChecksumAlgorithm, s.ChecksumValue, s3err = s3.getChecksumMode(option, pNumPath); s3err != nil {
			return nil, s3err
		}
		result := s.MakePutObjectResult()
	*/

	return &o, nil
}

func (bbs *Bb_server) UploadPartCopy(ctx context.Context, params *s3.UploadPartCopyInput, optFns ...func(*s3.Options)) (*s3.UploadPartCopyOutput, error) {
	fmt.Printf("*UploadPartCopy*\n")
	var o = s3.UploadPartCopyOutput{}

	// List of parameters.
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
