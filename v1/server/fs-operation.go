// fs-operation.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// MEMO: It avoids using "filepath" in other files that is OS dependent.

package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"io"
	"io/fs"
	"time"
	//"log"
	//"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Meta-information associated to an object.  It is stored in a hidden
// file.  Headers stores "x-amz-meta-".  Tags stores tagging tags.  It
// will be encoded in json.
type Meta_info struct {
	Headers map[string]string
	Tags    *types.Tagging
	//ETag *string
	//Checksum_algorithm types.ChecksumAlgorithm
	//Checksum *string
}

type Mpul_info struct {
	Upload_id          string
	Mtime              time.Time
	Checksum_type      types.ChecksumType
	Checksum_algorithm types.ChecksumAlgorithm
	Meta_info          *Meta_info
}

// (MultipartUpload)
type Mpul_catalog struct {
	Checksum_algorithm types.ChecksumAlgorithm
	Parts              []Mpul_part
}

// (types.CopyObjectResult, CopyPartResult)
type Mpul_part struct {
	Size     int64
	ETag     string
	Checksum string
	Mtime    time.Time
}

type upload_checks struct {
	size          int64
	checksum      types.ChecksumAlgorithm
	md5_to_check  []byte
	csum_to_check []byte
}

type copy_checks struct {
	size          int64
	checksum      *types.ChecksumAlgorithm
	md5_to_check  []byte
	csum_to_check []byte
}

func os_error_name(err error) string {
	if errors.Is(err, fs.ErrInvalid) {
		return "ErrInvalid"
	} else if errors.Is(err, fs.ErrPermission) {
		return "ErrPermission"
	} else if errors.Is(err, fs.ErrExist) {
		return "ErrExist"
	} else if errors.Is(err, fs.ErrNotExist) {
		return "ErrNotExist"
	} else if errors.Is(err, fs.ErrClosed) {
		return "ErrClosed"
	} else {
		// os.ErrNoDeadline
		// os.ErrDeadlineExceeded
		return "os-error-unknown"
	}
}

// Makes an AWS-S3 error from a given OS error.  Error codes may be
// replaced by a given map, to return something like
// "BucketAlreadyOwnedByYou" for fs.ErrExist.  A map accepts nil.
func map_os_error(location string, err1 error, m map[error]Aws_s3_error_code) *Aws_s3_error {
	var kind error
	var code1 Aws_s3_error_code
	if errors.Is(err1, fs.ErrInvalid) {
		kind = fs.ErrInvalid
		code1 = InvalidArgument
	} else if errors.Is(err1, fs.ErrPermission) {
		kind = fs.ErrPermission
		code1 = AccessDenied
	} else if errors.Is(err1, fs.ErrExist) {
		kind = fs.ErrExist
		code1 = InternalError
	} else if errors.Is(err1, fs.ErrNotExist) {
		kind = fs.ErrNotExist
		code1 = InternalError
	} else if errors.Is(err1, fs.ErrClosed) {
		kind = fs.ErrClosed
		code1 = InternalError
	} else {
		kind = nil
		code1 = InternalError
	}
	var code2, ok1 = m[kind]
	if ok1 {
		var err5 = &Aws_s3_error{Code: string(code2), Resource: location}
		return err5
	} else {
		var err5 = &Aws_s3_error{Code: string(code1), Resource: location,
			Message: err1.Error()}
		return err5
	}
}

func map_path_error(ctx context.Context, location string, err1 error, m map[error]Aws_s3_error_code) error {
	return err1
}

// MAKE_PATH_OF_BUCKET makes an OS-path to a bucket, by appending a
// pool-directory and a bucket.  Note Join() calls Clean().
func (bbs *Bb_server) make_path_of_bucket(bucket string) string {
	var pool_path = "."
	var path = filepath.Join(pool_path, bucket)
	return path
}

// MAKE_PATH_OF_OBJECT makes an OS-path to an object, by appending a
// pool-directory, a bucket, and a key.  A scratchkey is a random key,
// or it can be "meta" or "mpul" (multipart upload).
func (bbs *Bb_server) make_path_of_object(object string, scratchkey string) string {
	var prefix, suffix string
	if scratchkey == "" {
		prefix = ""
		suffix = ""
	} else {
		prefix = "."
		suffix = "@" + scratchkey
	}
	var dir, file = path.Split(object)
	var pool_path = "."
	var path = filepath.Join(pool_path, dir, (prefix + file + suffix))
	return path
}

