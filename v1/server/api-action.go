// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// This contains implementations of actions.

package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path"
	"slices"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func (bbs *Bbs_server) AbortMultipartUpload(ctx context.Context, i *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, *Aws_s3_error) {
	var o = s3.AbortMultipartUploadOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

	// List of parameters.
	// i.Bucket *string
	// i.Key *string
	// i.UploadId *string
	// i.ExpectedBucketOwner *string
	// i.IfMatchInitiatedTime *time.Time
	// i.RequestPayer types.RequestPayer

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
			RequestPayer:        i.RequestPayer,
		}
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	var mpulinfo, err3 = bbs.check_mpul_ongoing(rid, object, i.UploadId, true)
	if err3 != nil {
		return nil, err3
	}

	if i.IfMatchInitiatedTime != nil {
		var itime = *i.IfMatchInitiatedTime
		if !mpulinfo.Initiate_time.Equal(itime) {
			var errz = &Aws_s3_error{Code: PreconditionFailed,
				Resource: location}
			return nil, errz
		}
	}

	var err4 = bbs.discard_mpul_directory(rid, object)
	if err4 != nil {
		// IGNORE-ERRORS.
	}

	// o.RequestCharged types.RequestCharged

	return &o, nil
}

func (bbs *Bbs_server) CompleteMultipartUpload(ctx context.Context, i *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, *Aws_s3_error) {
	var o = s3.CompleteMultipartUploadOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
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
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var mpulinfo, err3 = bbs.check_mpul_ongoing(rid, object, i.UploadId, false)
	if err3 != nil {
		return nil, err3
	}

	var size_to_check int64
	if i.MpuObjectSize == nil {
		size_to_check = -1
	} else {
		size_to_check = *i.MpuObjectSize
	}

	// Check parts are nonempty.

	var partlist = i.MultipartUpload
	if partlist == nil || len(partlist.Parts) == 0 {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "Parts in request empty.",
			Resource: location}
		return nil, errz
	}

	// Check parts are sorted.

	var error_in_sorting *Aws_s3_error = nil
	var sorted = slices.IsSortedFunc(partlist.Parts,
		func(a, b types.CompletedPart) int {
			if a.PartNumber == nil || b.PartNumber == nil {
				error_in_sorting = &Aws_s3_error{Code: InvalidArgument,
					Message:  "Part number missing.",
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

	var catalog, err4 = bbs.fetch_mpul_catalog(rid, object)
	if err4 != nil {
		return nil, err4
	}

	var error_in_checking *Aws_s3_error = nil
	var nogood = slices.ContainsFunc(partlist.Parts,
		func(e types.CompletedPart) bool {
			// It returns true on an error to stop the loop.
			if e.PartNumber == nil || e.ETag == nil {
				error_in_checking = &Aws_s3_error{Code: InvalidArgument,
					Message:  "Part number or ETag missing.",
					Resource: location}
				// Return true to stop the loop.
				return true
			}
			var part = *e.PartNumber
			var etag = *bbs.fix_etag_quoting(e.ETag, rid)
			if !(0 <= (part-1) && (part-1) < int32(len(catalog.Parts)) &&
				catalog.Parts[part-1].ETag != "") {
				bbs.logger.Info("Part not uploaded",
					"rid", rid, "part", part)
				error_in_checking = &Aws_s3_error{Code: NoSuchUpload,
					Resource: location}
				// Return true to stop the loop.
				return true
			}
			var partinfo = catalog.Parts[part-1]
			if partinfo.ETag != etag {
				bbs.logger.Info("ETags mismatch in MPUL completion",
					"rid", rid, "part", part,
					"listed-etag", etag,
					"uploaded-etag", partinfo.ETag)
				error_in_checking = &Aws_s3_error{Code: InvalidPart,
					Resource: location}
				// Return true to stop the loop.
				return true
			}

			var csumset3 = &types.Checksum{
				ChecksumType:      types.ChecksumTypeFullObject,
				ChecksumCRC32:     e.ChecksumCRC32,
				ChecksumCRC32C:    e.ChecksumCRC32C,
				ChecksumCRC64NVME: e.ChecksumCRC64NVME,
				ChecksumSHA1:      e.ChecksumSHA1,
				ChecksumSHA256:    e.ChecksumSHA256,
			}
			var checksum3, csum_to_check3, err8 = bbs.decode_checksum_union(rid, object, csumset3)
			if err8 != nil {
				// IGNORE-ERRORS.
			}
			if csum_to_check3 != nil {
				if checksum3 != partinfo.Checksum {
					bbs.logger.Info("Checksum algorithm mismatch",
						"rid", rid, "object", object, "part", part,
						"mpul-completion", checksum3,
						"mpul-uploaded", partinfo.Checksum)
					error_in_checking = &Aws_s3_error{Code: InvalidPart,
						Message:  "Checksum algorithm mismatch.",
						Resource: location}
					// Return true to stop the loop.
					return true
				}
				var csum3, err7 = decode_base64(object, &partinfo.Csum)
				if err7 != nil {
					// IGNORE-ERRORS.
				}
				if bytes.Compare(csum_to_check3, csum3) != 0 {
					bbs.logger.Info("Checksum mismatch",
						"rid", rid, "object", object, "part", part,
						"algorithm", checksum3,
						"mpul-completion", hex.EncodeToString(csum_to_check3),
						"mpul-uploaded", hex.EncodeToString(csum3))
					error_in_checking = &Aws_s3_error{Code: InvalidPart,
						Message:  "Checksum mismatch.",
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
	if nogood {
		log.Fatal("BAD-IMPL: slices.ContainsFunc() returns something bad" +
			" in CompleteMultipartUpload")
	}

	var csumset = &types.Checksum{
		ChecksumType:      i.ChecksumType,
		ChecksumCRC32:     i.ChecksumCRC32,
		ChecksumCRC32C:    i.ChecksumCRC32C,
		ChecksumCRC64NVME: i.ChecksumCRC64NVME,
		ChecksumSHA1:      i.ChecksumSHA1,
		ChecksumSHA256:    i.ChecksumSHA256,
	}
	var checksum, csum_to_check, err8 = bbs.decode_checksum_union(rid, object, csumset)
	if err8 != nil {
		return nil, err8
	}

	if checksum != "" && checksum != mpulinfo.Checksum {
		bbs.logger.Info("Checksum algorithm differs MPUL creation/completion",
			"rid", rid, "creation", mpulinfo.Checksum, "completion", checksum)
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message:  "Checksum algorithm differs MPUL creation/completion.",
			Resource: location}
		return nil, errz
	}

	var checks = copy_checks{
		checksum:        checksum,
		csum_in_trailer: false,
		size_to_check:   size_to_check,
		csum_to_check:   csum_to_check,
	}
	var conditions = copy_conditions{
		some_match: i.IfMatch,
		none_match: i.IfNoneMatch,
	}

	var scratch = bbs.make_scratch_object_name(object, suffix)

	var md5v, csum, err6a = bbs.concatenate_scratch(ctx, object,
		mpulinfo, partlist, checks, conditions)
	if err6a != nil {
		return nil, err6a
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(rid, object, scratch)
		}
	}()

	{
		var err3 = bbs.compare_checksums(rid, object, scratch, checksum,
			md5v, csum, checks)
		if err3 != nil {
			return nil, err3
		}
	}

	// SERIALIZE-ACCESSES (in the concatenation routine).

	var etag, _, err6 = bbs.concatenate_object(ctx, object,
		mpulinfo, partlist, md5v, csum, checks, conditions, &cleanup_needed)
	if err6 != nil {
		return nil, err6
	}

	o.ETag = &etag

	var address string
	if bbs.config.Site_base_url != nil {
		var a, err1 = url.JoinPath(*bbs.config.Site_base_url, location)
		if err1 != nil {
			// IGNORE-ERRORS.
			address = location
		} else {
			address = a
		}
	} else {
		address = location
	}
	o.Location = &address

	{
		//var checksum2 = types.ChecksumAlgorithmCrc64nvme
		//var csumset2 *types.Checksum = fill_checksum_union(checksum2, csum)
		o.ChecksumType = csumset.ChecksumType
		o.ChecksumCRC32 = csumset.ChecksumCRC32
		o.ChecksumCRC32C = csumset.ChecksumCRC32C
		o.ChecksumCRC64NVME = csumset.ChecksumCRC64NVME
		o.ChecksumSHA1 = csumset.ChecksumSHA1
		o.ChecksumSHA256 = csumset.ChecksumSHA256
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

func (bbs *Bbs_server) CopyObject(ctx context.Context, i *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, *Aws_s3_error) {
	var o = s3.CopyObjectOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			ACL_object_canned:              i.ACL,
			BucketKeyEnabled:               i.BucketKeyEnabled,
			CopySourceSSECustomerAlgorithm: i.CopySourceSSECustomerAlgorithm,
			CopySourceSSECustomerKey:       i.CopySourceSSECustomerKey,
			CopySourceSSECustomerKeyMD5:    i.CopySourceSSECustomerKeyMD5,
			ExpectedBucketOwner:            i.ExpectedBucketOwner,
			ExpectedSourceBucketOwner:      i.ExpectedSourceBucketOwner,
			GrantFullControl:               i.GrantFullControl,
			GrantRead:                      i.GrantRead,
			GrantReadACP:                   i.GrantReadACP,
			GrantWriteACP:                  i.GrantWriteACP,
			ObjectLockLegalHoldStatus:      i.ObjectLockLegalHoldStatus,
			ObjectLockMode:                 i.ObjectLockMode,
			ObjectLockRetainUntilDate:      i.ObjectLockRetainUntilDate,
			RequestPayer:                   i.RequestPayer,
			SSECustomerAlgorithm:           i.SSECustomerAlgorithm,
			SSECustomerKey:                 i.SSECustomerKey,
			SSECustomerKeyMD5:              i.SSECustomerKeyMD5,
			SSEKMSEncryptionContext:        i.SSEKMSEncryptionContext,
			SSEKMSKeyId:                    i.SSEKMSKeyId,
			ServerSideEncryption:           i.ServerSideEncryption,
			WebsiteRedirectLocation:        i.WebsiteRedirectLocation,
		}
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}

		var ignored = option_check_list{}
		bbs.check_options_ignored(action, rid, location, &ignored)
	}

	var source, err25 = bbs.lookat_copy_source(rid, object, i.CopySource)
	if err25 != nil {
		return nil, err25
	}
	var source_entity, source_stat, err23 = bbs.check_object_exists(rid, source)
	if err23 != nil {
		return nil, err23
	}
	var source_etag, source_metainfo, err22 = bbs.fetch_object_etag(rid, source, source_entity)
	if err22 != nil {
		return nil, err22
	}

	var metainfo *Meta_info

	{
		metainfo = &Meta_info{}
		var headers = i.Metadata
		var tagging, err3 = bbs.parse_tags(rid, object, i.Tagging)
		if err3 != nil {
			return nil, err3
		}
		switch i.MetadataDirective {
		case "COPY":
			if source_metainfo != nil {
				metainfo.Headers = source_metainfo.Headers
			}
		case "REPLACE":
			metainfo.Headers = headers
		}
		switch i.TaggingDirective {
		case "COPY":
			if source_metainfo != nil {
				metainfo.Tags = source_metainfo.Tags
			}
		case "REPLACE":
			metainfo.Tags = tagging
		}

		var h = &Meta_info{
			CacheControl:       i.CacheControl,
			ContentDisposition: i.ContentDisposition,
			ContentEncoding:    i.ContentEncoding,
			ContentLanguage:    i.ContentLanguage,
			ContentType:        i.ContentType,
			Expires:            i.Expires,
		}
		metainfo = merge_metainfo_with_content_headers(metainfo, h)
	}

	var checksum = i.ChecksumAlgorithm
	if checksum == "" && source_metainfo != nil {
		checksum = source_metainfo.Checksum
	}

	// NOTE: Checking conditions on the source is not serialized.

	{
		var conditions = copy_conditions{
			some_match:      i.CopySourceIfMatch,
			none_match:      i.CopySourceIfNoneMatch,
			modified_after:  i.CopySourceIfModifiedSince,
			modified_before: i.CopySourceIfUnmodifiedSince,
		}
		var source_mtime = source_stat.ModTime()
		var source_size = source_stat.Size()
		var err7 = bbs.check_conditions(rid, source, source_etag,
			source_mtime, source_size, "read", conditions)
		if err7 != nil {
			return nil, err7
		}
	}

	var upload_id = ""
	var part int32 = 0
	var extent *[2]int64 = nil

	var scratch = bbs.make_scratch_object_name(object, suffix)

	var md5v, csum, err6a = bbs.copy_scratch(ctx, object, upload_id, part,
		source, source_entity, extent, metainfo, checksum)
	if err6a != nil {
		return nil, err6a
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(rid, object, scratch)
		}
	}()

	{
		var checks = copy_checks{}
		var err3 = bbs.compare_checksums(rid, object, scratch, checksum,
			md5v, csum, checks)
		if err3 != nil {
			return nil, err3
		}
	}

	// SERIALIZE-ACCESSES (in the copying routine).

	var etag, stat, _, err6 = bbs.copy_object(ctx, object, upload_id, part,
		source, source_entity, extent, metainfo, checksum, md5v, csum, &cleanup_needed)
	if err6 != nil {
		return nil, err6
	}

	var mtime = stat.ModTime()

	var csumset2 *types.Checksum = fill_checksum_union(checksum, csum)

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
		ChecksumCRC32:     csumset2.ChecksumCRC32,
		ChecksumCRC32C:    csumset2.ChecksumCRC32C,
		ChecksumCRC64NVME: csumset2.ChecksumCRC64NVME,
		ChecksumSHA1:      csumset2.ChecksumSHA1,
		ChecksumSHA256:    csumset2.ChecksumSHA256,
		ChecksumType:      csumset2.ChecksumType,
		ETag:              &etag,
		LastModified:      &mtime,
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

func (bbs *Bbs_server) CreateBucket(ctx context.Context, i *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, *Aws_s3_error) {
	var o = s3.CreateBucketOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}

		var ignored = option_check_list{
			ACL_bucket_canned:         i.ACL,
			CreateBucketConfiguration: i.CreateBucketConfiguration,
			ObjectOwnership:           i.ObjectOwnership,
		}
		bbs.check_options_ignored(action, rid, location, &ignored)
	}

	// Note serialization may not be necessary as mkdir() is atomic.

	// SERIALIZE-ACCESSES.

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
		// Note the error on an existing path is fs.PathError and not
		// fs.ErrExist.

		bbs.logger.Debug("os.Mkdir() for bucket failed",
			"rid", rid, "path", path, "error", err3)
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

func (bbs *Bbs_server) CreateMultipartUpload(ctx context.Context, i *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, *Aws_s3_error) {
	var o = s3.CreateMultipartUploadOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			StorageClass: i.StorageClass,
		}
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}

		var ignored = option_check_list{}
		bbs.check_options_ignored(action, rid, location, &ignored)
	}

	var metainfo *Meta_info

	{
		var headers = i.Metadata
		var tagging, err1 = bbs.parse_tags(rid, object, i.Tagging)
		if err1 != nil {
			return nil, err1
		}
		if headers != nil || tagging != nil {
			metainfo = &Meta_info{
				Entity_key: "",
				ETag:       "",
				Checksum:   "",
				Csum:       "",
				Headers:    headers,
				Tags:       tagging,
			}
		} else {
			metainfo = nil
		}
		var h = &Meta_info{
			CacheControl:       i.CacheControl,
			ContentDisposition: i.ContentDisposition,
			ContentEncoding:    i.ContentEncoding,
			ContentLanguage:    i.ContentLanguage,
			ContentType:        i.ContentType,
			Expires:            i.Expires,
		}
		metainfo = merge_metainfo_with_content_headers(metainfo, h)
	}

	var checksumtype types.ChecksumType
	var checksum types.ChecksumAlgorithm

	{
		var err1 = bbs.reject_composite_checksum(rid, object, i.ChecksumType)
		if err1 != nil {
			return nil, err1
		}
		checksumtype = types.ChecksumTypeFullObject
		checksum = i.ChecksumAlgorithm
		if checksum == "" {
			bbs.logger.Info("Fix checksum algorithm to CRC64NVME",
				"rid", rid)
			checksum = types.ChecksumAlgorithmCrc64nvme
		}
	}

	var uploadid string = bbs.make_new_upload_id()

	var now = time.Now()
	var mpul = &Mpul_info{
		//MultipartUpload: types.MultipartUpload{}
		Upload_id:     uploadid,
		Initiate_time: now,
		Checksum_type: checksumtype,
		Checksum:      checksum,
		Metainfo:      metainfo,
	}

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	{
		var err6 = bbs.create_mpul_directory(rid, object, suffix, mpul)
		if err6 != nil {
			return nil, err6
		}
		var cleanup_needed = true
		defer func() {
			if cleanup_needed {
				var _ = bbs.discard_mpul_directory(rid, object)
			}
		}()

		var catalog = &Mpul_catalog{
			// Empty catalog information.
		}
		var err7 = bbs.store_mpul_catalog(rid, object, suffix, catalog)
		if err7 != nil {
			return nil, err7
		}

		cleanup_needed = false
	}

	// This logging is printed in serialized region.

	if !bbs.config.Skip_trace_logging {
		bbs.logger.Log(context.Background(), LevelTrace,
			"Creating a multipart temporary",
			"rid", rid, "object", object, "mpul-info", mpul)
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

func (bbs *Bbs_server) DeleteBucket(ctx context.Context, i *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, *Aws_s3_error) {
	var o = s3.DeleteBucketOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

	// List of parameters.
	// i.Bucket *string
	// i.ExpectedBucketOwner *string

	var bucket, err2 = bbs.check_usual_bucket_setup(rid, i.Bucket)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + bucket

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
		}
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	// var rid uint64 = get_request_id(ctx)

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, bucket, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, bucket, rid)
	}

	var path = bbs.make_path_of_bucket(bucket)

	// Check some objects remain, when removing has failed.

	var err3 = os.Remove(path)
	if err3 != nil {
		bbs.logger.Info("os.Remove() for DeleteBucket failed",
			"rid", rid, "path", path, "error", err3)
		var err4 = bbs.check_bucket_empty(rid, path)
		if err4 != nil {
			return nil, err4
		}

		// Only files remain that start with a dot.  Remove them.

		bbs.logger.Info(("Try os.RemoveAll() after removing a bucket failed"),
			"rid", rid, "path", path)

		var err5 = os.RemoveAll(path)
		if err5 != nil {
			bbs.logger.Info("os.RemoveAll() for DeleteBucket failed",
				"rid", rid, "path", path, "error", err5)
			var errz = &Aws_s3_error{Code: BucketNotEmpty,
				Resource: location}
			return nil, errz
		}
	}

	return &o, nil
}

