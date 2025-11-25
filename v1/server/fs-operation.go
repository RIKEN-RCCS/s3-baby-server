// fs-operation.go
// Copyright 2025-2025 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// MEMO: It avoids using "filepath" in other files that is OS dependent.

package server

import (
	"context"
	"encoding/json"
	"bytes"
	"encoding/hex"
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
	Upload_id string
	Timestamp time.Time
	Checksum_type types.ChecksumType
	Checksum_algorithm types.ChecksumAlgorithm
	Meta_info *Meta_info
}

type Mpul_catalog struct {
	Checksum_algorithm types.ChecksumAlgorithm
	Parts []Mpul_part
}

type Mpul_part struct {
	Size int64
	ETag string
	Checksum string
	Mtime time.Time
}

type upload_checks struct {
	location string
	uploadid string
	size int64
	checksum types.ChecksumAlgorithm
	md5_to_check []byte
	csum_to_check []byte
	etag_condition [2]*string
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
		return "ErrUnknown"
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
			Message: os_error_name(kind)}
		return err5
	}
}

func map_path_error(ctx context.Context, location string, err1 error, m map[error]Aws_s3_error_code) error {
	return err1
}

// MAKE_PATH_OF_BUCKET makes an OS-path to a bucket, by appending a
// pool-directory and a bucket.  Note Join() calls Clean().
func (bbs *Bb_server) make_path_of_bucket(bucket string) string {
	var pool_path = bbs.pool_path
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
	var pool_path = bbs.pool_path
	var path = filepath.Join(pool_path, dir, (prefix + file + suffix))
	return path
}

func make_part_object_name(object string, part int32) string {
	var prefix = "."
	var suffix = "@meta"
	var partname = make_part_name(part)
	var dir, file = path.Split(object)
	var name = path.Join(dir, (prefix + file + suffix), partname)
	return name
}

func make_part_name(part int32) string {
	return fmt.Sprintf("part05d", part)
}

func (bbs *Bb_server) check_bucket_directory_exists(ctx context.Context, bucket string) error {
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
		var err5 = &Aws_s3_error{Code: NoSuchBucket,
			Resource: location}
		return err5
	}
	return nil
}

func (bbs *Bb_server) upload_file(ctx context.Context, object, scratchkey string, info *Meta_info, check upload_checks, body io.Reader) ([]byte, []byte, error) {
	var rid int64 = get_request_id(ctx)
	var location = check.location

	var err1 = bbs.make_intermediate_directories(object)
	if err1 != nil {
		return nil, nil, err1
	}

	var size int64 = check.size
	var err2 = bbs.upload_file_as_scratch(object, scratchkey, size, body)
	if err2 != nil {
		return nil, nil, err2
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discard_scratch_file(object, scratchkey)
		}
	}()

	var checksum  = check.checksum
	var md5, csum, err3 = bbs.calculate_csum2(checksum, object, scratchkey)
	if err3 != nil {
		return nil, nil, err3
	}

	var md5_to_check = check.md5_to_check
	if len(md5_to_check) != 0 && bytes.Compare(md5_to_check, md5) != 0 {
		bbs.Logger.Info("Digests mismatch",
			"algorithm", "MD5",
			"passed", hex.EncodeToString(md5_to_check),
			"calculated", hex.EncodeToString(md5))
		var errz = &Aws_s3_error{Code: BadDigest,
			Resource: location}
		return nil, nil, errz
	}

	var csum_to_check = check.csum_to_check
	if len(csum_to_check) != 0 && bytes.Compare(csum_to_check, csum) != 0 {
		bbs.Logger.Info("Checksums mismatch",
			"algorithm", checksum,
			"passed", hex.EncodeToString(csum_to_check),
			"calculated", hex.EncodeToString(csum))
		var errz = &Aws_s3_error{Code: BadDigest,
			Resource: location,
			Message:  "The checksum did not match what we received."}
		return nil, nil, errz
	}

	// It should be atomic on placing an uploaded file and saving a
	// meta-info file.  Failing to place an uploaded file will lose
	// old meta-info.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return nil, nil, timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	// Recheck the upload-id after exclusion, when an upload is for
	// UploadPart.

	if check.uploadid != "" {
		var mpul, err4 = bbs.fetch_mpul_info(object)
		if err4 != nil {
			var errz = &Aws_s3_error{Code: NoSuchUpload,
				Resource: location}
			return nil, nil, errz
		}
		if mpul.Upload_id != check.uploadid {
			var errz = &Aws_s3_error{Code: NoSuchUpload,
				Resource: location}
			return nil, nil, errz
		}
	}

	var err6 = bbs.place_scratch_file(object, scratchkey, info)
	if err6 != nil {
		return nil, nil, err6
	}

	cleanup_needed = false
	return md5, csum, nil
}

