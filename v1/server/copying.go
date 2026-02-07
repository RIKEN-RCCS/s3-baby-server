// copying.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Uploading and copying.  This is the main part of
// {CompleteMultipartUpload, CopyObject, PutObject, UploadPart,
// UploadPartCopy}.

// MEMO: Note io.MultiWriter is only a io.Writer, not io.Closer.

package server

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"hash"
	"io"
	"io/fs"
	"log"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const generate_md5_on_copy = true

type copy_checks struct {
	size_to_check int64
	checksum      types.ChecksumAlgorithm
	md5_to_check  []byte
	csum_to_check []byte
	//csum_ types.Checksum
}

type copy_conditionals struct {
	some_match      *string
	none_match      *string
	modified_after  *time.Time
	modified_before *time.Time
	modified_time   *time.Time
	size            *int64
}

type build_op int

const (
	BUILD_UPLOAD build_op = iota
	BUILD_COPY
	BUILD_LINK
	BUILD_CONCAT
)

// BUILD_SOURCE is an argument to copying or uploading.  (It actually
// be a sum type).  STREAM and LENGTH are effective on uploading.
// SOURCE, SOURCE_ENTITY, EXTENT are effective on copying/linking.
// PARTLIST and MPUL are are effective on concat.

type build_upload struct {
	stream io.Reader
	length int64
}

type build_copy struct {
	source        string
	source_entity string
	extent        *[2]int64
}

type build_concat struct {
	partlist *types.CompletedMultipartUpload
	mpul     *Mpul_info
}

type build_source struct {
	op     build_op
	upload build_upload
	copy   build_copy
	concat build_concat
}

// UPLOAD_OBJECT performs uploading.  Uploading is either for an
// object or an MPUL part file.  It returns stat, etag, and csum (of
// CRC64NVME).  Metainfo is only partially filled.
func (bbs *Bb_server) upload_object(ctx context.Context, object string, upload_id string, part int32, body io.Reader, metainfo *Meta_info, checks copy_checks, conditionals copy_conditionals) (fs.FileInfo, string, []byte, *Aws_s3_error) {
	var build = build_source{
		op: BUILD_UPLOAD,
		upload: build_upload{
			stream: body,
			length: checks.size_to_check,
		},
		copy: build_copy{
			source:        "",
			source_entity: "",
			extent:        nil,
		},
		concat: build_concat{
			partlist: nil,
			mpul:     nil,
		},
	}
	var checksum2 types.ChecksumAlgorithm = checks.checksum
	var stat, etag, csum2, err1 = bbs.build_object(ctx, object, upload_id,
		part, build, metainfo, checksum2, checks, conditionals)
	return stat, etag, csum2, err1
}

// COPY_OBJECT performs copying.  A copying target is either an object
// or an MPUL part file.  It returns stat and csum.  A checksum value
// is by the algorithm of CHECKSUM when copying is for MPUL.  Note
// checksum checks are not applied on copying.  Conditionals on the
// source object is checked by the caller.  Metainfo is only partially
// filled.
func (bbs *Bb_server) copy_object(ctx context.Context, object string, upload_id string, part int32, source string, source_entity string, extent *[2]int64, metainfo *Meta_info, checksum2 types.ChecksumAlgorithm) (fs.FileInfo, string, []byte, *Aws_s3_error) {
	var copy_or_link build_op
	var copy_file_by_linking = (extent == nil)
	if copy_file_by_linking {
		copy_or_link = BUILD_LINK
	} else {
		copy_or_link = BUILD_COPY
	}

	var build = build_source{
		op: copy_or_link,
		upload: build_upload{
			stream: nil,
			length: -1,
		},
		copy: build_copy{
			source:        source,
			source_entity: source_entity,
			extent:        extent,
		},
		concat: build_concat{
			partlist: nil,
			mpul:     nil,
		},
	}
	var checks = copy_checks{}
	var conditionals = copy_conditionals{}
	var stat, etag, csum2, err1 = bbs.build_object(ctx, object, upload_id,
		part, build, metainfo, checksum2, checks, conditionals)
	return stat, etag, csum2, err1
}

