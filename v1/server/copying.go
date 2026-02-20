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
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type copy_checks struct {
	checksum      types.ChecksumAlgorithm
	size_to_check int64
	md5_to_check  []byte
	csum_to_check []byte
}

type copy_conditions struct {
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
// be a sum type).  The concat part is effective on MPUL.

type build_source struct {
	op     build_op
	upload build_upload
	copy   build_copy
	concat build_concat
}

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

// UPLOAD_OBJECT performs uploading, where a target of uploading is
// either for an object or an MPUL part file.  It returns etag, stat,
// and csum (of CRC64NVME).  Metainfo is only partially filled.
func (bbs *Bb_server) upload_object(ctx context.Context, object string, upload_id string, part int32, body io.Reader, metainfo *Meta_info, checks copy_checks, conditions copy_conditions) (string, fs.FileInfo, []byte, *Aws_s3_error) {
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
	var checksum types.ChecksumAlgorithm = checks.checksum
	var etag, stat, csum2, err1 = bbs.build_object(ctx, object, upload_id,
		part, build, metainfo, checksum, checks, conditions)
	return etag, stat, csum2, err1
}

// COPY_OBJECT performs copying.  A copying target is either an object
// or an MPUL part file.  It returns etag, stat and csum.  Metainfo is
// only partially filled.  Specify a CHECKSUM when checksum
// calcualation is needed.  Note condition checks are on the source
// object, and is checked by the caller.
func (bbs *Bb_server) copy_object(ctx context.Context, object string, upload_id string, part int32, source string, source_entity string, extent *[2]int64, metainfo *Meta_info, checksum types.ChecksumAlgorithm) (string, fs.FileInfo, []byte, *Aws_s3_error) {
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
	var conditions = copy_conditions{}
	var etag, stat, csum2, err1 = bbs.build_object(ctx, object, upload_id,
		part, build, metainfo, checksum, checks, conditions)
	return etag, stat, csum2, err1
}

// CONCATENATE_OBJECT concatenates the parts to an MPUL object.  It
// returns etag, stat, and csum of CRC64NVME.
func (bbs *Bb_server) concatenate_object(ctx context.Context, object string, mpulinfo *Mpul_info, partlist *types.CompletedMultipartUpload, checks copy_checks, conditions copy_conditions) (string, fs.FileInfo, []byte, *Aws_s3_error) {
	//var _, rid, suffix = get_action_name(ctx)
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
			mpul:     mpulinfo,
		},
	}
	var upload_id = ""
	var part int32 = 0
	var metainfo *Meta_info = mpulinfo.Metainfo
	var checksum types.ChecksumAlgorithm = checks.checksum
	var etag, stat, csum2, err1 = bbs.build_object(ctx, object, upload_id,
		part, build, metainfo, checksum, checks, conditions)
	return etag, stat, csum2, err1
}