func make_mpul_part_name(object string, part int32) string {
	var prefix = "."
	var suffix = "@mpul"
	var partname = make_part_name(part)
	var dir, file = path.Split(object)
	var name = path.Join(dir, (prefix + file + suffix), partname)
	return name
}

func make_part_name(part int32) string {
	return fmt.Sprintf("part%05d", part)
}

func make_mpul_scratch_name(name string) string {
	var prefix = "."
	var suffix = "@mpul"
	return (prefix + name + suffix)
}

func adjust_mpul_scratch_to_object_name(path1 string) string {
	var prefix = "."
	var suffix = "@" + "mpul"
	var dir, name1 = path.Split(path1)
	var name2 = strings.TrimSuffix(strings.TrimPrefix(name1, prefix), suffix)
	var s2 = path.Join(dir, name2)
	return s2
}

func (bbs *Bb_server) check_bucket_directory_exists(ctx context.Context, bucket string) *Aws_s3_error {
	var location = "/" + bucket
	var path = bbs.make_path_of_bucket(bucket)
	var stat, err2 = os.Lstat(path)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			var err5 = &Aws_s3_error{Code: NoSuchBucket,
				Resource: location}
			return err5
		} else {
			var m = map[error]Aws_s3_error_code{}
			var err5 = map_os_error(location, err2, m)
			return err5
		}
	}
	if !stat.IsDir() {
		bbs.logger.Info("Bucket name inhabited by non-directory",
			"path", path)
		var err5 = &Aws_s3_error{Code: NoSuchBucket,
			Resource: location}
		return err5
	}
	return nil
}

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

// DELETE_FILE removes an object and its metainfo.
func (bbs *Bb_server) delete_file(object string) error {
	var location = "/" + object
	var err6 = bbs.store_metainfo(object, nil)
	if err6 != nil {
		return err6
	}
	var path = bbs.make_path_of_object(object, "")
	var err7 = os.Remove(path)
	if err7 != nil {
		bbs.logger.Warn("os.Remove() failed on object",
			"path", path, "error", err7)
		var errz = map_os_error(location, err7, nil)
		return errz
	}
	return nil
}

// CREATE_MPUL_DIRECTORY creates a scratch directory for MPUL and
// populates it with a info file.  It may overtake an existing
// directory when it already exists, and rewrites its upload-id.
func (bbs *Bb_server) create_mpul_directory(ctx context.Context, object string, mpul *Mpul_info) *Aws_s3_error {
	var location = "/" + object + "@mpul"
	var path = bbs.make_path_of_object(object, "mpul")

	// Make intermediate directories.

	var err1 = bbs.make_intermediate_directories(object)
	if err1 != nil {
		return err1
	}

	// Make or overtake a MPUL directory.

	var stat, err2 = os.Lstat(path)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
		} else {
			bbs.logger.Warn("os.Lstat() failed",
				"path", path, "error", err2)
			return map_os_error(location, err2, nil)
		}
	}
	if stat != nil && !stat.IsDir() {
		bbs.logger.Warn("A MPUL path is not a directory", "path", path)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "A MPUL path is not a directory",
			Resource: location}
		return errz
	}

	if err2 != nil {
		bb_assert(errors.Is(err2, fs.ErrNotExist))
		var err3 = os.Mkdir(path, 0755)
		if err3 != nil {
			bbs.logger.Warn("os.Mkdir() failed",
				"path", path, "error", err3)
			return map_os_error(location, err3, nil)
		}
	} else {
		// Overtake an existing directory.
		bbs.logger.Debug("Overtaking an existing MPUL directory", "path", path)
	}

	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			var err4 = os.RemoveAll(path)
			if err4 != nil {
				bbs.logger.Info("os.RemoveAll() failed", "path", path,
					"error", err4)
			}
		}
	}()

	// Store MPUL data.

	var err5 = bbs.store_mpul_info(object, mpul)
	if err5 != nil {
		return err5
	}

	cleanup_needed = false
	return nil
}

