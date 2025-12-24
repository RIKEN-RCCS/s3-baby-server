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
	"encoding/hex"
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
// of an object or a file of a MPUL part.
func (bbs *Bb_server) upload_object(ctx context.Context, object string, part int32, upload_id string, body io.Reader, info *Meta_info, checks copy_checks, conditionals copy_conditionals) (fs.FileInfo, string, *Aws_s3_error) {
	var location = "/" + object
	var action, rid = get_request_action(ctx)
	var scratchkey = bbs.make_scratch_suffix(rid)
	defer bbs.discharge_scratch_suffix(rid)

	// TARGET is the copy destination.  It can be either an object or
	// a MPUL part file.

	var target string
	if part != 0 {
		bb_assert(upload_id != "")
		target = make_mpul_part_name(object, part)
	} else {
		target = object
	}

	var scratch = bbs.make_scratch_object_name(object, scratchkey)

	var err1 = bbs.make_intervening_directories(object)
	if err1 != nil {
		return nil, "", err1
	}

	var md5, csum, err2 = bbs.upload_file_as_scratch(object, scratch,
		checks.size_to_check, checks.checksum, body)
	if err2 != nil {
		return nil, "", err2
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(object, scratch)
		}
	}()

	if checks.md5_to_check != nil {
		if bytes.Compare(checks.md5_to_check, md5) != 0 {
			bbs.logger.Info("Checksums mismatch",
				"algorithm", "MD5",
				"passed", hex.EncodeToString(checks.md5_to_check),
				"calculated", hex.EncodeToString(md5))
			var errz = &Aws_s3_error{Code: BadDigest,
				Message:  "The md5 did not match what we received.",
				Resource: location}
			return nil, "", errz
		}
	}
	if checks.csum_to_check != nil {
		if bytes.Compare(checks.csum_to_check, csum) != 0 {
			bbs.logger.Info("Checksums mismatch",
				"algorithm", checks.checksum,
				"passed", hex.EncodeToString(checks.csum_to_check),
				"calculated", hex.EncodeToString(csum))
			var errz = &Aws_s3_error{Code: BadDigest,
				Message:  "The checksum did not match what we received.",
				Resource: location}
			return nil, "", errz
		}
	}

	// SERIALIZE-ACCESSES.

	// It should be atomic on placing an uploaded file and saving a
	// metainfo file.  Failing to place an uploaded file will lose
	// old metainfo.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, "", timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	// Re-check the MPUL upload-id after exclusion, when an upload is
	// for MPUL.

	if part != 0 {
		var mpul, err4 = bbs.fetch_mpul_info(object, true)
		if err4 != nil || *mpul.UploadId != upload_id {
			bbs.logger.Info("Race on MPUL, MPUL gone",
				"action", action, "object", object)
			var errz = &Aws_s3_error{Code: NoSuchUpload,
				Resource: location}
			return nil, "", errz
		}
	}

	// Conditionals for uploading are i.IfMatch or i.IfNoneMatch.

	if part == 0 {
		var err15 = bbs.check_request_conditionals(target, "write",
			conditionals)
		if err15 != nil {
			return nil, "", err15
		}
	}

	{
		var err6 = bbs.place_scratch_file(object, scratch, target, info)
		if err6 != nil {
			return nil, "", err6
		}
		cleanup_needed = false
	}

	var stat, etag, err7 = bbs.fetch_object_status(target)
	if err7 != nil {
		return nil, "", err7
	}
	if stat == nil {
		bbs.logger.Error("Race: Object gone while serialized",
			"action", action, "object", object, "target", target)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "Uploaded object gone",
			Resource: location}
		return nil, "", errz
	}

	// Update MPUL catatlog information.

	var size = stat.Size()
	var mtime = stat.ModTime()
	if part != 0 {
		var csum1 = base64.StdEncoding.EncodeToString(csum)
		var partinfo = &Mpul_part{
			Size:     size,
			ETag:     etag,
			Checksum: csum1,
			Mtime:    mtime,
		}
		var err8 = bbs.update_mpul_catalog(object, part, partinfo)
		if err8 != nil {
			return nil, "", err8
		}
	}

	return stat, etag, nil
}