func (bbs *Bb_server) build_object(ctx context.Context, object string, upload_id string, part int32, build build_source, metainfo *Meta_info, checksum types.ChecksumAlgorithm, checks copy_checks, conditions copy_conditions) (string, fs.FileInfo, []byte, *Aws_s3_error) {
	var location = "/" + object
	var _, rid, suffix = get_action_name(ctx)

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

	var scratch = bbs.make_scratch_object_name(object, suffix)

	var err1 = bbs.make_intervening_directories(rid, object)
	if err1 != nil {
		return "", nil, nil, err1
	}

	var md5v []byte
	var csum1 []byte
	switch build.op {
	case BUILD_UPLOAD:
		fallthrough
	case BUILD_COPY:
		// Copy a file from a stream.
		var err2 *Aws_s3_error
		md5v, csum1, err2 = bbs.copy_file_as_scratch(ctx, object, scratch,
			build, checksum)
		if err2 != nil {
			return "", nil, nil, err2
		}
	case BUILD_LINK:
		// Copy a file by linking.
		var err4 *Aws_s3_error
		md5v, csum1, err4 = bbs.link_file_as_scratch(ctx, object, scratch,
			build, checksum)
		if err4 != nil {
			return "", nil, nil, err4
		}
	case BUILD_CONCAT:
		var err5 *Aws_s3_error
		md5v, csum1, err5 = bbs.concat_parts_as_scratch(ctx, object, scratch,
			build, checksum)
		if err5 != nil {
			return "", nil, nil, err5
		}
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(rid, object, scratch)
		}
	}()

	var etag = make_object_etag_from_md5(md5v)

	var csum2, err3 = bbs.compare_checksums(rid, object, scratch, checksum,
		md5v, csum1, checks)
	if err3 != nil {
		return "", nil, nil, err3
	}

	var target_entity string
	var target_etag string
	var target_stat fs.FileInfo
	if conditions != (copy_conditions{}) {
		// ETag and entity-key are empty strings on errors.
		bb_assert(part == 0 && target == object)

		var err1, err2 error
		target_entity, target_stat, err1 = bbs.fetch_object_status(rid, target, false)
		if err1 != nil {
			// IGNORE-ERRORS.
		}
		target_etag, _, err2 = bbs.fetch_object_etag(rid, target, target_entity)
		if err2 != nil {
			// IGNORE-ERRORS.
		}
	} else {
		target_entity = ""
		target_etag = ""
		target_stat = nil
	}

	var entity string
	var stat fs.FileInfo
	var size int64
	var mtime time.Time
	//var metainfo2 *Meta_info = nil

	// Prepare metainfo by inserting an entity-key and an etag.

	{
		var err7 *Aws_s3_error
		entity, stat, err7 = bbs.fetch_object_status(rid, scratch, true)
		if err7 != nil {
			return "", nil, nil, err7
		}
		bb_assert(stat != nil)

		size = stat.Size()
		mtime = stat.ModTime()

		// Insert an ETag into metainfo.  It stores metainfo, when the
		// object is large.

		if part != 0 {
			bb_assert(metainfo == nil)
		} else {
			if metainfo != nil {
				bb_assert(metainfo.Entity_key == "" && metainfo.ETag == "")
				metainfo.Entity_key = entity
				metainfo.ETag = etag
			} else if size >= byte_size(bbs.config.Etag_save_threshold) {
				metainfo = &Meta_info{
					Entity_key: entity,
					ETag:       etag,
					Checksum:   "",
					Csum:       "",
					Headers:    nil,
					Tags:       nil,
				}
			}
		}
	}

	// Prepare saving part information.

	var partinfo *Mpul_part = nil

	{
		if part != 0 {
			var csum = base64.StdEncoding.EncodeToString(csum2)
			partinfo = &Mpul_part{
				Entity_key: entity,
				ETag:       etag,
				Size:       size,
				Mtime:      mtime,
				Checksum:   checksum,
				Csum:       csum,
			}
		} else {
			partinfo = nil
		}
	}

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return "", nil, nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	// Checking conditions first checks the object is identical
	// before/after serialization.  Conditions for uploading are
	// i.IfMatch or i.IfNoneMatch.

	if conditions != (copy_conditions{}) {
		bb_assert(part == 0 && target == object)

		var entity3, _, err12 = bbs.fetch_object_status(rid, target, true)
		if err12 != nil {
			// IGNORE-ERRORS.
		}
		if entity3 != target_entity {
			bbs.logger.Info("Race: Target object changed during operation",
				"rid", rid, "object", object)
			var errz = &Aws_s3_error{Code: InternalError,
				Message:  "Target object changed during operation.",
				Resource: location}
			return "", nil, nil, errz
		}

		var target_mtime = target_stat.ModTime()
		var target_size = target_stat.Size()
		var err7 = bbs.check_conditions(rid, target, target_etag,
			target_mtime, target_size, "write", conditions)
		if err7 != nil {
			return "", nil, nil, err7
		}
	}

	// Check the source does not change before/after serialization.

	if build.op == BUILD_COPY {
		var source = build.copy.source
		var source_entity = build.copy.source_entity
		var entity4, _, err3 = bbs.fetch_object_status(rid, source, false)
		if err3 != nil {
			return "", nil, nil, err3
		}
		if entity4 != source_entity {
			bbs.logger.Info("Race: Source object changed during operation",
				"rid", rid, "source", source)
			var errz = &Aws_s3_error{Code: InternalError,
				Message:  "Source object changed during operation.",
				Resource: location}
			return "", nil, nil, errz
		}
	}

	// Re-check the MPUL upload-id after exclusion.

	if part != 0 {
		var _, err3 = bbs.check_mpul_ongoing(rid, object, &upload_id, true)
		if err3 != nil {
			return "", nil, nil, err3
		}
	}

	{
		var err6 = bbs.place_scratch_file(rid, object, scratch,
			target, suffix, metainfo, (build.op == BUILD_LINK))
		if err6 != nil {
			return "", nil, nil, err6
		}
		cleanup_needed = false
	}

	// Update MPUL catatlog file.

	if part != 0 {
		var err8 = bbs.update_mpul_catalog(rid, object, part, suffix, partinfo)
		if err8 != nil {
			return "", nil, nil, err8
		}
	}

	if build.op == BUILD_CONCAT {
		var err2 = bbs.discard_mpul_directory(rid, object)
		if err2 != nil {
			// IGNORE-ERRORS.
		}
	}

	// This logging is printed in serialized region.

	if bbs.config.Verbose_debug_logging {
		if part == 0 {
			bbs.logger.Debug("Creating an object",
				"rid", rid, "object", object, "build", build.op,
				"metainfo", metainfo)
		} else {
			bbs.logger.Debug("Creating a multipart part",
				"rid", rid, "object", object, "build", build.op,
				"upload_id", upload_id, "part", part,
				"partinfo", partinfo)
		}
	}

	return etag, stat, csum2, nil
}