// CONCATENATE_OBJECT concatenates the parts to an MPUL object.  It
// returns stat, etag, and csum of CRC64NVME.
func (bbs *Bb_server) concatenate_object(ctx context.Context, object string, partlist *types.CompletedMultipartUpload, mpul *Mpul_info, checks copy_checks, conditionals copy_conditionals) (fs.FileInfo, string, []byte, *Aws_s3_error) {
	var build = build_source{
		op: BUILD_CONCAT,
		upload: build_upload{
			stream: nil,
			length: 0,
		},
		copy: build_copy{
			source:        "",
			source_entity: "",
			extent:        nil,
		},
		concat: build_concat{
			partlist: partlist,
			mpul:     mpul,
		},
	}
	var upload_id = ""
	var part int32 = 0
	var info *Meta_info = nil
	var checksum2 types.ChecksumAlgorithm = checks.checksum
	var stat, etag, csum2, err1 = bbs.build_object(ctx, object, upload_id,
		part, build, info, checksum2, checks, conditionals)

	{
		var err2 = bbs.discard_mpul_directory(object)
		if err2 != nil {
			// IGNORE-ERRORS.
		}
	}

	return stat, etag, csum2, err1
}

func (bbs *Bb_server) build_object(ctx context.Context, object string, upload_id string, part int32, build build_source, metainfo *Meta_info, checksum2 types.ChecksumAlgorithm, checks copy_checks, conditionals copy_conditionals) (fs.FileInfo, string, []byte, *Aws_s3_error) {
	var location = "/" + object
	var _, rid = get_action_name(ctx)
	var scratchkey = bbs.make_scratch_suffix(rid)
	defer bbs.discharge_scratch_suffix(rid)

	bb_assert(build.op == BUILD_UPLOAD || build.op == BUILD_COPY ||
		build.op == BUILD_LINK || build.op == BUILD_CONCAT)

	// An MPUL part does not have metainfo.

	bb_assert((part == 0) == (upload_id == ""))
	bb_assert(!(part != 0) || (metainfo == nil))

	// TARGET is a copy destination.  It can be either an object or an
	// MPUL part file.

	var target string
	if part == 0 {
		target = object
	} else {
		target = make_mpul_part_name(object, part)
	}

	var scratch = bbs.make_scratch_object_name(object, scratchkey)

	var err1 = bbs.make_intervening_directories(object)
	if err1 != nil {
		return nil, "", nil, err1
	}

	var md5v []byte
	var csum1 []byte
	switch build.op {
	case BUILD_UPLOAD:
		var err2 *Aws_s3_error
		md5v, csum1, err2 = bbs.copy_file_as_scratch(ctx, object, scratch,
			build, checksum2)
		if err2 != nil {
			return nil, "", nil, err2
		}
	case BUILD_COPY:
		var err3 *Aws_s3_error
		md5v, csum1, err3 = bbs.copy_file_as_scratch(ctx, object, scratch,
			build, checksum2)
		if err3 != nil {
			return nil, "", nil, err3
		}
	case BUILD_LINK:
		// Copy a file by linking.
		var err4 *Aws_s3_error
		md5v, csum1, err4 = bbs.link_file_as_scratch(object, scratch,
			build, checksum2)
		if err4 != nil {
			return nil, "", nil, err4
		}
	case BUILD_CONCAT:
		var err5 *Aws_s3_error
		md5v, csum1, err5 = bbs.concat_parts_as_scratch(object, scratch,
			build, checksum2)
		if err5 != nil {
			return nil, "", nil, err5
		}
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(object, scratch)
		}
	}()

	//var checks = copy_checks{}
	var csum2, err3 = bbs.compare_checksums(object, scratch, checksum2,
		md5v, csum1, checks)
	if err3 != nil {
		return nil, "", nil, err3
	}

	var etag = make_object_etag_from_md5(md5v)

	var target_etag string
	var target_entity string
	if conditionals == (copy_conditionals{}) {
		target_etag = ""
		target_entity = ""
	} else {
		bb_assert(part == 0 && target == object)

		var _, entity2, err1 = bbs.fetch_object_status(target, false)
		if err1 != nil {
			// IGNORE-ERRORS.
		}
		var etag2, err2 = bbs.fetch_object_etag(target, entity2)
		if err2 != nil {
			// IGNORE-ERRORS.
		}
		// ETag and entity-key are empty strings on errors.
		target_etag = etag2
		target_entity = entity2
	}

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, "", nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	// Checking conditionals first checks the object is identical
	// before/after serialization.  Conditionals for uploading are
	// i.IfMatch or i.IfNoneMatch.

	if conditionals != (copy_conditionals{}) {
		bb_assert(part == 0 && target == object)

		var _, entity3, err12 = bbs.fetch_object_status(target, true)
		if err12 != nil {
			// IGNORE-ERRORS.
		}
		if entity3 != target_entity {
			bbs.logger.Info("Race: Target object changed during operation",
				"object", object)
			var errz = &Aws_s3_error{Code: InternalError,
				Message:  "Target object changed during operation.",
				Resource: location}
			return nil, "", nil, errz
		}

		var err7 = bbs.check_conditionals(target, target_etag, "write",
			conditionals)
		if err7 != nil {
			return nil, "", nil, err7
		}
	}

	// Check the source does not change before/after serialization.

	if build.op == BUILD_COPY {
		var source = build.copy.source
		var source_entity = build.copy.source_entity
		var _, entity4, err3 = bbs.fetch_object_status(source, false)
		if err3 != nil {
			return nil, "", nil, err3
		}
		if entity4 != source_entity {
			bbs.logger.Info("Race: Source object changed during operation",
				"source", source)
			var errz = &Aws_s3_error{Code: InternalError,
				Message:  "Source object changed during operation.",
				Resource: location}
			return nil, "", nil, errz
		}
	}

	// Re-check the MPUL upload-id after exclusion.

	if part != 0 {
		var _, err3 = bbs.check_upload_ongoing(object, &upload_id, true)
		if err3 != nil {
			return nil, "", nil, err3
		}
	}

	var stat fs.FileInfo
	var size int64
	var mtime time.Time

	{
		var stat5, entity5, err7 = bbs.fetch_object_status(scratch, true)
		if err7 != nil {
			return nil, "", nil, err7
		}
		stat = stat5
		bb_assert(stat != nil)

		size = stat.Size()
		mtime = stat.ModTime()

		// Insert an ETag into metainfo.

		var metainfo2 *Meta_info
		if metainfo != nil {
			metainfo2 = &Meta_info{
				Entity_key: entity5,
				ETag:       etag,
				Headers:    metainfo.Headers,
				Tags:       metainfo.Tags,
			}
		} else if size >= byte_size(bbs.config.Record_etag_threshold) {
			metainfo2 = &Meta_info{
				Entity_key: entity5,
				ETag:       etag,
				Headers:    nil,
				Tags:       nil,
			}
		}

		var err6 = bbs.place_scratch_file(object, scratch, target, metainfo2)
		if err6 != nil {
			return nil, "", nil, err6
		}
		cleanup_needed = false
	}

	/*
		var stat, _, err7 = bbs.fetch_object_status(target, true)
		if err7 != nil {
			return nil, "", nil, err7
		}
		bb_assert(stat != nil)
	*/

	// Update MPUL catatlog information.

	if part != 0 {
		var csums = base64.StdEncoding.EncodeToString(csum2)
		var partinfo = &Mpul_part{
			Size:     size,
			ETag:     etag,
			Checksum: csums,
			Mtime:    mtime,
		}
		var err8 = bbs.update_mpul_catalog(object, part, partinfo)
		if err8 != nil {
			return nil, "", nil, err8
		}
	}

	return stat, etag, csum2, nil
}