// COPY_OBJECT performs copying.  Copying is either for a file of an
// object or a MPUL part.  The argument to copy_checks is always nil.
// See also upload_object().
func (bbs *Bb_server) copy_object(ctx context.Context, object string, part int32, upload_id string, source string, extent *[2]int64, info *Meta_info, checks copy_checks) (fs.FileInfo, string, *Aws_s3_error) {
	var location = "/" + object
	var action, rid = get_request_action(ctx)
	var scratchkey = bbs.make_scratch_suffix(rid)
	defer bbs.discharge_scratch_suffix(rid)

	// A MPUL part does not have metainfo.

	bb_assert(!(part != 0) || info == nil)

	var copy_file_by_linking = (extent == nil)

	// TARGET is the copy destination.  It can be either an object or
	// a MPUL part file.

	var target string
	if part != 0 {
		bb_assert(upload_id != "")
		target = make_mpul_part_name(object, part)
	} else {
		target = object
	}

	var scratch = bbs.make_scratch_object_name(object, scratchkey)

	if !copy_file_by_linking {
		var _, err6 = bbs.copy_file_as_scratch(object, scratch,
			source, nil)
		if err6 != nil {
			return nil, "", err6
		}
	} else {
		var s_path = bbs.make_path_of_object(source, "")
		var t_path = bbs.make_path_of_object(scratch, "")
		var err3 = os.Remove(t_path)
		if err3 != nil && !errors.Is(err3, fs.ErrNotExist) {
			bbs.logger.Error("os.Remove() failed on a scratch file",
				"path", t_path, "error", err3)
			return nil, "", map_os_error(location, err3, nil)
		}
		var err4 = os.Link(s_path, t_path)
		if err4 != nil {
			bbs.logger.Error("os.Link() failed on a scratch file",
				"source", s_path, "target", t_path, "error", err4)
			return nil, "", map_os_error(location, err4, nil)
		}
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(object, scratch)
		}
	}()

	if false {
		if bbs.conf.Verify_fs_write {
			var md5, _, err1 = bbs.calculate_csum2(object, "", scratch)
			if err1 != nil {
				return nil, "", err1
			}
			if bytes.Compare(checks.md5_to_check, md5) != 0 {
				bbs.logger.Error("Copying file failed, MD5 values differ",
					"source", hex.EncodeToString(checks.md5_to_check),
					"target", hex.EncodeToString(md5))
				var errz = &Aws_s3_error{
					Code:     InternalError,
					Message:  "Copying file failed",
					Resource: location}
				return nil, "", errz
			}
		}
	}

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, "", timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	// Re-check the MPUL upload-id after exclusion, when an upload is
	// for MPUL.

	var mpul *Mpul_info
	if part != 0 {
		var mpul1, err4 = bbs.fetch_mpul_info(object, true)
		if err4 != nil || *mpul1.UploadId != upload_id {
			bbs.logger.Info("Race on MPUL, MPUL gone",
				"action", action, "object", object)
			var errz = &Aws_s3_error{Code: NoSuchUpload,
				Resource: location}
			return nil, "", errz
		}
		mpul = mpul1
	}

	{
		var err16 = bbs.place_scratch_file(object, scratch, target, info)
		if err16 != nil {
			return nil, "", err16
		}
		cleanup_needed = false
	}

	var stat, etag, err7 = bbs.fetch_object_status(target)
	if err7 != nil {
		return nil, "", err7
	}
	if stat == nil {
		bbs.logger.Error("Race: Object gone while serialized",
			"action", action, "object", target)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "Uploaded object gone",
			Resource: location}
		return nil, "", errz
	}

	if part != 0 {
		var size = stat.Size()
		var mtime = stat.ModTime()

		var checksum = mpul.ChecksumAlgorithm
		var csum string
		if checksum != "" {
			var _, csum1, err1 = bbs.calculate_csum2(object, checksum, target)
			if err1 != nil {
				return nil, "", err1
			}
			csum = base64.StdEncoding.EncodeToString(csum1)
		} else {
			csum = ""
		}

		// Update MPUL parts catatlog information.

		var partinfo = &Mpul_part{
			Size:     size,
			ETag:     etag,
			Checksum: csum,
			Mtime:    mtime,
		}
		var err8 = bbs.update_mpul_catalog(object, part, partinfo)
		if err8 != nil {
			return nil, "", err8
		}
	}

	return stat, etag, nil
}

