// fs-aid.go

package server

import (
	"context"
	"bytes"
	"crypto/md5"
	"errors"
	//"crypto/rand"
	//"encoding/base64"
	//"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	//"math/big"
	"os"
	"path"
	"path/filepath"
	//"regexp"
	//"s3-baby-server/pkg/utils"
	//"strconv"
	//"strings"
)

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

func (bbs *Bb_server) upload_file(ctx context.Context, bucket, object string, size int64, md5 *string, body io.Reader) error {
	var location = "/" + bucket
	var dir1, filename = path.Split(object)
	var dir2, err1 = filepath.Localize(dir1)
	if err1 != nil {
		return err1
	}
	var pool_path = bbs.S3.FileSystem.RootPath
	var dirpath = filepath.Join(pool_path, bucket, dir2)

	var info, err2 = os.Lstat(dirpath)
	if err2 != nil {
		if errors.Is(err2, fs.ErrNotExist) {
			// OK.
		} else {
			var m = map[error]Aws_s3_error_code{}
			var errz = map_os_error(ctx, location, err2, m)
			return errz
		}
	}
	if !info.IsDir() {
		var errz = Aws_s3_Error{Code: AccessDenied,
			Resource: location}
		return errz
	}

	if err2 != nil {
		// assert(errors.Is(err2, fs.ErrNotExist))
		var err3 = os.MkdirAll(dirpath, 0755)
		if err3 != nil {
			bbs.Logger.Info("os.Mkdir() failed", "error", err3)
			var m = map[error]Aws_s3_error_code{}
			var errz = map_os_error(ctx, location, err3, m)
			return errz
		}
	}

	// Copy data to a temporary file.

	var scratchname = "." + filename + "@scratch"
	var scratchpath = filepath.Join(dirpath, scratchname)

	var f, err4 = os.Create(scratchpath)
	if err4 != nil {
		bbs.Logger.Info("os.Create() failed", "error", err4)
		var m = map[error]Aws_s3_error_code{}
		var errz = map_os_error(ctx, location, err4, m)
		return errz
	}
	defer func() {
		f.Close()
		os.Remove(scratchpath)
	}()

	var cc, err5 = io.Copy(f, body)
	if err5 != nil {
		bbs.Logger.Info("io.Copy() failed", "error", err5)
		var m = map[error]Aws_s3_error_code{}
		var errz = map_os_error(ctx, location, err5, m)
		return errz
	}
	var err6 = f.Close()
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

	if (md5 != nil) {
		var md5x = []byte(*md5)
		var md5y, err7 = calculate_md5(scratchpath, bbs.Logger)
		if err7 != nil {
			var errz = Aws_s3_Error{Code: InternalError,
				Resource: location,
				Message: fmt.Sprintf("md5 calculation failed")}
			return errz
		}
		if bytes.Compare(md5x, md5y) != 0 {
			var errz = Aws_s3_Error{Code: IncompleteBody,
				Resource: location,
				Message: fmt.Sprintf("Body md5 unmatch")}
			return errz
		}
	}

	// Move a temporary file to actual file.

	var filepath = filepath.Join(dirpath, filename)
	var err8 = os.Rename(scratchname, filepath)
	if err8 != nil {
		bbs.Logger.Info("io.Rename() failed", "error", err8)
		var m = map[error]Aws_s3_error_code{}
		var errz = map_os_error(ctx, location, err8, m)
		return errz
	}

	return nil
}