// DISCARD_MPUL_DIRECTORY removes a directory for MPUL.  It does not
// remove intermediate directories.  Errors are ignored.
func (bbs *Bb_server) discard_mpul_directory(object string) error {
	var location = "/" + object + "@mpul"
	var path = bbs.make_path_of_object(object, "mpul")

	var stat, err2 = os.Lstat(path)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
			return nil
		} else {
			// Ignore an error in taking stat.
			bbs.logger.Warn("os.Lstat() failed",
				"path", path, "error", err2)
		}
	} else if !stat.IsDir() {
		bbs.logger.Warn("A MPUL path is not a directory", "path", path)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "A MPUL path is not a directory",
			Resource: location}
		return errz
	}

	var infopath = filepath.Join(path, "info")
	var err3 = os.Remove(infopath)
	if err3 != nil {
		// Ignore an error.
		bbs.logger.Warn("os.Remove() failed on MPUL info",
			"path", infopath, "error", err3)
	}
	var err4 = os.RemoveAll(path)
	if err4 != nil {
		bbs.logger.Info("os.RemoveAll() failed on MPUL directory",
			"path", path, "error", err4)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "Removing a MPUL scratch directory failed.",
			Resource: location}
		return errz
	}

	return nil
}

// MAKE_INTERMEDIATE_DIRECTORIES makes directories to the object.
func (bbs *Bb_server) make_intermediate_directories(object string) *Aws_s3_error {
	var location = "/" + object

	var err1 = bbs.check_path_is_link_free(object)
	if err1 != nil {
		return err1
	}

	var path = bbs.make_path_of_object(object, "")
	var dir, _ = filepath.Split(path)
	var stat, err2 = os.Lstat(dir)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
			var err3 = os.MkdirAll(dir, 0755)
			if err3 != nil {
				bbs.logger.Info("os.MkdirAll() failed in making directories",
					"path", dir, "error", err3)
				return map_os_error(location, err3, nil)
			}
			return nil
		} else {
			bbs.logger.Warn("os.Lstat() failed",
				"path", dir, "error", err2)
			return map_os_error(location, err2, nil)
		}
	}
	if !stat.IsDir() {
		bbs.logger.Warn("Path is not a directory", "path", dir)
		var errz = &Aws_s3_error{Code: AccessDenied,
			Resource: location}
		return errz
	}
	return nil
}

// CHECK_PATH_IS_LINK_FREE makes sure the given os-path to the object
// does not contain symbolic-links.  It includes the check of the
// object is a symbolic-link.
func (bbs *Bb_server) check_path_is_link_free(object string) *Aws_s3_error {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, "")
	var pathlist = strings.Split(path, string(os.PathSeparator))
	var name = ""
	for _, e := range pathlist {
		name = filepath.Join(name, e)
		var info, err1 = os.Lstat(name)
		if err1 != nil {
			if errors.Is(err1, fs.ErrNotExist) {
				// OK.
				return nil
			} else {
				bbs.logger.Warn("os.Lstat() failed",
					"path", name, "error", err1)
				return map_os_error(location, err1, nil)
			}
		}
		var mode = info.Mode()
		if mode&fs.ModeSymlink != 0 {
			var errz = &Aws_s3_error{Code: AccessDenied,
				Message:  "Path to object contains symbolic links.",
				Resource: location}
			return errz
		}
	}
	return nil
}

// FETCH_METAINFO fetches a metainfo file.  It returns nil if metainfo
// file does not exist.  (The object path is guaranteed its
// properness).
func (bbs *Bb_server) fetch_metainfo(object string) (*Meta_info, *Aws_s3_error) {
	//var location = "/" + object
	var path = bbs.make_path_of_object(object, "meta")

	var info Meta_info
	var err5 = bbs.fetch_json_data(object, path, &info)
	if err5 == io.EOF {
		return nil, nil
	} else if err5 != nil {
		var errz = err5.(*Aws_s3_error)
		return nil, errz
	}
	return &info, nil
}

// STORE_METAINFO stores a metainfo file.  Passing nil deletes a
// metainfo file.  Also, deletes a metainfo file all elements are nil.
func (bbs *Bb_server) store_metainfo(object string, info *Meta_info) *Aws_s3_error {
	//var location = "/" + object
	if info != nil && (info.Headers == nil && info.Tags == nil) {
		info = nil
	}

	var path = bbs.make_path_of_object(object, "meta")
	var err5 = bbs.store_json_data(object, path, info)
	if err5 != nil {
		return err5
	}
	return nil
}

