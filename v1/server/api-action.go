// api-action.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// This contains implementations of actions.

package server

import (
	"context"
	//"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"time"
	//"github.com/riken-rccs/s3-baby-server/pkg/httpaide"
	"bytes"
	"encoding/base64"
	//"encoding/binary"
	"encoding/hex"
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

func (bbs *Bb_server) AbortMultipartUpload(ctx context.Context, i *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, *Aws_s3_error) {
	var action = "AbortMultipartUpload"
	fmt.Printf("*AbortMultipartUpload*\n")
	var o = s3.AbortMultipartUploadOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.Key *string
	// i.UploadId *string
	// i.ExpectedBucketOwner *string
	// i.IfMatchInitiatedTime *time.Time
	// i.RequestPayer types.RequestPayer

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
			RequestPayer:        i.RequestPayer,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var rid int64 = get_request_id(ctx)

	// SERIALIZE ACCESS.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	var mpul, err3 = bbs.check_upload_ongoing(object, i.UploadId)
	if err3 != nil {
		return nil, err3
	}

	if i.IfMatchInitiatedTime != nil {
		var itime = *i.IfMatchInitiatedTime
		if !mpul.Mtime.Equal(itime) {
			var errz = &Aws_s3_error{Code: PreconditionFailed,
				Resource: location}
			return nil, errz
		}
	}

	var err4 = bbs.discard_mpul_directory(object)
	if err4 != nil {
		// Ignore errors.
	}

	// o.RequestCharged types.RequestCharged

	return &o, nil
}

func (bbs *Bb_server) CompleteMultipartUpload(ctx context.Context, i *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, *Aws_s3_error) {
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

	// Errors: EntityTooSmall, InvalidPart, InvalidPartOrder,
	// NoSuchUpload

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner:  i.ExpectedBucketOwner,
			RequestPayer:         i.RequestPayer,
			SSECustomerAlgorithm: i.SSECustomerAlgorithm,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var mpul1, err3 = bbs.check_upload_ongoing(object, i.UploadId)
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

	var error_in_sorting *Aws_s3_error = nil
	var sorted = slices.IsSortedFunc(partlist.Parts,
		func(a, b types.CompletedPart) int {
			if a.PartNumber == nil || b.PartNumber == nil {
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
	var error_in_checking *Aws_s3_error = nil
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
				bbs.logger.Info("Part not uploaded",
					"action", action)
				error_in_checking = &Aws_s3_error{Code: NoSuchUpload,
					Resource: location}
				// Return true to stop the loop.
				return true
			}
			if catalog.Parts[part].ETag != etag {
				bbs.logger.Info("ETags mismatch",
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
					bbs.logger.Info("Checksums mismatch",
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
	var scratchkey = bbs.make_scratch_suffix(rid)
	defer bbs.discharge_scratch_suffix(rid)

	var err5 = bbs.concat_parts_as_scratch(ctx, object, scratchkey, partlist, mpul1)
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

	var csum_given = types.Checksum{
		//ChecksumType: types.ChecksumTypeFullObject,
		ChecksumType:      i.ChecksumType,
		ChecksumCRC32:     i.ChecksumCRC32,
		ChecksumCRC32C:    i.ChecksumCRC32C,
		ChecksumSHA1:      i.ChecksumCRC64NVME,
		ChecksumSHA256:    i.ChecksumSHA1,
		ChecksumCRC64NVME: i.ChecksumSHA256,
	}
	var _ = csum_given

	var checksum = types.ChecksumAlgorithmCrc64nvme
	if i.ChecksumType != types.ChecksumTypeFullObject {
		bbs.logger.Info("Checksum by not full-object unsuppored, ignored",
			"checksum-type", i.ChecksumType)
		checksum = ""
	}

	var md5, csum, err6 = bbs.calculate_csum2(checksum, object, scratchkey)
	if err6 != nil {
		return nil, err6
	}

	if checksum != "" {
		var csum1 *string
		switch checksum {
		case types.ChecksumAlgorithmCrc32:
			csum1 = i.ChecksumCRC32
		case types.ChecksumAlgorithmCrc32c:
			csum1 = i.ChecksumCRC32C
		case types.ChecksumAlgorithmCrc64nvme:
			csum1 = i.ChecksumCRC64NVME
		case types.ChecksumAlgorithmSha1:
			csum1 = i.ChecksumSHA1
		case types.ChecksumAlgorithmSha256:
			csum1 = i.ChecksumSHA256
		}
		var csum_to_check []byte
		if csum1 != nil {
			var csum2, err5 = base64.StdEncoding.DecodeString(*csum1)
			if err5 != nil {
				var errz = &Aws_s3_error{Code: InvalidArgument,
					Resource: location,
					Message:  "Checksum value is illegal."}
				return nil, errz
			}
			csum_to_check = csum2
		}
		if len(csum_to_check) != 0 && bytes.Compare(csum_to_check, csum) != 0 {
			bbs.logger.Info("Checksums mismatch",
				"algorithm", checksum,
				"passed", hex.EncodeToString(csum_to_check),
				"calculated", hex.EncodeToString(csum))
			var errz = &Aws_s3_error{Code: BadDigest,
				Resource: location,
				Message:  "The checksum did not match what we received."}
			return nil, errz
		}
	}

	var etag = make_etag_from_md5(md5)

	var _, err7 = bbs.check_request_conditions(&etag, nil, "POST",
		i.IfMatch, i.IfNoneMatch,
		nil, nil)
	if err7 != nil {
		return nil, err7
	}

	// SERIALIZE ACCESS.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	{
		// Check the upload-id again after exclusion.

		var mpul2, err3 = bbs.check_upload_ongoing(object, i.UploadId)
		if err3 != nil {
			return nil, err3
		}
		var info = mpul2.Meta_info

		var err1 = bbs.place_scratch_file(object, scratchkey, info)
		if err1 != nil {
			return nil, err1
		}

		cleanup_needed = false

		var err2 = bbs.discard_mpul_directory(object)
		if err2 != nil {
			// IGNORE ERRORS.
		}
	}

	o.ETag = &etag

	var address string
	if bbs.conf.Site_base_url != nil {
		var a, err1 = url.JoinPath(*bbs.conf.Site_base_url, location)
		if err1 != nil {
			// Ignore errors.
			address = location
		} else {
			address = a
		}
	} else {
		address = location
	}
	o.Location = &address

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
		o.Key = i.Key
	}

	// o.BucketKeyEnabled *bool
	// o.Expiration *string
	// o.RequestCharged types.RequestCharged
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption
	// o.VersionId *string

	return &o, nil
}

func (bbs *Bb_server) CopyObject(ctx context.Context, i *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, *Aws_s3_error) {
	var action = "CopyObject"
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

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			// i.ACL types.ObjectCannedACL
			// i.BucketKeyEnabled *bool
			// i.CacheControl *string
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
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var source, err15 = bbs.lookat_copy_source(object, i.CopySource)
	if err15 != nil {
		return nil, err15
	}

	var checksum types.ChecksumAlgorithm = i.ChecksumAlgorithm

	var s_stat, info, err3 = bbs.check_object_status(source)
	if err3 != nil {
		return nil, err3
	}
	var md5, csum, err4 = bbs.calculate_csum2(checksum, source, "")
	if err4 != nil {
		return nil, err4
	}
	var csum_calculated = fill_checksum_record(checksum, csum)

	var s_mtime = s_stat.ModTime()
	var etag = make_etag_from_md5(md5)

	var _, err5 = bbs.check_request_conditions(&etag, &s_mtime, "PUT",
		i.CopySourceIfMatch, i.CopySourceIfNoneMatch,
		i.CopySourceIfModifiedSince, i.CopySourceIfUnmodifiedSince)
	if err5 != nil {
		return nil, err5
	}

	var t_mtime time.Time
	if !copy_file_by_linking {
		var rid int64 = get_request_id(ctx)
		var scratchkey = bbs.make_scratch_suffix(rid)
		defer bbs.discharge_scratch_suffix(rid)

		var err6 = bbs.copy_file_as_scratch(ctx, object, scratchkey,
			source, nil)
		if err6 != nil {
			return nil, err6
		}
		var cleanup_needed = true
		defer func() {
			if cleanup_needed {
				bbs.discard_scratch_file(object, scratchkey)
			}
		}()

		if bbs.conf.Verify_fs_write {
			var md5x, _, err1 = bbs.calculate_csum2("", object, scratchkey)
			if err1 != nil {
				return nil, err1
			}
			if bytes.Compare(md5, md5x) != 0 {
				bbs.logger.Warn("Copying file failed, MD5 values differ",
					"source", hex.EncodeToString(md5),
					"target", hex.EncodeToString(md5x))
				var errz = &Aws_s3_error{
					Code:     InternalError,
					Message:  "Copying file failed",
					Resource: location}
				return nil, errz
			}
		}

		// SERIALIZE ACCESS.

		{
			var timeout = bbs.serialize_access(ctx, object, rid)
			if timeout != nil {
				return nil, timeout
			}
			defer bbs.release_access(ctx, object, rid)
		}

		{
			var err1 = bbs.place_scratch_file(object, scratchkey, info)
			if err1 != nil {
				return nil, err1
			}
		}

		var t_stat, err9 = bbs.fetch_object_status(object)
		if err9 != nil {
			return nil, err9
		}
		t_mtime = t_stat.ModTime()
		cleanup_needed = false
	} else {
		// copy_file_by_linking = true.

		var rid int64 = get_request_id(ctx)
		// var scratchkey = bbs.make_scratch_suffix(rid)
		// defer bbs.discharge_scratch_suffix(rid)

		// SERIALIZE ACCESS.

		{
			var timeout = bbs.serialize_access(ctx, object, rid)
			if timeout != nil {
				return nil, timeout
			}
			defer bbs.release_access(ctx, object, rid)
		}

		var err1 = bbs.store_metainfo(object, info)
		if err1 != nil {
			return nil, err1
		}

		var s_path = bbs.make_path_of_object(source, "")
		var t_path = bbs.make_path_of_object(object, "")

		var err2 = os.Link(s_path, t_path)
		if err2 != nil {
			bbs.logger.Warn("os.Link() failed on an object",
				"source", s_path, "object", t_path, "error", err2)
			return nil, map_os_error(location, err2, nil)
		}

		var t_stat, err9 = bbs.fetch_object_status(object)
		if err9 != nil {
			return nil, err9
		}
		t_mtime = t_stat.ModTime()
	}

	o.CopyObjectResult = &types.CopyObjectResult{
		// types.CopyObjectResult:
		// - ChecksumCRC32 *string
		// - ChecksumCRC32C *string
		// - ChecksumCRC64NVME *string
		// - ChecksumSHA1 *string
		// - ChecksumSHA256 *string
		// - ChecksumType ChecksumType
		// - ETag: *string
		// - LastModified *time.Time
		ChecksumCRC32:     csum_calculated.ChecksumCRC32,
		ChecksumCRC32C:    csum_calculated.ChecksumCRC32C,
		ChecksumCRC64NVME: csum_calculated.ChecksumCRC64NVME,
		ChecksumSHA1:      csum_calculated.ChecksumSHA1,
		ChecksumSHA256:    csum_calculated.ChecksumSHA256,
		ChecksumType:      csum_calculated.ChecksumType,
		ETag:              &etag,
		LastModified:      &t_mtime,
	}

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

func (bbs *Bb_server) CreateBucket(ctx context.Context, i *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, *Aws_s3_error) {
	var action = "CreateBucket"
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
		var err2 = &Aws_s3_error{Code: InvalidBucketName}
		return nil, err2
	}
	var location = "/" + bucket

	{
		var unsupported = option_check_list{
			GrantFullControl:           i.GrantFullControl,
			GrantRead:                  i.GrantRead,
			GrantReadACP:               i.GrantReadACP,
			GrantWrite:                 i.GrantWrite,
			GrantWriteACP:              i.GrantWriteACP,
			ObjectLockEnabledForBucket: i.ObjectLockEnabledForBucket,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}

		var ignored = option_check_list{
			ACL_bucket_canned:         i.ACL,
			CreateBucketConfiguration: i.CreateBucketConfiguration,
			ObjectOwnership:           i.ObjectOwnership,
		}
		bbs.check_options_ignored(action, location, &ignored)
	}

	var rid int64 = get_request_id(ctx)
	// var scratchkey = bbs.make_scratch_suffix(rid)
	// defer bbs.discharge_scratch_suffix(rid)

	// Note serialization may not be necessary as mkdir() is atomic.

	// SERIALIZE ACCESS.

	{
		var timeout = bbs.serialize_access(ctx, bucket, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, bucket, rid)
	}

	var path = bbs.make_path_of_bucket(bucket)
	var err3 = os.Mkdir(path, 0755)
	if err3 != nil {
		// Note the error on existing path is fs.PathError and not
		// fs.ErrExist.

		/*if errors.As(err2, &err3) {*/
		/*if !errors.Is(err2, fs.ErrExist) {*/
		/*var err4, ok = err3.Err.(syscall.Errno)*/

		bbs.logger.Debug("os.Mkdir() failed", "path", path, "error", err3)
		var m = map[error]Aws_s3_error_code{
			fs.ErrExist: BucketAlreadyOwnedByYou}
		var errz = map_os_error(location, err3, m)
		return nil, errz
	}

	{
		o.Location = &location
	}

	return &o, nil
}

func (bbs *Bb_server) CreateMultipartUpload(ctx context.Context, i *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, *Aws_s3_error) {
	var action = "CreateMultipartUpload"
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

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object
	var _ = location

	{
		var unsupported = option_check_list{
			StorageClass: i.StorageClass,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var info, err3 = make_metainfo(i.Metadata, i.Tagging, location)
	if err3 != nil {
		return nil, err3
	}

	var checksum = i.ChecksumAlgorithm
	var checksumtype = i.ChecksumType
	if checksumtype != types.ChecksumTypeFullObject {
		bbs.logger.Info("Change checksum-type",
			"requested", checksumtype,
			"employed", types.ChecksumTypeFullObject)
		checksumtype = types.ChecksumTypeFullObject
	}

	var rid int64 = get_request_id(ctx)
	var scratchkey = bbs.make_scratch_suffix(rid)
	defer bbs.discharge_scratch_suffix(rid)

	var uploadid = scratchkey

	// SERIALIZE ACCESS.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	var mpul = &Mpul_info{
		Upload_id:          uploadid,
		Mtime:              time.Now(),
		Checksum_type:      checksumtype,
		Checksum_algorithm: checksum,
		Meta_info:          info,
	}

	{
		var err6 = bbs.create_mpul_directory(ctx, object, mpul)
		if err6 != nil {
			return nil, err6
		}
		var cleanup_needed = true
		defer func() {
			if cleanup_needed {
				bbs.discard_mpul_directory(object)
			}
		}()

		var catalog = &Mpul_catalog{
			Checksum_algorithm: checksum,
		}
		var err7 = bbs.store_mpul_catalog(object, catalog)
		if err7 != nil {
			return nil, err7
		}

		cleanup_needed = false
	}

	{
		o.Bucket = i.Bucket
		o.Key = i.Key
		o.UploadId = &uploadid
		o.ChecksumType = checksumtype
		o.ChecksumAlgorithm = checksum
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

func (bbs *Bb_server) DeleteBucket(ctx context.Context, i *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, *Aws_s3_error) {
	var action = "DeleteBucket"
	fmt.Printf("*DeleteBucket*\n")
	var o = s3.DeleteBucketOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.ExpectedBucketOwner *string

	var bucket, err2 = check_usual_bucket_setup(ctx, bbs, i.Bucket)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + bucket

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var rid int64 = get_request_id(ctx)

	// SERIALIZE ACCESS.

	{
		var timeout = bbs.serialize_access(ctx, bucket, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, bucket, rid)
	}

	var path = bbs.make_path_of_bucket(bucket)
	var err3 = os.Remove(path)

	// Check some objects remain, when removing has failed.

	if err3 != nil {
		var err4 = bbs.check_bucket_empty(path)
		if err4 != nil {
			return nil, err4
		}

		// Only files remain that start with a dot.  Remove them.

		bbs.logger.Info(("Try os.RemoveAll() after removing a bucket failed"),
			"path", path)

		var err5 = os.RemoveAll(path)
		if err5 != nil {
			bbs.logger.Info("os.RemoveAll() failed", "path", path,
				"error", err5)
			var errz = &Aws_s3_error{Code: BucketNotEmpty,
				Resource: location}
			return nil, errz
		}
	}

	return &o, nil
}

func (bbs *Bb_server) DeleteObject(ctx context.Context, i *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, *Aws_s3_error) {
	var action = "DeleteObject"
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

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			MFA:                 i.MFA,
			ExpectedBucketOwner: i.ExpectedBucketOwner,
			VersionId:           i.VersionId,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var _, _, err3 = bbs.check_object_status(object)
	if err3 != nil {
		return nil, err3
	}

	var md5, _, err4 = bbs.calculate_csum2("", object, "")
	if err4 != nil {
		return nil, err4
	}
	var etag = make_etag_from_md5(md5)

	var _, err5 = bbs.check_request_conditions(&etag, nil, "POST",
		i.IfMatch, nil,
		nil, nil)
	if err5 != nil {
		return nil, err5
	}

	var rid int64 = get_request_id(ctx)

	// SERIALIZE ACCESS.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	{
		var err1 = bbs.store_metainfo(object, nil)
		if err1 != nil {
			// IGNORE ERRORS.
		}
		var path = bbs.make_path_of_object(object, "")
		var err2 = os.Remove(path)
		if err2 != nil {
			bbs.logger.Warn("os.Remove() failed on an object",
				"file", path, "error", err2)
			return nil, map_os_error(location, err2, nil)
		}
	}

	// o.DeleteMarker *bool
	// o.RequestCharged types.RequestCharged
	// o.VersionId *string

	return &o, nil
}

func (bbs *Bb_server) DeleteObjects(ctx context.Context, i *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, *Aws_s3_error) {
	var action = "DeleteObjects"
	fmt.Printf("*DeleteObjects*\n")
	var o = s3.DeleteObjectsOutput{}

	// i.Bucket *string
	// i.Delete *types.Delete
	// i.BypassGovernanceRetention *bool
	// i.ChecksumAlgorithm types.ChecksumAlgorithm
	// i.ExpectedBucketOwner *string
	// i.MFA *string
	// i.RequestPayer types.RequestPayer

	// Note "i.ChecksumAlgorithm" is passed by
	// "x-amz-sdk-checksum-algorithm" ("sdk" with it).

	var dummy = "dummy"
	var _, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, &dummy)
	if err2 != nil {
		return nil, err2
	}
	var bucket = *i.Bucket
	var location = "/" + bucket

	{
		var unsupported = option_check_list{
			BypassGovernanceRetention: i.BypassGovernanceRetention,
			ChecksumAlgorithm:         i.ChecksumAlgorithm,
			ExpectedBucketOwner:       i.ExpectedBucketOwner,
			MFA:                       i.MFA,
			RequestPayer:              i.RequestPayer,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	if i.Delete == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message: "Body of DeteteObjects is missing."}
		return nil, errz
	}

	// types.Delete is:
	// - Objects []ObjectIdentifier
	// - Quiet *bool

	var quiet bool = ((*i.Delete).Quiet != nil && *(*i.Delete).Quiet)

	var deletestate = make([]struct {
		object string
		error  types.Error
	}, len(i.Delete.Objects))

	// Check conditions of objects.

	{
	loop1:
		for i, e := range i.Delete.Objects {
			// e : types.ObjectIdentifier.
			// - Key *string
			// - ETag *string
			// - LastModifiedTime *time.Time
			// - Size *int64
			// - VersionId *string

			deletestate[i].error.Key = e.Key
			var u, err3 = url.Parse(*e.Key)
			if err3 != nil {
				var errz = &Aws_s3_error{Code: InvalidArgument,
					Message: "Bad key to DeleteObjects."}
				deletestate[i].error.Code = &errz.Code
				deletestate[i].error.Message = &errz.Message
				continue loop1
			}
			var key = u.Path
			var object = path.Join(bucket, key)
			if !check_object_naming(object) {
				var errz = &Aws_s3_error{Code: InvalidArgument,
					Message: "Bad object naming to DeleteObjects."}
				deletestate[i].error.Code = &errz.Code
				deletestate[i].error.Message = &errz.Message
				continue loop1
			}
			deletestate[i].object = object

			if e.VersionId != nil {
				var errz = &Aws_s3_error{Code: NotImplemented,
					Message: "VersionID is not implemented."}
				deletestate[i].error.Code = &errz.Code
				deletestate[i].error.Message = &errz.Message
				continue loop1
			}

			var stat, _, err4 = bbs.check_object_status(object)
			if err4 != nil {
				deletestate[i].error.Code = &err4.Code
				deletestate[i].error.Message = &err4.Message
				continue loop1
			}

			var mtime = stat.ModTime()
			if e.LastModifiedTime != nil && !mtime.Equal(*e.LastModifiedTime) {
				var errz = &Aws_s3_error{Code: PreconditionFailed,
					Message: "LastModifiedTime does not match."}
				deletestate[i].error.Code = &errz.Code
				deletestate[i].error.Message = &errz.Message
				continue loop1
			}

			var size = stat.Size()
			if e.Size != nil && size != *e.Size {
				var errz = &Aws_s3_error{Code: PreconditionFailed,
					Message: "Size does not match."}
				deletestate[i].error.Code = &errz.Code
				deletestate[i].error.Message = &errz.Message
				continue loop1
			}

			if e.ETag != nil {
				var md5, _, err4 = bbs.calculate_csum2("", object, "")
				if err4 != nil {
					deletestate[i].error.Code = &err4.Code
					deletestate[i].error.Message = &err4.Message
					continue loop1
				}
				var etag = make_etag_from_md5(md5)
				if etag != *e.ETag {
					var errz = &Aws_s3_error{Code: PreconditionFailed,
						Message: "ETag does not match."}
					deletestate[i].error.Code = &errz.Code
					deletestate[i].error.Message = &errz.Message
					continue loop1
				}
			}
		}
	}

	var rid int64 = get_request_id(ctx)
	// var scratchkey = bbs.make_scratch_suffix(rid)
	// defer bbs.discharge_scratch_suffix(rid)

	// Deleting files and checking conditions are slack.  It
	// serializes on a bucket.  Also, ETag calculation takes time and
	// it is placed outside of serialization.

	// SERIALIZE ACCESS.

	{
		var timeout = bbs.serialize_access(ctx, bucket, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, bucket, rid)
	}

	// Perform deletes.

	{
	loop2:
		for i, e := range deletestate {
			if e.object != "" && e.error.Code == nil {
				var object = e.object
				var err6 = bbs.store_metainfo(object, nil)
				if err6 != nil {
					// IGNORE ERRORS.
					// deletestate[i].error.Code = &err6.Code
					// deletestate[i].error.Message = &err6.Message
					// continue loop2
				}
				var path = bbs.make_path_of_object(object, "")
				var err7 = os.Remove(path)
				if err7 != nil {
					bbs.logger.Warn("os.Remove() failed on an object",
						"file", path, "error", err7)
					var errz = map_os_error(location, err7, nil)
					deletestate[i].error.Code = &errz.Code
					deletestate[i].error.Message = &errz.Message
					continue loop2
				}
			}
		}
	}

	// Fill the return record: o.Deleted and o.Errors.

	var deletelist []types.DeletedObject
	var errorlist []types.Error
	{
		for _, e := range deletestate {
			if e.error.Code == nil {
				if e.object == "" {
					log.Fatal("BAD-IMPL")
				}
				var d = types.DeletedObject{
					// d : types.DeletedObject.
					// - DeleteMarker *bool
					// - DeleteMarkerVersionId *string
					// - Key *string
					// - VersionId *string
					Key: e.error.Key,
				}
				deletelist = append(deletelist, d)
			} else {
				var d = types.Error{
					// d : types.Error.
					// - Code *string
					// - Key *string
					// - Message *string
					// - VersionId *string
					Key:     e.error.Key,
					Code:    e.error.Code,
					Message: e.error.Message,
				}
				errorlist = append(errorlist, d)
			}
		}
	}

	{
		if !quiet {
			o.Deleted = deletelist
		}
		o.Errors = errorlist
	}

	// o.RequestCharged types.RequestCharged
	// o.ResultMetadata middleware.Metadata

	return &o, nil
}

func (bbs *Bb_server) DeleteObjectTagging(ctx context.Context, i *s3.DeleteObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectTaggingOutput, *Aws_s3_error) {
	var action = "DeleteObjectTagging"
	fmt.Printf("*DeleteObjectTagging*\n")
	var o = s3.DeleteObjectTaggingOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.Key *string
	// i.ExpectedBucketOwner *string
	// i.VersionId *string

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	//var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
			VersionId:           i.VersionId,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var rid int64 = get_request_id(ctx)

	// SERIALIZE ACCESS.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	var _, info, err3 = bbs.check_object_status(object)
	if err3 != nil {
		return nil, err3
	}

	// Modify metainfo, and remove the file when it become nothing.

	if info != nil && info.Tags != nil {
		info.Tags = nil
		if info.Headers == nil {
			info = nil
		}
		var err7 = bbs.store_metainfo(object, info)
		if err7 != nil {
			return nil, err7
		}
	}

	// o.VersionId *string

	return &o, nil
}

func (bbs *Bb_server) GetObject(ctx context.Context, i *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, *Aws_s3_error) {
	var action = "GetObject"
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

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
			PartNumber:          i.PartNumber,
			VersionId:           i.VersionId,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

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

	var mtime = stat.ModTime()
	var etag = make_etag_from_md5(md5)

	var _, err6 = bbs.check_request_conditions(&etag, &mtime, "GET",
		i.IfMatch, i.IfNoneMatch,
		i.IfModifiedSince, i.IfUnmodifiedSince)
	if err6 != nil {
		return nil, err6
	}

	// NO SERIALIZE-ACCESS.

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

	o.LastModified = &mtime
	o.ETag = &etag

	if i.ChecksumMode == types.ChecksumModeEnabled {
		o.ChecksumType = types.ChecksumTypeFullObject
		var crc = base64.StdEncoding.EncodeToString(csum)
		o.ChecksumCRC64NVME = &crc
	}

	if info != nil && info.Headers != nil {
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

func (bbs *Bb_server) GetObjectAttributes(ctx context.Context, i *s3.GetObjectAttributesInput, optFns ...func(*s3.Options)) (*s3.GetObjectAttributesOutput, *Aws_s3_error) {
	var action = "GetObjectAttributes"
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

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner:  i.ExpectedBucketOwner,
			RequestPayer:         i.RequestPayer,
			SSECustomerAlgorithm: i.SSECustomerAlgorithm,
			SSECustomerKey:       i.SSECustomerKey,
			SSECustomerKeyMD5:    i.SSECustomerKeyMD5,
			VersionId:            i.VersionId,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}

		var ignored = option_check_list{
			MaxParts:         i.MaxParts,
			PartNumberMarker: i.PartNumberMarker,
		}
		bbs.check_options_ignored(action, location, &ignored)
	}

	var stat, info, err3 = bbs.check_object_status(object)
	if err3 != nil {
		return nil, err3
	}
	var _ = info

	var rid int64 = get_request_id(ctx)
	var _ = rid
	//var scratchkey = bbs.make_scratch_suffix(rid)
	//defer bbs.discharge_scratch_suffix(rid)

	// NO SERIALIZE-ACCESS.

	var checksum = types.ChecksumAlgorithmCrc64nvme
	var md5, csum, err6 = bbs.calculate_csum2(checksum, object, "")
	if err6 != nil {
		return nil, err6
	}

	var attributes = i.ObjectAttributes
	if slices.Contains(attributes, types.ObjectAttributesEtag) {
		var etag = make_etag_from_md5(md5)
		o.ETag = &etag
	}
	if slices.Contains(attributes, types.ObjectAttributesChecksum) {
		var csum_calculated = fill_checksum_record(checksum, csum)
		o.Checksum = csum_calculated
	}
	if slices.Contains(attributes, types.ObjectAttributesObjectParts) {
		o.ObjectParts = nil
	}
	if slices.Contains(attributes, types.ObjectAttributesStorageClass) {
		o.StorageClass = types.StorageClassStandard
	}
	if slices.Contains(attributes, types.ObjectAttributesObjectSize) {
		var size = stat.Size()
		o.ObjectSize = &size
	}
	var mtime = stat.ModTime()
	o.LastModified = &mtime

	// parts : types.GetObjectAttributesParts
	// - IsTruncated *bool
	// - MaxParts *int32
	// - NextPartNumberMarker *string
	// - PartNumberMarker *string
	// - Parts []types.ObjectPart
	// - TotalPartsCount *int32
	// types.ObjectPart:
	// - ChecksumCRC32 *string
	// - ChecksumCRC32C *string
	// - ChecksumCRC64NVME *string
	// - ChecksumSHA1 *string
	// - ChecksumSHA256 *string
	// - PartNumber *int32
	// - Size *int64

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

func (bbs *Bb_server) GetObjectTagging(ctx context.Context, i *s3.GetObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.GetObjectTaggingOutput, *Aws_s3_error) {
	var action = "GetObjectTagging"
	fmt.Printf("*GetObjectTagging*\n")
	var o = s3.GetObjectTaggingOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.Key *string
	// i.ExpectedBucketOwner *string
	// i.RequestPayer types.RequestPayer
	// i.VersionId *string

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	//var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
			RequestPayer:        i.RequestPayer,
			VersionId:           i.VersionId,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var _, info, err3 = bbs.check_object_status(object)
	if err3 != nil {
		return nil, err3
	}

	// NO SERIALIZE-ACCESS.

	if info != nil && info.Tags != nil {
		o.TagSet = info.Tags.TagSet
	}

	// o.VersionId *string

	return &o, nil
}

func (bbs *Bb_server) HeadBucket(ctx context.Context, i *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, *Aws_s3_error) {
	var action = "HeadBucket"
	fmt.Printf("*HeadBucket*\n")
	var o = s3.HeadBucketOutput{}

	// List of parameters.
	// i.Bucket *string
	// i.ExpectedBucketOwner *string

	var _, err2 = check_usual_bucket_setup(ctx, bbs, i.Bucket)
	if err2 != nil {
		return nil, err2
	}

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	// NO SERIALIZE-ACCESS.

	// o.AccessPointAlias *bool
	// o.BucketArn *string
	// o.BucketLocationName *string
	// o.BucketLocationType types.LocationType
	// o.BucketRegion *string

	return &o, nil
}

func (bbs *Bb_server) HeadObject(ctx context.Context, i *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, *Aws_s3_error) {
	var action = "HeadObject"
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

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			PartNumber:          i.PartNumber,
			ExpectedBucketOwner: i.ExpectedBucketOwner,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

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

	var mtime = stat.ModTime()
	var etag = make_etag_from_md5(md5)

	var _, err6 = bbs.check_request_conditions(&etag, &mtime, "HEAD",
		i.IfMatch, i.IfNoneMatch,
		i.IfModifiedSince, i.IfUnmodifiedSince)
	if err6 != nil {
		return nil, err6
	}

	// NO SERIALIZE-ACCESS.

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

	o.ETag = &etag

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

func (bbs *Bb_server) ListBuckets(ctx context.Context, i *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, *Aws_s3_error) {
	var action = "ListBuckets"
	fmt.Printf("*ListBuckets*\n")
	var o = s3.ListBucketsOutput{}

	// List of parameters.
	// i.BucketRegion *string
	// i.ContinuationToken *string
	// i.MaxBuckets *int32
	// i.Prefix *string

	{
		var unsupported = option_check_list{}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var start int
	if i.ContinuationToken != nil {
		var n, err1 = strconv.ParseInt(*i.ContinuationToken, 10, 32)
		if err1 != nil {
			var err2 = make_parameter_error("continuation-token", err1)
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

	var prefix string
	if i.Prefix != nil {
		prefix = *i.Prefix
	} else {
		prefix = ""
	}

	// NO SERIALIZE-ACCESS.

	var buckets, continuation, err3 = bbs.list_buckets(start, max_buckets,
		prefix)
	if err3 != nil {
		return nil, err3
	}

	o.Buckets = buckets
	if continuation != 0 {
		var scontinuation = strconv.FormatInt(int64(continuation), 10)
		o.ContinuationToken = &scontinuation
	}

	{
		o.Prefix = i.Prefix
	}

	// o.Owner

	return &o, nil
}

func (bbs *Bb_server) ListMultipartUploads(ctx context.Context, i *s3.ListMultipartUploadsInput, optFns ...func(*s3.Options)) (*s3.ListMultipartUploadsOutput, *Aws_s3_error) {
	var action = "ListMultipartUploads"
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

	var bucket, err2 = check_usual_bucket_setup(ctx, bbs, i.Bucket)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + bucket
	var _ = location

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
			RequestPayer:        i.RequestPayer,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var marker string
	var maxkeys int
	var delimiter string
	var prefix string
	var urlencode bool

	if i.KeyMarker != nil {
		marker = *i.KeyMarker
	}
	if i.MaxUploads != nil {
		maxkeys = int(min(list_objects_limit, *i.MaxUploads))
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

	// NO SERIALIZE-ACCESS.

	var objects, commons, nextmarker, err5 = bbs.list_mpuls_flat(
		bucket, marker, maxkeys, delimiter, prefix, urlencode)
	if err5 != nil {
		return nil, err5
	}

	var istruncated = (nextmarker != "")

	o.Uploads = objects
	o.CommonPrefixes = commons
	o.IsTruncated = &istruncated
	o.KeyMarker = &nextmarker
	o.NextUploadIdMarker = nil

	{
		o.Bucket = i.Bucket
		o.Delimiter = i.Delimiter
		o.EncodingType = i.EncodingType
		o.MaxUploads = i.MaxUploads
		o.Prefix = i.Prefix
		o.UploadIdMarker = i.UploadIdMarker
	}

	// o.RequestCharged types.RequestCharged

	return &o, nil
}

func (bbs *Bb_server) ListObjects(ctx context.Context, i *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, *Aws_s3_error) {
	var action = "ListObjects"
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

	var bucket, err2 = check_usual_bucket_setup(ctx, bbs, i.Bucket)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + bucket
	var _ = location

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner:      i.ExpectedBucketOwner,
			OptionalObjectAttributes: i.OptionalObjectAttributes,
			RequestPayer:             i.RequestPayer,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var index = 0
	var marker string
	var maxkeys int
	var delimiter string
	var prefix string
	var urlencode bool

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
	if i.EncodingType == types.EncodingTypeUrl {
		urlencode = true
	}

	// NO SERIALIZE-ACCESS.

	var entries []object_list_entry
	var nextindex int
	var nextmarker string
	var err3 *Aws_s3_error
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
		entries, bucket, delimiter, prefix, urlencode)
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

func (bbs *Bb_server) ListObjectsV2(ctx context.Context, i *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, *Aws_s3_error) {
	var action = "ListObjectsV2"
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

	var bucket, err2 = check_usual_bucket_setup(ctx, bbs, i.Bucket)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + bucket
	var _ = location

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner:      i.ExpectedBucketOwner,
			FetchOwner:               i.FetchOwner,
			OptionalObjectAttributes: i.OptionalObjectAttributes,
			RequestPayer:             i.RequestPayer,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
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
			var err4 = make_parameter_error("continuation-token", err3)
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

	// NO SERIALIZE-ACCESS.

	var entries []object_list_entry
	var nextindex int
	var nextmarker string
	var err5 *Aws_s3_error
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

func (bbs *Bb_server) ListParts(ctx context.Context, i *s3.ListPartsInput, optFns ...func(*s3.Options)) (*s3.ListPartsOutput, *Aws_s3_error) {
	var action = "ListParts"
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

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	//var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
			RequestPayer:        i.RequestPayer,
			// i.SSECustomerAlgorithm *string
			// i.SSECustomerKey *string
			// i.SSECustomerKeyMD5 *string
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	/*
		var mpul, err3 = bbs.check_upload_ongoing(object, i.UploadId)
		if err3 != nil {
			return nil, err3
		}
	*/

	var count int32 = -1
	if i.MaxParts != nil {
		count = *i.MaxParts
	}
	var index int32
	if i.PartNumberMarker != nil {
		var n, err3 = strconv.ParseInt(*i.PartNumberMarker, 10, 32)
		if err3 != nil {
			var err4 = make_parameter_error("part-number-marker", err3)
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message: err4.Error()}
			return nil, errz
		}
		index = int32(n)
	}

	var catalog, err4 = bbs.fetch_mpul_catalog(object)
	if err4 != nil {
		return nil, err4
	}

	// NO SERIALIZE-ACCESS.

	// Copy MPUL catalog to a result record.

	var partlist []types.Part
	var checksum = catalog.Checksum_algorithm
	var parts = catalog.Parts
	var endindex int32
	if count != -1 {
		var start = min(index, int32(len(parts)))
		var end = min(index+count, int32(len(parts)))
		//truncated = (end >= int32(len(parts)))
		parts = parts[start:end]
		endindex = end
	} else {
		var start = min(index, int32(len(parts)))
		parts = parts[start:]
		endindex = -1
	}

	for i, e := range parts {
		// Part is counted by base one.
		var no = int32(i + 1)
		var p = types.Part{
			// p : types.Part
			// - ChecksumCRC32 *string
			// - ChecksumCRC32C *string
			// - ChecksumCRC64NVME *string
			// - ChecksumSHA1 *string
			// - ChecksumSHA256 *string
			// - ETag *string
			// - LastModified *time.Time
			// - PartNumber *int32
			// - Size *int64

			ETag:         &e.ETag,
			LastModified: &e.Mtime,
			PartNumber:   &no,
			Size:         &e.Size,
		}
		var csum1 = &e.Checksum
		switch checksum {
		case types.ChecksumAlgorithmCrc32:
			p.ChecksumCRC32 = csum1
		case types.ChecksumAlgorithmCrc32c:
			p.ChecksumCRC32C = csum1
		case types.ChecksumAlgorithmSha1:
			p.ChecksumSHA1 = csum1
		case types.ChecksumAlgorithmSha256:
			p.ChecksumSHA256 = csum1
		case types.ChecksumAlgorithmCrc64nvme:
			p.ChecksumCRC64NVME = csum1
		}
		partlist = append(partlist, p)
	}

	{
		o.Key = i.Key
		o.MaxParts = i.MaxParts
		o.PartNumberMarker = i.PartNumberMarker
		o.UploadId = i.UploadId
		if endindex != -1 {
			var truncated = true
			var n string = fmt.Sprintf("%d", endindex)
			o.IsTruncated = &truncated
			o.NextPartNumberMarker = &n
		}
		o.ChecksumAlgorithm = checksum
		o.ChecksumType = types.ChecksumTypeFullObject
		o.StorageClass = types.StorageClassStandard
		o.Parts = partlist
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

	return &o, nil
}

func (bbs *Bb_server) PutObject(ctx context.Context, i *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, *Aws_s3_error) {
	var action = "PutObject"
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

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
			StorageClass:        i.StorageClass,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	// AHO ?? Check "Cache-Control" which only accepts "no-cache".

	if i.CacheControl != nil {
		if !strings.EqualFold(*i.CacheControl, "no-cache") {
			var errz = &Aws_s3_error{Code: InvalidStorageClass,
				Message:  "Bad Cache-Control",
				Resource: location}
			return nil, errz
		}
	}

	var info, err3 = make_metainfo(i.Metadata, i.Tagging, location)
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
	var scratchkey = bbs.make_scratch_suffix(rid)
	defer bbs.discharge_scratch_suffix(rid)

	// SERIALIZE ACCESS (in the uploading routine).

	var check = upload_checks{
		upload_id:      "",
		size:           size,
		checksum:       checksum,
		md5_to_check:   md5_to_check,
		csum_to_check:  csum_to_check,
		etag_condition: [2]*string{i.IfMatch, i.IfNoneMatch},
	}
	var md5, csum, err6 = bbs.upload_file(ctx, object, scratchkey, object,
		info, check, i.Body)
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

	var etag = make_etag_from_md5(md5)
	o.ETag = &etag

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

func (bbs *Bb_server) PutObjectTagging(ctx context.Context, i *s3.PutObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.PutObjectTaggingOutput, *Aws_s3_error) {
	var action = "PutObjectTagging"
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

	// ERRORS:
	// - InvalidTag
	// - MalformedXML
	// - OperationAborted
	// - InternalError

	// IGNORE i.ContentMD5.

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	//var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
			RequestPayer:        i.RequestPayer,
			VersionId:           i.VersionId,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	bbs.logger.Debug("Tagging", "action", action, "tagging", i.Tagging)

	var rid int64 = get_request_id(ctx)

	// SERIALIZE ACCESS.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	{
		var _, info, err3 = bbs.check_object_status(object)
		if err3 != nil {
			return nil, err3
		}

		if info == nil {
			info = &Meta_info{
				Headers: nil,
				Tags:    nil,
			}
		}

		info.Tags = i.Tagging
		var err7 = bbs.store_metainfo(object, info)
		if err7 != nil {
			return nil, err7
		}
	}

	// o.VersionId *string

	return &o, nil
}

func (bbs *Bb_server) UploadPart(ctx context.Context, i *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, *Aws_s3_error) {
	var action = "UploadPart"
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

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner:  i.ExpectedBucketOwner,
			RequestPayer:         i.RequestPayer,
			SSECustomerAlgorithm: i.SSECustomerAlgorithm,
			SSECustomerKey:       i.SSECustomerKey,
			SSECustomerKeyMD5:    i.SSECustomerKeyMD5,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

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

	var mpul, err3 = bbs.check_upload_ongoing(object, i.UploadId)
	if err3 != nil {
		return nil, err3
	}
	var part, err4 = bbs.lookat_part_number(object, i.PartNumber)
	if err4 != nil {
		return nil, err4
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
	var scratchkey = bbs.make_scratch_suffix(rid)
	defer bbs.discharge_scratch_suffix(rid)

	var partobject = make_mpul_part_name(object, part)

	// SERIALIZE ACCESS (in the uploading routine).

	var check = upload_checks{
		upload_id:      mpul.Upload_id,
		size:           size,
		checksum:       checksum,
		md5_to_check:   md5_to_check,
		csum_to_check:  csum_to_check,
		etag_condition: [2]*string{nil, nil},
	}
	var md5, csum, err6 = bbs.upload_file(ctx, partobject, scratchkey, object,
		nil, check, i.Body)
	if err6 != nil {
		return nil, err6
	}

	var stat, _, err7 = bbs.check_object_status(partobject)
	if err7 != nil {
		return nil, err7
	}

	{
		var catalog, err4 = bbs.fetch_mpul_catalog(object)
		if err4 != nil {
			return nil, err4
		}

		if part > int32(len(catalog.Parts)) {
			var n = (part - int32(len(catalog.Parts)))
			var adds = make([]Mpul_part, n)
			catalog.Parts = append(catalog.Parts, adds...)
		}

		var etag = make_etag_from_md5(md5)

		catalog.Parts[part-1] = Mpul_part{
			Size:     size,
			ETag:     etag,
			Checksum: base64.StdEncoding.EncodeToString(csum),
			Mtime:    stat.ModTime(),
		}

		var err8 = bbs.store_mpul_catalog(object, catalog)
		if err8 != nil {
			return nil, err8
		}
	}

	var etag = make_etag_from_md5(md5)
	o.ETag = &etag

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

	// o.BucketKeyEnabled *bool
	// o.RequestCharged types.RequestCharged
	// o.SSECustomerAlgorithm *string
	// o.SSECustomerKeyMD5 *string
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption

	return &o, nil
}

func (bbs *Bb_server) UploadPartCopy(ctx context.Context, i *s3.UploadPartCopyInput, optFns ...func(*s3.Options)) (*s3.UploadPartCopyOutput, *Aws_s3_error) {
	var action = "UploadPartCopy"
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

	var object, err2 = check_usual_object_setup(ctx, bbs, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			CopySourceSSECustomerAlgorithm: i.CopySourceSSECustomerAlgorithm,
			ExpectedBucketOwner:            i.ExpectedBucketOwner,
			ExpectedSourceBucketOwner:      i.ExpectedSourceBucketOwner,
			RequestPayer:                   i.RequestPayer,
			SSECustomerAlgorithm:           i.SSECustomerAlgorithm,
		}
		var err1 = check_options_unsupported(action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var _, err3 = bbs.check_upload_ongoing(object, i.UploadId)
	if err3 != nil {
		return nil, err3
	}
	var part, err4 = bbs.lookat_part_number(object, i.PartNumber)
	if err4 != nil {
		return nil, err4
	}

	var source, err5 = bbs.lookat_copy_source(object, i.CopySource)
	if err5 != nil {
		return nil, err5
	}
	var s_stat, _, err13 = bbs.check_object_status(source)
	if err13 != nil {
		return nil, err13
	}

	{
		var md5, _, err14 = bbs.calculate_csum2("", source, "")
		if err14 != nil {
			return nil, err14
		}
		//var csum_calculated = fill_checksum_record(checksum, csum)

		var s_mtime = s_stat.ModTime()
		var s_etag = make_etag_from_md5(md5)

		var _, err15 = bbs.check_request_conditions(&s_etag, &s_mtime, "PUT",
			i.CopySourceIfMatch, i.CopySourceIfNoneMatch,
			i.CopySourceIfModifiedSince, i.CopySourceIfUnmodifiedSince)
		if err15 != nil {
			return nil, err15
		}
	}

	var size = s_stat.Size()
	var extent, err24 = scan_range(i.CopySourceRange, size, location)
	if err24 != nil {
		return nil, err24
	}

	var rid int64 = get_request_id(ctx)
	var scratchkey = bbs.make_scratch_suffix(rid)
	defer bbs.discharge_scratch_suffix(rid)

	var partobject = make_mpul_part_name(object, part)

	var err6 = bbs.copy_file_as_scratch(ctx, partobject, scratchkey,
		source, extent)
	if err6 != nil {
		return nil, err6
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(partobject, scratchkey)
		}
	}()

	// SERIALIZE ACCESS.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	var t_mtime time.Time
	{
		var err1 = bbs.place_scratch_file(object, scratchkey, nil)
		if err1 != nil {
			return nil, err1
		}

		var t_stat, err9 = bbs.fetch_object_status(object)
		if err9 != nil {
			return nil, err9
		}
		t_mtime = t_stat.ModTime()

		cleanup_needed = false
	}

	o.CopyPartResult = &types.CopyPartResult{
		// - ChecksumCRC32 *string
		// - ChecksumCRC32C *string
		// - ChecksumCRC64NVME *string
		// - ChecksumSHA1 *string
		// - ChecksumSHA256 *string
		// - ETag *string
		// - LastModified *time.Time

		LastModified: &t_mtime,
	}

	// o.BucketKeyEnabled *bool
	// o.CopySourceVersionId *string
	// o.RequestCharged types.RequestCharged
	// o.SSECustomerAlgorithm *string
	// o.SSECustomerKeyMD5 *string
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption

	return &o, nil
}
