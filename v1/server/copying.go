// copying.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Uploading and copying.  This is the main part of {CopyObject,
// PutObject, UploadPart, UploadPartCopy}.

package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	//"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"io"
	"io/fs"
	"time"
	//"log"
	//"net/url"
	"os"
	//"path"
	"path/filepath"
	//"strings"
)

// UPLOAD_OBJECT performs uploading.  Uploading is either for a file
// of an object or a file of a MPUL part.
func (bbs *Bb_server) upload_object(ctx context.Context, object string, part int32, upload_id string, body io.Reader, info *Meta_info, check upload_checks) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object
	var rid int64 = get_request_id(ctx)
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

	var err1 = bbs.make_intermediate_directories(target)
	if err1 != nil {
		return nil, nil, err1
	}

	var size int64 = check.size
	var err2 = bbs.upload_file_as_scratch(target, scratchkey, size, body)
	if err2 != nil {
		return nil, nil, err2
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(target, scratchkey)
		}
	}()

	var checksum = check.checksum
	var md5, csum1, err3 = bbs.calculate_csum2(checksum, target, scratchkey)
	if err3 != nil {
		return nil, nil, err3
	}

	var md5_to_check = check.md5_to_check
	if len(md5_to_check) != 0 && bytes.Compare(md5_to_check, md5) != 0 {
		bbs.logger.Info("Digests mismatch",
			"algorithm", "MD5",
			"passed", hex.EncodeToString(md5_to_check),
			"calculated", hex.EncodeToString(md5))
		var errz = &Aws_s3_error{Code: BadDigest,
			Resource: location}
		return nil, nil, errz
	}

	var csum_to_check = check.csum_to_check
	if len(csum_to_check) != 0 && bytes.Compare(csum_to_check, csum1) != 0 {
		bbs.logger.Info("Checksums mismatch",
			"algorithm", checksum,
			"passed", hex.EncodeToString(csum_to_check),
			"calculated", hex.EncodeToString(csum1))
		var errz = &Aws_s3_error{Code: BadDigest,
			Resource: location,
			Message:  "The checksum did not match what we received."}
		return nil, nil, errz
	}

	// SERIALIZE-ACCESSES.

	// It should be atomic on placing an uploaded file and saving a
	// metainfo file.  Failing to place an uploaded file will lose
	// old metainfo.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	// Re-check the MPUL upload-id after exclusion, when an upload is
	// for MPUL.

	var mpul *Mpul_info
	if part != 0 {
		var mpul1, err4 = bbs.fetch_mpul_info(object)
		if err4 != nil || mpul1.Upload_id != upload_id {
			bbs.logger.Info("Race on MPUL parts",
				"object", object)
			var errz = &Aws_s3_error{Code: NoSuchUpload,
				Resource: location}
			return nil, nil, errz
		}
		mpul = mpul1
	}

	var err6 = bbs.place_scratch_file(target, scratchkey, info)
	if err6 != nil {
		return nil, nil, err6
	}
	cleanup_needed = false

	var stat, _, err7 = bbs.check_object_status(target)
	if err7 != nil {
		return nil, nil, err7
	}
	var mtime = stat.ModTime()

	//var checksum types.ChecksumAlgorithm
	if part != 0 {
		checksum = mpul.Checksum_algorithm
	} else {
		checksum = ""
	}
	//var md5, csum1, err1 = bbs.calculate_csum2(checksum, target, "")
	//if err1 != nil {
	//	return nil, err1
	//}
	var csum = base64.StdEncoding.EncodeToString(csum1)
	var etag = make_etag_from_md5(md5)

	// Update MPUL parts catatlog information.

	if part != 0 {
		var partinfo = &Mpul_part{
			Size:     size,
			ETag:     etag,
			Checksum: csum,
			Mtime:    mtime,
		}
		var err8 = bbs.update_mpul_catalog(object, part, partinfo)
		if err8 != nil {
			return nil, nil, err8
		}
	}

	return md5, csum1, nil
}