func (bbs *Bb_server) copy_file_as_scratch(ctx context.Context, object string, scratch string, build build_source, checksum2 types.ChecksumAlgorithm) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object

	//var copy_file_by_linking = (extent == nil)
	var body2 io.Reader
	var size int64
	var source_name string

	if build.op == BUILD_UPLOAD {
		// Modify reader of the body when Transfer-Encoding is chunked.

		size = build.upload.length
		source_name = "--"

		var body1 io.Reader = build.upload.stream
		var _, r = get_handler_arguments(ctx)
		var enc = r.TransferEncoding
		if len(enc) == 1 && strings.EqualFold(enc[0], "chunked") {
			body2 = httputil.NewChunkedReader(body1)
		} else if size != -1 {
			body2 = &io.LimitedReader{R: body1, N: size}
		} else {
			body2 = body1
		}
	} else {
		var source = build.copy.source
		var extent = build.copy.extent
		var path = bbs.make_path_of_object(source, "")
		size = extent[1] - extent[0]
		source_name = path

		var body1, err1 = os.Open(path)
		if err1 != nil {
			bbs.logger.Warn("os.Open() failed for copy source",
				"path", path, "error", err1)
			return nil, nil, map_os_error(location, err1, nil)
		}
		body2 = New_range_reader(body1, extent)
		defer func() {
			var err4 = body1.Close()
			if err4 != nil {
				bbs.logger.Warn("op.Close() failed",
					"path", path, "error", err4)
				// IGNORE-ERRORS.
			}
		}()
	}

	var md5v, csumv, err6 = bbs.copy_content_stream(object, scratch,
		size, source_name, checksum2, body2)
	if err6 != nil {
		return nil, nil, err6
	}
	return md5v, csumv, err6
}

