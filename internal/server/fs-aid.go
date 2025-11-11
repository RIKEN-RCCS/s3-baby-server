// fs-aid.go
// Copyright 2025-2025 RIKEN R-CCS.
// SPDX-License-Identifier: BSD-2-Clause

// MEMO: It avoids using "filepath" in other files that is OS dependent.

package server

import (
	"context"
	"bytes"
	"crypto/md5"
	"errors"
	//"crypto/rand"
	//"encoding/base64"
	"encoding/json"
	//"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	//"math/big"
	"os"
	"path"
	"path/filepath"
	//"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	//"regexp"
	//"s3-baby-server/pkg/utils"
	//"strconv"
	//"strings"
)

type File_meta_info struct {
	Tags *types.Tagging
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
func map_os_error(ctx context.Context, location string, err1 error, m map[error]Aws_s3_error_code) error {
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
		var err5 = Aws_s3_Error{Code: code2, Resource: location}
		return &err5
	} else {
		var err5 = Aws_s3_Error{Code: code1, Resource: location,
			Message: os_error_name(kind)}
		return &err5
	}
}

func map_path_error(ctx context.Context, location string, err1 error, m map[error]Aws_s3_error_code) error {
	return err1
}

func calculate_md5(file string, logger *slog.Logger) ([]byte, error) {
	var info, err1 = os.Lstat(file)
	if err1 != nil {
		logger.Info("os.Lstat() for md5 failed", "error", err1)
		return nil, err1
	}
	var f, err2 = os.Open(file)
	if err2 != nil {
		logger.Info("os.Open() for md5 failed", "error", err2)
		return nil, err2
	}
	defer f.Close()
	var h = md5.New()
	var cc, err3 = io.Copy(h, f)
	if err3 != nil {
		logger.Info("io.Copy() for md5 failed", "error", err3)
		return nil, err3
	}
	if cc != info.Size() {
		logger.Info("io.Copy() for md5 failed, bad size")
		var err4 = fmt.Errorf("io.Copy() for md5 failed, bad size")
		return nil, err4
	}

	var sum []byte = h.Sum(nil)
	return sum, nil
}

// Appends a pool-directory and a bucket, where a bucket is assumed as
// a legal name.
func (bbs *Bb_server) make_path(bucket string) string {
	// file.Clean(path)
	var pool_path = bbs.S3.FileSystem.RootPath
	var path = filepath.Join(pool_path, bucket)
	return path
}

func (bbs *Bb_server) check_bucket_directory_exists(ctx context.Context, bucket string) error {
	var location = "/" + bucket
	var path = bbs.make_path(bucket)
	var info, err2 = os.Lstat(path)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			var err5 = Aws_s3_Error{Code: NoSuchBucket,
				Resource: location}
			return err5
		} else {
			var m = map[error]Aws_s3_error_code{}
			var err5 = map_os_error(ctx, location, err2, m)
			return err5
		}
	}
	if !info.IsDir() {
		var err5 = Aws_s3_Error{Code: NoSuchBucket,
			Resource: location}
		return err5
	}
	return nil
}

func (bbs *Bb_server) upload_file(ctx context.Context, object, suffix string, size int64, md5a []byte, body io.Reader) error {
	var location = "/" + object
	var dir1, filename = path.Split(object)
	var dir2, err1 = filepath.Localize(dir1)
	if err1 != nil {
		return err1
	}
	var pool_path = bbs.S3.FileSystem.RootPath
	var dirpath = filepath.Join(pool_path, dir2)
	var name = filepath.Join(dirpath, ("." + filename + "@" + suffix))

	var info, err2 = os.Lstat(dirpath)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
		} else {
			bbs.Logger.Info("os.Lstat() failed",
				"file", dirpath, "error", err2)
			return map_os_error(ctx, location, err2, nil)
		}
	}
	if !info.IsDir() {
		bbs.Logger.Warn("Path is not a directory", "path", dirpath)
		var errz = Aws_s3_Error{Code: AccessDenied,
			Resource: location}
		return errz
	}
	if err2 != nil {
		// assert(errors.Is(err2, fs.ErrNotExist))
		var err3 = os.MkdirAll(dirpath, 0755)
		if err3 != nil {
			bbs.Logger.Info("os.Mkdir() failed", "path", dirpath,
				"error", err3)
			return map_os_error(ctx, location, err3, nil)
		}
	}

	// Copy data to a temporary file.

	var f1, err4 = os.Create(name)
	if err4 != nil {
		bbs.Logger.Info("os.Create() failed", "file", name, "error", err4)
		return map_os_error(ctx, location, err4, nil)
	}
	var cleanup_needed = new(bool)
	defer func() {
		var err2 = f1.Close()
		if err2 != nil&& !errors.Is(err2, fs.ErrClosed) {
			bbs.Logger.Warn("op.Close() failed", "file", name,
				"error", err2)
		}
		if *cleanup_needed {
			var _ = os.Remove(name)
		}
	}()

	var cc, err5 = io.Copy(f1, body)
	if err5 != nil {
		bbs.Logger.Info("io.Copy() failed", "error", err5)
		var m = map[error]Aws_s3_error_code{}
		var errz = map_os_error(ctx, location, err5, m)
		return errz
	}
	var err6 = f1.Close()
	if err6 != nil {
		bbs.Logger.Info("os.Close() failed", "error", err6)
		var m = map[error]Aws_s3_error_code{}
		var errz = map_os_error(ctx, location, err6, m)
		return errz
	}
	if cc != size {
		bbs.Logger.Info("Transfer failed")
		var errz = Aws_s3_Error{Code: IncompleteBody,
			Resource: location,
			Message: fmt.Sprintf("Body expected length=%d but received length=%d",
				size, cc)}
		return errz
	}

	// Check MD5 of a temporary file.

	if (len(md5a) != 0) {
		var md5b, err7 = calculate_md5(name, bbs.Logger)
		if err7 != nil {
			var errz = Aws_s3_Error{Code: InternalError,
				Resource: location,
				Message: fmt.Sprintf("md5 calculation failed")}
			return errz
		}
		if bytes.Compare(md5a, md5b) != 0 {
			var errz = Aws_s3_Error{Code: IncompleteBody,
				Resource: location,
				Message: fmt.Sprintf("Body md5 unmatch")}
			return errz
		}
	}

	// The work of renaming a temporary file to an actual file is
	// separated.  It should be in coordination with the meta-info
	// file.

	return nil
}

