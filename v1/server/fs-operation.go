// fs-operation.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	//"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

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

// MAKE_PATH_OF_BUCKET makes an os-dependent path to a bucket, by
// appending a pool-directory and a bucket.  Note filepath.Join()
// calls filepath.Clean().
func (bbs *Bb_server) make_path_of_bucket(bucket string) string {
	var pool_directory = "."
	var path = filepath.Join(pool_directory, bucket)
	return path
}

// MAKE_SCRATCH_OBJECT_NAME makes an object name by appending a marker
// suffix.  A marker can be one of a null string, "@" plus a random
// for a scratch file, "@meta", or "@mpul" (mpul is for multipart
// upload).
func (bbs *Bb_server) make_scratch_object_name(object string, marker string) string {
	var dir, file = path.Split(object)
	var name = prefix_suffix_by_marker(file, marker)
	var scratch = path.Join(dir, name)
	return scratch
}

// MAKE_PATH_OF_OBJECT makes an os-dependent path to an object, by
// appending an object name with a marker suffix.  A marker can be one
// of a null string, a random for a scratch file, "@meta", or "@mpul"
// (mpul is for multipart upload).
func (bbs *Bb_server) make_path_of_object(object string, marker string) string {
	var dir, file = path.Split(object)
	var pool_directory = "."
	var name = prefix_suffix_by_marker(file, marker)
	var path = filepath.Join(pool_directory, dir, name)
	return path
}

func prefix_suffix_by_marker(file, marker string) string {
	var prefix, suffix string
	if marker == "" {
		prefix = ""
		suffix = ""
	} else {
		bb_assert(marker[0] == '@')
		prefix = "."
		suffix = marker
	}
	return (prefix + file + suffix)
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

func adjust_mpul_scratch_to_object_name(object string) string {
	var prefix = "."
	var suffix = "@" + "mpul"
	var dir, name1 = path.Split(object)
	var name2 = strings.TrimSuffix(strings.TrimPrefix(name1, prefix), suffix)
	var s2 = path.Join(dir, name2)
	return s2
}

func (bbs *Bb_server) check_bucket_directory_exists(rid uint64, bucket string) *Aws_s3_error {
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
			"rid", rid, "path", path)
		var err5 = &Aws_s3_error{Code: NoSuchBucket,
			Resource: location}
		return err5
	}
	return nil
}

// CREATE_MPUL_DIRECTORY creates a scratch directory for MPUL and
// populates it with an info file.  It may overtake an existing
// directory when it already exists, and rewrites its upload-id.
func (bbs *Bb_server) create_mpul_directory(rid uint64, object string, mpul *Mpul_info) *Aws_s3_error {
	var location = "/" + object + "@mpul"
	var path = bbs.make_path_of_object(object, "@mpul")

	// Make intermediate directories.

	var err1 = bbs.make_intervening_directories(rid, object)
	if err1 != nil {
		return err1
	}

	// Make or overtake an MPUL directory.

	var stat, err2 = os.Lstat(path)
	if err2 == nil && stat != nil && !stat.IsDir() {
		bbs.logger.Warn("MPUL temporary path is not a directory",
			"rid", rid, "path", path)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "An MPUL path is not a directory",
			Resource: location}
		return errz
	}
	if err2 != nil && !errors.Is(err2, fs.ErrNotExist) {
		bbs.logger.Warn("os.Lstat() on MPUL temporary failed",
			"rid", rid, "path", path, "error", err2)
		return map_os_error(location, err2, nil)
	}

	if err2 != nil {
		// Make a new directory.
		bb_assert(errors.Is(err2, fs.ErrNotExist))
		var err3 = os.Mkdir(path, 0755)
		if err3 != nil {
			bbs.logger.Warn("os.Mkdir() MPUL temporary failed",
				"rid", rid, "path", path, "error", err3)
			return map_os_error(location, err3, nil)
		}
	} else {
		// Overtake an existing directory.
		bbs.logger.Debug("Overtaking an existing MPUL directory",
			"rid", rid, "path", path)

		var oldmpul, err4 = bbs.fetch_mpul_info(rid, object, false)
		if err4 == nil {
			if *oldmpul.Upload_id == *mpul.Upload_id {
				bbs.logger.Error("Upload-ID conflicting (should never happen)",
					"rid", rid, "object", object)
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
			var err4 = bbs.discard_mpul_directory(rid, object)
			if err4 != nil {
				// IGNORE-ERRORS.
			}
		}
	}()

	// Store MPUL data.

	var err5 = bbs.store_mpul_info(rid, object, mpul)
	if err5 != nil {
		return err5
	}
	cleanup_needed = false

	return nil
}