// UPLOAD_FILE_AS_SCRATCH stores the contents as a scratch file.  The
// work of renaming a scratch file to an actual file will be done in
// serialization.  Also, renaming should be in coordination with the
// the meta-info file.
func (bbs *Bb_server) upload_file_as_scratch(object, scratchkey string, size int64, body io.Reader) error {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, scratchkey)

	// Copy data to a temporary file.

	var f1, err4 = os.Create(path)
	if err4 != nil {
		bbs.Logger.Info("os.Create() failed", "file", path, "error", err4)
		return map_os_error(location, err4, nil)
	}
	var cleanup_needed = true
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.Logger.Warn("op.Close() failed", "file", path,
				"error", err2)
		}
		if cleanup_needed {
			var _ = os.Remove(path)
		}
	}()

	var cc, err5 = io.Copy(f1, body)
	if err5 != nil {
		bbs.Logger.Info("io.Copy() failed", "file", path, "error", err5)
		var m = map[error]Aws_s3_error_code{}
		var errz = map_os_error(location, err5, m)
		return errz
	}
	var err6 = f1.Close()
	if err6 != nil {
		bbs.Logger.Info("os.Close() failed", "file", path, "error", err6)
		var m = map[error]Aws_s3_error_code{}
		var errz = map_os_error(location, err6, m)
		return errz
	}
	if cc != size {
		var msg = fmt.Sprintf("Body expected size=%d but received size=%d.",
			size, cc)
		bbs.Logger.Info("Transfer failed", "message", msg)
		var errz = &Aws_s3_error{Code: IncompleteBody,
			Resource: location}
		return errz
	}

	cleanup_needed = false
	return nil
}

func (bbs *Bb_server) concat_parts_as_scratch(ctx context.Context, object, scratchkey string, partlist *types.CompletedMultipartUpload, mpul *Mpul_info) error {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, scratchkey)

	bbs.Logger.Warn("IMPLEMENTATION OF CONCAT_PARTS() IS NAIVE AND SLOW")

	// Copy data to a temporary file.

	var f1, err1 = os.Create(path)
	if err1 != nil {
		bbs.Logger.Info("os.Create() failed", "file", path, "error", err1)
		return map_os_error(location, err1, nil)
	}
	var cleanup_needed = true
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.Logger.Warn("op.Close() failed", "file", path,
				"error", err2)
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
			bbs.Logger.Warn("os.Open() failed for MPUL data",
				"file", partpath, "error", err1)
			return map_os_error(location, err1, nil)
		}
		var _, err2 = io.Copy(f1, f2)
		if err2 != nil {
			bbs.Logger.Warn("io.Copy() failed for MPUL data",
				"file", partpath, "error", err2)
			return map_os_error(location, err2, nil)
		}
		var err3 = f2.Close()
		if err3 != nil {
			bbs.Logger.Warn("op.Close() failed", "file", partpath,
				"error", err3)
			// Ignore an error.
		}
	}

	var err4 = f1.Close()
	if err4 != nil {
		bbs.Logger.Warn("op.Close() failed", "file", path,
			"error", err4)
		// Ignore an error.
	}

	cleanup_needed = false

	var err5 = os.Chtimes(path, time.Time{}, mpul.Timestamp)
	if err5 != nil {
		bbs.Logger.Warn("op.Chtimes() failed", "file", path,
			"error", err5)
		// Ignore an error.
	}

	return nil
}