func (bbs *Bbs_server) DeleteObject(ctx context.Context, i *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, *Aws_s3_error) {
	var o = s3.DeleteObjectOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	//var location = "/" + object

	{
		var unsupported = option_check_list{
			BypassGovernanceRetention: i.BypassGovernanceRetention,
			ExpectedBucketOwner:       i.ExpectedBucketOwner,
			MFA:                       i.MFA,
			RequestPayer:              i.RequestPayer,
			VersionId:                 i.VersionId,
		}
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	// SERIALIZE-ACCESSES (in the deletion routine).

	var conditions = copy_conditions{
		some_match:    i.IfMatch,
		modified_time: i.IfMatchLastModifiedTime,
		size:          i.IfMatchSize,
	}
	var err6 = bbs.delete_object(ctx, object, conditions)
	if err6 != nil {
		return nil, err6
	}

	// o.DeleteMarker *bool
	// o.RequestCharged types.RequestCharged
	// o.VersionId *string

	return &o, nil
}

func (bbs *Bbs_server) DeleteObjects(ctx context.Context, i *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, *Aws_s3_error) {
	var o = s3.DeleteObjectsOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

	// i.Bucket *string
	// i.Delete *types.Delete
	// i.BypassGovernanceRetention *bool
	// i.ChecksumAlgorithm types.ChecksumAlgorithm
	// i.ExpectedBucketOwner *string
	// i.MFA *string
	// i.RequestPayer types.RequestPayer

	// Note i.ChecksumAlgorithm shall be ignored because it is
	// "x-amz-sdk-checksum-algorithm".

	var dummy = "dummy"
	var _, err2 = bbs.check_usual_object_setup(rid, i.Bucket, &dummy)
	if err2 != nil {
		return nil, err2
	}
	var bucket = *i.Bucket
	//var location = "/" + bucket

	{
		var unsupported = option_check_list{
			BypassGovernanceRetention: i.BypassGovernanceRetention,
			ExpectedBucketOwner:       i.ExpectedBucketOwner,
			MFA:                       i.MFA,
			RequestPayer:              i.RequestPayer,
		}
		var err1 = check_options_unsupported(bbs, action, &unsupported)
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

	// Delete objects.

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

			// SERIALIZE-ACCESSES (in the deletion routine).

			var conditions = copy_conditions{
				some_match:    bbs.fix_etag_quoting(e.ETag, rid),
				modified_time: e.LastModifiedTime,
				size:          e.Size,
			}
			var err6 = bbs.delete_object(ctx, object, conditions)
			if err6 != nil {
				deletestate[i].error.Code = &err6.Code
				deletestate[i].error.Message = &err6.Message
				continue loop1
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

func (bbs *Bbs_server) DeleteObjectTagging(ctx context.Context, i *s3.DeleteObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectTaggingOutput, *Aws_s3_error) {
	var o = s3.DeleteObjectTaggingOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

	// List of parameters.
	// i.Bucket *string
	// i.Key *string
	// i.ExpectedBucketOwner *string
	// i.VersionId *string

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	//var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
			VersionId:           i.VersionId,
		}
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	// var rid uint64 = get_request_id(ctx)

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	var entity, _, err3 = bbs.check_object_exists(rid, object)
	if err3 != nil {
		return nil, err3
	}
	var metainfo, err5 = bbs.fetch_object_metainfo(rid, object, entity)
	if err5 != nil {
		return nil, err5
	}

	// Modify metainfo, and remove the metainfo file when it become
	// nothing.

	if metainfo != nil && metainfo.Tags != nil {
		metainfo.Tags = nil
		//metainfo = metainfo_null_for_zero(metainfo)
		var err7 = bbs.store_object_metainfo(rid, object, suffix, metainfo)
		if err7 != nil {
			return nil, err7
		}
	}

	// o.VersionId *string

	return &o, nil
}

func (bbs *Bbs_server) GetObject(ctx context.Context, i *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, *Aws_s3_error) {
	var o = s3.GetObjectOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	//var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner:  i.ExpectedBucketOwner,
			RequestPayer:         i.RequestPayer,
			SSECustomerAlgorithm: i.SSECustomerAlgorithm,
			SSECustomerKey:       i.SSECustomerKey,
			SSECustomerKeyMD5:    i.SSECustomerKeyMD5,
			VersionId:            i.VersionId,
		}
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var entity, stat, err3 = bbs.check_object_exists(rid, object)
	if err3 != nil {
		return nil, err3
	}
	var etag, metainfo, err31 = bbs.fetch_object_etag(rid, object, entity)
	if err31 != nil {
		return nil, err31
	}
	var mtime = stat.ModTime()
	var size = stat.Size()
	var extent, err4 = scan_range(object, i.Range, size)
	if err4 != nil {
		return nil, err4
	}

	var _, err15 = bbs.lookat_part_number(object, i.PartNumber, false)
	if err15 != nil {
		return nil, err15
	}

	// Store an ETag in a metainfo file when the object is large.
	// Storing metainfo serializes accesses inside the routine.

	if metainfo == nil && size >= byte_size(bbs.config.Etag_save_threshold) {
		var err6 = bbs.store_etag_as_metainfo(ctx, object, suffix, entity, etag)
		if err6 != nil {
			return nil, err6
		}
	}

	// NO SERIALIZE-ACCESSES.

	var err7 = bbs.check_conditions(rid, object, etag,
		mtime, size, "read",
		copy_conditions{
			some_match:      i.IfMatch,
			none_match:      i.IfNoneMatch,
			modified_after:  i.IfModifiedSince,
			modified_before: i.IfUnmodifiedSince,
		})
	if err7 != nil {
		return nil, err7
	}

	var csum []byte
	if i.ChecksumMode == types.ChecksumModeEnabled {
		var checksum = types.ChecksumAlgorithmCrc64nvme
		var _, crc1, _, err8 = bbs.calculate_csum2(rid, object, checksum, object, extent)
		if err8 != nil {
			return nil, err8
		}
		csum = crc1
	}

	{
		var f1, err9 = bbs.make_file_stream(rid, object, extent, entity)
		if err9 != nil {
			return nil, err9
		}
		o.Body = f1
	}

	if extent != nil {
		var length = extent[1] - extent[0]
		o.ContentLength = &length
		var subrange = fmt.Sprintf("bytes %d-%d/%d",
			extent[0], (extent[1] - 1), size)
		o.ContentRange = &subrange
	} else {
		o.ContentLength = &size
	}

	o.LastModified = &mtime
	o.ETag = &etag

	if i.ChecksumMode == types.ChecksumModeEnabled {
		var checksum2 = types.ChecksumAlgorithmCrc64nvme
		var csumset *types.Checksum = fill_checksum_union(checksum2, csum)
		o.ChecksumType = csumset.ChecksumType
		o.ChecksumCRC32 = csumset.ChecksumCRC32
		o.ChecksumCRC32C = csumset.ChecksumCRC32C
		o.ChecksumCRC64NVME = csumset.ChecksumCRC64NVME
		o.ChecksumSHA1 = csumset.ChecksumSHA1
		o.ChecksumSHA256 = csumset.ChecksumSHA256
	}

	if metainfo != nil && metainfo.Headers != nil {
		// Always leave "MissingMeta" nil for zero.
		o.Metadata = metainfo.Headers
		o.MissingMeta = nil
	}
	if metainfo != nil && metainfo.Tags != nil {
		var count = int32(len(metainfo.Tags.TagSet))
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
			var expires = i.ResponseExpires.Format(time.RFC1123)
			o.ExpiresString = &expires
		}
	}

	return &o, nil
}

func (bbs *Bbs_server) GetObjectAttributes(ctx context.Context, i *s3.GetObjectAttributesInput, optFns ...func(*s3.Options)) (*s3.GetObjectAttributesOutput, *Aws_s3_error) {
	var o = s3.GetObjectAttributesOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
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
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}

		var ignored = option_check_list{
			MaxParts:         i.MaxParts,
			PartNumberMarker: i.PartNumberMarker,
		}
		bbs.check_options_ignored(action, rid, location, &ignored)
	}

	var entity, stat, err3 = bbs.check_object_exists(rid, object)
	if err3 != nil {
		return nil, err3
	}

	// NO SERIALIZE-ACCESSES.

	var attributes = i.ObjectAttributes
	if slices.Contains(attributes, types.ObjectAttributesEtag) {
		var etag, _, err31 = bbs.fetch_object_etag(rid, object, entity)
		if err31 != nil {
			return nil, err31
		}
		o.ETag = &etag
	}
	if slices.Contains(attributes, types.ObjectAttributesChecksum) {
		var checksum = types.ChecksumAlgorithmCrc64nvme
		var _, crc1, _, err8 = bbs.calculate_csum2(rid, object, checksum, object, nil)
		if err8 != nil {
			return nil, err8
		}
		var csumset *types.Checksum = fill_checksum_union(checksum, crc1)
		o.Checksum = csumset
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

	// o.DeleteMarker *bool
	// o.ObjectParts *types.GetObjectAttributesParts
	// o.RequestCharged types.RequestCharged
	// o.VersionId *string

	return &o, nil
}