func (bbs *Bb_server) link_file_as_scratch(object string, scratch string, build build_source, checksum2 types.ChecksumAlgorithm) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object

	bb_assert(build.copy.extent == nil)

	var source = build.copy.source
	var source_path = bbs.make_path_of_object(source, "")
	var target_path = bbs.make_path_of_object(scratch, "")
	var err3 = os.Remove(target_path)
	if err3 != nil && !errors.Is(err3, fs.ErrNotExist) {
		bbs.logger.Error("os.Remove() on a scratch file failed",
			"path", target_path, "error", err3)
		return nil, nil, map_os_error(location, err3, nil)
	}
	var err4 = os.Link(source_path, target_path)
	if err4 != nil {
		bbs.logger.Error("os.Link() on a scratch file failed",
			"source", source_path, "target", target_path, "error", err4)
		return nil, nil, map_os_error(location, err4, nil)
	}

	var md5v, crc1, _, err8 = bbs.calculate_csum2(object, checksum2, scratch, nil)
	if err8 != nil {
		return nil, nil, err8
	}
	return md5v, crc1, nil
}

func (bbs *Bb_server) concat_parts_as_scratch(object string, scratch string, build build_source, checksum2 types.ChecksumAlgorithm) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(scratch, "")

	var partlist *types.CompletedMultipartUpload = build.concat.partlist
	var mpul *Mpul_info = build.concat.mpul

	/*bbs.logger.Warn("IMPLEMENTATION OF CONCAT_PARTS() IS NAIVE AND SLOW")*/

	// Copy data to a temporary file.

	var f1, err1 = os.Create(path)
	if err1 != nil {
		bbs.logger.Info("os.Create() failed for concat parts",
			"path", path, "error", err1)
		return nil, nil, map_os_error(location, err1, nil)
	}
	var cleanup_needed = true
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.logger.Warn("op.Close() failed",
				"path", path, "error", err2)
		}
		if cleanup_needed {
			var _ = os.Remove(path)
		}
	}()

	var f2 io.Writer
	var hash1 hash.Hash
	var hash2 hash.Hash
	{
		var writers []io.Writer
		writers = append(writers, f1)
		if checksum2 != "" {
			hash1 = checksum_algorithm(checksum2)
			writers = append(writers, hash1)
		}
		if generate_md5_on_copy {
			hash2 = md5.New()
			writers = append(writers, hash2)
		}
		if len(writers) == 1 {
			f2 = f1
		} else {
			f2 = io.MultiWriter(writers...)
		}
	}

	var mpulpath = bbs.make_path_of_object(object, "@mpul")
	for _, p := range partlist.Parts {
		// p : types.CompletedPart
		// - ChecksumCRC32 *string
		// - ChecksumCRC32C *string
		// - ChecksumCRC64NVME *string
		// - ChecksumSHA1 *string
		// - ChecksumSHA256 *string
		// - ETag *string
		// - PartNumber *int32

		var part = *p.PartNumber
		var partname = make_part_name(part)
		var partpath = filepath.Join(mpulpath, partname)
		var f3, err1 = os.Open(partpath)
		if err1 != nil {
			bbs.logger.Warn("os.Open() failed for MPUL data",
				"path", partpath, "error", err1)
			return nil, nil, map_os_error(location, err1, nil)
		}
		defer func() {
			var err3 = f3.Close()
			if err3 != nil {
				bbs.logger.Warn("op.Close() failed",
					"path", partpath, "error", err3)
				// IGNORE-ERRORS.
			}
		}()
		var _, err2 = io.Copy(f2, f3)
		if err2 != nil {
			bbs.logger.Warn("io.Copy() failed for MPUL data",
				"path", partpath, "error", err2)
			return nil, nil, map_os_error(location, err2, nil)
		}

		//bbs.logger.Debug("concat copied", "count", cc)
	}

	cleanup_needed = false

	bb_assert(mpul.Initiate_time != nil)

	var err5 = os.Chtimes(path, time.Time{}, *mpul.Initiate_time)
	if err5 != nil {
		bbs.logger.Warn("op.Chtimes() failed",
			"path", path, "error", err5)
		// IGNORE-ERRORS.
	}

	var md5 []byte
	var csum []byte
	if hash1 != nil {
		md5 = hash1.Sum(nil)
	}
	if hash2 != nil {
		csum = hash2.Sum(nil)
	}
	return md5, csum, nil
}

