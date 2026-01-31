// fs-operation.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// MEMO: It avoids using "filepath" in other files that is OS dependent.

package server

import (
	//"bytes"
	"context"
	//"encoding/base64"
	//"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"io"
	"io/fs"
	"log"
	"time"
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

	//ContentDisposition *string
	//ContentEncoding    *string
	//ContentLanguage    *string
	//ContentType        *string
	//Expires            *time.Time
}

// MPUL-information.  It is stored in a file "info".  It corresponds
// to "types.MultipartUpload".
type Mpul_info struct {
	types.MultipartUpload
	MetaInfo *Meta_info
	// "types.MultipartUpload":
	// - ChecksumAlgorithm ChecksumAlgorithm
	// - ChecksumType ChecksumType
	// - Initiated *time.Time
	// - Initiator *Initiator
	// - Key *string
	// - Owner *Owner
	// - StorageClass StorageClass
	// - UploadId *string
}

// MPUL-catalog.  It is stored in a file "list".
type Mpul_catalog struct {
	//ChecksumAlgorithm types.ChecksumAlgorithm
	Parts []Mpul_part
}

// (types.CopyObjectResult, CopyPartResult)
type Mpul_part struct {
	Size     int64
	ETag     string
	Checksum string
	Mtime    time.Time
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

// MAKE_SCRATCH_OBJECT_NAME makes an object name by appending a marker
// suffix.  A marker can be one of a null string, a random for a
// scratch file, "@meta", or "@mpul" (mpul is for multipart upload).
func (bbs *Bb_server) make_scratch_object_name(object string, marker string) string {
	var prefix, suffix string
	if marker == "" {
		prefix = ""
		suffix = ""
	} else if marker[0] == '@' {
		prefix = "."
		suffix = marker
	} else {
		prefix = "."
		suffix = "@" + marker
	}
	var dir, file = path.Split(object)
	var scratch = path.Join(dir, (prefix + file + suffix))
	return scratch
}

// MAKE_PATH_OF_OBJECT makes an OS-path to an object, by appending an
// object name with a marker suffix.  A marker can be one of a null
// string, a random for a scratch file, "@meta", or "@mpul" (mpul is
// for multipart upload).
func (bbs *Bb_server) make_path_of_object(object string, marker string) string {
	var prefix, suffix string
	if marker == "" {
		prefix = ""
		suffix = ""
	} else if marker[0] == '@' {
		prefix = "."
		suffix = marker
	} else {
		prefix = "."
		suffix = "@" + marker
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
// populates it with an info file.  It may overtake an existing
// directory when it already exists, and rewrites its upload-id.
func (bbs *Bb_server) create_mpul_directory(ctx context.Context, object string, mpul *Mpul_info) *Aws_s3_error {
	var location = "/" + object + "@mpul"
	var path = bbs.make_path_of_object(object, "@mpul")

	// Make intermediate directories.

	var err1 = bbs.make_intervening_directories(object)
	if err1 != nil {
		return err1
	}

	// Make or overtake a MPUL directory.

	var stat, err2 = os.Lstat(path)
	if err2 == nil && stat != nil && !stat.IsDir() {
		bbs.logger.Warn("A MPUL path is not a directory", "path", path)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "A MPUL path is not a directory",
			Resource: location}
		return errz
	}
	if err2 != nil && !errors.Is(err2, fs.ErrNotExist) {
		bbs.logger.Warn("os.Lstat() failed",
			"path", path, "error", err2)
		return map_os_error(location, err2, nil)
	}

	if err2 != nil {
		// Make a new directory.
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

		var oldmpul, err4 = bbs.fetch_mpul_info(object, false)
		if err4 == nil {
			if *oldmpul.UploadId == *mpul.UploadId {
				bbs.logger.Error("Upload-ID conflicting (should never happen)",
					"object", object)
				var errz = &Aws_s3_error{Code: InternalError,
					Message:  "Upload-ID conflicting.",
					Resource: location}
				return errz
			}
		}
	}

	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			var err4 = bbs.discard_mpul_directory(object)
			if err4 != nil {
				// IGNORE-ERRORS.
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
	var path = bbs.make_path_of_object(object, "@mpul")

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
		bbs.logger.Warn("os.Remove() failed on MPUL info",
			"path", infopath, "error", err3)
		// IGNORE-ERRORS.
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

// MAKE_INTERVENING_DIRECTORIES makes directories to the object.
func (bbs *Bb_server) make_intervening_directories(object string) *Aws_s3_error {
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
	var path = bbs.make_path_of_object(object, "@meta")

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
// metainfo file.  Also, it deletes a metainfo file when all elements
// are nil.  IMPLEMENTATION NOTE: Do not assign info=nil, but use
// data=nil instead.  This is to avoid the issue of typed-nil!=nil.
func (bbs *Bb_server) store_metainfo(object string, info *Meta_info) *Aws_s3_error {
	//var location = "/" + object
	var data any
	if info == nil {
		data = nil
	} else if info.Headers == nil && info.Tags == nil {
		data = nil
	} else {
		data = info
	}
	var path = bbs.make_path_of_object(object, "@meta")
	var err5 = bbs.store_json_data(object, path, data)
	if err5 != nil {
		return err5
	}
	return nil
}

// FETCH_MPUL_INFO fetches the stored MPUL information file.  It
// checks the required fields are non-nil.  It would remove a MPUL
// scratch directory when the content is broken (should never happen).
// SERIALIZING indicates it is called in access serialization.  The
// serializing status can be imprecise, and it is acceptable calling
// with serializing=false when the status is not certain.
func (bbs *Bb_server) fetch_mpul_info(object string, serializing bool) (*Mpul_info, error) {
	var location = "/" + object + "@mpul"
	var mpulpath = bbs.make_path_of_object(object, "@mpul")
	var path = filepath.Join(mpulpath, "info")
	var mpul Mpul_info
	var err5 = bbs.fetch_json_data(object, path, &mpul)
	if err5 == io.EOF {
		bbs.logger.Warn("MPUL info file missing",
			"path", path)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "MPUL record missing.",
			Resource: location}
		return nil, errz
	} else if err5 != nil {
		var errz = err5.(*Aws_s3_error)
		return nil, errz
	}

	// Check if the fields of MPUL info is Okay.

	if mpul.Initiated == nil || mpul.UploadId == nil {
		bbs.logger.Error("MPUL info file broken",
			"path", path)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "MPUL record broken.",
			Resource: location}

		// REMOVE MPUL DIRECTORY when in an access serialization.

		if serializing {
			var err6 = bbs.discard_mpul_directory(object)
			if err6 != nil {
				// IGNORE-ERRORS.
			}
		}

		return nil, errz
	}

	return &mpul, nil
}

func (bbs *Bb_server) store_mpul_info(object string, mpul *Mpul_info) *Aws_s3_error {
	//var location = "/" + object + "@mpul"
	var mpulpath = bbs.make_path_of_object(object, "@mpul")
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
	var mpulpath = bbs.make_path_of_object(object, "@mpul")
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
	var mpulpath = bbs.make_path_of_object(object, "@mpul")
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
// on-going upload.  SERIALIZING is the serialization status that is
// passed to fetch_mpul_info.
func (bbs *Bb_server) check_upload_ongoing(object string, uploadid *string, serializing bool) (*Mpul_info, *Aws_s3_error) {
	var location = "/" + object
	if uploadid == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "UploadId missing.",
			Resource: location}
		return nil, errz
	}
	var mpul, err1 = bbs.fetch_mpul_info(object, false)
	if err1 != nil || *mpul.UploadId != *uploadid {
		if serializing {
			bbs.logger.Info("Race on MPUL, MPUL gone",
				"object", object)
		}
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
		//fmt.Printf("extent==nil\n")
		return f1, nil
	} else {
		var f2 = New_range_reader(f1, extent)
		return f2, nil
	}
}

// CHECK_OBJECT_EXISTS takes a stat() and etag on an object.  It
// differs from fetch_object_status() as it returns an error, when an
// object does not exist.
func (bbs *Bb_server) check_object_exists(object string) (fs.FileInfo, string, *Aws_s3_error) {
	var location = "/" + object
	var stat, etag, err1 = bbs.fetch_object_status(object, false)
	if err1 != nil {
		return stat, etag, err1
	}
	if stat == nil {
		var errz = &Aws_s3_error{Code: NoSuchKey,
			Resource: location}
		return stat, etag, errz
	}
	return stat, etag, err1
}

// FETCH_OBJECT_STATUS takes a stat() and etag on an object.  It can
// be used for checking existence.  It may return stat=nil when an
// object does not exist.  Note non-regular files are never an object.
// SERIALIZING is the serialization status.  Fetching should not fail
// when serializing.
func (bbs *Bb_server) fetch_object_status(object string, serializing bool) (fs.FileInfo, string, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, "")

	var stat, err1 = os.Lstat(path)
	if err1 != nil && !errors.Is(err1, fs.ErrNotExist) {
		bbs.logger.Error("os.Lstat() failed",
			"path", path, "error", err1)
		return nil, "", map_os_error(location, err1, nil)
	}
	if err1 != nil && errors.Is(err1, fs.ErrNotExist) {
		if serializing {
			bbs.logger.Error("RACE: Object gone while serialized",
				"object", object)
			var errz = &Aws_s3_error{Code: InternalError,
				Message:  "Uploaded object gone.",
				Resource: location}
			return nil, "", errz
		} else {
			return nil, "", nil
		}
	}

	var mode = stat.Mode()
	switch {
	case mode.IsRegular():
		// OK.
	case mode.IsDir():
		fallthrough
	case mode&fs.ModeSymlink != 0:
		fallthrough
	default:
		bbs.logger.Info("Object is not a regular file",
			"path", path, "mode", mode)
		var errz = &Aws_s3_error{Code: InvalidObjectState,
			Message:  "Named object is a non-regular file.",
			Resource: location}
		return nil, "", errz
	}

	var ino, ok = file_ino(stat, path)
	if !ok {
		log.Fatal("BAD-IMPL: Cannot take inode number")
	}
	var etag = make_etag_from_stat(stat, ino)

	return stat, etag, nil
}