func (bbs *Bbs_server) GetObjectTagging(ctx context.Context, i *s3.GetObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.GetObjectTaggingOutput, *Aws_s3_error) {
	var o = s3.GetObjectTaggingOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

	// List of parameters.
	// i.Bucket *string
	// i.Key *string
	// i.ExpectedBucketOwner *string
	// i.RequestPayer types.RequestPayer
	// i.VersionId *string

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
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
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var entity, _, err3 = bbs.check_object_exists(rid, object)
	if err3 != nil {
		return nil, err3
	}
	var metainfo, err5 = bbs.fetch_object_metainfo(rid, object, entity)
	if err5 != nil {
		return nil, err5
	}

	// NO SERIALIZE-ACCESSES.

	if metainfo != nil && metainfo.Tags != nil {
		o.TagSet = metainfo.Tags.TagSet
	}

	// o.VersionId *string

	return &o, nil
}

func (bbs *Bbs_server) HeadBucket(ctx context.Context, i *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, *Aws_s3_error) {
	var o = s3.HeadBucketOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

	// List of parameters.
	// i.Bucket *string
	// i.ExpectedBucketOwner *string

	var _, err2 = bbs.check_usual_bucket_setup(rid, i.Bucket)
	if err2 != nil {
		return nil, err2
	}

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner: i.ExpectedBucketOwner,
		}
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	// NO SERIALIZE-ACCESSES.

	// o.AccessPointAlias *bool
	// o.BucketArn *string
	// o.BucketLocationName *string
	// o.BucketLocationType types.LocationType
	// o.BucketRegion *string

	return &o, nil
}

