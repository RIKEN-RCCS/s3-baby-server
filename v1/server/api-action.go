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
	"slices"
	//"log/slog"
	//"math/rand"
	//"net/http"
	"net/url"
	"strconv"
	"strings"
	//"sync"
)

func (bbs *Bb_server) AbortMultipartUpload(ctx context.Context, i *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
	fmt.Printf("*AbortMultipartUpload*\n")
	var o = s3.AbortMultipartUploadOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.Key *string
	// i.UploadId *string
	// i.ExpectedBucketOwner *string
	// i.IfMatchInitiatedTime *time.Time
	// i.RequestPayer types.RequestPayer

	var unsupported = unsupported_checks {
		expectedbucketowner: i.ExpectedBucketOwner,
		// i.RequestPayer types.RequestPayer
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	var rid int64 = get_request_id(ctx)

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	var mpul, err3 = bbs.check_upload_id(object, i.UploadId)
	if err3 != nil {
		return nil, err3
	}
	if i.IfMatchInitiatedTime != nil {
		if !mpul.Timestamp.Equal(*i.IfMatchInitiatedTime) {
			var errz = &Aws_s3_error{Code: PreconditionFailed,
				Resource: location}
			return nil, errz
		}
	}

	var err4 = bbs.discharge_mpul_directory(object)
	if err4 != nil {
		// Ignore errors.
	}

	// o.RequestCharged types.RequestCharged

	return &o, nil
}

func (bbs *Bb_server) CompleteMultipartUpload(ctx context.Context, i *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	var action = "CompleteMultipartUpload"
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

	var unsupported = unsupported_checks {
		expectedbucketowner: i.ExpectedBucketOwner,
		// i.RequestPayer types.RequestPayer
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

	// Error Code: EntityTooSmall
	// Error Code: InvalidPart
	// Error Code: InvalidPartOrder
	// Error Code: NoSuchUpload

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	var mpul, err3 = bbs.check_upload_id(object, i.UploadId)
	if err3 != nil {
		return nil, err3
	}

	if i.MpuObjectSize == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "x-amz-mp-object-size missing.",
			Resource: location}
		return nil, errz
	}
	var size = *i.MpuObjectSize
	var _ = size

	var partlist = i.MultipartUpload
	if partlist == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "Request body missing.",
			Resource: location}
		return nil, errz
	}

	// Check parts are sorted.

	var error_in_sorting error = nil
	var sorted = slices.IsSortedFunc(partlist.Parts,
		func(a, b types.CompletedPart) int {
			if a.PartNumber == nil || b.PartNumber == nil {
				// x_assert(error_in_sorting == nil)
				error_in_sorting = &Aws_s3_error{Code: InvalidArgument,
					Message:  "PartNumber missing.",
					Resource: location}
				// Return a positive to stop the loop.
				return 1
			}
			return int(*a.PartNumber - *b.PartNumber)
		})
	if error_in_sorting != nil {
		return nil, error_in_sorting
	}
	if !sorted {
		var errz = &Aws_s3_error{Code: InvalidPartOrder,
			Resource: location}
		return nil, errz
	}

	// Check etags and checksums of parts.

	var catalog, err4 = bbs.fetch_mpul_catalog(object)
	if err4 != nil {
		return nil, err4
	}
	var error_in_checking error = nil
	var ng = slices.ContainsFunc(partlist.Parts,
		func(e types.CompletedPart) bool {
			// It returns true on an error to stop the loop.
			if e.PartNumber == nil || e.ETag == nil {
				error_in_checking = &Aws_s3_error{Code: InvalidArgument,
					Message:  "PartNumber/ETag missing.",
					Resource: location}
				// Return true to stop the loop.
				return true
			}
			var part = *e.PartNumber
			var etag = *e.ETag
			if part >= int32(len(catalog.Parts)) {
				bbs.Logger.Info("Part not uploaded",
					"action", action)
				error_in_checking = &Aws_s3_error{Code: NoSuchUpload,
					Resource: location}
				// Return true to stop the loop.
				return true
			}
			if catalog.Parts[part].ETag != etag {
				bbs.Logger.Info("ETags mismatch",
					"action", action)
				error_in_checking = &Aws_s3_error{Code: InvalidPart,
					Resource: location}
				// Return true to stop the loop.
				return true
			}
			var csum *string
			var checksum = catalog.Checksum_algorithm
			if checksum != "" {
				switch checksum {
				case types.ChecksumAlgorithmCrc32:
					csum = e.ChecksumCRC32
				case types.ChecksumAlgorithmCrc32c:
					csum = e.ChecksumCRC32C
				case types.ChecksumAlgorithmCrc64nvme:
					csum = e.ChecksumCRC64NVME
				case types.ChecksumAlgorithmSha1:
					csum = e.ChecksumSHA1
				case types.ChecksumAlgorithmSha256:
					csum = e.ChecksumSHA256
				}
				if csum == nil {
					error_in_checking = &Aws_s3_error{Code: InvalidArgument,
						Message:  "Checksum missing in multipart upload",
						Resource: location}
					// Return true to stop the loop.
					return true
				}
				if catalog.Parts[part].Checksum != *csum {
					bbs.Logger.Info("Checksums mismatch",
						"action", action)
					error_in_checking = &Aws_s3_error{Code: InvalidPart,
						Resource: location}
					// Return true to stop the loop.
					return true
				}
			}
			return false
		})
	if error_in_checking != nil {
		return nil, error_in_checking
	}
	if ng {
		log.Fatal("BAD-IMPL: slices.ContainsFunc() returns something bad" +
			" in CompleteMultipartUpload")
	}

	var rid int64 = get_request_id(ctx)
	var scratchkey = bbs.make_file_suffix(rid)
	defer bbs.discharge_file_suffix(rid)

	var err5 = bbs.concat_parts_as_scratch(ctx, object, scratchkey, partlist, mpul)
	if err5 != nil {
		return nil, err5
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(object, scratchkey)
		}
	}()

	// Baby-server can only handle "types.ChecksumTypeFullObject".
	// The checksum of the input is ignored when it is not the case.
	// The returned checksum is always for full object.

	if i.ChecksumType != types.ChecksumTypeFullObject {
	}

	var checksum = types.ChecksumAlgorithmCrc64nvme
	var md5, csum, err6 = bbs.calculate_csum2(checksum, object, scratchkey)
	if err6 != nil {
		return nil, err6
	}

	var _, err7 = bbs.check_conditions(ctx, i.IfMatch, i.IfNoneMatch,
		nil, nil, md5)
	if err7 != nil {
		return nil, err7
	}

	var info = mpul.Meta_info

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	var err9 = bbs.place_scratch_file(object, scratchkey, info)
	if err9 != nil {
		return nil, err9
	}

	cleanup_needed = false

	var err10 = bbs.discharge_mpul_directory(object)
	if err10 != nil {
		// Ignore errors.
	}

	o.ETag = make_etag_from_md5(md5)

	if checksum != "" {
		var csum1 = base64.StdEncoding.EncodeToString(csum)
		switch checksum {
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
		o.Bucket = i.Bucket
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

	return &o, nil
}

func (bbs *Bb_server) CopyObject(ctx context.Context, i *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
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

	var unsupported = unsupported_checks {
		// i.CopySource *string
		// i.ACL types.ObjectCannedACL
		// i.BucketKeyEnabled *bool
		// i.CacheControl *string
		// i.ChecksumAlgorithm types.ChecksumAlgorithm
		// i.ContentDisposition *string
		// i.ContentEncoding *string
		// i.ContentLanguage *string
		// i.ContentType *string
		// i.CopySourceSSECustomerAlgorithm *string
		// i.ExpectedBucketOwner *string
		// i.ExpectedSourceBucketOwner *string
		// i.Expires *time.Time
		// i.GrantFullControl *string
		// i.GrantRead *string
		// i.GrantReadACP *string
		// i.GrantWriteACP *string
		// i.MetadataDirective types.MetadataDirective
		// i.ObjectLockLegalHoldStatus types.ObjectLockLegalHoldStatus
		// i.ObjectLockMode types.ObjectLockMode
		// i.ObjectLockRetainUntilDate *time.Time
		// i.RequestPayer types.RequestPayer
		// i.SSECustomerAlgorithm *string
		// i.ServerSideEncryption types.ServerSideEncryption
		// i.TaggingDirective types.TaggingDirective
		// i.WebsiteRedirectLocation *string
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}
	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}

	if i.CopySource == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message: "No x-amz-copy-source supplied."}
		return nil, errz
	}
	var u, err3 = url.Parse(*i.CopySource)
	if err3 != nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message: "Bad x-amz-copy-source."}
		return nil, errz
	}
	var source = u.Path

	{
		var d1 = strings.Split(object, "/")
		var s1 = strings.Split(source, "/")
		if check_object_naming(source) {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message: "Bad x-amz-copy-source."}
			return nil, errz
		}
		if !(len(d1) >= 2 && len(s1) >= 2 && d1[0] == s1[0]) {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message: "x-amz-copy-source must be in the same bucket."}
			return nil, errz
		}
	}

	var _, info, err4 = bbs.check_object_status(source)
	if err4 != nil {
		return nil, err4
	}

	var rid int64 = get_request_id(ctx)
	var scratchkey = bbs.make_file_suffix(rid)
	defer bbs.discharge_file_suffix(rid)

	var err6 = bbs.copy_file_as_scratch(ctx, object, scratchkey, source)
	if err6 != nil {
		return nil, err6
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(object, scratchkey)
		}
	}()

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	var md5, _, err7 = bbs.calculate_csum2("", object, scratchkey)
	if err7 != nil {
		return nil, err7
	}

	var err9 = bbs.place_scratch_file(object, scratchkey, info)
	if err9 != nil {
		return nil, err9
	}

	var d_stat, err10 = bbs.fetch_object_status(object)
	if err10 != nil {
		return nil, err10
	}

	var mtime = d_stat.ModTime()
	var etag = make_etag_from_md5(md5)

	o.CopyObjectResult = &types.CopyObjectResult {
		// ChecksumCRC32 *string
		// ChecksumCRC32C *string
		// ChecksumCRC64NVME *string
		// ChecksumSHA1 *string
		// ChecksumSHA256 *string
		// ChecksumType ChecksumType
		ETag: etag,
		LastModified: &mtime,
	}

	cleanup_needed = false

	// o.BucketKeyEnabled *bool
	// o.CopySourceVersionId *string
	// o.Expiration *string
	// o.RequestCharged types.RequestCharged
	// o.SSECustomerAlgorithm *string
	// o.SSECustomerKeyMD5 *string
	// o.SSEKMSEncryptionContext *string
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption
	// o.VersionId *string

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

	var unsupported = unsupported_checks {
		// expectedbucketowner: i.ExpectedBucketOwner
		// i.RequestPayer types.RequestPayer
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

	if i.Bucket == nil {
		log.Fatalf("BAD-IMPL: Bucket parameter missing")
	}
	var bucket = *i.Bucket
	if !check_bucket_naming(bucket) {
		var err2 = &Aws_s3_error{Code: InvalidBucketName}
		return nil, err2
	}

	var location = "/" + bucket

	var path = bbs.make_path_of_bucket(bucket)
	var err3 = os.Mkdir(path, 0755)
	if err3 != nil {
		// Note the error on existing path is fs.PathError and not
		// fs.ErrExist.

		/*if errors.As(err2, &err3) {*/
		/*if !errors.Is(err2, fs.ErrExist) {*/
		/*var err4, ok = err3.Err.(syscall.Errno)*/

		bbs.Logger.Info("os.Mkdir() failed", "error", err3)
		var m = map[error]Aws_s3_error_code{
			fs.ErrExist: BucketAlreadyOwnedByYou}
		var errz = map_os_error(location, err3, m)
		return nil, errz
	}

	o.Location = &location
	return &o, nil
}

