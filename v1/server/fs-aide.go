// fs-aide.go
// Copyright 2025-2025 RIKEN R-CCS.
// SPDX-License-Identifier: BSD-2-Clause

// MEMO: It avoids using "filepath" in other files that is OS dependent.

package server

import (
	//"bytes"
	"context"
	//"crypto/md5"
	//"crypto/sha1"
	//"crypto/sha256"
	//"encoding/base64"
	"errors"
	//"hash"
	//"hash/crc32"
	//"hash/crc64"
	//"log/slog"
	//"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	//"log"
	"os"
	"path"
	"path/filepath"
	//"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	//"regexp"
	//"s3-baby-server/pkg/utils"
	//"strconv"
	"strings"
)

// Meta-information associated to objects.  It is stored in a hidden
// file.  Headers stores "x-amz-meta-".  Tags stores tagging tags.  It
// will be encoded i json.
type Meta_info struct {
	Headers map[string]string
	Tags *types.Tagging
}

func make_meta_file_name(file string) string {
	return ("." + file + "@meta")
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

// Appends a pool-directory and a bucket, where a bucket is assumed as
// a legal name.
func (bbs *Bb_server) make_path(bucket string) string {
	// file.Clean(path)
	var pool_path = bbs.pool_path
	var path = filepath.Join(pool_path, bucket)
	return path
}

func (bbs *Bb_server) check_bucket_directory_exists(ctx context.Context, bucket string) error {
	var location = "/" + bucket
	var path = bbs.make_path(bucket)
	var info, err2 = os.Lstat(path)
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
	if !info.IsDir() {
		var err5 = &Aws_s3_error{Code: NoSuchBucket,
			Resource: location}
		return err5
	}
	return nil
}

func (bbs *Bb_server) upload_file(ctx context.Context, object, scratch string, size int64, body io.Reader) error {
	var location = "/" + object
	//var dir1, filename = path.Split(object)
	//var dir2 = filepath.Clean(dir1)
	//var pool_path = bbs.pool_path
	//var dirpath = filepath.Join(pool_path, dir2)
	//var name = filepath.Join(dirpath, ("." + filename + "@" + suffix))
	var name = bbs.make_file_name_of_object(object, scratch)
	var dir, _ = filepath.Split(name)

	var info, err2 = os.Lstat(dir)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
		} else {
			bbs.Logger.Info("os.Lstat() failed in upload_file",
				"path", dir, "error", err2)
			return map_os_error(location, err2, nil)
		}
	}
	if !info.IsDir() {
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

	// Copy data to a temporary file.

	var f1, err4 = os.Create(name)
	if err4 != nil {
		bbs.Logger.Info("os.Create() failed", "file", name, "error", err4)
		return map_os_error(location, err4, nil)
	}
	var cleanup_needed = new(bool)
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.Logger.Warn("op.Close() failed", "file", name,
				"error", err2)
		}
		if *cleanup_needed {
			var _ = os.Remove(name)
		}
	}()

	var cc, err5 = io.Copy(f1, body)
	if err5 != nil {
		bbs.Logger.Info("io.Copy() failed", "file", name, "error", err5)
		var m = map[error]Aws_s3_error_code{}
		var errz = map_os_error(location, err5, m)
		return errz
	}
	var err6 = f1.Close()
	if err6 != nil {
		bbs.Logger.Info("os.Close() failed", "file", name, "error", err6)
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

	// Check MD5 of a temporary file.

	/*
	var md5, err7 = bbs.calculate_csum("MD5", name, "")
	if err7 != nil {
		var errz = &Aws_s3_error{Code: InternalError,
			Resource: location,
			Message:  fmt.Sprintf("md5 calculation failed")}
		return nil, errz
	}
	if len(md5_to_check) != 0 {
		if bytes.Compare(md5_to_check, md5) != 0 {
			var errz = &Aws_s3_error{Code: IncompleteBody,
				Resource: location,
				Message:  fmt.Sprintf("Body md5 unmatch")}
			return nil, errz
		}
	}
	*/

	// The work of renaming a temporary file to an actual file is
	// separated.  It should be in coordination with the meta-info
	// file.

	return nil
}