func (bbs *Bb_server) place_uploaded(ctx context.Context, object, suffix string) error {
	var location = "/" + object
	var dir1, file = path.Split(object)
	var dir2, err1 = filepath.Localize(dir1)
	if err1 != nil {
		var errz = map_path_error(ctx, location, err1, nil)
		return errz
	}
	var pool_path = bbs.S3.FileSystem.RootPath
	var name1 = filepath.Join(pool_path, dir2, ("." + file + "@" + suffix))
	var name2 = filepath.Join(pool_path, dir2, file)

	var err8 = os.Rename(name1, name2)
	if err8 != nil {
		bbs.Logger.Info("io.Rename() failed", "error", err8)
		var errz = map_os_error(ctx, location, err8, nil)
		return errz
	}

	return nil
}

// Fetches a info file.  The object path includes its bucket.
func (bbs *Bb_server) fetch_file_meta_info(ctx context.Context, object string) (*File_meta_info, error) {
	var location = "/" + object
	var dir1, file = path.Split(object)
	var dir2, err1 = filepath.Localize(dir1)
	if err1 != nil {
		return nil, map_path_error(ctx, location, err1, nil)
	}
	var pool_path = bbs.S3.FileSystem.RootPath
	var name = filepath.Join(pool_path, dir2, ("." + file + "@meta"))

	var f1, err2 = os.Open(name)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
			return nil, nil
		} else {
			bbs.Logger.Warn("os.Open() failed", "file", name,
				"error", err2)
			return nil, map_os_error(ctx, location, err2, nil)
		}
	}
	defer func() {
		var err2 = f1.Close()
		if err2 != nil&& !errors.Is(err2, fs.ErrClosed) {
			bbs.Logger.Warn("op.Close() failed", "file", name,
				"error", err2)
		}
	}()

	var dec = json.NewDecoder(f1)
	var meta File_meta_info
	var err4 = dec.Decode(&meta)
	if err4 != nil {
		bbs.Logger.Warn("BAD META-INFO FILE: The content broken",
			"file", name, "error", err4)
		return nil, map_os_error(ctx, location, err4, nil)
	}
	return &meta, nil
}

// Stores a info file.  The object path includes its bucket.
func (bbs *Bb_server) store_file_meta_info(ctx context.Context, object string, info *File_meta_info) error {
	var location = "/" + object
	var dir1, file = path.Split(object)
	var dir2, err1 = filepath.Localize(dir1)
	if err1 != nil {
		var errz = map_path_error(ctx, location, err1, nil)
		return errz
	}
	var pool_path = bbs.S3.FileSystem.RootPath
	var name = filepath.Join(pool_path, dir2, ("." + file + "@meta"))

	if info == nil {
		// Remove a info file if exists.
		var _, err2 = os.Lstat(name)
		if err2 != nil {
			if errors.Is(err2, fs.ErrNotExist) {
				// OK.
				return nil
			} else {
				bbs.Logger.Warn("os.Lstat() failed", "file", name,
					"error", err2)
				return map_os_error(ctx, location, err2, nil)
			}
		}
		var err3 = os.Remove(name)
		if err3 != nil {
			bbs.Logger.Warn("os.Remove() failed", "file", name, "error", err3)
			return map_os_error(ctx, location, err3, nil)
		}
		return nil
	} else {
		// Make a info file.
		var f1, err1 = os.Create(name)
		if err1 != nil {
			bbs.Logger.Warn("os.Create() failed", "file", name, "error", err1)
			return map_os_error(ctx, location, err1, nil)
		}
		defer func() {
			var err2 = f1.Close()
			if err2 != nil&& !errors.Is(err2, fs.ErrClosed) {
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
			return map_os_error(ctx, location, err4, nil)
		}
		return nil
	}
}