func (bbs *Bb_server) copy_file_as_scratch(ctx context.Context, object string, scratch string, build build_source, checksum types.ChecksumAlgorithm) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object
	var _, rid, _ = get_action_name(ctx)

	bb_assert(build.op == BUILD_UPLOAD || build.op == BUILD_COPY)

	var body2 io.Reader
	var size int64
	var source_name string

	if build.op == BUILD_UPLOAD {
		// Modify reader of the body when Transfer-Encoding is chunked.

		size = build.upload.length
		source_name = "--stream--"

		var body1 io.Reader = build.upload.stream
		var _, r = get_handler_arguments(ctx)

		var bodyc, chunked, length, err1 = bbs.make_chunked_reader(object, rid,
			body1, r)
		if err1 != nil {
			return nil, nil, err1
		}
		if chunked != CHUNKED_NO {
			bbs.logger.Info("Body stream with chunked-reader",
				"rid", rid, "object", object, "chunked", chunked,
				"body", bodyc)
		}

		switch chunked {
		case CHUNKED_HTTP1:
			size = -1
			body2 = bodyc
		case CHUNKED_AWSS3:
			size = length
			body2 = bodyc
		case CHUNKED_NO:
			fallthrough
		default:
			bb_assert(bodyc == body1)
			if size != -1 {
				body2 = &io.LimitedReader{R: bodyc, N: size}
			} else {
				body2 = bodyc
			}
		}
	} else if build.op == BUILD_COPY {
		var source = build.copy.source
		var extent = build.copy.extent
		var path = bbs.make_path_of_object(source, "")
		size = extent[1] - extent[0]
		source_name = path

		var body1, err1 = os.Open(path)
		if err1 != nil {
			bbs.logger.Warn("os.Open() for copy source failed",
				"rid", rid, "path", path, "error", err1)
			return nil, nil, map_os_error(location, err1, nil)
		}
		body2 = New_range_reader(body1, extent)
		defer func() {
			var err4 = body1.Close()
			if err4 != nil {
				bbs.logger.Warn("op.Close() on copy source failed",
					"rid", rid, "path", path, "error", err4)
				// IGNORE-ERRORS.
			}
		}()
	} else {
		Fatalf("never")
	}

	var md5v, csumv, err6 = bbs.copy_content_stream(rid, object, scratch,
		size, source_name, checksum, body2)
	if err6 != nil {
		return nil, nil, err6
	}
	return md5v, csumv, err6
}