func (bbs *Bb_server) place_uploaded(ctx context.Context, object, suffix string) error {
	var location = "/" + object
	var dir1, file = path.Split(object)
	//var dir2, err1 = filepath.Localize(dir1)
	var dir2 = filepath.Clean(dir1)
	//if err1 != nil {
	//var errz = map_path_error(ctx, location, err1, nil)
	//return errz
	//}
	var pool_path = bbs.pool_path
	var name1 = filepath.Join(pool_path, dir2, ("." + file + "@" + suffix))
	var name2 = filepath.Join(pool_path, dir2, file)

	var err8 = os.Rename(name1, name2)
	if err8 != nil {
		bbs.Logger.Info("io.Rename() failed", "error", err8)
		var errz = map_os_error(location, err8, nil)
		return errz
	}

	return nil
}

// DISCHARGE_SCRATCH_FILE removes a scatch file as well as file
// suffixes for associated to a request-id.
func (bbs *Bb_server) discharge_scratch_file(ctx context.Context, object, scratch string, cleanup_needed *bool) {
	var rid int64 = get_request_id(ctx)

	bbs.mutex.Lock()
	defer bbs.mutex.Unlock()
	for k, v := range bbs.suffixes {
		if v.rid == rid {
			delete(bbs.suffixes, k)
		}
	}
	if *cleanup_needed {
		var name = bbs.make_file_name_of_object(object, scratch)
		var err1 = os.Remove(name)
		if err1 != nil {
			bbs.Logger.Info("os.Remove() failed on scratch file",
				"file", name, "error", err1)
		}
	}
}

// Fetches a meta-info file.  It returns nil if meta-info does not
// exist.  The object path is guaranteed its properness.
func (bbs *Bb_server) fetch_metainfo(ctx context.Context, object string) (*Meta_info, error) {
	var location = "/" + object
	var dir1, file = path.Split(object)
	var dir2 = filepath.Clean(dir1)
	var pool_path = bbs.pool_path
	var name = filepath.Join(pool_path, dir2, make_meta_file_name(file))

	var f1, err2 = os.Open(name)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
			return nil, nil
		} else {
			bbs.Logger.Warn("os.Open() failed", "file", name,
				"error", err2)
			return nil, map_os_error(location, err2, nil)
		}
	}
	defer func() {
		var err2 = f1.Close()
		if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
			bbs.Logger.Warn("op.Close() failed", "file", name,
				"error", err2)
		}
	}()

	var dec = json.NewDecoder(f1)
	var info Meta_info
	var err4 = dec.Decode(&info)
	if err4 != nil {
		bbs.Logger.Warn("BAD META-INFO FILE: The content broken",
			"file", name, "error", err4)
		return nil, map_os_error(location, err4, nil)
	}
	return &info, nil
}