func (bbs *Bbs_server) HeadObject(ctx context.Context, i *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, *Aws_s3_error) {
	var o = s3.HeadObjectOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
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
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var entity, stat, err3 = bbs.check_object_exists(rid, object)
	if err3 != nil {
		return nil, err3
	}
	var etag, metainfo, err5 = bbs.fetch_object_etag(rid, object, entity)
	if err5 != nil {
		return nil, err5
	}
	var mtime = stat.ModTime()
	var size = stat.Size()
	var extent, err4 = scan_range(object, i.Range, size)
	if err4 != nil {
		return nil, err4
	}

	var _, err15 = bbs.lookat_part_number(object, i.PartNumber, false)
	if err15 != nil {
		return nil, err15
	}

	// Store an ETag in metainfo when the object is large.  Storing
	// metainfo serializes accesses inside the routine.

	if metainfo == nil && size >= byte_size(bbs.config.Etag_save_threshold) {
		var err6 = bbs.store_etag_as_metainfo(ctx, object, suffix, entity, etag)
		if err6 != nil {
			return nil, err6
		}
	}

	// NO SERIALIZE-ACCESSES.

	var err7 = bbs.check_conditions(rid, object, etag,
		mtime, size, "read",
		copy_conditions{
			some_match:      i.IfMatch,
			none_match:      i.IfNoneMatch,
			modified_after:  i.IfModifiedSince,
			modified_before: i.IfUnmodifiedSince,
		})
	if err7 != nil {
		return nil, err7
	}

	if i.ChecksumMode == types.ChecksumModeEnabled && metainfo != nil {
		var checksum = metainfo.Checksum
		var crc1, err1 = hex.DecodeString(metainfo.Csum)
		if err1 != nil {
			bbs.logger.Info("hex.DecodeString() on metainfo checksum failed",
				"rid", rid, "error", err1)
			var errz = &Aws_s3_error{
				Code:     InvalidObjectState,
				Message:  "Metainfo file broken.",
				Resource: location}
			return nil, errz
		}
		var csumset *types.Checksum = fill_checksum_union(checksum, crc1)

		o.ChecksumType = csumset.ChecksumType
		o.ChecksumCRC32 = csumset.ChecksumCRC32
		o.ChecksumCRC32C = csumset.ChecksumCRC32C
		o.ChecksumCRC64NVME = csumset.ChecksumCRC64NVME
		o.ChecksumSHA1 = csumset.ChecksumSHA1
		o.ChecksumSHA256 = csumset.ChecksumSHA256
	}

	if extent != nil {
		var length = extent[1] - extent[0]
		o.ContentLength = &length
		var subrange = fmt.Sprintf("bytes %d-%d/%d",
			extent[0], (extent[1] - 1), size)
		o.ContentRange = &subrange
	} else {
		o.ContentLength = &size
	}
	var one int32 = 1
	o.PartsCount = &one

	o.ETag = &etag
	o.LastModified = &mtime

	if metainfo != nil {
		// Always leave "MissingMeta" nil for zero.
		o.Metadata = metainfo.Headers
		o.MissingMeta = nil
		if metainfo.Tags != nil {
			var count = int32(len(metainfo.Tags.TagSet))
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
			var expires = i.ResponseExpires.Format(time.RFC1123)
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

func (bbs *Bbs_server) ListBuckets(ctx context.Context, i *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, *Aws_s3_error) {
	var o = s3.ListBucketsOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

	// List of parameters.
	// i.BucketRegion *string
	// i.ContinuationToken *string
	// i.MaxBuckets *int32
	// i.Prefix *string

	{
		var unsupported = option_check_list{}
		var err1 = check_options_unsupported(bbs, action, &unsupported)
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

	// NO SERIALIZE-ACCESSES.

	var buckets, continuation, err3 = bbs.list_buckets(rid, start, max_buckets,
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

func (bbs *Bbs_server) ListMultipartUploads(ctx context.Context, i *s3.ListMultipartUploadsInput, optFns ...func(*s3.Options)) (*s3.ListMultipartUploadsOutput, *Aws_s3_error) {
	var o = s3.ListMultipartUploadsOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	var bucket, err2 = bbs.check_usual_bucket_setup(rid, i.Bucket)
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
		var err1 = check_options_unsupported(bbs, action, &unsupported)
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

	// NO SERIALIZE-ACCESSES.

	var objects, commons, nextmarker, err5 = bbs.list_mpuls_flat(
		rid, bucket, marker, maxkeys, delimiter, prefix, urlencode)
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

func (bbs *Bbs_server) ListObjects(ctx context.Context, i *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, *Aws_s3_error) {
	var o = s3.ListObjectsOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	var bucket, err2 = bbs.check_usual_bucket_setup(rid, i.Bucket)
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
		var err1 = check_options_unsupported(bbs, action, &unsupported)
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

	// NO SERIALIZE-ACCESSES.

	var entries []object_list_entry
	var nextindex int
	var nextmarker string
	var err3 *Aws_s3_error
	if !always_use_flat_lister && delimiter == "/" {
		entries, nextindex, nextmarker, err3 = bbs.list_objects_delimited(
			rid, bucket, index, marker, maxkeys, delimiter, prefix)
	} else {
		entries, nextindex, nextmarker, err3 = bbs.list_objects_flat(
			rid, bucket, index, marker, maxkeys, delimiter, prefix)
	}
	if err3 != nil {
		return nil, err3
	}

	var contents, commonprefixes, err4 = bbs.make_list_objects_entries(
		rid, entries, bucket, delimiter, prefix, urlencode)
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

func (bbs *Bbs_server) ListObjectsV2(ctx context.Context, i *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, *Aws_s3_error) {
	var o = s3.ListObjectsV2Output{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	var bucket, err2 = bbs.check_usual_bucket_setup(rid, i.Bucket)
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
		var err1 = check_options_unsupported(bbs, action, &unsupported)
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

	// NO SERIALIZE-ACCESSES.

	var entries []object_list_entry
	var nextindex int
	var nextmarker string
	var err5 *Aws_s3_error
	if !always_use_flat_lister && delimiter == "/" {
		entries, nextindex, nextmarker, err5 = bbs.list_objects_delimited(
			rid, bucket, index, marker, maxkeys, delimiter, prefix)
	} else {
		entries, nextindex, nextmarker, err5 = bbs.list_objects_flat(
			rid, bucket, index, marker, maxkeys, delimiter, prefix)
	}
	if err5 != nil {
		return nil, err5
	}
	var _ = nextmarker

	var contents, commonprefixes, err6 = bbs.make_list_objects_entries(
		rid, entries, bucket, delimiter, prefix, urlencode)
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

func (bbs *Bbs_server) ListParts(ctx context.Context, i *s3.ListPartsInput, optFns ...func(*s3.Options)) (*s3.ListPartsOutput, *Aws_s3_error) {
	var o = s3.ListPartsOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
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
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var mpulinfo, err3 = bbs.check_mpul_ongoing(rid, object, i.UploadId, false)
	if err3 != nil {
		return nil, err3
	}

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

	var catalog, err4 = bbs.fetch_mpul_catalog(rid, object)
	if err4 != nil {
		return nil, err4
	}

	// NO SERIALIZE-ACCESSES.

	// Copy MPUL catalog to a result record.

	var partlist []types.Part
	var checksum = mpulinfo.Checksum
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
		var csum, err5 = base64.StdEncoding.DecodeString(e.Csum)
		if err5 != nil {
			// IGNORE-ERRORS.
		}
		var csumset *types.Checksum = fill_checksum_union(checksum, csum)
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
			ChecksumCRC32:     csumset.ChecksumCRC32,
			ChecksumCRC32C:    csumset.ChecksumCRC32C,
			ChecksumCRC64NVME: csumset.ChecksumCRC64NVME,
			ChecksumSHA1:      csumset.ChecksumSHA1,
			ChecksumSHA256:    csumset.ChecksumSHA256,
			ETag:              &e.ETag,
			LastModified:      &e.Mtime,
			PartNumber:        &no,
			Size:              &e.Size,
		}
		partlist = append(partlist, p)
	}

	{
		o.Bucket = i.Bucket
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
	// o.Initiator *types.Initiator
	// o.Owner *types.Owner
	// o.RequestCharged types.RequestCharged

	return &o, nil
}

func (bbs *Bbs_server) PutObject(ctx context.Context, i *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, *Aws_s3_error) {
	var o = s3.PutObjectOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	// Note i.ChecksumAlgorithm shall be ignored because it is
	// "x-amz-sdk-checksum-algorithm".

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	var location = "/" + object

	{
		var unsupported = option_check_list{
			ACL_object_canned:   i.ACL,
			BucketKeyEnabled:    i.BucketKeyEnabled,
			ExpectedBucketOwner: i.ExpectedBucketOwner,
			//Expires: i.Expires,
			ObjectLockLegalHoldStatus: i.ObjectLockLegalHoldStatus,
			ObjectLockMode:            i.ObjectLockMode,
			ObjectLockRetainUntilDate: i.ObjectLockRetainUntilDate,
			RequestPayer:              i.RequestPayer,
			SSECustomerAlgorithm:      i.SSECustomerAlgorithm,
			SSECustomerKey:            i.SSECustomerKey,
			SSECustomerKeyMD5:         i.SSECustomerKeyMD5,
			SSEKMSEncryptionContext:   i.SSEKMSEncryptionContext,
			SSEKMSKeyId:               i.SSEKMSKeyId,
			ServerSideEncryption:      i.ServerSideEncryption,
			WebsiteRedirectLocation:   i.WebsiteRedirectLocation,
			WriteOffsetBytes:          i.WriteOffsetBytes,
		}
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}

		var ignored = option_check_list{
			CacheControl: i.CacheControl,
			// i.GrantFullControl *string
			// i.GrantRead *string
			// i.GrantReadACP *string
			// i.GrantWriteACP *string
			StorageClass: i.StorageClass,
		}
		bbs.check_options_ignored(action, rid, location, &ignored)
	}

	var size_to_check int64
	if i.ContentLength != nil {
		size_to_check = *i.ContentLength
	} else {
		size_to_check = -1
	}

	var md5_to_check, err7 = decode_base64(object, i.ContentMD5)
	if err7 != nil {
		return nil, err7
	}

	var csumset = &types.Checksum{
		ChecksumType:      types.ChecksumTypeFullObject,
		ChecksumCRC32:     i.ChecksumCRC32,
		ChecksumCRC32C:    i.ChecksumCRC32C,
		ChecksumCRC64NVME: i.ChecksumCRC64NVME,
		ChecksumSHA1:      i.ChecksumSHA1,
		ChecksumSHA256:    i.ChecksumSHA256,
	}

	var trailer_checksum types.ChecksumAlgorithm
	var checksum types.ChecksumAlgorithm
	var csum_to_check []byte

	{
		// Handle the case a checksum is in the trailer.

		var checksum1, err1 = bbs.check_trailer_checksum(ctx, rid, object)
		if err1 != nil {
			return nil, err1
		}
		trailer_checksum = checksum1
		if trailer_checksum != "" {
			checksum = trailer_checksum
			csum_to_check = nil
		} else {
			var checksum2, csum2, err2 = bbs.decode_checksum_union(rid, object, csumset)
			if err2 != nil {
				return nil, err2
			}
			checksum = checksum2
			csum_to_check = csum2
		}
	}

	var metainfo *Meta_info

	{
		var csum = hex.EncodeToString(csum_to_check)
		var headers = i.Metadata
		var tagging, err1 = bbs.parse_tags(rid, object, i.Tagging)
		if err1 != nil {
			return nil, err1
		}
		metainfo = &Meta_info{
			Entity_key: "",
			ETag:       "",
			Checksum:   checksum,
			Csum:       csum,
			Headers:    headers,
			Tags:       tagging,
		}
		var h = &Meta_info{
			CacheControl:       i.CacheControl,
			ContentDisposition: i.ContentDisposition,
			ContentEncoding:    i.ContentEncoding,
			ContentLanguage:    i.ContentLanguage,
			ContentType:        i.ContentType,
			Expires:            i.Expires,
		}
		metainfo = merge_metainfo_with_content_headers(metainfo, h)
	}

	var part int32 = 0
	var upload_id = ""
	var checks = copy_checks{
		checksum:        checksum,
		csum_in_trailer: (trailer_checksum != ""),
		size_to_check:   size_to_check,
		md5_to_check:    md5_to_check,
		csum_to_check:   csum_to_check,
	}
	var conditions = copy_conditions{
		some_match: i.IfMatch,
		none_match: i.IfNoneMatch,
	}

	var scratch = bbs.make_scratch_object_name(object, suffix)

	var md5v, csum, err6a = bbs.upload_scratch(ctx, object,
		upload_id, part, i.Body, metainfo, checks, conditions)
	if err6a != nil {
		return nil, err6a
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(rid, object, scratch)
		}
	}()

	if trailer_checksum != "" {
		// Handle the case a checksum is in the trailer.

		var err9 = bbs.refill_put_object_input_by_trailer(ctx, rid, object,
			i, trailer_checksum)
		if err9 != nil {
			return nil, err9
		}

		// UPDATE CSUMSET TO RETURN AS A RESPONSE.

		csumset = &types.Checksum{
			ChecksumType:      types.ChecksumTypeFullObject,
			ChecksumCRC32:     i.ChecksumCRC32,
			ChecksumCRC32C:    i.ChecksumCRC32C,
			ChecksumCRC64NVME: i.ChecksumCRC64NVME,
			ChecksumSHA1:      i.ChecksumSHA1,
			ChecksumSHA256:    i.ChecksumSHA256,
		}
		var checksum2, csum_to_check2, err2 = bbs.decode_checksum_union(rid, object, csumset)
		if err2 != nil {
			return nil, err2
		}
		if checksum2 != checksum || csum_to_check2 == nil {
			bbs.logger.Warn("Trailer checksum missing",
				"rid", rid, "action", action, "object", object)
		}
		checks.csum_to_check = csum_to_check2

		if metainfo != nil && metainfo.Checksum != "" {
			var csum = hex.EncodeToString(csum_to_check2)
			//var csum, _ = encode_base64(object, csum1)
			metainfo.Csum = csum
		}
	}

	{
		var err3 = bbs.compare_checksums(rid, object, scratch, checksum,
			md5v, csum, checks)
		if err3 != nil {
			return nil, err3
		}
	}

	// SERIALIZE-ACCESSES (in the uploading routine).

	var etag, stat, err6 = bbs.upload_object(ctx, object,
		upload_id, part, i.Body, md5v, csum, metainfo, checks, conditions, &cleanup_needed)
	if err6 != nil {
		return nil, err6
	}

	var size = stat.Size()
	o.ETag = &etag
	o.Size = &size

	{
		//var checksum2 = types.ChecksumAlgorithmCrc64nvme
		//var csumset2 *types.Checksum = fill_checksum_union(checksum2, csum2)
		o.ChecksumType = csumset.ChecksumType
		o.ChecksumCRC32 = csumset.ChecksumCRC32
		o.ChecksumCRC32C = csumset.ChecksumCRC32C
		o.ChecksumCRC64NVME = csumset.ChecksumCRC64NVME
		o.ChecksumSHA1 = csumset.ChecksumSHA1
		o.ChecksumSHA256 = csumset.ChecksumSHA256
	}

	// o.BucketKeyEnabled *bool
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

func (bbs *Bbs_server) PutObjectTagging(ctx context.Context, i *s3.PutObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.PutObjectTaggingOutput, *Aws_s3_error) {
	var o = s3.PutObjectTaggingOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	// Note i.ChecksumAlgorithm shall be ignored because it is
	// "x-amz-sdk-checksum-algorithm".

	// i.ContentMD5 is implicitly checked in unmarshaling body.

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
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
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	//bbs.logger.Debug("Tagging", "action", action, "tagging", i.Tagging)

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	{
		var entity, _, err3 = bbs.check_object_exists(rid, object)
		if err3 != nil {
			return nil, err3
		}
		var etag, metainfo, err2 = bbs.fetch_object_etag(rid, object, entity)
		if err2 != nil {
			return nil, err2
		}

		if metainfo == nil {
			metainfo = &Meta_info{
				Entity_key: entity,
				ETag:       etag,
				Checksum:   "",
				Csum:       "",
				Headers:    nil,
				Tags:       nil,
			}
		}
		metainfo.Tags = i.Tagging
		var err7 = bbs.store_object_metainfo(rid, object, suffix, metainfo)
		if err7 != nil {
			return nil, err7
		}
	}

	// o.VersionId *string

	return &o, nil
}

func (bbs *Bbs_server) UploadPart(ctx context.Context, i *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, *Aws_s3_error) {
	var o = s3.UploadPartOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	// Note i.ChecksumAlgorithm shall be ignored because it is
	// "x-amz-sdk-checksum-algorithm".

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	//var location = "/" + object

	{
		var unsupported = option_check_list{
			ExpectedBucketOwner:  i.ExpectedBucketOwner,
			RequestPayer:         i.RequestPayer,
			SSECustomerAlgorithm: i.SSECustomerAlgorithm,
			SSECustomerKey:       i.SSECustomerKey,
			SSECustomerKeyMD5:    i.SSECustomerKeyMD5,
		}
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var mpulinfo, err3 = bbs.check_mpul_ongoing(rid, object, i.UploadId, false)
	if err3 != nil {
		return nil, err3
	}
	var part, err4 = bbs.lookat_part_number(object, i.PartNumber, true)
	if err4 != nil {
		return nil, err4
	}

	var size_to_check int64
	if i.ContentLength != nil {
		size_to_check = *i.ContentLength
	} else {
		size_to_check = -1
	}

	var md5_to_check, err7 = decode_base64(object, i.ContentMD5)
	if err7 != nil {
		return nil, err7
	}

	var csumset1 = &types.Checksum{
		ChecksumType:      types.ChecksumTypeFullObject,
		ChecksumCRC32:     i.ChecksumCRC32,
		ChecksumCRC32C:    i.ChecksumCRC32C,
		ChecksumCRC64NVME: i.ChecksumCRC64NVME,
		ChecksumSHA1:      i.ChecksumSHA1,
		ChecksumSHA256:    i.ChecksumSHA256,
	}

	var trailer_checksum types.ChecksumAlgorithm
	var checksum types.ChecksumAlgorithm
	var csum_to_check []byte

	{
		// Handle the case a checksum is in the trailer.

		var checksum1, err1 = bbs.check_trailer_checksum(ctx, rid, object)
		if err1 != nil {
			return nil, err1
		}
		trailer_checksum = checksum1
		if checksum1 != "" {
			checksum = checksum1
			csum_to_check = nil
		} else {
			var checksum2, csum2, err2 = bbs.decode_checksum_union(rid, object, csumset1)
			if err2 != nil {
				return nil, err2
			}
			checksum = checksum2
			csum_to_check = csum2
		}
	}

	var upload_id = mpulinfo.Upload_id

	var metainfo *Meta_info = nil
	var checks = copy_checks{
		checksum:        checksum,
		csum_in_trailer: (trailer_checksum != ""),
		size_to_check:   size_to_check,
		md5_to_check:    md5_to_check,
		csum_to_check:   csum_to_check,
	}
	var conditions = copy_conditions{}

	var scratch = bbs.make_scratch_object_name(object, suffix)

	var md5v, csum, err6a = bbs.upload_scratch(ctx, object,
		upload_id, part, i.Body, metainfo, checks, conditions)
	if err6a != nil {
		return nil, err6a
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(rid, object, scratch)
		}
	}()

	if trailer_checksum != "" {
		// Handle the case a checksum is in the trailer.

		var err9 = bbs.refill_upload_part_input_by_trailer(ctx, rid, object,
			i, trailer_checksum)
		if err9 != nil {
			return nil, err9
		}

		var csumset2 = &types.Checksum{
			ChecksumType:      types.ChecksumTypeFullObject,
			ChecksumCRC32:     i.ChecksumCRC32,
			ChecksumCRC32C:    i.ChecksumCRC32C,
			ChecksumCRC64NVME: i.ChecksumCRC64NVME,
			ChecksumSHA1:      i.ChecksumSHA1,
			ChecksumSHA256:    i.ChecksumSHA256,
		}
		var checksum2, csum_to_check2, err2 = bbs.decode_checksum_union(rid, object, csumset2)
		if err2 != nil {
			return nil, err2
		}
		if checksum2 != checksum || csum_to_check2 == nil {
			bbs.logger.Warn("Trailer checksum missing",
				"rid", rid, "action", action, "object", object)
		}
		checks.csum_to_check = csum_to_check2

		var err3 = bbs.compare_checksums(rid, object, scratch, checksum,
			md5v, csum, checks)
		if err3 != nil {
			return nil, err3
		}

		if metainfo != nil && metainfo.Checksum != "" {
			var csum = hex.EncodeToString(csum_to_check2)
			//var csum, _ = encode_base64(object, csum1)
			metainfo.Csum = csum
		}
	}

	// SERIALIZE-ACCESSES (in the uploading routine).

	var etag, _, err6 = bbs.upload_object(ctx, object,
		upload_id, part, i.Body, md5v, csum, metainfo, checks, conditions, &cleanup_needed)
	if err6 != nil {
		return nil, err6
	}

	o.ETag = &etag

	{
		// var csumset2 *types.Checksum = fill_checksum_union(checksum, csum2)
		// No o.ChecksumType in the output record.
		// o.ChecksumCRC32 = csumset2.ChecksumCRC32
		// o.ChecksumCRC32C = csumset2.ChecksumCRC32C
		// o.ChecksumCRC64NVME = csumset2.ChecksumCRC64NVME
		// o.ChecksumSHA1 = csumset2.ChecksumSHA1
		// o.ChecksumSHA256 = csumset2.ChecksumSHA256
		o.ChecksumCRC32 = i.ChecksumCRC32
		o.ChecksumCRC32C = i.ChecksumCRC32C
		o.ChecksumCRC64NVME = i.ChecksumCRC64NVME
		o.ChecksumSHA1 = i.ChecksumSHA1
		o.ChecksumSHA256 = i.ChecksumSHA256
	}

	// o.BucketKeyEnabled *bool
	// o.RequestCharged types.RequestCharged
	// o.SSECustomerAlgorithm *string
	// o.SSECustomerKeyMD5 *string
	// o.SSEKMSKeyId *string
	// o.ServerSideEncryption types.ServerSideEncryption

	return &o, nil
}

func (bbs *Bbs_server) UploadPartCopy(ctx context.Context, i *s3.UploadPartCopyInput, optFns ...func(*s3.Options)) (*s3.UploadPartCopyOutput, *Aws_s3_error) {
	var o = s3.UploadPartCopyOutput{}
	var action, rid, suffix = get_action_name(ctx)
	bbs.logger.Info("Serving", "action", action, "rid", rid, "suffix", suffix)

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

	var object, err2 = bbs.check_usual_object_setup(rid, i.Bucket, i.Key)
	if err2 != nil {
		return nil, err2
	}
	//var location = "/" + object

	{
		var unsupported = option_check_list{
			CopySourceSSECustomerAlgorithm: i.CopySourceSSECustomerAlgorithm,
			CopySourceSSECustomerKey:       i.CopySourceSSECustomerKey,
			CopySourceSSECustomerKeyMD5:    i.CopySourceSSECustomerKeyMD5,
			ExpectedBucketOwner:            i.ExpectedBucketOwner,
			ExpectedSourceBucketOwner:      i.ExpectedSourceBucketOwner,
			RequestPayer:                   i.RequestPayer,
			SSECustomerAlgorithm:           i.SSECustomerAlgorithm,
			SSECustomerKey:                 i.SSECustomerKey,
			SSECustomerKeyMD5:              i.SSECustomerKeyMD5,
		}
		var err1 = check_options_unsupported(bbs, action, &unsupported)
		if err1 != nil {
			return nil, err1
		}
	}

	var mpulinfo, err3 = bbs.check_mpul_ongoing(rid, object, i.UploadId, false)
	if err3 != nil {
		return nil, err3
	}
	var part, err4 = bbs.lookat_part_number(object, i.PartNumber, true)
	if err4 != nil {
		return nil, err4
	}

	var source, err25 = bbs.lookat_copy_source(rid, object, i.CopySource)
	if err25 != nil {
		return nil, err25
	}
	var source_entity, source_stat, err23 = bbs.check_object_exists(rid, source)
	if err23 != nil {
		return nil, err23
	}
	var source_etag, _, err21 = bbs.fetch_object_etag(rid, source, source_entity)
	if err21 != nil {
		return nil, err21
	}

	// NOTE: Checking conditions on the source is not serialized.

	{
		var source_mtime = source_stat.ModTime()
		var source_size = source_stat.Size()
		var err7 = bbs.check_conditions(rid, source, source_etag,
			source_mtime, source_size, "read",
			copy_conditions{
				some_match:      i.CopySourceIfMatch,
				none_match:      i.CopySourceIfNoneMatch,
				modified_after:  i.CopySourceIfModifiedSince,
				modified_before: i.CopySourceIfUnmodifiedSince,
			})
		if err7 != nil {
			return nil, err7
		}
	}

	var size = source_stat.Size()
	var extent, err24 = scan_range(object, i.CopySourceRange, size)
	if err24 != nil {
		return nil, err24
	}

	var checksum2 types.ChecksumAlgorithm = mpulinfo.Checksum

	var upload_id = mpulinfo.Upload_id

	var metainfo *Meta_info = nil

	var scratch = bbs.make_scratch_object_name(object, suffix)

	var md5v, csum, err6a = bbs.copy_scratch(ctx, object, upload_id, part,
		source, source_entity, extent, metainfo, checksum2)
	if err6a != nil {
		return nil, err6a
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(rid, object, scratch)
		}
	}()

	/*
		{
			var err3 = bbs.compare_checksums(rid, object, scratch, checksum,
				md5v, csum1, checks)
			if err3 != nil {
				return nil, nil, err3
			}
		}
	*/

	// SERIALIZE-ACCESSES (in the copying routine)

	var etag, stat, _, err6 = bbs.copy_object(ctx, object, upload_id, part,
		source, source_entity, extent, metainfo, checksum2, md5v, csum, &cleanup_needed)
	if err6 != nil {
		return nil, err6
	}

	var mtime = stat.ModTime()

	var csumset2 *types.Checksum = fill_checksum_union(checksum2, csum)

	o.CopyPartResult = &types.CopyPartResult{
		// - ChecksumCRC32 *string
		// - ChecksumCRC32C *string
		// - ChecksumCRC64NVME *string
		// - ChecksumSHA1 *string
		// - ChecksumSHA256 *string
		// - ETag *string
		// - LastModified *time.Time
		ChecksumCRC32:     csumset2.ChecksumCRC32,
		ChecksumCRC32C:    csumset2.ChecksumCRC32C,
		ChecksumCRC64NVME: csumset2.ChecksumCRC64NVME,
		ChecksumSHA1:      csumset2.ChecksumSHA1,
		ChecksumSHA256:    csumset2.ChecksumSHA256,
		ETag:              &etag,
		LastModified:      &mtime,
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