func (bbs *Bb_server) link_file_as_scratch(ctx context.Context, object string, scratch string, build build_source, checksum types.ChecksumAlgorithm) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object
	var _, rid, _ = get_action_name(ctx)

	bb_assert(build.copy.extent == nil)

	var source = build.copy.source
	var source_path = bbs.make_path_of_object(source, "")
	var target_path = bbs.make_path_of_object(scratch, "")
	var err3 = os.Remove(target_path)
	if err3 != nil && !errors.Is(err3, fs.ErrNotExist) {
		bbs.logger.Error("os.Remove() on a scratch file failed",
			"rid", rid, "path", target_path, "error", err3)
		return nil, nil, map_os_error(location, err3, nil)
	}
	var err4 = os.Link(source_path, target_path)
	if err4 != nil {
		bbs.logger.Error("os.Link() on a scratch file failed",
			"rid", rid, "source", source_path, "target", target_path, "error", err4)
		return nil, nil, map_os_error(location, err4, nil)
	}

	var md5v, crc1, _, err8 = bbs.calculate_csum2(rid, object, checksum, scratch, nil)
	if err8 != nil {
		return nil, nil, err8
	}
	return md5v, crc1, nil
}

func (bbs *Bb_server) concat_parts_as_scratch(ctx context.Context, object string, scratch string, build build_source, checksum types.ChecksumAlgorithm) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object
	var _, rid, _ = get_action_name(ctx)

	var path = bbs.make_path_of_object(scratch, "")

	var partlist *types.CompletedMultipartUpload = build.concat.partlist
	var mpul *Mpul_info = build.concat.mpul

	// Copy data to a temporary file.

	var f1, err1 = os.Create(path)
	if err1 != nil {
		bbs.logger.Info("os.Create() for concat parts failed",
			"rid", rid, "path", path, "error", err1)
		return nil, nil, map_os_error(location, err1, nil)
	}
	var cleanup_needed = true
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.logger.Warn("op.Close() on concat parts failed",
				"rid", rid, "path", path, "error", err2)
		}
		if cleanup_needed {
			var _ = os.Remove(path)
		}
	}()

	var f2 io.Writer
	var hash_md5 hash.Hash
	var hash_crc hash.Hash
	{
		var writers []io.Writer
		writers = append(writers, f1)
		if true {
			hash_md5 = md5.New()
			writers = append(writers, hash_md5)
		}
		if checksum != "" {
			hash_crc = checksum_algorithm(checksum)
			writers = append(writers, hash_crc)
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
			bbs.logger.Warn("os.Open() for MPUL data failed",
				"rid", rid, "path", partpath, "error", err1)
			return nil, nil, map_os_error(location, err1, nil)
		}
		defer func() {
			var err3 = f3.Close()
			if err3 != nil {
				bbs.logger.Warn("op.Close() on MPUL data failed",
					"rid", rid, "path", partpath, "error", err3)
				// IGNORE-ERRORS.
			}
		}()
		var _, err2 = io.Copy(f2, f3)
		if err2 != nil {
			bbs.logger.Warn("io.Copy() for MPUL data failed",
				"rid", rid, "path", partpath, "error", err2)
			return nil, nil, map_os_error(location, err2, nil)
		}

		//bbs.logger.Debug("concat copied", "count", cc)
	}

	cleanup_needed = false

	var err5 = os.Chtimes(path, time.Time{}, mpul.Initiate_time)
	if err5 != nil {
		bbs.logger.Warn("op.Chtimes() failed",
			"rid", rid, "path", path, "error", err5)
		// IGNORE-ERRORS.
	}

	var md5 []byte
	var csum []byte
	if hash_md5 != nil {
		md5 = hash_md5.Sum(nil)
	}
	if hash_crc != nil {
		csum = hash_crc.Sum(nil)
	}
	return md5, csum, nil
}