// COPY_CONTENT_STREAM copies the stream data (for uploading or
// copying) to a temporary scratch file.  SOURCE_NAME indicates a
// source object for logging, which is either "--" (uploading) or an
// object name (copying).
func (bbs *Bb_server) copy_content_stream(object string, scratch string, size int64, source_name string, checksum2 types.ChecksumAlgorithm, body io.Reader) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(scratch, "")

	// Copy data to a temporary file.

	var f1, err1 = os.Create(path)
	if err1 != nil {
		bbs.logger.Info("os.Create() failed for copying",
			"path", path, "error", err1)
		return nil, nil, map_os_error(location, err1, nil)
	}
	var cleanup_needed = true
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.logger.Warn("op.Close() failed",
				"path", path, "error", err2)
		}
		if cleanup_needed {
			var _ = os.Remove(path)
		}
	}()

	var hash1 hash.Hash = md5.New()
	var hash2 hash.Hash = checksum_algorithm(checksum2)
	var f2 io.Writer
	{
		if hash2 != nil {
			f2 = io.MultiWriter(f1, hash1, hash2)
		} else {
			f2 = io.MultiWriter(f1, hash1)
		}
	}

	{
		var cc, err3 = io.Copy(f2, body)
		if err3 != nil {
			bbs.logger.Warn("io.Copy() failed for copying object",
				"path", source_name, "error", err3)
			return nil, nil, map_os_error(location, err3, nil)
		}
		if size != -1 && cc != size {
			bbs.logger.Info("Transfer truncated",
				"expected", size, "received", cc)
			var errz = &Aws_s3_error{Code: IncompleteBody,
				Resource: location}
			return nil, nil, errz
		}
	}

	cleanup_needed = false

	var md5 []byte
	var csum []byte
	if hash1 != nil {
		md5 = hash1.Sum(nil)
	}
	if hash2 != nil {
		csum = hash2.Sum(nil)
	}
	return md5, csum, nil
}

// PLACE_SCRATCH_FILE renames a scratch file to a target file.
func (bbs *Bb_server) place_scratch_file(object string, scratch string, target string, info *Meta_info) *Aws_s3_error {
	var location = "/" + object
	var path1 = bbs.make_path_of_object(scratch, "")
	var path2 = bbs.make_path_of_object(target, "")

	if info != nil {
		var err5 = bbs.store_metainfo(object, info)
		if err5 != nil {
			return err5
		}
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed && info != nil {
			var _ = bbs.store_metainfo(object, nil)
		}
	}()

	var err8 = os.Rename(path1, path2)
	if err8 != nil {
		bbs.logger.Error("io.Rename() failed", "error", err8)
		var errz = map_os_error(location, err8, nil)
		return errz
	}
	cleanup_needed = false

	return nil
}

// DISCARD_SCRATCH_FILE removes a scratch file.  It is called from a
// deferred call.  Errors are ignored.
func (bbs *Bb_server) discard_scratch_file(object string, scratch string) error {
	var path1 = bbs.make_path_of_object(scratch, "")
	var err1 = os.Remove(path1)
	if err1 != nil {
		bbs.logger.Warn("os.Remove() failed on scratch file",
			"path", path1, "error", err1)
		// IGNORE-ERRORS.
	}
	return nil
}