func (bbs *Bb_server) fetch_mpul_info(object string) (*Mpul_info, error) {
	var location = "/" + object + "@mpul"
	var mpulpath = bbs.make_path_of_object(object, "mpul")
	var path = filepath.Join(mpulpath, "info")
	var mpul Mpul_info
	var err5 = bbs.fetch_json_data(object, path, &mpul)
	if err5 == io.EOF {
		bbs.logger.Warn("Metainfo file of MPUL missing",
			"path", path)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "Metainfo file of MPUL missing.",
			Resource: location}
		return nil, errz
	} else if err5 != nil {
		var errz = err5.(*Aws_s3_error)
		return nil, errz
	}
	return &mpul, nil
}

func (bbs *Bb_server) store_mpul_info(object string, mpul *Mpul_info) *Aws_s3_error {
	//var location = "/" + object + "@mpul"
	var mpulpath = bbs.make_path_of_object(object, "mpul")
	var path = filepath.Join(mpulpath, "info")
	var err5 = bbs.store_json_data(object, path, mpul)
	if err5 != nil {
		return err5
	}
	return nil
}

func (bbs *Bb_server) update_mpul_catalog(object string, part int32, partinfo *Mpul_part) *Aws_s3_error {
	var catalog, err4 = bbs.fetch_mpul_catalog(object)
	if err4 != nil {
		return err4
	}
	if part > int32(len(catalog.Parts)) {
		var n = (part - int32(len(catalog.Parts)))
		var adds = make([]Mpul_part, n)
		catalog.Parts = append(catalog.Parts, adds...)
	}
	catalog.Parts[part-1] = *partinfo
	var err8 = bbs.store_mpul_catalog(object, catalog)
	if err8 != nil {
		return err8
	}
	return nil
}

func (bbs *Bb_server) fetch_mpul_catalog(object string) (*Mpul_catalog, *Aws_s3_error) {
	var location = "/" + object + "@mpul"
	var mpulpath = bbs.make_path_of_object(object, "mpul")
	var path = filepath.Join(mpulpath, "list")
	var catalog Mpul_catalog
	var err5 = bbs.fetch_json_data(object, path, &catalog)
	if err5 == io.EOF {
		bbs.logger.Warn("Catalog file of MPUL missing",
			"path", path)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "Catalog file of MPUL missing.",
			Resource: location}
		return nil, errz
	} else if err5 != nil {
		var errz = err5.(*Aws_s3_error)
		return nil, errz
	}
	return &catalog, nil
}

func (bbs *Bb_server) store_mpul_catalog(object string, catalog *Mpul_catalog) *Aws_s3_error {
	//var location = "/" + object + "@mpul"
	var mpulpath = bbs.make_path_of_object(object, "mpul")
	var path = filepath.Join(mpulpath, "list")
	var err5 = bbs.store_json_data(object, path, catalog)
	if err5 != nil {
		return err5
	}
	return nil
}

// FETCH_JSON_DATA fetches the content in a metainfo file.  It returns
// io.EOF when the file does not exist.
func (bbs *Bb_server) fetch_json_data(object, path string, data any) error {
	var location = "/" + object
	var f1, err1 = os.Open(path)
	if err1 != nil {
		if errors.Is(err1, fs.ErrNotExist) {
			// Metainfo file does not exist.
			return io.EOF
		} else {
			var datatype = fmt.Sprintf("%T", data)
			bbs.logger.Error("os.Open() failed for metainfo",
				"path", path, "type", datatype, "error", err1)
			return map_os_error(location, err1, nil)
		}
	}
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.logger.Warn("op.Close() failed",
				"path", path, "error", err2)
		}
	}()

	var d = json.NewDecoder(f1)
	var err4 = d.Decode(&data)
	if err4 != nil {
		if err1 == io.EOF {
			return io.EOF
		} else {
			// The content broken.
			var datatype = fmt.Sprintf("%T", data)
			bbs.logger.Error("json.Decode() failed on metainfo",
				"path", path, "type", datatype, "error", err4)
			return map_os_error(location, err4, nil)
		}
	}
	return nil
}