// COPY_CONTENT_STREAM copies the stream data (for uploading or
// copying) to a temporary scratch file.  SOURCE_NAME indicates a
// source object for logging, which is either "--" (uploading) or an
// object name (copying).
func (bbs *Bb_server) copy_content_stream(rid uint64, object string, scratch string, size int64, source_name string, checksum types.ChecksumAlgorithm, body io.Reader) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(scratch, "")

	// Copy data to a temporary file.

	var f1, err1 = os.Create(path)
	if err1 != nil {
		bbs.logger.Info("os.Create() for copying object failed",
			"rid", rid, "path", path, "error", err1)
		return nil, nil, map_os_error(location, err1, nil)
	}
	var cleanup_needed = true
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.logger.Warn("op.Close() on copying object failed",
				"rid", rid, "path", path, "error", err2)
		}
		if cleanup_needed {
			var _ = os.Remove(path)
		}
	}()

	var hash1 hash.Hash = md5.New()
	var hash2 hash.Hash = checksum_algorithm(checksum)
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
			bbs.logger.Warn("io.Copy() for copying object failed",
				"rid", rid, "path", source_name, "error", err3)
			return nil, nil, map_os_error(location, err3, nil)
		}
		if size != -1 && cc != size {
			bbs.logger.Info("Transfer truncated",
				"rid", rid, "expected", size, "received", cc)
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

// PLACE_SCRATCH_FILE renames a scratch file to a target.  Notice the
// section of code with os.SameFile().  Copying source=target results
// in the same file (a scratch file is a hard-link of a source).  And,
// it causes renaming to fail, because rename(2) in Unix does nothing
// on the same file (with success).  os.SameFile() checks the
// condition and it removes the target to handle the case.
func (bbs *Bb_server) place_scratch_file(rid uint64, object string, scratch string, target string, suffix string, metainfo *Meta_info, copy_file_by_linking bool) *Aws_s3_error {
	var location = "/" + object
	var path1 = bbs.make_path_of_object(scratch, "")
	var path2 = bbs.make_path_of_object(target, "")

	if metainfo != nil {
		var err5 = bbs.store_object_metainfo(rid, object, suffix, metainfo)
		if err5 != nil {
			return err5
		}
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed && metainfo != nil {
			var err1 = bbs.store_object_metainfo(rid, object, suffix, nil)
			if err1 != nil {
				// IGNORE-ERRORS.
			}
		}
	}()

	// Remove the target as os.Rename() fails on the same file.  Look
	// at second file first.

	if copy_file_by_linking {
		var stat1, stat2 fs.FileInfo
		var err1, err2 error
		stat2, err2 = os.Stat(path2)
		if err2 != nil {
			// A missing target is normal. Okay.
			// IGNORE-ERRORS.
		} else {
			stat1, err1 = os.Stat(path1)
			if err1 != nil {
				bbs.logger.Error("os.Stat() in placing the file failed",
					"rid", rid, "error", err1)
				var errz = map_os_error(location, err1, nil)
				return errz
			}
		}
		if stat1 != nil && stat2 != nil && os.SameFile(stat1, stat2) {
			bbs.logger.Debug("Special handling of the same file in copying",
				"path1", path1, "path2", path2)
			var err3 = os.Remove(path2)
			if err3 != nil {
				bbs.logger.Error("os.Remove() in placing the same file failed",
					"rid", rid, "error", err3)
				var errz = map_os_error(location, err3, nil)
				return errz
			}
		}
	}

	var err8 = os.Rename(path1, path2)
	if err8 != nil {
		bbs.logger.Error("os.Rename() in placing a scratch file failed",
			"rid", rid, "error", err8)
		var errz = map_os_error(location, err8, nil)
		return errz
	}
	cleanup_needed = false

	return nil
}

// DISCARD_SCRATCH_FILE removes a scratch file.  It is called from a
// deferred call.  Errors are ignored.
func (bbs *Bb_server) discard_scratch_file(rid uint64, object string, scratch string) error {
	var path1 = bbs.make_path_of_object(scratch, "")
	var err1 = os.Remove(path1)
	if err1 != nil {
		bbs.logger.Warn("os.Remove() on scratch file failed",
			"rid", rid, "path", path1, "error", err1)
		// IGNORE-ERRORS.
	}
	return nil
}

// CALCULATE_CSUM2 calculates two checksums of a file TARGET, md5 and
// one requested.  It skips one when a checksum algorithm CHECKSUM="".
// An algorithm is one of {CRC32, CRC32C, CRC64NVME, SHA1, SHA256}.
// The file range EXTENT is checked being within the file size by the
// caller.  It also returns an entity-key of a file.
func (bbs *Bb_server) calculate_csum2(rid uint64, object string, checksum types.ChecksumAlgorithm, target string, extent *[2]int64) ([]byte, []byte, string, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(target, "")

	var f1, err2 = os.Open(path)
	if err2 != nil {
		bbs.logger.Warn("os.Open() for calculating csum failed",
			"rid", rid, "path", path, "error", err2)
		return nil, nil, "", map_os_error(location, err2, nil)
	}
	defer func() {
		var err3 = f1.Close()
		if err3 != nil {
			bbs.logger.Warn("os.Close() on calculating csum failed",
				"rid", rid, "path", path, "error", err3)
		}
	}()
	var f2 = New_range_reader(f1, extent)

	var stat, err1 = f1.Stat()
	if err1 != nil {
		bbs.logger.Info("fs.File.Stat() in calculating csum failed",
			"rid", rid, "path", path, "error", err1)
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
		bbs.logger.Info("io.Copy() in calculating csum failed; bad copy size",
			"rid", rid, "path", path, "extent-size", size, "copied-size", cc)
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
func (bbs *Bb_server) compare_checksums(rid uint64, object string, scratch string, checksum1 types.ChecksumAlgorithm, md5a []byte, csum1 []byte, checks copy_checks) ([]byte, *Aws_s3_error) {
	var location = "/" + object
	if checks.md5_to_check != nil {
		if bytes.Compare(checks.md5_to_check, md5a) != 0 {
			bbs.logger.Info("Checksums mismatch",
				"rid", rid, "algorithm", "MD5", "object", object)
			var errz = &Aws_s3_error{Code: BadDigest,
				Message:  "The md5 did not match what we received.",
				Resource: location}
			return nil, errz
		}
	}
	if checks.csum_to_check != nil {
		if bytes.Compare(checks.csum_to_check, csum1) != 0 {
			bbs.logger.Info("Checksums mismatch",
				"rid", rid, "algorithm", checksum1, "object", object)
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

		var md5b, crc1, _, err8 = bbs.calculate_csum2(rid, object, checksum2, scratch, nil)
		if err8 != nil {
			return nil, err8
		}
		if bytes.Compare(md5a, md5b) != 0 {
			bbs.logger.Error("Copying file unverified, MD5 values differ",
				"rid", rid, "object", object,
				"md5", hex.EncodeToString(md5a[:]),
				"md5", hex.EncodeToString(md5b[:]))
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

func (bbs *Bb_server) make_chunked_reader(object string, rid uint64, body io.Reader, q *http.Request) (io.Reader, Chunked_type, int64, *Aws_s3_error) {
	var location = "/" + object
	var r2, chunked, length, err1 = New_chunked_reader(q, body, rid, bbs)
	if err1 != nil {
		bbs.logger.Info("Making chunked-reader failed",
			"rid", rid, "object", object, "chunked", "AWSS3",
			"error", err1)
		var errz = &Aws_s3_error{Code: InvalidRequest,
			Message:  "Making chunked-reader failed.",
			Resource: location}
		return nil, CHUNKED_NO, 0, errz
	} else {
		return r2, chunked, length, nil
	}
}
