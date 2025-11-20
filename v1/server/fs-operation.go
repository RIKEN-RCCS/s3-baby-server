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
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type object_list_entry struct {
	key  string
	stat fs.FileInfo
}

// Meta-information associated to objects.  It is stored in a hidden
// file.  Headers stores "x-amz-meta-".  Tags stores tagging tags.  It
// will be encoded i json.
type Meta_info struct {
	Headers *map[string]string
	Tags    *types.Tagging
}

type upload_checks struct {
	location string
	size int64
	checksum types.ChecksumAlgorithm
	md5_to_check []byte
	csum_to_check []byte
}

type MPUL_info struct {
	Upload_id string
	Headers *map[string]string
	Tags    *types.Tagging
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
func map_os_error(location string, err1 error, m map[error]Aws_s3_error_code) error {
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
		var err5 = &Aws_s3_error{Code: code2, Resource: location}
		return err5
	} else {
		var err5 = &Aws_s3_error{Code: code1, Resource: location,
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
	var partname = fmt.Sprintf("part05d", part)
	var dir, file = path.Split(object)
	var name = path.Join(dir, (prefix + file + suffix), partname)
	return name
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
	var err2 = bbs.upload_file_scratch(object, scratchkey, size, body)
	if err2 != nil {
		return nil, nil, err2
	}
	var cleanup_needed = true
	defer func() {
		if cleanup_needed {
			bbs.discharge_scratch_file(object, scratchkey)
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

	var err4 = bbs.serialize_access(ctx, object, rid)
	if err4 != nil {
		return nil, nil, err4
	}
	defer bbs.release_access(ctx, object, rid)

	if info != nil {
		var err5 = bbs.store_metainfo(object, info)
		if err5 != nil {
			return nil, nil, err5
		}
	}
	var err6 = bbs.place_uploaded(object, scratchkey)
	if err6 != nil && info != nil {
		var _ = bbs.store_metainfo(object, nil)
	}
	if err6 != nil {
		return nil, nil, err6
	}

	cleanup_needed = false
	return md5, csum, nil
}

func (bbs *Bb_server) upload_file_scratch(object, scratchkey string, size int64, body io.Reader) error {
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
		bbs.Logger.Info("Transfer failed")
		var errz = &Aws_s3_error{Code: IncompleteBody,
			Resource: location,
			Message: fmt.Sprintf("Body expected length=%d but received length=%d.",
				size, cc)}
		return errz
	}

	// The work of renaming a temporary file to an actual file is
	// separated.  It should be in coordination with the meta-info
	// file.

	cleanup_needed = false
	return nil
}

func (bbs *Bb_server) place_uploaded(object, scratchkey string) error {
	var location = "/" + object
	var path1 = bbs.make_path_of_object(object, scratchkey)
	var path2 = bbs.make_path_of_object(object, "")

	var err8 = os.Rename(path1, path2)
	if err8 != nil {
		bbs.Logger.Info("io.Rename() failed", "error", err8)
		var errz = map_os_error(location, err8, nil)
		return errz
	}

	return nil
}

// CREATE_UPLOAD_DIRECTORY creates a scratch directory for MPUL.  It
// overtakes an existing directory when it already exists, and
// rewrites its upload-id.  It creates an empty file in the directory,
// which will later become a true file.  It is to keep ctime now.
func (bbs *Bb_server) create_upload_directory(ctx context.Context, object, uploadid string) error {
	var location = "/" + object + "@mpul"
	var path = bbs.make_path_of_object(object, "mpul")

	// Make intermediate directories.

	var err1 = bbs.make_intermediate_directories(object)
	if err1 != nil {
		return err1
	}

	// Make (or overtake) an MPUL directory.

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
		bbs.Logger.Warn("MPUL path is not a directory", "path", path)
		var errz = &Aws_s3_error{Code: AccessDenied,
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

	// Make MPUL data.

	var mpul = MPUL_info{uploadid, nil, nil}
	var infopath = filepath.Join(path, "info")
	var err5 = bbs.store_json_data(object, infopath, &mpul)
	if err5 != nil {
		return map_os_error(location, err5, nil)
	}

	// Create an empty file.

	var datapath = filepath.Join(path, "data")
	var err6 = bbs.create_empty_file(datapath)
	if err6 != nil {
		return map_os_error(location, err6, nil)
	}

	cleanup_needed = false
	return nil
}

// DISCHARGE_SCRATCH_FILE removes a scratch file and a metainfo file.
// Errors are ignored.
func (bbs *Bb_server) discharge_scratch_file(object, scratchkey string) error {
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
			bbs.Logger.Warn("os.Remove() failed",
				"file", path2, "error", err3)
			// Ignore an error.
		}
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
func (bbs *Bb_server) fetch_metainfo(ctx context.Context, object string) (*Meta_info, error) {
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
func (bbs *Bb_server) store_metainfo(object string, info *Meta_info) error {
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

func (bbs *Bb_server) fetch_mpul_info(object string) (*MPUL_info, error) {
	var location = "/" + object + "@mpul"
	var path = bbs.make_path_of_object(object, "mpul")
	var mpul = MPUL_info{}
	var infopath = filepath.Join(path, "info")
	var err5 = bbs.fetch_json_data(object, infopath, &mpul)
	if err5 != nil {
		return nil, map_os_error(location, err5, nil)
	}
	return &mpul, nil
}

func (bbs *Bb_server) make_file_stream(ctx context.Context, object string, extent []int64) (io.ReadCloser, error) {
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
		/*
			var pos, err1 = f1.Seek(extent[0], 0)
			if err1 != nil {
				return nil, err1
			}
			if pos < extent[0] {
				log.Fatalf("os.Seek returned incomplete")
				return nil, io.ErrUnexpectedEOF
			}
			var f2 = &io.LimitedReader{R: f1, N: extent[1] - extent[0]}
			return f2, nil
		*/

		var f2, err3 = New_range_reader(f1, [2]int64(extent[0:2]))
		return f2, err3
	}
}

func (bbs *Bb_server) fetch_file_stat(object string) (fs.FileInfo, error) {
	var location = "/" + object
	var path = bbs.make_path_of_object(object, "")

	var stat, err1 = os.Lstat(path)
	if err1 != nil {
		bbs.Logger.Info("os.Lstat() failed in fetch_file_stat",
			"file", path, "error", err1)
		return nil, map_os_error(location, err1, nil)
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

func (bbs *Bb_server) create_empty_file(path string) error {
	var f1, err6 = os.Create(path)
	if err6 != nil {
		bbs.Logger.Warn("os.Create() failed", "file", path, "error", err6)
		return err6
	}
	var err7 = f1.Close()
	if err7 != nil {
		bbs.Logger.Warn("op.Close() failed", "file", path, "error", err7)
		// Ignore.
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

// LIST_OBJECTS_DELIMITED makes listing for "/"-delimiter case.  It
// works with regard to the directory hierarchy.  A start-index and a
// start-marker indicates a start point.  Note the entries ReadDir()
// returns are sorted.  It returns a next start-index and a next
// start-marker, in addition to the entries.  THE ENTRIES INCLUDE
// DIRECTORIES EVEN IF THEY ARE EMPTY.
func (bbs *Bb_server) list_objects_delimited(bucket string, index int, marker string, maxkeys int, delimiter string, prefix string) ([]object_list_entry, int, string, error) {
	if delimiter != "/" {
		log.Fatalf("BAD-IMPL: list_objects_delimited with non-slash")
	}

	var location = "/" + bucket
	var dir1, fileprefix = path.Split(path.Clean(prefix))
	var pool_path = bbs.pool_path
	var name = path.Join(pool_path, bucket, dir1)

	var dir2, filemarker = path.Split(path.Clean(marker))
	if marker != "" {
		if dir1 != dir2 {
			// Nothing matches to the start-marker, return empty.
			return nil, 0, "", nil
		}
	}

	var entries1, err1 = os.ReadDir(name)
	if err1 != nil {
		return nil, 0, "", map_os_error(location, err1, nil)
	}

	// Filter entries by a prefix.

	var entries2 []os.DirEntry
	if fileprefix != "" {
		for _, e := range entries1 {
			if strings.HasPrefix(e.Name(), fileprefix) {
				entries2 = append(entries2, e)
			}
		}
	} else {
		entries2 = entries1
	}

	// Find a position of a start-marker, or use a start-index.

	var start int = -1
	if filemarker != "" {
		for i, e := range entries2 {
			if e.Name() == filemarker {
				start = i
				break
			}
		}
	} else {
		start = 0
	}
	if start == -1 {
		// Nothing matches to the start-marker, return empty.
		return nil, 0, "", nil
	}

	start = max(index, start)
	var entries3 = entries2[start:]
	var entries4 = entries3[:min(len(entries3), maxkeys)]
	var nextindex int
	var nextmarker string
	if len(entries4) < len(entries3) {
		nextindex = start + len(entries4)
		nextmarker = entries3[len(entries4)].Name()
	} else {
		nextindex = 0
		nextmarker = ""
	}

	var entries5 []object_list_entry
	for _, e := range entries4 {
		var key = path.Join(dir1, e.Name())
		var stat, err2 = e.Info()
		if err2 != nil {
			bbs.Logger.Info("os.Lstat() failed on os.DirEntry",
				"direntry", e, "error", err2)
			continue
		}
		entries5 = append(entries5, object_list_entry{key, stat})
	}

	return entries5, nextindex, nextmarker, nil
}

// LIST_OBJECTS_FLAT makes listing for general delimiter case (it
// works for both slash and non-slash delimiter).  It scans all the
// files in the bucket.  It uses WalkDir() in "io/fs" as it returns
// slash-paths (not os-specific).  In the scanning loop, it does not
// count directory entries.  COUNT counts files visited and it is used
// to check a start-index.  MEMO: A prefix should not have a
// preceeding delimiter.  A common-prefix will have a trailing
// delimiter.
func (bbs *Bb_server) list_objects_flat(bucket string, index int, marker string, maxkeys int, delimiter string, prefix string) ([]object_list_entry, int, string, error) {
	var location = "/" + bucket
	var pool_path = bbs.pool_path
	var name = path.Join(pool_path, bucket)

	var entries []object_list_entry
	var nextindex int = 0
	var nextmarker string = ""
	var count int = 0
	var collecting bool = false
	var commonprefix string = ""

	var bucket1 = os.DirFS(name)
	var err1 = fs.WalkDir(bucket1, "", func(path1 string, e fs.DirEntry, err1 error) error {
		// Skip errors or directories. (Don't count directories).

		if err1 != nil {
			bbs.Logger.Info("os.DirFS() callbacks with error",
				"bucket", name, "path", path1, "error", err1)
			return nil
		}
		if e.IsDir() {
			return nil
		}

		defer func() {
			count++
		}()

		// Check the start-marker first, then check the start-index.

		if marker != "" && !collecting {
			if marker == path1 {
				collecting = true
			} else {
				return nil
			}
		}
		if count < index {
			return nil
		}

		// Check the prefix.

		if !strings.HasPrefix(path1, prefix) {
			return nil
		}

		// Check a common prefix, and already encountered.

		var commonpart = check_common_prefix(path1, delimiter, prefix)
		if commonpart != "" {
			if commonprefix == commonpart {
				// Skip if it is the one encountered.
				return nil
			}
			commonprefix = commonpart
		}

		// Don't finish when fully collected.  It needs one extra
		// entry to check truncation.

		if len(entries) < maxkeys {
			var key = path1
			var stat, err2 = e.Info()
			if err2 != nil {
				bbs.Logger.Info("os.Lstat() failed on os.DirEntry",
					"direntry", e, "error", err2)
				return nil
			}
			entries = append(entries, object_list_entry{key, stat})
			return nil
		} else {
			nextindex = count
			nextmarker = path1
			return fs.SkipAll
		}
	})
	if err1 != nil {
		return nil, 0, "", map_os_error(location, err1, nil)
	}

	return entries, nextindex, nextmarker, nil
}

// It calculates MD5.
func (bbs *Bb_server) make_list_objects_entries(entries []object_list_entry, bucket string, delimiter string, prefix string, urlencode bool) ([]types.Object, []types.CommonPrefix, error) {
	var contents []types.Object
	var commonprefixes []types.CommonPrefix
	for _, e := range entries {
		var object = path.Join(bucket, e.key)
		var commonpart = check_common_prefix(e.key, delimiter, prefix)
		if commonpart == "" {
			var md5, _, err3 = bbs.calculate_csum2("", object, "")
			var etag *string
			if err3 != nil {
				bbs.Logger.Warn("MD5 calculation failed",
					"file", object, "error", err3)
				etag = nil
			} else {
				etag = make_etag_from_md5(md5)
			}
			var key string
			if urlencode {
				key = url.QueryEscape(e.key)
			} else {
				key = e.key
			}
			var size int64 = e.stat.Size()
			var mtime = e.stat.ModTime()
			var s = types.Object{
				// - ChecksumAlgorithm []ChecksumAlgorithm
				// - ChecksumType ChecksumType
				// - ETag *string
				// - Key *string
				// - LastModified *time.Time
				// - Owner *Owner
				// - RestoreStatus *RestoreStatus
				// - Size *int64
				// - StorageClass ObjectStorageClass
				Key:          &key,
				ETag:         etag,
				Size:         &size,
				LastModified: &mtime,
				StorageClass: types.ObjectStorageClassStandard}
			contents = append(contents, s)
		} else {
			var s = types.CommonPrefix{
				// - Prefix *string
				Prefix: &commonpart}
			commonprefixes = append(commonprefixes, s)
		}
	}
	return contents, commonprefixes, nil
}