// DISCARD_MPUL_DIRECTORY removes a directory for MPUL.  It does not
// remove intermediate directories.  Errors are ignored.
func (bbs *Bb_server) discard_mpul_directory(rid uint64, object string) error {
	var location = "/" + object + "@mpul"
	var path = bbs.make_path_of_object(object, "@mpul")

	var stat, err2 = os.Lstat(path)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
			return nil
		} else {
			// Ignore an error in taking stat.
			bbs.logger.Warn("os.Lstat() MPUL temporary failed",
				"rid", rid, "path", path, "error", err2)
		}
	} else if !stat.IsDir() {
		bbs.logger.Warn("An MPUL path is not a directory",
			"rid", rid, "path", path)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "An MPUL path is not a directory",
			Resource: location}
		return errz
	}

	var infopath = filepath.Join(path, "info")
	var err3 = os.Remove(infopath)
	if err3 != nil {
		bbs.logger.Warn("os.Remove() on MPUL info failed",
			"rid", rid, "path", infopath, "error", err3)
		// IGNORE-ERRORS.
	}
	var err4 = os.RemoveAll(path)
	if err4 != nil {
		bbs.logger.Info("os.RemoveAll() on MPUL temporary failed",
			"rid", rid, "path", path, "error", err4)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "Removing an MPUL scratch directory failed.",
			Resource: location}
		return errz
	}

	return nil
}

// MAKE_INTERVENING_DIRECTORIES makes directories to the object.
func (bbs *Bb_server) make_intervening_directories(rid uint64, object string) *Aws_s3_error {
	var location = "/" + object

	var err1 = bbs.check_path_is_link_free(rid, object)
	if err1 != nil {
		return err1
	}

	var path = bbs.make_path_of_object(object, "")
	var dirpath, _ = filepath.Split(path)
	var stat, err2 = os.Lstat(dirpath)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
			var err3 = os.MkdirAll(dirpath, 0755)
			if err3 != nil {
				bbs.logger.Info("os.MkdirAll() in making directories failed",
					"rid", rid, "path", dirpath, "error", err3)
				return map_os_error(location, err3, nil)
			}
			return nil
		} else {
			bbs.logger.Warn("os.Lstat() in making directories failed",
				"rid", rid, "path", dirpath, "error", err2)
			return map_os_error(location, err2, nil)
		}
	}
	if !stat.IsDir() {
		bbs.logger.Warn("Path is not a directory",
			"rid", rid, "path", dirpath)
		var errz = &Aws_s3_error{Code: AccessDenied,
			Resource: location}
		return errz
	}
	return nil
}

