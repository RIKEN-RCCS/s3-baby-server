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
	"encoding/binary"
	//"encoding/hex"
	//"encoding/json"
	//"encoding/xml"
	"errors"
	//"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"hash"
	"io"
	"io/fs"
	//"log"
	"time"
	//"net/url"
	"os"
	//"path"
	"path/filepath"
	//"strings"
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

// UPLOAD_OBJECT performs uploading.  Uploading is either for a file
// of an object or a file of a MPUL part.  It returns stat, etag, and
// csum (of CRC64NVME).
func (bbs *Bb_server) upload_object(ctx context.Context, object string, part int32, upload_id string, body io.Reader, info *Meta_info, checks copy_checks, conditionals copy_conditionals) (fs.FileInfo, string, []byte, *Aws_s3_error) {
	//var location = "/" + object
	var _, rid = get_request_action(ctx)
	var scratchkey = bbs.make_scratch_suffix(rid)
	defer bbs.discharge_scratch_suffix(rid)

	bb_assert(!(part != 0) || upload_id != "")

	// TARGET is the copy destination.  It can be either an object or
	// a MPUL part file.

	var target string
	if part != 0 {
		target = make_mpul_part_name(object, part)
	} else {
		target = object
	}

	var scratch = bbs.make_scratch_object_name(object, scratchkey)

	var err1 = bbs.make_intervening_directories(object)
	if err1 != nil {
		return nil, "", nil, err1
	}

	var checksum1 types.ChecksumAlgorithm
	if checks.checksum != "" {
		checksum1 = checks.checksum
	} else {
		checksum1 = types.ChecksumAlgorithmCrc64nvme
	}

	var md5a, csum1, err2 = bbs.upload_file_as_scratch(object, scratch,
		checks.size_to_check, checksum1, body)
	if err2 != nil {
		return nil, "", nil, err2
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(object, scratch)
		}
	}()

	var csum2, err3 = bbs.compare_checksums(object, scratch, checksum1,
		md5a, csum1, checks)
	if err3 != nil {
		return nil, "", nil, err3
	}

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, "", nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	// Re-check the MPUL upload-id after exclusion, when an upload is
	// for MPUL.

	if part != 0 {
		var _, err3 = bbs.check_upload_ongoing(object, &upload_id, true)
		if err3 != nil {
			return nil, "", nil, err3
		}
	}

	// Conditionals for uploading are i.IfMatch or i.IfNoneMatch.

	if part == 0 {
		var err15 = bbs.check_request_conditionals(target, "write",
			conditionals)
		if err15 != nil {
			return nil, "", nil, err15
		}
	}

	{
		var err6 = bbs.place_scratch_file(object, scratch, target, info)
		if err6 != nil {
			return nil, "", nil, err6
		}
		cleanup_needed = false
	}

	var stat, etag, err7 = bbs.fetch_object_status(target, true)
	if err7 != nil {
		return nil, "", nil, err7
	}
	bb_assert(stat != nil)

	// Update MPUL catatlog information.

	if part != 0 {
		var size = stat.Size()
		var mtime = stat.ModTime()
		var csums = base64.StdEncoding.EncodeToString(csum1)
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

// COPY_OBJECT performs copying.  Copying is either for a file of an
// object or a MPUL part.  It returns stat, etag, and csum.  A
// checksum value is by the algorithm of CHECKSUM, which is one for
// MPUL when copying is for MPUL.  Note checksum checks are not
// applied on copying.  CONDITIONALS are on the source object.
func (bbs *Bb_server) copy_object(ctx context.Context, object string, part int32, upload_id string, source string, extent *[2]int64, info *Meta_info, checksum types.ChecksumAlgorithm, conditionals copy_conditionals) (fs.FileInfo, string, []byte, *Aws_s3_error) {
	//var location = "/" + object
	var _, rid = get_request_action(ctx)
	var scratchkey = bbs.make_scratch_suffix(rid)
	defer bbs.discharge_scratch_suffix(rid)

	// A MPUL part does not have metainfo.

	bb_assert(!(part != 0) || (info == nil && upload_id != ""))

	// TARGET is the copy destination.  It can be either an object or
	// a MPUL part file.

	var target string
	if part != 0 {
		target = make_mpul_part_name(object, part)
	} else {
		target = object
	}

	var scratch = bbs.make_scratch_object_name(object, scratchkey)

	var checksum2 = checksum

	var md5a, csum1, err6 = bbs.copy_file_as_scratch(object, scratch,
		source, extent, checksum2)
	if err6 != nil {
		return nil, "", nil, err6
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(object, scratch)
		}
	}()

	var checks = copy_checks{}
	var csum2, err3 = bbs.compare_checksums(object, scratch, checksum2,
		md5a, csum1, checks)
	if err3 != nil {
		return nil, "", nil, err3
	}

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, "", nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	// Re-check the MPUL upload-id after exclusion, when an upload is
	// for MPUL.

	if part != 0 {
		var _, err3 = bbs.check_upload_ongoing(object, &upload_id, true)
		if err3 != nil {
			return nil, "", nil, err3
		}
	}

	var err5 = bbs.check_request_conditionals(source, "read", conditionals)
	if err5 != nil {
		return nil, "", nil, err5
	}

	{
		var err6 = bbs.place_scratch_file(object, scratch, target, info)
		if err6 != nil {
			return nil, "", nil, err6
		}
		cleanup_needed = false
	}

	var stat, etag, err7 = bbs.fetch_object_status(target, true)
	if err7 != nil {
		return nil, "", nil, err7
	}
	bb_assert(stat != nil)

	// Update MPUL catatlog information.

	if part != 0 {
		var size = stat.Size()
		var mtime = stat.ModTime()
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

// CONCATENATE_OBJECT concatenates the parts to an object.  It returns
// stat, etag, and csum of CRC64NVME.
func (bbs *Bb_server) concatenate_object(ctx context.Context, object string, partlist *types.CompletedMultipartUpload, mpul *Mpul_info, checks copy_checks, conditionals copy_conditionals) (fs.FileInfo, string, []byte, *Aws_s3_error) {
	//var location = "/" + object
	var _, rid = get_request_action(ctx)
	var scratchkey = bbs.make_scratch_suffix(rid)
	defer bbs.discharge_scratch_suffix(rid)

	var scratch = bbs.make_scratch_object_name(object, scratchkey)
	var target = object

	var checksum1 types.ChecksumAlgorithm
	if checks.checksum != "" {
		checksum1 = checks.checksum
	} else {
		checksum1 = types.ChecksumAlgorithmCrc64nvme
	}

	var md5a, csum1, err5 = bbs.concat_parts_as_scratch(object, scratch, partlist, mpul, checksum1)
	if err5 != nil {
		return nil, "", nil, err5
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(object, scratch)
		}
	}()

	var csum2, err3 = bbs.compare_checksums(object, scratch, checksum1,
		md5a, csum1, checks)
	if err3 != nil {
		return nil, "", nil, err3
	}

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, "", nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	// Re-check the MPUL upload-id after exclusion.

	{
		var mpul2, err3 = bbs.check_upload_ongoing(object, mpul.UploadId, true)
		if err3 != nil {
			return nil, "", nil, err3
		}
		var info = mpul2.MetaInfo

		var err7 = bbs.check_request_conditionals(object, "write",
			conditionals)
		if err7 != nil {
			return nil, "", nil, err7
		}

		var err6 = bbs.place_scratch_file(object, scratch, target, info)
		if err6 != nil {
			return nil, "", nil, err6
		}
		cleanup_needed = false

		var err2 = bbs.discard_mpul_directory(object)
		if err2 != nil {
			// IGNORE-ERRORS.
		}
	}

	var stat, etag, err7 = bbs.fetch_object_status(object, true)
	if err7 != nil {
		return nil, "", nil, err7
	}
	bb_assert(stat != nil)

	return stat, etag, csum2, nil
}

// UPLOAD_FILE_AS_SCRATCH stores the contents as a scratch file.
// Renaming a scratch file to an actual file will be done in
// serialization.
func (bbs *Bb_server) upload_file_as_scratch(object string, scratch string, size int64, checksum2 types.ChecksumAlgorithm, body io.Reader) ([]byte, []byte, *Aws_s3_error) {
	//var location = "/" + object
	//var path = bbs.make_path_of_object(scratch, "")

	var body2 io.Reader
	if size != -1 {
		body2 = &io.LimitedReader{R: body, N: size}
	} else {
		body2 = body
	}

	var md5, csum, err6 = bbs.copy_content_stream(object, scratch, "",
		size, checksum2, body2)
	if err6 != nil {
		return nil, nil, err6
	}
	return md5, csum, nil
}

func (bbs *Bb_server) copy_file_as_scratch(object string, scratch string, source string, extent *[2]int64, checksum2 types.ChecksumAlgorithm) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object
	var copy_file_by_linking = (extent == nil)

	if !copy_file_by_linking {
		var size int64 = extent[1] - extent[0]
		var sourcepath = bbs.make_path_of_object(source, "")
		var f3, err1 = os.Open(sourcepath)
		if err1 != nil {
			bbs.logger.Warn("os.Open() failed for copy source",
				"path", sourcepath, "error", err1)
			return nil, nil, map_os_error(location, err1, nil)
		}
		var f4 = New_range_reader(f3, extent)
		defer func() {
			var err4 = f4.Close()
			if err4 != nil {
				bbs.logger.Warn("op.Close() failed",
					"path", sourcepath, "error", err4)
				// IGNORE-ERRORS.
			}
		}()

		var md5, csum, err6 = bbs.copy_content_stream(object, scratch, source,
			size, checksum2, f4)
		if err6 != nil {
			return nil, nil, err6
		}
		return md5, csum, nil
	} else {
		var s_path = bbs.make_path_of_object(source, "")
		var t_path = bbs.make_path_of_object(scratch, "")
		var err3 = os.Remove(t_path)
		if err3 != nil && !errors.Is(err3, fs.ErrNotExist) {
			bbs.logger.Error("os.Remove() failed on a scratch file",
				"path", t_path, "error", err3)
			return nil, nil, map_os_error(location, err3, nil)
		}
		var err4 = os.Link(s_path, t_path)
		if err4 != nil {
			bbs.logger.Error("os.Link() failed on a scratch file",
				"source", s_path, "target", t_path, "error", err4)
			return nil, nil, map_os_error(location, err4, nil)
		}

		var md5, csum, err1 = bbs.calculate_csum2(object, checksum2, scratch, nil)
		if err1 != nil {
			return nil, nil, err1
		}
		return md5, csum, nil
	}
}