// CONCATENATE_OBJECT concatenates the parts to an object.  It returns
// stat and etag.
func (bbs *Bb_server) concatenate_object(ctx context.Context, object string, partlist *types.CompletedMultipartUpload, mpul *Mpul_info, checks copy_checks, conditionals copy_conditionals) (fs.FileInfo, string, *Aws_s3_error) {
	var location = "/" + object
	var action, rid = get_request_action(ctx)
	var scratchkey = bbs.make_scratch_suffix(rid)
	defer bbs.discharge_scratch_suffix(rid)

	var target = object
	var scratch = bbs.make_scratch_object_name(object, scratchkey)

	var checksum = checks.checksum
	var _, csum, err5 = bbs.concat_parts_as_scratch(object, scratch, partlist, mpul, checksum)
	if err5 != nil {
		return nil, "", err5
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(object, scratch)
		}
	}()

	if checks.csum_to_check != nil {
		bb_assert(csum != nil)
		if bytes.Compare(checks.csum_to_check, csum) != 0 {
			bbs.logger.Info("Checksums mismatch",
				"algorithm", checksum,
				"passed", hex.EncodeToString(checks.csum_to_check),
				"calculated", hex.EncodeToString(csum))
			var errz = &Aws_s3_error{Code: BadDigest,
				Message:  "The checksum did not match what we received.",
				Resource: location}
			return nil, "", errz
		}
	}

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, "", timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	{
		// Re-check the upload-id again after exclusion.

		var mpul2, err3 = bbs.check_upload_ongoing(object, mpul.UploadId)
		if err3 != nil {
			return nil, "", err3
		}
		var info = mpul2.MetaInfo

		var err7 = bbs.check_request_conditionals(object, "write",
			conditionals)
		if err7 != nil {
			return nil, "", err7
		}

		var err1 = bbs.place_scratch_file(object, scratch, target, info)
		if err1 != nil {
			return nil, "", err1
		}
		cleanup_needed = false

		var err2 = bbs.discard_mpul_directory(object)
		if err2 != nil {
			// IGNORE-ERRORS.
		}
	}

	var stat, etag, err7 = bbs.fetch_object_status(object)
	if err7 != nil {
		return nil, "", err7
	}
	if stat == nil {
		bbs.logger.Error("Race: Object gone while serialized",
			"action", action, "object", object)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "Uploaded object gone",
			Resource: location}
		return nil, "", errz
	}

	return stat, etag, nil
}

// UPLOAD_FILE_AS_SCRATCH stores the contents as a scratch file.
// Renaming a scratch file to an actual file will be done in
// serialization.
func (bbs *Bb_server) upload_file_as_scratch(object string, scratch string, size int64, checksum types.ChecksumAlgorithm, body io.Reader) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(scratch, "")

	// Copy data to a temporary file.

	var f1, err4 = os.Create(path)
	if err4 != nil {
		bbs.logger.Info("os.Create() failed for uploading",
			"path", path, "error", err4)
		return nil, nil, map_os_error(location, err4, nil)
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

	var body2 io.Reader
	if size != -1 {
		body2 = &io.LimitedReader{R: body, N: size}
	} else {
		body2 = body
	}

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

	var cc, err5 = io.Copy(f2, body2)
	if err5 != nil {
		bbs.logger.Info("io.Copy() failed for uploading",
			"path", path, "error", err5)
		var m = map[error]Aws_s3_error_code{}
		var errz = map_os_error(location, err5, m)
		return nil, nil, errz
	}
	if cc != size {
		bbs.logger.Info("Transfer truncated",
			"expected", size, "received", cc)
		var errz = &Aws_s3_error{Code: IncompleteBody,
			Resource: location}
		return nil, nil, errz
	}

	cleanup_needed = false

	var md5 []byte
	if hash1 != nil {
		md5 = hash1.Sum(nil)
	}
	var csum []byte
	if hash1 != nil {
		csum = hash2.Sum(nil)
	}
	return md5, csum, nil
}