func (bbs *Bb_server) CreateMultipartUpload(ctx context.Context, i *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	//var action = "CreateMultipartUpload"
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

	var unsupported = unsupported_checks {
		storageclass: i.StorageClass,
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object
	var _ = location

	var info, err3 = make_meta_info(i.Metadata, i.Tagging, location)
	if err3 != nil {
		return nil, err3
	}

	var checksum = i.ChecksumAlgorithm
	var checksumtype = i.ChecksumType
	if checksumtype != types.ChecksumTypeFullObject {
		bbs.Logger.Info("Change checksum-type",
			"requested", checksumtype,
			"employed", types.ChecksumTypeFullObject)
		checksumtype = types.ChecksumTypeFullObject
	}

	var rid int64 = get_request_id(ctx)
	var scratchkey = bbs.make_file_suffix(rid)
	defer bbs.discharge_file_suffix(rid)
	var uploadid = scratchkey

	var mpul = &Mpul_info{
		Upload_id:          uploadid,
		Checksum_type:      checksumtype,
		Checksum_algorithm: checksum,
		Meta_info:          info}

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	var err6 = bbs.create_mpul_directory(ctx, object, mpul)
	if err6 != nil {
		return nil, err6
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discharge_mpul_directory(object)
		}
	}()

	var err7 = bbs.store_metainfo(object, info)
	if err7 != nil {
		return nil, err7
	}

	{
		o.Bucket = i.Bucket
		o.Key = i.Key
		o.UploadId = &uploadid
		o.ChecksumType = checksumtype
		o.ChecksumAlgorithm = i.ChecksumAlgorithm
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

	return &o, nil
}

func (bbs *Bb_server) DeleteBucket(ctx context.Context, i *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
	fmt.Printf("*DeleteBucket*\n")
	var o = s3.DeleteBucketOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.ExpectedBucketOwner *string

	var unsupported = unsupported_checks {
		expectedbucketowner: i.ExpectedBucketOwner,
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

	var bucket, err2 = check_usual_bucket_setup(ctx, bbs, i.Bucket)
	if err2 != nil {
		return nil, err2
	}
	var object = bucket
	var location = "/" + object

	var rid int64 = get_request_id(ctx)

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	var path = bbs.make_path_of_bucket(bucket)
	var err3 = os.Remove(path)

	// Check some objects remain, when removing has failed.

	if err3 != nil {
		var filelist, err4 = os.ReadDir(path)
		if err4 != nil {
			//if errors.Is(err4, fs.ErrNotExist)
			bbs.Logger.Info("Reading in a bucket failed",
				"path", path, "error", err4)
			var errz = &Aws_s3_error{Code: InternalError,
				Message: "Listing in a bucket failed.",
				Resource: location}
			return nil, errz
		}
		for _, e := range filelist {
			if strings.HasPrefix(e.Name(), ".") {
				continue
			}
			var errz = &Aws_s3_error{Code: BucketNotEmpty,
				Resource: location}
			return nil, errz
		}

		// Nothing but files start with a dot.  Remove them.

		var err5 = os.RemoveAll(path)
		if err5 != nil {
			bbs.Logger.Info("os.RemoveAll() failed", "path", path,
				"error", err5)
			var errz = &Aws_s3_error{Code: BucketNotEmpty,
				Resource: location}
			return nil, errz
		}
	}

	return &o, nil
}

func (bbs *Bb_server) DeleteObject(ctx context.Context, i *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
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

	var unsupported = unsupported_checks {
		mfa: i.MFA,
		expectedbucketowner: i.ExpectedBucketOwner,
		versionid: i.VersionId,
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object
	var _, _, err3 = bbs.check_object_status(object)
	if err3 != nil {
		return nil, err3
	}

	var md5, _, err4 = bbs.calculate_csum2("", object, "")
	if err4 != nil {
		return nil, err4
	}

	var _, err5 = bbs.check_conditions(ctx, i.IfMatch, nil,
		nil, nil, md5)
	if err5 != nil {
		return nil, err5
	}

	var rid int64 = get_request_id(ctx)

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	var err6 = bbs.store_metainfo(object, nil)
	if err6 != nil {
		return nil, err6
	}
	var path = bbs.make_path_of_object(object, "")
	var err7 = os.Remove(path)
	if err7 != nil {
		bbs.Logger.Warn("os.Remove() failed on an object",
			"file", path, "error", err7)
		var errz = map_os_error(location, err7, nil)
		return nil, errz
	}

	// o.DeleteMarker *bool
	// o.RequestCharged types.RequestCharged
	// o.VersionId *string

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

	var unsupported = unsupported_checks {
		//mfa: i.MFA,
		expectedbucketowner: i.ExpectedBucketOwner,
		//partnumber: i.PartNumber
		versionid: i.VersionId,
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object
	var stat, info, err3 = bbs.check_object_status(object)
	if err3 != nil {
		return nil, err3
	}

	var size = stat.Size()
	var extent, err4 = scan_range(i.Range, size, location)
	if err4 != nil {
		return nil, err4
	}

	var checksum = types.ChecksumAlgorithmCrc64nvme
	var md5, csum, err5 = bbs.calculate_csum2(checksum, object, "")
	if err5 != nil {
		return nil, err5
	}
	var _, err6 = bbs.check_conditions(ctx, i.IfMatch, i.IfNoneMatch,
		i.IfModifiedSince, i.IfUnmodifiedSince, md5)
	if err6 != nil {
		return nil, err6
	}

	var f1, err7 = bbs.make_file_stream(ctx, object, extent)
	if err7 != nil {
		return nil, err7
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

	o.ETag = make_etag_from_md5(md5)

	if i.ChecksumMode == types.ChecksumModeEnabled {
		o.ChecksumType = types.ChecksumTypeFullObject
		var crc = base64.StdEncoding.EncodeToString(csum)
		o.ChecksumCRC64NVME = &crc
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

	return &o, nil
}

func (bbs *Bb_server) GetObjectAttributes(ctx context.Context, params *s3.GetObjectAttributesInput, optFns ...func(*s3.Options)) (*s3.GetObjectAttributesOutput, error) {
	fmt.Printf("*GetObjectAttributes*\n")
	var o = s3.GetObjectAttributesOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.Key *string
	// i.ObjectAttributes []types.ObjectAttributes
	// i.ExpectedBucketOwner *string
	// i.MaxParts *int32
	// i.PartNumberMarker *string
	// i.RequestPayer types.RequestPayer
	// i.SSECustomerAlgorithm *string
	// i.SSECustomerKey *string
	// i.SSECustomerKeyMD5 *string
	// i.VersionId *string

	// o.Checksum *types.Checksum
	// o.DeleteMarker *bool
	// o.ETag *string
	// o.LastModified *time.Time
	// o.ObjectParts *types.GetObjectAttributesParts
	// o.ObjectSize *int64
	// o.RequestCharged types.RequestCharged
	// o.StorageClass types.StorageClass
	// o.VersionId *string

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

	// List of parameters.
	// i.Bucket *string
	// i.ExpectedBucketOwner *string

	var unsupported = unsupported_checks {
		expectedbucketowner: i.ExpectedBucketOwner,
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

	var _, err2 = check_usual_bucket_setup(ctx, bbs, i.Bucket)
	if err2 != nil {
		return nil, err2
	}

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

	var unsupported = unsupported_checks {
		partnumber: i.PartNumber,
		expectedbucketowner: i.ExpectedBucketOwner,
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object
	var stat, info, err3 = bbs.check_object_status(object)
	if err3 != nil {
		return nil, err3
	}

	var size = stat.Size()
	var extent, err4 = scan_range(i.Range, size, location)
	if err4 != nil {
		return nil, err4
	}

	var checksum = types.ChecksumAlgorithmCrc64nvme
	var md5, csum, err5 = bbs.calculate_csum2(checksum, object, "")
	if err5 != nil {
		return nil, err5
	}

	var _, err6 = bbs.check_conditions(ctx, i.IfMatch, i.IfNoneMatch,
		i.IfModifiedSince, i.IfUnmodifiedSince, md5)
	if err6 != nil {
		return nil, err6
	}

	var mtime = stat.ModTime()
	o.LastModified = &mtime

	if extent != nil {
		var length = extent[1] - extent[0]
		o.ContentLength = &length
		var srange = fmt.Sprintf("bytes %d-%d/%d", extent[0], extent[1], size)
		o.ContentRange = &srange
	} else {
		o.ContentLength = &size
	}
	var one int32 = 1
	o.PartsCount = &one

	if i.ChecksumMode == types.ChecksumModeEnabled {
		o.ChecksumType = types.ChecksumTypeFullObject
		var crc = base64.StdEncoding.EncodeToString(csum)
		o.ChecksumCRC64NVME = &crc
	}
	o.ETag = make_etag_from_md5(md5)

	if info != nil {
		// Always leave "MissingMeta" nil for zero.
		o.Metadata = info.Headers
		o.MissingMeta = nil
		if info.Tags != nil {
			var count = int32(len(info.Tags.TagSet))
			if count > 0 {
				o.TagCount = &count
			}
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

	var unsupported = unsupported_checks {
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

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

	var unsupported = unsupported_checks {
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

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

	var unsupported = unsupported_checks {
		expectedbucketowner: i.ExpectedBucketOwner,
		optionalobjectattributes: i.OptionalObjectAttributes,
		requestpayer: i.RequestPayer,
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

	var bucket, err2 = check_usual_bucket_setup(ctx, bbs, i.Bucket)
	if err2 != nil {
		return nil, err2
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
	var err3 error
	if !always_use_flat_lister && delimiter == "/" {
		entries, nextindex, nextmarker, err3 = bbs.list_objects_delimited(
			bucket, index, marker, maxkeys, delimiter, prefix)
	} else {
		entries, nextindex, nextmarker, err3 = bbs.list_objects_flat(
			bucket, index, marker, maxkeys, delimiter, prefix)
	}
	if err3 != nil {
		return nil, err3
	}

	var contents, commonprefixes, err4 = bbs.make_list_objects_entries(
		entries, bucket, delimiter, prefix, false)
	var _ = err4
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

	var unsupported = unsupported_checks {
		expectedbucketowner: i.ExpectedBucketOwner,
		optionalobjectattributes: i.OptionalObjectAttributes,
		requestpayer: i.RequestPayer,
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

	var bucket, err2 = check_usual_bucket_setup(ctx, bbs, i.Bucket)
	if err2 != nil {
		return nil, err2
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
		var n, err3 = strconv.ParseInt(*i.ContinuationToken, 10, 32)
		if err3 != nil {
			var err4 = Bb_input_error{"continuation-token", err3}
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message: err4.Error()}
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
	var err5 error
	if !always_use_flat_lister && delimiter == "/" {
		entries, nextindex, nextmarker, err5 = bbs.list_objects_delimited(
			bucket, index, marker, maxkeys, delimiter, prefix)
	} else {
		entries, nextindex, nextmarker, err5 = bbs.list_objects_flat(
			bucket, index, marker, maxkeys, delimiter, prefix)
	}
	if err5 != nil {
		return nil, err5
	}
	var _ = nextmarker

	var contents, commonprefixes, err6 = bbs.make_list_objects_entries(
		entries, bucket, delimiter, prefix, urlencode)
	var _ = err6
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

	var unsupported = unsupported_checks {
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

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

	var unsupported = unsupported_checks {
		expectedbucketowner: i.ExpectedBucketOwner,
		storageclass: i.StorageClass,
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
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

	var check = upload_checks{
		location:       location,
		uploadid:	    "",
		size:           size,
		checksum:       checksum,
		md5_to_check:   md5_to_check,
		csum_to_check:  csum_to_check,
		etag_condition: [2]*string{i.IfMatch, i.IfNoneMatch},
	}
	var md5, csum, err6 = bbs.upload_file(ctx, object, scratchkey, info,
		check, i.Body)
	if err6 != nil {
		return nil, err6
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

	var unsupported = unsupported_checks {
		expectedbucketowner: i.ExpectedBucketOwner,
		requestpayer: i.RequestPayer,
		// i.SSECustomerAlgorithm *string
	}
	var err1 = check_unsupported_options(&unsupported)
	if err1 != nil {
		return nil, err1
	}

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
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

	var mpul, err3 = bbs.check_upload_id(object, i.UploadId)
	if err3 != nil {
		return nil, err3
	}

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

	var size int64
	if i.ContentLength != nil {
		size = *i.ContentLength
	} else {
		size = -1
	}

	var md5_to_check []byte
	if i.ContentMD5 != nil {
		var md5, err4 = base64.StdEncoding.DecodeString(*i.ContentMD5)
		if err4 != nil {
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

	var partobject = make_part_object_name(object, part)

	var check = upload_checks{
		location:      location,
		uploadid:	   mpul.Upload_id,
		size:          size,
		checksum:      checksum,
		md5_to_check:  md5_to_check,
		csum_to_check: csum_to_check,
		etag_condition: [2]*string{nil, nil},
	}
	var md5, csum, err6 = bbs.upload_file(ctx, partobject, scratchkey, nil,
		check, i.Body)
	if err6 != nil {
		return nil, err6
	}

	if checksum != "" {
		var csum1 = base64.StdEncoding.EncodeToString(csum)
		switch checksum {
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
	// o.RequestCharged types.RequestCharged
	// o.SSECustomerAlgorithm *string
	// o.SSECustomerKeyMD5 *string
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption

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