// COPY_CONTENT_STREAM copies the stream data (for upload or copy) to
// a temporary file.
func (bbs *Bb_server) copy_content_stream(object string, scratch string, source string, size int64, checksum2 types.ChecksumAlgorithm, body io.Reader) ([]byte, []byte, *Aws_s3_error) {
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
		var sourcepath string
		if source != "" {
			// Copying from a source file.
			sourcepath = bbs.make_path_of_object(source, "")
		} else {
			// Uploading from a request stream.
			sourcepath = "-"
		}

		var cc, err3 = io.Copy(f2, body)
		if err3 != nil {
			bbs.logger.Warn("io.Copy() failed for copying object",
				"path", sourcepath, "error", err3)
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

func (bbs *Bb_server) concat_parts_as_scratch(object string, scratch string, partlist *types.CompletedMultipartUpload, mpul *Mpul_info, checksum types.ChecksumAlgorithm) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(scratch, "")

	bbs.logger.Warn("IMPLEMENTATION OF CONCAT_PARTS() IS NAIVE AND SLOW")

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
		if checksum != "" {
			hash1 = checksum_algorithm(checksum)
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

	bb_assert(mpul.Initiated != nil)

	var err5 = os.Chtimes(path, time.Time{}, *mpul.Initiated)
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

// CALCULATE_CSUM2 calculates two checksums, md5 and one requested.
// It skips one when a checksum algorithm CHECKSUM="".  An algorithm
// is one of {CRC32, CRC32C, CRC64NVME, SHA1, SHA256}.  The file range
// EXTENT is checked being within the file size by the caller.
func (bbs *Bb_server) calculate_csum2(object string, checksum types.ChecksumAlgorithm, target string, extent *[2]int64) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(target, "")

	var stat, err1 = os.Lstat(path)
	if err1 != nil {
		bbs.logger.Info("os.Lstat() failed in calculating csum",
			"path", path, "error", err1)
		return nil, nil, map_os_error(location, err1, nil)
	}
	var f1, err2 = os.Open(path)
	if err2 != nil {
		bbs.logger.Warn("os.Open() failed", "path", path, "error", err2)
		return nil, nil, map_os_error(location, err2, nil)
	}
	var f2 = New_range_reader(f1, extent)
	defer func() {
		var err3 = f1.Close()
		if err3 != nil {
			bbs.logger.Warn("os.Close() failed", "path", path, "error", err3)
		}
	}()

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
		return nil, nil, map_os_error(location, err4, nil)
	}
	if cc != size {
		bbs.logger.Info("io.Copy() failed in calculating csum, bad copy size",
			"path", path, "extent-size", size, "copied-size", cc)
		var err5 = &Aws_s3_error{Code: InternalError,
			Message:  "io.Copy() failed, incomplete copy",
			Resource: location}
		return nil, nil, err5
	}

	var csum1 []byte = hash1.Sum(nil)
	var csum2 []byte
	if hash2 != nil {
		csum2 = hash2.Sum(nil)
	}
	return csum1, csum2, nil
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

		var md5b, crc, err1 = bbs.calculate_csum2(object, checksum2, scratch, nil)
		if err1 != nil {
			return nil, err1
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
		csum2 = crc
	} else {
		csum2 = csum1
	}
	return csum2, nil
}

// Note ETags are strong always.  Do not confuse md5.Sum(b) and
// md5.New().Sum(b).
func make_etag_from_stat(stat fs.FileInfo, ino uint64) string {
	var size = stat.Size()
	var mtime = stat.ModTime().UnixMicro()
	var b2 = make([]byte, 32)
	binary.LittleEndian.PutUint64(b2[0:], uint64(size))
	binary.LittleEndian.PutUint64(b2[8:], uint64(mtime))
	binary.LittleEndian.PutUint64(b2[16:], ino)
	binary.LittleEndian.PutUint64(b2[24:], uint64(0xdeadbeefdeadbeef))
	var md5v = md5.Sum(b2)
	return "\"" + base64.StdEncoding.EncodeToString(md5v[:]) + "\""
}