func (bbs *Bb_server) copy_file_as_scratch(ctx context.Context, object, scratchkey string, source string) error {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, scratchkey)

	bbs.Logger.Warn("IMPLEMENTATION OF CONCAT_PARTS() IS NAIVE AND SLOW")

	// Copy data to a temporary file.

	var f1, err1 = os.Create(path)
	if err1 != nil {
		bbs.Logger.Info("os.Create() failed", "file", path, "error", err1)
		return map_os_error(location, err1, nil)
	}
	var cleanup_needed = true
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.Logger.Warn("op.Close() failed", "file", path,
				"error", err2)
		}
		if cleanup_needed {
			var _ = os.Remove(path)
		}
	}()

	{
		var sourcepath = bbs.make_path_of_object(source, "")
		var f2, err1 = os.Create(sourcepath)
		if err1 != nil {
			bbs.Logger.Warn("os.Open() failed for CopyObject",
				"file", sourcepath, "error", err1)
			return map_os_error(location, err1, nil)
		}
		var _, err2 = io.Copy(f1, f2)
		if err2 != nil {
			bbs.Logger.Warn("io.Copy() failed for CopyObject",
				"file", sourcepath, "error", err2)
			return map_os_error(location, err2, nil)
		}
		var err3 = f1.Close()
		if err3 != nil {
			bbs.Logger.Warn("op.Close() failed", "file", path,
				"error", err3)
			// Ignore an error.
		}
	}

	var err4 = f1.Close()
	if err4 != nil {
		bbs.Logger.Warn("op.Close() failed", "file", path,
			"error", err4)
		// Ignore an error.
	}

	cleanup_needed = false
	return nil
}

func (bbs *Bb_server) place_scratch_file(object, scratchkey string, info *Meta_info) error {
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
		bbs.Logger.Info("io.Rename() failed", "error", err8)
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
		bbs.Logger.Warn("os.Remove() failed on a scratch file",
			"file", path1, "error", err1)
		// Ignore an error.
	}

	var path2 = bbs.make_path_of_object(object, "meta")
	var _, err2 = os.Lstat(path2)
	if err2 == nil || !errors.Is(err2, fs.ErrNotExist) {
		var err3 = os.Remove(path2)
		if err3 != nil {
			bbs.Logger.Warn("os.Remove() failed on a metainfo file",
				"file", path2, "error", err3)
			// Ignore an error.
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
		bbs.Logger.Warn("os.Remove() failed on an object",
			"file", path, "error", err7)
		var errz = map_os_error(location, err7, nil)
		return errz
	}
	return nil
}

// CREATE_MPUL_DIRECTORY creates a scratch directory for MPUL and
// populates it with a info file.  It may overtake an existing
// directory when it already exists, and rewrites its upload-id.
func (bbs *Bb_server) create_mpul_directory(ctx context.Context, object string, mpul *Mpul_info) error {
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
			bbs.Logger.Warn("os.Lstat() failed on MPUL path",
				"path", path, "error", err2)
			return map_os_error(location, err2, nil)
		}
	}
	if !stat.IsDir() {
		bbs.Logger.Warn("A MPUL path is not a directory", "path", path)
		var errz = &Aws_s3_error{Code: InternalError,
			Message: "A MPUL path is not a directory",
			Resource: location}
		return errz
	}
	if err2 != nil {
		// Overtake an existing directory
		// assert(errors.Is(err2, fs.ErrNotExist))
		bbs.Logger.Debug("Overtaking an existing MPUL directory", "path", path)
	} else {
		var err3 = os.Mkdir(path, 0755)
		if err3 != nil {
			bbs.Logger.Info("os.Mkdir() failed", "path", path,
				"error", err3)
			return map_os_error(location, err3, nil)
		}
	}

	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			var err4 = os.RemoveAll(path)
			if err4 != nil {
				bbs.Logger.Info("os.RemoveAll() failed", "path", path,
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
			bbs.Logger.Warn("os.Lstat() failed on a MPUL directory",
				"path", path, "error", err2)
		}
	} else if !stat.IsDir() {
		bbs.Logger.Warn("A MPUL path is not a directory", "path", path)
		var errz = &Aws_s3_error{Code: InternalError,
			Message: "A MPUL path is not a directory",
			Resource: location}
		return errz
	}

	var infopath = filepath.Join(path, "info")
	var err3 = os.Remove(infopath)
	if err3 != nil {
		// Ignore an error.
		bbs.Logger.Warn("os.Remove() failed on a MPUL info file",
			"file", infopath, "error", err3)
	}
	var err4 = os.RemoveAll(path)
	if err4 != nil {
		bbs.Logger.Info("os.RemoveAll() failed on a MPUL directory",
			"path", path, "error", err4)
		var errz = &Aws_s3_error{Code: InternalError,
			Message: "Removing a MPUL scratch directory failed.",
			Resource: location}
		return errz
	}

	return nil
}