// CHECK_PATH_IS_LINK_FREE makes sure the given os-path to the object
// does not contain symbolic-links.  It includes the check of the
// object is a symbolic-link.
func (bbs *Bb_server) check_path_is_link_free(rid uint64, object string) *Aws_s3_error {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, "")
	var pathparts = strings.Split(path, string(os.PathSeparator))
	var descendpath = ""
	for _, e := range pathparts {
		descendpath = filepath.Join(descendpath, e)
		var info, err1 = os.Lstat(descendpath)
		if err1 != nil {
			if errors.Is(err1, fs.ErrNotExist) {
				// OK.
				return nil
			} else {
				bbs.logger.Warn("os.Lstat() in checking links failed",
					"rid", rid, "path", descendpath, "error", err1)
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

// FETCH_OBJECT_METAINFO fetches a metainfo file.  It returns nil if
// metainfo file does not exist or outdated.  (The object path is
// guaranteed for its properness by the caller).
func (bbs *Bb_server) fetch_object_metainfo(rid uint64, object string, entity string) (*Meta_info, *Aws_s3_error) {
	//var location = "/" + object
	var path = bbs.make_path_of_object(object, "@meta")

	var metainfo Meta_info
	var err5 = bbs.fetch_json_data(rid, object, path, &metainfo)
	if err5 == io.EOF {
		return nil, nil
	} else if err5 != nil {
		var errz = err5.(*Aws_s3_error)
		return nil, errz
	}
	if metainfo.Entity_key != entity {
		bbs.logger.Info("Metainfo file outdated",
			"rid", rid, "object", object, "entity1", entity,
			"entity2", metainfo.Entity_key)
		return nil, nil
	}
	return &metainfo, nil
}

// STORE_ETAG_AS_METAINFO() stores an ETag in metainfo (it is
// otherwise empty).  It is called when the object is large.  Storing
// metainfo serializes accesses inside the routine.
func (bbs *Bb_server) store_etag_as_metainfo(ctx context.Context, object string, entity string, etag string) *Aws_s3_error {
	var metainfo2 = &Meta_info{
		Entity_key:         entity,
		ETag:               etag,
		Checksum_algorithm: "",
		Checksum:           "",
		Headers:            nil,
		Tags:               nil,
	}
	var err6 = bbs.store_metainfo_serialized(ctx, object, metainfo2)
	if err6 != nil {
		return err6
	}
	return nil
}

// STORE_METAINFO_SERIALIZED calls store_object_metainfo() after
// serialization.  It check the entity-key as the object is not
// outdated.
func (bbs *Bb_server) store_metainfo_serialized(ctx context.Context, object string, metainfo *Meta_info) *Aws_s3_error {
	var location = "/" + object
	var _, rid = get_action_name(ctx)

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	var entity, _, err3 = bbs.check_object_exists(rid, object)
	if err3 != nil {
		return err3
	}
	if entity != metainfo.Entity_key {
		bbs.logger.Info("Race: Source object changed during operation",
			"rid", rid, "object", object)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "Source object changed during operation.",
			Resource: location}
		return errz
	}
	var err7 = bbs.store_object_metainfo(rid, object, metainfo)
	return err7
}

// STORE_OBJECT_METAINFO stores a metainfo file.  Passing nil deletes
// a metainfo file.  IMPORTANT NOTE: USE data=nil HERE FOR CALLING
// store_json_data().  This is to avoid the issue of typed-nil cannot
// be compared with untyped-nil.
func (bbs *Bb_server) store_object_metainfo(rid uint64, object string, metainfo *Meta_info) *Aws_s3_error {
	//var location = "/" + object
	// USE UNTYPED-NIL.
	var data any
	if metainfo == nil {
		data = nil
	} else {
		data = metainfo
	}
	var path = bbs.make_path_of_object(object, "@meta")
	var err5 = bbs.store_json_data(rid, object, path, data)
	if err5 != nil {
		return err5
	}
	return nil
}

// FETCH_MPUL_INFO fetches the stored MPUL information file.  It
// checks the required fields are non-nil.  It would remove an MPUL
// scratch directory when the content is broken (should never happen).
// SERIALIZING indicates it is called in access serialization.  The
// serializing status can be imprecise, and it is acceptable calling
// with serializing=false when the status is not certain.
func (bbs *Bb_server) fetch_mpul_info(rid uint64, object string, serializing bool) (*Mpul_info, error) {
	var location = "/" + object + "@mpul"
	var mpulpath = bbs.make_path_of_object(object, "@mpul")
	var path = filepath.Join(mpulpath, "info")
	var mpul Mpul_info
	var err5 = bbs.fetch_json_data(rid, object, path, &mpul)
	if err5 == io.EOF {
		bbs.logger.Warn("MPUL info file missing",
			"rid", rid, "path", path)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "MPUL record missing.",
			Resource: location}
		return nil, errz
	} else if err5 != nil {
		var errz = err5.(*Aws_s3_error)
		return nil, errz
	}

	// Check if the fields of MPUL info is Okay.

	if mpul.Initiate_time == nil || mpul.Upload_id == nil {
		bbs.logger.Error("MPUL info file broken",
			"rid", rid, "path", path)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "MPUL record broken.",
			Resource: location}

		// REMOVE MPUL DIRECTORY when in an access serialization.

		if serializing {
			var err6 = bbs.discard_mpul_directory(rid, object)
			if err6 != nil {
				// IGNORE-ERRORS.
			}
		}

		return nil, errz
	}

	return &mpul, nil
}

