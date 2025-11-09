// fs-helper.go

package server

import (
	//"bytes"
	//"crypto/md5"
	//"crypto/rand"
	//"encoding/base64"
	//"encoding/hex"
	//"fmt"
	//"io"
	//"io/fs"
	//"log/slog"
	//"math/big"
	//"os"
	"path/filepath"
	//"regexp"
	//"s3-baby-server/pkg/utils"
	//"strconv"
	//"strings"
)

// Appends a pool-directory and a bucket, where a bucket is assumed as
// a legal name.
func (bbs *Bb_server) make_path(bucket string) string {
	// filepath.Clean(path)
	var pool_path = bbs.S3.FileSystem.RootPath
	var path = filepath.Join(pool_path, bucket)
	return path
}