// MAKE_INTERMEDIATE_DIRECTORIES makes intermediate directories.
func (bbs *Bb_server) make_intermediate_directories(object string) error {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, "")
	var dir, _ = filepath.Split(path)
	var stat, err2 = os.Lstat(dir)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
		} else {
			bbs.Logger.Info("os.Lstat() failed in making directories",
				"path", dir, "error", err2)
			return map_os_error(location, err2, nil)
		}
	}
	if !stat.IsDir() {
		bbs.Logger.Warn("Path is not a directory", "path", dir)
		var errz = &Aws_s3_error{Code: AccessDenied,
			Resource: location}
		return errz
	}
	if err2 != nil {
		// assert(errors.Is(err2, fs.ErrNotExist))
		var err3 = os.MkdirAll(dir, 0755)
		if err3 != nil {
			bbs.Logger.Info("os.Mkdir() failed", "path", dir,
				"error", err3)
			return map_os_error(location, err3, nil)
		}
	}
	return nil
}

// Fetches a meta-info file.  It returns nil if meta-info does not
// exist.  (The object path is guaranteed its properness).
func (bbs *Bb_server) fetch_metainfo(object string) (*Meta_info, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, "meta")

	var f1, err2 = os.Open(path)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
			return nil, nil
		} else {
			bbs.Logger.Warn("os.Open() failed", "file", path,
				"error", err2)
			return nil, map_os_error(location, err2, nil)
		}
	}
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.Logger.Warn("op.Close() failed", "file", path,
				"error", err2)
		}
	}()

	var dec = json.NewDecoder(f1)
	var info Meta_info
	var err4 = dec.Decode(&info)
	if err4 != nil {
		bbs.Logger.Warn("BAD META-INFO FILE: The content broken",
			"file", path, "error", err4)
		return nil, map_os_error(location, err4, nil)
	}
	return &info, nil
}

// Stores a meta-info file.  The object path includes its bucket.
// Passing nil deletes a meta-info file.
func (bbs *Bb_server) store_metainfo(object string, info *Meta_info) *Aws_s3_error {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, "meta")

	if info == nil {
		// Remove a info file if exists.
		var _, err2 = os.Lstat(path)
		if err2 != nil {
			if errors.Is(err2, fs.ErrNotExist) {
				// OK.
				return nil
			} else {
				bbs.Logger.Warn("os.Lstat() failed in store_metainfo",
					"file", path, "error", err2)
				return map_os_error(location, err2, nil)
			}
		}
		var err3 = os.Remove(path)
		if err3 != nil {
			bbs.Logger.Warn("os.Remove() failed", "file", path, "error", err3)
			return map_os_error(location, err3, nil)
		}
		return nil
	} else {
		// Make a info file.
		var f1, err1 = os.Create(path)
		if err1 != nil {
			bbs.Logger.Warn("os.Create() failed", "file", path, "error", err1)
			return map_os_error(location, err1, nil)
		}
		defer func() {
			var err2 = f1.Close()
			if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
				bbs.Logger.Warn("op.Close() failed", "file", path,
					"error", err2)
			}
		}()

		var enc = json.NewEncoder(f1)
		var err4 = enc.Encode(info)
		if err4 != nil {
			bbs.Logger.Info("json.Encode() failed", "error", err4)
			var err5 = f1.Close()
			if err5 != nil {
				bbs.Logger.Warn("op.Close() failed", "file", path,
					"error", err5)
			}
			var err6 = os.Remove(path)
			if err6 != nil {
				bbs.Logger.Warn("op.Remove() failed", "file", path,
					"error", err6)
			}
			return map_os_error(location, err4, nil)
		}
		return nil
	}
}

func (bbs *Bb_server) fetch_mpul_info(object string) (*Mpul_info, error) {
	var location = "/" + object + "@mpul"
	var path = bbs.make_path_of_object(object, "mpul")
	var infopath = filepath.Join(path, "info")
	var mpul Mpul_info
	var err5 = bbs.fetch_json_data(object, infopath, &mpul)
	if err5 != nil {
		return nil, map_os_error(location, err5, nil)
	}
	return &mpul, nil
}

func (bbs *Bb_server) store_mpul_info(object string, mpul *Mpul_info) error {
	var location = "/" + object + "@mpul"
	var path = bbs.make_path_of_object(object, "mpul")
	var infopath = filepath.Join(path, "info")
	var err5 = bbs.store_json_data(object, infopath, mpul)
	if err5 != nil {
		return map_os_error(location, err5, nil)
	}
	return nil
}

