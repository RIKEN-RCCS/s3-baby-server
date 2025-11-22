// listing.go
// Copyright 2025-2025 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// MEMO: It avoids using "filepath" that is OS dependent.

package server

import (
	//"context"
	//"encoding/json"
	//"bytes"
	//"encoding/hex"
	//"errors"
	//"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	//"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path"
	//"path/filepath"
	"strings"
)

type object_list_entry struct {
	key  string
	stat fs.FileInfo
}

const always_use_flat_lister = true

// LIST_OBJECTS_DELIMITED makes listing for "/"-delimiter case.  It
// works with regard to a directory hierarchy.  A start-index and a
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
// files in the bucket.  It uses fs.WalkDir() in "io/fs" as it returns
// slash-paths (not OS-specific).  In the scanning loop, it does not
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

// make_list_objects_entries converts a list to response data.  It
// calculates MD5.
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