// COPY_OBJECT performs copying.  Copying is either for a file of an
// object or a MPUL part.  See also upload_object().
func (bbs *Bb_server) copy_object(ctx context.Context, object string, part int32, upload_id string, source string, extent *[2]int64, info *Meta_info, check copy_checks) (*time.Time, *Aws_s3_error) {
	var location = "/" + object

	// A MPUL part does not have metainfo.

	bb_assert(!(part != 0) || info == nil)

	var copy_file_by_linking = (extent == nil)

	var rid int64 = get_request_id(ctx)
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

	var cleanup_needed = false
	if !copy_file_by_linking {
		var err6 = bbs.copy_file_as_scratch(ctx, target, scratchkey,
			source, nil)
		if err6 != nil {
			return nil, err6
		}
		cleanup_needed = true
		defer func() {
			if cleanup_needed {
				bbs.discard_scratch_file(target, scratchkey)
			}
		}()
	}

	if false {
		if bbs.conf.Verify_fs_write {
			var md5, _, err1 = bbs.calculate_csum2("", target, scratchkey)
			if err1 != nil {
				return nil, err1
			}
			if bytes.Compare(check.md5_to_check, md5) != 0 {
				bbs.logger.Error("Copying file failed, MD5 values differ",
					"source", hex.EncodeToString(check.md5_to_check),
					"target", hex.EncodeToString(md5))
				var errz = &Aws_s3_error{
					Code:     InternalError,
					Message:  "Copying file failed",
					Resource: location}
				return nil, errz
			}
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

	// Re-check the MPUL upload-id after exclusion, when an upload is
	// for MPUL.

	var mpul *Mpul_info
	if part != 0 {
		var mpul1, err4 = bbs.fetch_mpul_info(object)
		if err4 != nil || mpul1.Upload_id != upload_id {
			bbs.logger.Info("Race on MPUL parts",
				"object", object)
			var errz = &Aws_s3_error{Code: NoSuchUpload,
				Resource: location}
			return nil, errz
		}
		mpul = mpul1
	}

	if !copy_file_by_linking {
		var err16 = bbs.place_scratch_file(target, scratchkey, info)
		if err16 != nil {
			return nil, err16
		}
		cleanup_needed = false
	} else {
		var err1 = bbs.store_metainfo(target, info)
		if err1 != nil {
			return nil, err1
		}

		var s_path = bbs.make_path_of_object(source, "")
		var t_path = bbs.make_path_of_object(target, "")

		var err2 = os.Link(s_path, t_path)
		if err2 != nil {
			bbs.logger.Error("os.Link() failed on an object",
				"source", s_path, "object", t_path, "error", err2)
			return nil, map_os_error(location, err2, nil)
		}
	}

	var stat, _, err7 = bbs.check_object_status(target)
	if err7 != nil {
		return nil, err7
	}
	var size = stat.Size()
	var mtime = stat.ModTime()

	var checksum types.ChecksumAlgorithm
	if part != 0 {
		checksum = mpul.Checksum_algorithm
	} else {
		checksum = ""
	}
	var md5, csum1, err1 = bbs.calculate_csum2(checksum, target, "")
	if err1 != nil {
		return nil, err1
	}
	var csum = base64.StdEncoding.EncodeToString(csum1)
	var etag = make_etag_from_md5(md5)

	// Update MPUL parts catatlog information.

	if part != 0 {
		var partinfo = &Mpul_part{
			Size:     size,
			ETag:     etag,
			Checksum: csum,
			Mtime:    mtime,
		}
		var err8 = bbs.update_mpul_catalog(object, part, partinfo)
		if err8 != nil {
			return nil, err8
		}
	}

	return &mtime, nil
}

// UPLOAD_FILE_AS_SCRATCH stores the contents as a scratch file.  The
// work of renaming a scratch file to an actual file will be done in
// serialization.  Also, renaming should be in coordination with the
// the metainfo file.
func (bbs *Bb_server) upload_file_as_scratch(object, scratchkey string, size int64, body io.Reader) *Aws_s3_error {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, scratchkey)

	// Copy data to a temporary file.

	var f1, err4 = os.Create(path)
	if err4 != nil {
		bbs.logger.Info("os.Create() failed for uploading",
			"path", path, "error", err4)
		return map_os_error(location, err4, nil)
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

	var cc, err5 = io.Copy(f1, body)
	if err5 != nil {
		bbs.logger.Info("io.Copy() failed for uploading",
			"path", path, "error", err5)
		var m = map[error]Aws_s3_error_code{}
		var errz = map_os_error(location, err5, m)
		return errz
	}
	var err6 = f1.Close()
	if err6 != nil {
		bbs.logger.Info("os.Close() failed",
			"path", path, "error", err6)
		var m = map[error]Aws_s3_error_code{}
		var errz = map_os_error(location, err6, m)
		return errz
	}
	if cc != size {
		var msg = fmt.Sprintf("Body expected size=%d but received size=%d.",
			size, cc)
		bbs.logger.Info("Transfer failed", "message", msg)
		var errz = &Aws_s3_error{Code: IncompleteBody,
			Resource: location}
		return errz
	}

	cleanup_needed = false
	return nil
}

func (bbs *Bb_server) copy_file_as_scratch(ctx context.Context, object, scratchkey string, source string, extent *[2]int64) *Aws_s3_error {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, scratchkey)

	// Copy data to a temporary file.

	var f1, err1 = os.Create(path)
	if err1 != nil {
		bbs.logger.Info("os.Create() failed for copying",
			"path", path, "error", err1)
		return map_os_error(location, err1, nil)
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

	{
		var sourcepath = bbs.make_path_of_object(source, "")
		var f2, err1 = os.Open(sourcepath)
		if err1 != nil {
			bbs.logger.Warn("os.Open() failed for copy source",
				"path", sourcepath, "error", err1)
			return map_os_error(location, err1, nil)
		}
		var f3 = New_range_reader(f2, extent)
		/*
			if err2 != nil {
				bbs.logger.Warn("New_range_reader() failed for copying",
					"path", sourcepath, "error", err2)
				return map_os_error(location, err2, nil)
			}
		*/
		var _, err3 = io.Copy(f1, f3)
		if err3 != nil {
			bbs.logger.Warn("io.Copy() failed for copying object",
				"path", sourcepath, "error", err3)
			return map_os_error(location, err3, nil)
		}
		var err4 = f1.Close()
		if err4 != nil {
			bbs.logger.Warn("op.Close() failed",
				"path", path, "error", err4)
			// Ignore an error.
		}
	}

	var err4 = f1.Close()
	if err4 != nil {
		bbs.logger.Warn("op.Close() failed",
			"path", path, "error", err4)
		// IGNORE ERRORS.
	}

	cleanup_needed = false
	return nil
}

func (bbs *Bb_server) concat_parts_as_scratch(ctx context.Context, object, scratchkey string, partlist *types.CompletedMultipartUpload, mpul *Mpul_info) *Aws_s3_error {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, scratchkey)

	bbs.logger.Warn("IMPLEMENTATION OF CONCAT_PARTS() IS NAIVE AND SLOW")

	// Copy data to a temporary file.

	var f1, err1 = os.Create(path)
	if err1 != nil {
		bbs.logger.Info("os.Create() failed for concat parts",
			"path", path, "error", err1)
		return map_os_error(location, err1, nil)
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

	var mpulpath = bbs.make_path_of_object(object, "mpul")
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
		var f2, err1 = os.Open(partpath)
		if err1 != nil {
			bbs.logger.Warn("os.Open() failed for MPUL data",
				"path", partpath, "error", err1)
			return map_os_error(location, err1, nil)
		}
		var _, err2 = io.Copy(f1, f2)
		if err2 != nil {
			bbs.logger.Warn("io.Copy() failed for MPUL data",
				"path", partpath, "error", err2)
			return map_os_error(location, err2, nil)
		}
		var err3 = f2.Close()
		if err3 != nil {
			bbs.logger.Warn("op.Close() failed",
				"path", partpath, "error", err3)
			// Ignore an error.
		}
	}

	var err4 = f1.Close()
	if err4 != nil {
		bbs.logger.Warn("op.Close() failed",
			"path", path, "error", err4)
		// Ignore an error.
	}

	cleanup_needed = false

	var err5 = os.Chtimes(path, time.Time{}, mpul.Mtime)
	if err5 != nil {
		bbs.logger.Warn("op.Chtimes() failed",
			"path", path, "error", err5)
		// Ignore an error.
	}

	return nil
}

func (bbs *Bb_server) place_scratch_file(object, scratchkey string, info *Meta_info) *Aws_s3_error {
	var location = "/" + object
	var path1 = bbs.make_path_of_object(object, scratchkey)
	var path2 = bbs.make_path_of_object(object, "")

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

// DISCARD_SCRATCH_FILE removes a scratch file and a metainfo file.
// Errors are ignored.
func (bbs *Bb_server) discard_scratch_file(object, scratchkey string) error {
	var path1 = bbs.make_path_of_object(object, scratchkey)
	var err1 = os.Remove(path1)
	if err1 != nil {
		bbs.logger.Warn("os.Remove() failed on scratch file",
			"path", path1, "error", err1)
		// IGNORE ERRORS.
	}

	var path2 = bbs.make_path_of_object(object, "meta")
	var _, err2 = os.Lstat(path2)
	if err2 == nil || !errors.Is(err2, fs.ErrNotExist) {
		var err3 = os.Remove(path2)
		if err3 != nil {
			bbs.logger.Warn("os.Remove() failed on metainfo",
				"path", path2, "error", err3)
			// IGNORE ERRORS.
		}
	}
	return nil
}