// CALCULATE_CSUM2 calculates two checksums of a file TARGET, md5 and
// one requested.  It skips one when a checksum algorithm CHECKSUM="".
// An algorithm is one of {CRC32, CRC32C, CRC64NVME, SHA1, SHA256}.
// The file range EXTENT is checked being within the file size by the
// caller.  It also returns an entity-key of a file.
func (bbs *Bb_server) calculate_csum2(object string, checksum types.ChecksumAlgorithm, target string, extent *[2]int64) ([]byte, []byte, string, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(target, "")

	var f1, err2 = os.Open(path)
	if err2 != nil {
		bbs.logger.Warn("os.Open() failed", "path", path, "error", err2)
		return nil, nil, "", map_os_error(location, err2, nil)
	}
	defer func() {
		var err3 = f1.Close()
		if err3 != nil {
			bbs.logger.Warn("os.Close() failed", "path", path, "error", err3)
		}
	}()
	var f2 = New_range_reader(f1, extent)

	var stat, err1 = f1.Stat()
	if err1 != nil {
		bbs.logger.Info("fs.File.Stat() failed in calculating csum",
			"path", path, "error", err1)
		return nil, nil, "", map_os_error(location, err1, nil)
	}
	var ino, ok = file_ino(stat, path)
	if !ok {
		log.Fatal("BAD-IMPL: Cannot take inode number")
	}
	var entity = hash_entity_key(stat, ino)

	var size int64
	if extent == nil {
		size = stat.Size()
	} else {
		size = extent[1] - extent[0]
	}

	var hash1 hash.Hash = md5.New()
	var hash2 hash.Hash = checksum_algorithm(checksum)
	var f3 io.Writer
	{
		if hash2 != nil {
			f3 = io.MultiWriter(hash1, hash2)
		} else {
			f3 = hash1
		}
	}
	var cc, err4 = io.Copy(f3, f2)
	if err4 != nil {
		return nil, nil, "", map_os_error(location, err4, nil)
	}
	if cc != size {
		bbs.logger.Info("io.Copy() failed in calculating csum, bad copy size",
			"path", path, "extent-size", size, "copied-size", cc)
		var err5 = &Aws_s3_error{Code: InternalError,
			Message:  "io.Copy() failed, incomplete copy",
			Resource: location}
		return nil, nil, "", err5
	}

	var csum1 []byte = hash1.Sum(nil)
	var csum2 []byte
	if hash2 != nil {
		csum2 = hash2.Sum(nil)
	}
	return csum1, csum2, entity, nil
}

// COMPARE_CHECKSUMS compares checksums between ones passed and
// calculated.  It will calculate the checksum of a SCRATCH when one is
// needed by CRC64NVME.
func (bbs *Bb_server) compare_checksums(object string, scratch string, checksum1 types.ChecksumAlgorithm, md5a []byte, csum1 []byte, checks copy_checks) ([]byte, *Aws_s3_error) {
	var location = "/" + object
	if checks.md5_to_check != nil {
		if bytes.Compare(checks.md5_to_check, md5a) != 0 {
			bbs.logger.Info("Checksums mismatch",
				"algorithm", "MD5", "object", object)
			var errz = &Aws_s3_error{Code: BadDigest,
				Message:  "The md5 did not match what we received.",
				Resource: location}
			return nil, errz
		}
	}
	if checks.csum_to_check != nil {
		if bytes.Compare(checks.csum_to_check, csum1) != 0 {
			bbs.logger.Info("Checksums mismatch",
				"algorithm", checksum1, "object", object)
			var errz = &Aws_s3_error{Code: BadDigest,
				Message:  "The checksum did not match what we received.",
				Resource: location}
			return nil, errz
		}
	}

	var checksum2 = types.ChecksumAlgorithmCrc64nvme
	var csum2 []byte
	if bbs.config.Verify_fs_write || checksum1 != checksum2 {
		// HERE NEEDS A FS CACHE PURGE CALL (or flock for NFS).

		var md5b, crc1, _, err8 = bbs.calculate_csum2(object, checksum2, scratch, nil)
		if err8 != nil {
			return nil, err8
		}
		if bytes.Compare(md5a, md5b) != 0 {
			bbs.logger.Error("Copying file unverified, MD5 values differ",
				"object", object)
			var errz = &Aws_s3_error{
				Code:     InternalError,
				Message:  "Copying file unverified",
				Resource: location}
			return nil, errz
		}
		csum2 = crc1
	} else {
		csum2 = csum1
	}
	return csum2, nil
}