func (bbs *Bb_server) store_mpul_info(rid uint64, object string, mpul *Mpul_info) *Aws_s3_error {
	//var location = "/" + object + "@mpul"
	var mpulpath = bbs.make_path_of_object(object, "@mpul")
	var path = filepath.Join(mpulpath, "info")
	var err5 = bbs.store_json_data(rid, object, path, mpul)
	if err5 != nil {
		return err5
	}
	return nil
}

func (bbs *Bb_server) update_mpul_catalog(rid uint64, object string, part int32, partinfo *Mpul_part) *Aws_s3_error {
	var catalog, err4 = bbs.fetch_mpul_catalog(rid, object)
	if err4 != nil {
		return err4
	}
	if part > int32(len(catalog.Parts)) {
		var n = (part - int32(len(catalog.Parts)))
		var adds = make([]Mpul_part, n)
		catalog.Parts = append(catalog.Parts, adds...)
	}
	catalog.Parts[part-1] = *partinfo
	var err8 = bbs.store_mpul_catalog(rid, object, catalog)
	if err8 != nil {
		return err8
	}
	return nil
}

func (bbs *Bb_server) fetch_mpul_catalog(rid uint64, object string) (*Mpul_catalog, *Aws_s3_error) {
	var location = "/" + object + "@mpul"
	var mpulpath = bbs.make_path_of_object(object, "@mpul")
	var path = filepath.Join(mpulpath, "list")
	var catalog Mpul_catalog
	var err5 = bbs.fetch_json_data(rid, object, path, &catalog)
	if err5 == io.EOF {
		bbs.logger.Warn("Catalog file of MPUL missing",
			"rid", rid, "path", path)
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

func (bbs *Bb_server) store_mpul_catalog(rid uint64, object string, catalog *Mpul_catalog) *Aws_s3_error {
	//var location = "/" + object + "@mpul"
	var mpulpath = bbs.make_path_of_object(object, "@mpul")
	var path = filepath.Join(mpulpath, "list")
	var err5 = bbs.store_json_data(rid, object, path, catalog)
	if err5 != nil {
		return err5
	}
	return nil
}

// FETCH_JSON_DATA fetches the content in a metainfo file.  It returns
// io.EOF when the file does not exist.
func (bbs *Bb_server) fetch_json_data(rid uint64, object, path string, data any) error {
	var location = "/" + object
	var f1, err1 = os.Open(path)
	if err1 != nil {
		if errors.Is(err1, fs.ErrNotExist) {
			// Metainfo file does not exist.
			return io.EOF
		} else {
			var datatype = fmt.Sprintf("%T", data)
			bbs.logger.Error("os.Open() on fetching metafile failed",
				"rid", rid, "path", path, "type", datatype, "error", err1)
			return map_os_error(location, err1, nil)
		}
	}
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.logger.Warn("op.Close() on fetching metafile failed",
				"rid", rid, "path", path, "error", err2)
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
			bbs.logger.Error("json.Decode() on fetching metafile failed",
				"rid", rid, "path", path, "type", datatype, "error", err4)
			return map_os_error(location, err4, nil)
		}
	}
	return nil
}