// STORE_JSON_DATA stores the data in a metainfo file.  It removes a
// metainfo file when data=nil.
func (bbs *Bb_server) store_json_data(object, path string, data any) *Aws_s3_error {
	var location = "/" + object
	if data == nil {
		// Remove a metainfo file if exists.
		var _, err2 = os.Lstat(path)
		if err2 != nil {
			if errors.Is(err2, fs.ErrNotExist) {
				// OK.
				return nil
			} else {
				bbs.logger.Warn("os.Lstat() failed on metainfo",
					"path", path, "error", err2)
				return map_os_error(location, err2, nil)
			}
		}
		var err3 = os.Remove(path)
		if err3 != nil {
			bbs.logger.Warn("os.Remove() failed on metainfo",
				"path", path, "error", err3)
			return map_os_error(location, err3, nil)
		}
		return nil
	} else {
		var f1, err1 = os.Create(path)
		if err1 != nil {
			var datatype = fmt.Sprintf("%T", data)
			bbs.logger.Warn("os.Create() failed for metainfo",
				"path", path, "type", datatype, "error", err1)
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
				var err3 = os.Remove(path)
				if err3 != nil {
					bbs.logger.Warn("os.Remove() failed on metainfo",
						"path", path, "error", err3)
				}
			}
		}()

		var e = json.NewEncoder(f1)
		var err4 = e.Encode(data)
		if err4 != nil {
			var datatype = fmt.Sprintf("%T", data)
			bbs.logger.Info("json.Encode() failed on metainfo",
				"path", path, "type", datatype, "error", err4)
			return map_os_error(location, err4, nil)
		}
		cleanup_needed = false
		return nil
	}
}

// CHECK_UPLOAD_ONGOING checks "params.UploadId" is a currently
// on-going upload.
func (bbs *Bb_server) check_upload_ongoing(object string, uploadid *string) (*Mpul_info, *Aws_s3_error) {
	var location = "/" + object
	if uploadid == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "UploadId missing.",
			Resource: location}
		return nil, errz
	}
	var mpul, err1 = bbs.fetch_mpul_info(object)
	if err1 != nil || mpul.Upload_id != *uploadid {
		var errz = &Aws_s3_error{Code: NoSuchUpload,
			Resource: location}
		return nil, errz
	}
	return mpul, nil
}

func (bbs *Bb_server) make_file_stream(ctx context.Context, object string, extent *[2]int64) (io.ReadCloser, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, "")

	var f1, err2 = os.Open(path)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
			bbs.logger.Info("os.Open() failed for payload",
				"path", path, "error", err2)
			var errz = &Aws_s3_error{Code: NoSuchKey,
				Resource: location}
			return nil, errz
		} else {
			bbs.logger.Warn("os.Open() failed for payload",
				"path", path, "error", err2)
			return nil, map_os_error(location, err2, nil)
		}
	}
	if extent == nil {
		fmt.Printf("extent==nil\n")
		return f1, nil
	} else {
		var f2 = New_range_reader(f1, extent)
		return f2, nil
	}
}

// Takes a stat() on an object.  It is used to check the existence of
// an object.  It also checks the existence of metainfo.  Metainfo may
// be nil.
func (bbs *Bb_server) check_object_status(object string) (fs.FileInfo, *Meta_info, *Aws_s3_error) {
	var stat, err1 = bbs.fetch_object_status(object)
	if err1 != nil {
		return nil, nil, err1
	}
	var info, err2 = bbs.fetch_metainfo(object)
	if err2 != nil {
		return nil, nil, err2
	}
	return stat, info, nil
}

// Takes a stat() on an object.  Non-regular files are not an object.
func (bbs *Bb_server) fetch_object_status(object string) (fs.FileInfo, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, "")

	var stat, err1 = os.Lstat(path)
	if err1 != nil {
		bbs.logger.Info("os.Lstat() failed",
			"path", path, "error", err1)
		if errors.Is(err1, fs.ErrNotExist) {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message:  "No object as named.",
				Resource: location}
			return nil, errz
		} else {
			return nil, map_os_error(location, err1, nil)
		}
	}
	var mode = stat.Mode()
	switch {
	case mode.IsRegular():
		// OK.
	case mode&fs.ModeSymlink != 0:
		fallthrough
	default:
		bbs.logger.Info("An object path is not a regular file",
			"path", path, "mode", mode)
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "No object as named.",
			Resource: location}
		return nil, errz
	}
	return stat, nil
}

// check_common_prefix checks if a path has common-prefix part.  It
// returns a common-prefix or "".
func check_common_prefix(path, delimiter, prefix string) string {
	if delimiter == "" {
		return ""
	}
	var suffix = strings.TrimPrefix(path, prefix)
	var s2 = strings.SplitAfter(suffix, delimiter)
	if strings.HasSuffix(s2[0], delimiter) {
		return strings.Join([]string{prefix, s2[0]}, "")
	} else {
		return ""
	}
}