func (bbs *Bb_server) fetch_mpul_catalog(object string) (*Mpul_catalog, error) {
	var location = "/" + object + "@mpul"
	var path = bbs.make_path_of_object(object, "mpul")
	var infopath = filepath.Join(path, "list")
	var catalog Mpul_catalog
	var err5 = bbs.fetch_json_data(object, infopath, &catalog)
	if err5 != nil {
		return nil, map_os_error(location, err5, nil)
	}
	return &catalog, nil
}

func (bbs *Bb_server) store_mpul_catalog(object string, catalog *Mpul_catalog) error {
	var location = "/" + object + "@mpul"
	var path = bbs.make_path_of_object(object, "mpul")
	var infopath = filepath.Join(path, "list")
	var err5 = bbs.store_json_data(object, infopath, catalog)
	if err5 != nil {
		return map_os_error(location, err5, nil)
	}
	return nil
}

func (bbs *Bb_server) fetch_json_data(object, path string, data any) error {
	var location = "/" + object
	var f1, err1 = os.Open(path)
	if err1 != nil {
		bbs.Logger.Warn("os.Open() failed", "file", path,
			"error", err1)
		return map_os_error(location, err1, nil)
	}
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.Logger.Warn("op.Close() failed", "file", path,
				"error", err2)
		}
	}()

	var d = json.NewDecoder(f1)
	var err4 = d.Decode(&data)
	if err4 != nil {
		var datatype = fmt.Sprintf("%T", data)
		bbs.Logger.Warn("json.Decode() failed",
			"file", path, "type", datatype, "error", err4)
		return map_os_error(location, err4, nil)
	}
	return nil
}

func (bbs *Bb_server) store_json_data(object, path string, data any) error {
	var location = "/" + object
	var f1, err1 = os.Create(path)
	if err1 != nil {
		bbs.Logger.Warn("os.Create() failed", "file", path, "error", err1)
		return map_os_error(location, err1, nil)
	}
	var cleanup_needed = true
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.Logger.Warn("op.Close() failed", "file", path,
				"error", err2)
		}
		if cleanup_needed {
			var err3 = os.Remove(path)
			if err3 != nil {
				bbs.Logger.Warn("os.Remove() failed",
					"file", path, "error", err3)
			}
		}
	}()

	var e = json.NewEncoder(f1)
	var err4 = e.Encode(data)
	if err4 != nil {
		var datatype = fmt.Sprintf("%T", data)
		bbs.Logger.Info("json.Encode() failed",
			"file", path, "type", datatype, "error", err4)
		return map_os_error(location, err4, nil)
	}
	cleanup_needed = false
	return nil
}

// CHECK_UPLOAD_GOING checks "params.UploadId" is a currently on-going
// upload.
func (bbs *Bb_server) check_upload_going(object string, uploadid *string) (*Mpul_info, error) {
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

func (bbs *Bb_server) make_file_stream(ctx context.Context, object string, extent *[2]int64) (io.ReadCloser, error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, "")

	var f1, err2 = os.Open(path)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
			bbs.Logger.Info("os.Open() failed", "file", path,
				"error", err2)
			var errz = &Aws_s3_error{Code: NoSuchKey,
				Resource: location}
			return nil, errz
		} else {
			bbs.Logger.Warn("os.Open() failed", "file", path,
				"error", err2)
			return nil, map_os_error(location, err2, nil)
		}
	}
	if extent == nil {
		fmt.Printf("extent==nil\n")
		return f1, nil
	} else {
		var f2, err3 = New_range_reader(f1, [2]int64(extent[0:2]))
		return f2, err3
	}
}

// Takes a stat() on an object.  It is used to check the existence of
// an object.
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
		bbs.Logger.Info("os.Lstat() failed on an object",
			"file", path, "error", err1)
		if errors.Is(err1, fs.ErrNotExist) {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message: "No object as named.",
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
		bbs.Logger.Info("An object path is not a regular file",
			"file", path, "mode", mode)
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message: "No object as named.",
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

func (bbs *Bb_server) create_empty_file(object, path string) error {
	var location = "/" + object
	var f1, err6 = os.Create(path)
	if err6 != nil {
		bbs.Logger.Warn("os.Create() failed", "file", path, "error", err6)
		return map_os_error(location, err6, nil)
	}
	var err7 = f1.Close()
	if err7 != nil {
		bbs.Logger.Warn("op.Close() failed", "file", path, "error", err7)
		// Ignore.
	}
	return nil
}