// Stores a meta-info file.  The object path includes its bucket.
// Passing nil deletes a meta-info file.
func (bbs *Bb_server) store_metainfo(ctx context.Context, object string, info *Meta_info) error {
	var location = "/" + object
	var dir1, file = path.Split(object)
	var dir2 = filepath.Clean(dir1)
	var pool_path = bbs.pool_path
	var name = filepath.Join(pool_path, dir2, make_meta_file_name(file))

	if info == nil {
		// Remove a info file if exists.
		var _, err2 = os.Lstat(name)
		if err2 != nil {
			if errors.Is(err2, fs.ErrNotExist) {
				// OK.
				return nil
			} else {
				bbs.Logger.Warn("os.Lstat() failed in store_metainfo",
					"file", name, "error", err2)
				return map_os_error(location, err2, nil)
			}
		}
		var err3 = os.Remove(name)
		if err3 != nil {
			bbs.Logger.Warn("os.Remove() failed", "file", name, "error", err3)
			return map_os_error(location, err3, nil)
		}
		return nil
	} else {
		// Make a info file.
		var f1, err1 = os.Create(name)
		if err1 != nil {
			bbs.Logger.Warn("os.Create() failed", "file", name, "error", err1)
			return map_os_error(location, err1, nil)
		}
		defer func() {
			var err2 = f1.Close()
			if err2 != nil && !errors.Is(err2, fs.ErrClosed) {
				bbs.Logger.Warn("op.Close() failed", "file", name,
					"error", err2)
			}
		}()

		var enc = json.NewEncoder(f1)
		var err4 = enc.Encode(info)
		if err4 != nil {
			bbs.Logger.Info("json.Encode() failed", "error", err4)
			var err5 = f1.Close()
			if err5 != nil {
				bbs.Logger.Warn("op.Close() failed", "file", name,
					"error", err5)
			}
			var err6 = os.Remove(name)
			if err6 != nil {
				bbs.Logger.Warn("op.Remove() failed", "file", name,
					"error", err6)
			}
			return map_os_error(location, err4, nil)
		}
		return nil
	}
}

func (bbs *Bb_server) make_file_name_of_object(object string, scratch string) string {
	var prefix, suffix string
	if scratch == "" {
		prefix = ""
		suffix = ""
	} else {
		prefix = "."
		suffix = "@" + scratch
	}
	var dir1, file = path.Split(object)
	var dir2 = filepath.Clean(dir1)
	var pool_path = bbs.pool_path
	var name = filepath.Join(pool_path, dir2, (prefix + file + suffix))
	return name
}

func (bbs *Bb_server) make_file_stream(ctx context.Context, object string, extent []int64) (io.ReadCloser, error) {
	var location = "/" + object
	var dir1, file = path.Split(object)
	var dir2 = filepath.Clean(dir1)
	var pool_path = bbs.pool_path
	var name = filepath.Join(pool_path, dir2, file)

	var f1, err2 = os.Open(name)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
			bbs.Logger.Info("os.Open() failed", "file", name,
				"error", err2)
			var errz = &Aws_s3_error{Code: NoSuchKey,
				Resource: location}
			return nil, errz
		} else {
			bbs.Logger.Warn("os.Open() failed", "file", name,
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
	var dir1, file = path.Split(object)
	var dir2 = filepath.Clean(dir1)
	var pool_path = bbs.pool_path
	var name = filepath.Join(pool_path, dir2, file)

	var info, err1 = os.Lstat(name)
	if err1 != nil {
		bbs.Logger.Info("os.Lstat() failed in fetch_file_stat",
			"file", name, "error", err1)
		return nil, map_os_error(location, err1, nil)
	}
	return info, nil
}

// LIST_OBJECTS_delimited makes listing for "/"-delimiter case.  An
// index and a marker indicates a start point.
func (bbs *Bb_server) list_objects_delimited(bucket string, prefix string, index int, marker string, maxkeys int) ([]os.DirEntry, error) {
	var location = "/" + bucket
	var dir1, fileprefix = path.Split(path.Clean(prefix))
	var pool_path = bbs.pool_path
	var name = path.Join(pool_path, bucket, dir1)

	var dir2, filemarker = path.Split(path.Clean(marker))
	if marker != "" {
		if dir1 != dir2 || !strings.HasPrefix(filemarker, fileprefix) {
			// Nothing matches to the start-marker, thus, empty.
			return nil, nil
		}
	}

	// Note the entries ReadDir() returns are sorted.

	var entries1, err1 = os.ReadDir(name)
	if err1 != nil {
		return nil, map_os_error(location, err1, nil)
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

	var start int
	if filemarker != "" {
		for i, e := range entries2 {
			if e.Name() == filemarker {
				start = i
				break
			}
		}
	} else {
		start = index
	}
	var entries3 = entries2[start:]
	var entries4 = entries3[:min(len(entries3), maxkeys)]

	return entries4, nil
}