// STORE_JSON_DATA stores the data in a metainfo file.  It removes a
// metainfo file when data=nil.
func (bbs *Bb_server) store_json_data(rid uint64, object, path string, data any) *Aws_s3_error {
	var location = "/" + object
	if data == nil {
		// Remove a metainfo file if exists.
		var _, err2 = os.Lstat(path)
		if err2 != nil {
			if errors.Is(err2, fs.ErrNotExist) {
				// OK.
				return nil
			} else {
				bbs.logger.Warn("os.Lstat() on storing metafile failed",
					"rid", rid, "path", path, "error", err2)
				return map_os_error(location, err2, nil)
			}
		}
		var err3 = os.Remove(path)
		if err3 != nil {
			bbs.logger.Warn("os.Remove() on storing metafile failed",
				"rid", rid, "path", path, "error", err3)
			return map_os_error(location, err3, nil)
		}
		return nil
	} else {
		var f1, err1 = os.Create(path)
		if err1 != nil {
			var datatype = fmt.Sprintf("%T", data)
			bbs.logger.Warn("os.Create() on storing metafile failed",
				"rid", rid, "path", path, "type", datatype, "error", err1)
			return map_os_error(location, err1, nil)
		}
		var cleanup_needed = true
		defer func() {
			var err2 = f1.Close()
			if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
				bbs.logger.Warn("fs.File.Close() on storing metafile failed",
					"rid", rid, "path", path, "error", err2)
			}
			if cleanup_needed {
				var err3 = os.Remove(path)
				if err3 != nil {
					bbs.logger.Warn("os.Remove() on storing metafile failed",
						"rid", rid, "path", path, "error", err3)
				}
			}
		}()

		var e = json.NewEncoder(f1)
		var err4 = e.Encode(data)
		if err4 != nil {
			var datatype = fmt.Sprintf("%T", data)
			bbs.logger.Info("json.Encode() on storing metafile failed",
				"rid", rid, "path", path, "type", datatype, "error", err4)
			return map_os_error(location, err4, nil)
		}
		cleanup_needed = false
		return nil
	}
}

// CHECK_UPLOAD_ONGOING checks "params.UploadId" is a currently
// on-going upload.  SERIALIZING is the serialization status that is
// passed to fetch_mpul_info.
func (bbs *Bb_server) check_upload_ongoing(rid uint64, object string, uploadid *string, serializing bool) (*Mpul_info, *Aws_s3_error) {
	var location = "/" + object
	if uploadid == nil {
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "UploadId missing.",
			Resource: location}
		return nil, errz
	}
	var mpul, err1 = bbs.fetch_mpul_info(rid, object, false)
	if err1 != nil || *mpul.Upload_id != *uploadid {
		if serializing {
			bbs.logger.Info("Race on MPUL, MPUL gone",
				"rid", rid, "object", object)
		}
		var errz = &Aws_s3_error{Code: NoSuchUpload,
			Resource: location}
		return nil, errz
	}
	return mpul, nil
}

// MAKE_FILE_STREAM makes a file stream of an object.  It obtains an
// entity-key after making a stream for checking consistency.  It
// fails when it detects changes on the object after checking
// conditions.
func (bbs *Bb_server) make_file_stream(rid uint64, object string, extent *[2]int64, entity string) (io.ReadCloser, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, "")

	var f1, err2 = os.Open(path)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			bbs.logger.Info("os.Open() for payload failed",
				"rid", rid, "path", path, "error", err2)
			var errz = &Aws_s3_error{Code: NoSuchKey,
				Resource: location}
			return nil, errz
		} else {
			bbs.logger.Warn("os.Open() for payload failed",
				"rid", rid, "path", path, "error", err2)
			return nil, map_os_error(location, err2, nil)
		}
	}

	var entity2, err3 = bbs.make_entity_key(rid, object, f1)
	if err3 != nil {
		return nil, err3
	}
	if entity2 != entity {
		var err4 = f1.Close()
		if err4 != nil {
			bbs.logger.Warn("fs.File.Close() for payload failed",
				"rid", rid, "path", path, "error", err4)
			// IGNORE-ERRORS.
		}
		bbs.logger.Info("Race: Source object changed during operation",
			"rid", rid, "object", object)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "Source object changed during operation.",
			Resource: location}
		return nil, errz
	}

	if extent == nil {
		return f1, nil
	} else {
		var f2 = New_range_reader(f1, extent)
		return f2, nil
	}
}

// CHECK_OBJECT_EXISTS obtains an entity-key and a stat on an object.
// It differs from fetch_object_status() as it returns an error, when
// an object does not exist.
func (bbs *Bb_server) check_object_exists(rid uint64, object string) (string, fs.FileInfo, *Aws_s3_error) {
	var location = "/" + object
	var entity, stat, err1 = bbs.fetch_object_status(rid, object, false)
	if err1 != nil {
		return entity, stat, err1
	}
	if stat == nil {
		var errz = &Aws_s3_error{Code: NoSuchKey,
			Resource: location}
		return entity, stat, errz
	}
	return entity, stat, err1
}