func (bbs *Bb_server) copy_file_as_scratch(object string, scratch string, source string, extent *[2]int64) ([]byte, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(scratch, "")

	// Copy data to a temporary file.

	var f1, err1 = os.Create(path)
	if err1 != nil {
		bbs.logger.Info("os.Create() failed for copying",
			"path", path, "error", err1)
		return nil, map_os_error(location, err1, nil)
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
	if generate_md5_on_copy {
		hash1 = md5.New()
		f2 = io.MultiWriter(f1, hash1)
	} else {
		f2 = f1
	}

	{
		var sourcepath = bbs.make_path_of_object(source, "")
		var f3, err1 = os.Open(sourcepath)
		if err1 != nil {
			bbs.logger.Warn("os.Open() failed for copy source",
				"path", sourcepath, "error", err1)
			return nil, map_os_error(location, err1, nil)
		}
		var f4 = New_range_reader(f3, extent)
		var _, err3 = io.Copy(f2, f4)
		if err3 != nil {
			bbs.logger.Warn("io.Copy() failed for copying object",
				"path", sourcepath, "error", err3)
			return nil, map_os_error(location, err3, nil)
		}
		var err4 = f4.Close()
		if err4 != nil {
			bbs.logger.Warn("op.Close() failed",
				"path", sourcepath, "error", err4)
			// IGNORE-ERRORS.
		}
	}

	cleanup_needed = false

	var md5 []byte
	if hash1 != nil {
		md5 = hash1.Sum(nil)
	}
	return md5, nil
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
		var _, err2 = io.Copy(f2, f3)
		if err2 != nil {
			bbs.logger.Warn("io.Copy() failed for MPUL data",
				"path", partpath, "error", err2)
			return nil, nil, map_os_error(location, err2, nil)
		}
		var err3 = f3.Close()
		if err3 != nil {
			bbs.logger.Warn("op.Close() failed",
				"path", partpath, "error", err3)
			// IGNORE-ERRORS.
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

	/*
	var path2 = bbs.make_path_of_object(object, "@meta")
	var _, err2 = os.Lstat(path2)
	if err2 == nil || !errors.Is(err2, fs.ErrNotExist) {
		var err3 = os.Remove(path2)
		if err3 != nil {
			bbs.logger.Warn("os.Remove() failed on metainfo",
				"path", path2, "error", err3)
			// IGNORE-ERRORS.
		}
	}
	*/

	return nil
}

// CALCULATE_CSUM2 calculates two checksums, md5 and one requested.
// It skips one when algorithm="".  An algorithm is
// types.ChecksumAlgorithm and one of {CRC32, CRC32C, CRC64NVME, SHA1,
// SHA256}.
func (bbs *Bb_server) calculate_csum2(object string, checksum types.ChecksumAlgorithm, target string) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(target, "")

	var stat, err1 = os.Lstat(path)
	if err1 != nil {
		bbs.logger.Info("os.Lstat() failed in calculate_csum2",
			"path", path, "error", err1)
		return nil, nil, map_os_error(location, err1, nil)
	}
	var f1, err2 = os.Open(path)
	if err2 != nil {
		bbs.logger.Warn("os.Open() failed", "path", path, "error", err2)
		return nil, nil, map_os_error(location, err2, nil)
	}
	defer func() {
		var err3 = f1.Close()
		if err3 != nil {
			bbs.logger.Warn("os.Close() failed", "path", path, "error", err3)
		}
	}()

	var hash1 hash.Hash = md5.New()
	var hash2 hash.Hash = checksum_algorithm(checksum)
	var writer io.Writer
	if hash2 != nil {
		writer = io.MultiWriter(hash1, hash2)
	} else {
		writer = hash1
	}
	var count, err4 = io.Copy(writer, f1)
	if err4 != nil {
		return nil, nil, map_os_error(location, err4, nil)
	}
	if count != stat.Size() {
		bbs.logger.Info("io.Copy() failed, bad copy size")
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

// Note ETags are strong always.  Do not confuse md5.Sum(b) and
// md5.New().Sum(b).
func make_etag_from_stat(stat fs.FileInfo, ino uint64) string {
	var size = stat.Size()
	var mtime = stat.ModTime().UnixMicro()
	var b1 = []byte("The quick brown fox jumps over the lazy dog")
	var c = len(b1)
	var b2 = make([]byte, c+24)
	binary.LittleEndian.PutUint64(b2[0:], uint64(size))
	binary.LittleEndian.PutUint64(b2[8:], uint64(mtime))
	binary.LittleEndian.PutUint64(b2[16:], ino)
	copy(b2[24:], b1)
	var md5v = md5.Sum(b2)
	return "\"" + base64.StdEncoding.EncodeToString(md5v[:]) + "\""
}