// FETCH_OBJECT_STATUS obtains an entity-key and a stat on an object.
// It will return (entity="",stat=nil,error=nil) when an object does
// not exist.  It can be used for checking existence.  Note
// non-regular files are never an object.  SERIALIZING is the status
// of access serialization.  Non-existence should be an error while
// serialization.
func (bbs *Bb_server) fetch_object_status(rid uint64, object string, serializing bool) (string, fs.FileInfo, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, "")

	var stat, err1 = os.Lstat(path)
	if err1 != nil && !errors.Is(err1, fs.ErrNotExist) {
		bbs.logger.Error("os.Lstat() on object failed",
			"rid", rid, "path", path, "error", err1)
		return "", nil, map_os_error(location, err1, nil)
	} else if err1 != nil {
		// errors.Is(err1, fs.ErrNotExist).
		if serializing {
			bbs.logger.Error("Race: Target object changed during operation",
				"rid", rid, "object", object)
			var errz = &Aws_s3_error{Code: InternalError,
				Message:  "Target object changed during operation.",
				Resource: location}
			return "", nil, errz
		} else {
			return "", nil, nil
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
			"rid", rid, "path", path, "mode", mode)
		var errz = &Aws_s3_error{Code: InvalidObjectState,
			Message:  "Named object is a non-regular file.",
			Resource: location}
		return "", nil, errz
	}

	var ino, ok = file_ino(stat, path)
	if !ok {
		log.Fatal("BAD-IMPL: Cannot take inode number")
	}
	var entity = hash_entity_key(stat, ino)
	return entity, stat, nil
}

// MAKE_OBJECT_ETAG_FROM_MD5 makes an ETag string from an MD5 value.
// Note ETags are strong always.
func make_object_etag_from_md5(md5v []byte) string {
	// return "\"" + base64.StdEncoding.EncodeToString(md5v[:]) + "\""
	return "\"" + hex.EncodeToString(md5v[:]) + "\""
}

// FETCH_OBJECT_ETAG returns an ETag and metainfo.  It calculates an
// MD5 sum of an object when metainfo is not stored.  IT TOOK TIME!
func (bbs *Bb_server) fetch_object_etag(rid uint64, object string, entity string) (string, *Meta_info, *Aws_s3_error) {
	// var location = "/" + object
	var metainfo, err2 = bbs.fetch_object_metainfo(rid, object, entity)
	if err2 != nil {
		// IGNORE-ERRORS.
	}
	if metainfo != nil {
		return metainfo.ETag, metainfo, nil
	}

	var checksum types.ChecksumAlgorithm = ""
	var md5v, _, _, err8 = bbs.calculate_csum2(rid, object, checksum, object, nil)
	if err8 != nil {
		return "", nil, err8
	}
	var etag = make_object_etag_from_md5(md5v)

	return etag, nil, nil
}

// MAKE_ENTITY_KEY returns an entity-key of a stream.
func (bbs *Bb_server) make_entity_key(rid uint64, object string, f fs.File) (string, *Aws_s3_error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, "")
	var stat, err1 = f.Stat()
	if err1 != nil {
		bbs.logger.Warn("fs.File.Stat() on object failed",
			"rid", rid, "path", path, "error", err1)
		return "", map_os_error(location, err1, nil)
	}
	var ino, ok = file_ino(stat, path)
	if !ok {
		log.Fatal("BAD-IMPL: Cannot take inode number")
	}
	var entity = hash_entity_key(stat, ino)

	return entity, nil
}

// MEMO: Do not confuse md5.Sum(b) and md5.New().Sum(b).
func hash_entity_key(stat fs.FileInfo, ino uint64) string {
	var size = stat.Size()
	var mtime = stat.ModTime().UnixMicro()
	var b2 = make([]byte, 32)
	binary.LittleEndian.PutUint64(b2[0:], uint64(size))
	binary.LittleEndian.PutUint64(b2[8:], uint64(mtime))
	binary.LittleEndian.PutUint64(b2[16:], ino)
	binary.LittleEndian.PutUint64(b2[24:], uint64(0xdeadbeefdeadbeef))
	var md5v = md5.Sum(b2)
	// return "\"" + base64.StdEncoding.EncodeToString(md5v[:]) + "\""
	return hex.EncodeToString(md5v[:])
	//return md5v[:]
}
