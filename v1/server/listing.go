// listing.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Listing.  This is for {ListBuckets, ListMultipartUploads,
// ListObjects, ListObjectsV2, ListParts}.  It prefers libraries
// "io/fs" that is mostly os-independent (slash-delimited) over "os"
// and "filepath".

// MEMO: io/fs.WalkDir does not follow symbolic links.  It is explicit
// in "https://pkg.go.dev/io/fs#WalkDir"

package server

import (
	//"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	//"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type object_list_entry struct {
	key  string
	stat fs.FileInfo
}

const always_use_flat_lister = true

var mpul_scratch_pattern = ".*@mpul"
var scratch_file_pattern = ".*"

// LIST_BUCKETS makes a list of buckets.
func (bbs *Bb_server) list_buckets(start int, count int, prefix string) ([]types.Bucket, int, *Aws_s3_error) {
	var pool_path = "."
	var entries1, err1 = os.ReadDir(pool_path)
	if err1 != nil {
		bbs.logger.Info("os.ReadDir() failed in ListBuckets", "error", err1)
		var errz = map_os_error("/", err1, nil)
		return nil, 0, errz
	}

	// Filter and keep only directories that satisfy bucket naming.
	// Checking a dot is redundant, because check_bucket_naming()
	// filters them out.  HasPrefix() is true when prefix="".

	var entries2 = []fs.DirEntry{}
	for _, e := range entries1 {
		var name = e.Name()
		var stat, err2 = e.Info()
		if err2 != nil {
			bbs.logger.Info("os.Lstat() failed on fs.DirEntry",
				"direntry", e, "error", err2)
			// IGNORE ERRORS.
			continue
		}

		if !e.IsDir() {
			continue
		}
		if check_special_file(stat) {
			continue
		}
		if check_metainfo_name(name) {
			continue
		}

		if !check_bucket_naming(name) {
			continue
		}
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		entries2 = append(entries2, e)
	}

	var entries3 []fs.DirEntry
	var continuation int
	if start < len(entries2) {
		var end = min(start+count, len(entries2))
		entries3 = entries2[start:end]
		if end < len(entries2) {
			continuation = end
		} else {
			continuation = 0
		}
	} else {
		entries3 = []fs.DirEntry{}
		continuation = 0
	}

	var buckets = []types.Bucket{}
	for _, e := range entries3 {
		var stat, err2 = e.Info()
		if err2 != nil {
			// Skip the entry because it may be removed after scanning
			// directory.  SHOULD CHECK errors.Is(err, ErrNotExist).
			continue
		}
		var times, ok = file_time(stat)
		if !ok {
			var t0 = stat.ModTime()
			times = [3]time.Time{t0, t0, t0}
		}
		var name = e.Name()
		var b = types.Bucket{
			// b : types.Bucket.
			// - BucketArn *string
			// - BucketRegion *string
			// - CreationDate *time.Time
			// - Name *string
			CreationDate: &times[1],
			Name:         &name,
		}
		buckets = append(buckets, b)
	}
	return buckets, continuation, nil
}

// LIST_OBJECTS_DELIMITED makes listing for "/"-delimiter case.  It
// works with regard to a directory hierarchy.  A start-index and a
// start-marker indicates a start point.  Note the entries ReadDir()
// returns are sorted.  It returns a next start-index and a next
// start-marker, in addition to the entries.  THE ENTRIES INCLUDE
// DIRECTORIES EVEN IF THEY ARE EMPTY.
func (bbs *Bb_server) list_objects_delimited(bucket string, index int, marker string, maxkeys int, delimiter string, prefix string) ([]object_list_entry, int, string, *Aws_s3_error) {
	if delimiter != "/" {
		log.Fatalf("BAD-IMPL: list_objects_delimited with non-slash")
	}

	var location = "/" + bucket
	var dir1, fileprefix = path.Split(path.Clean(prefix))
	var dir2, filemarker = path.Split(path.Clean(marker))

	if marker != "" {
		if dir1 != dir2 {
			// Nothing won't match the start-marker, return empty.
			return nil, 0, "", nil
		}
	}

	var pool_path = "."
	var directory = filepath.Join(pool_path, bucket, dir1)
	var entries1, err1 = os.ReadDir(directory)
	if err1 != nil {
		bbs.logger.Info("os.ReadDir() failed",
			"path", directory, "error", err1)
		return nil, 0, "", map_os_error(location, err1, nil)
	}

	// Filter entries by name.  This skips directories for MPUL.

	var entries2 []fs.DirEntry
	{
		for _, e := range entries1 {
			var name = e.Name()
			var stat, err2 = e.Info()
			if err2 != nil {
				bbs.logger.Info("os.Lstat() failed on fs.DirEntry",
					"direntry", e, "error", err2)
				// IGNORE ERRORS.
				continue
			}

			if check_special_file(stat) {
				continue
			}
			if check_metainfo_name(name) {
				continue
			}

			if fileprefix != "" {
				if strings.HasPrefix(name, fileprefix) {
					entries2 = append(entries2, e)
				}
			} else {
				entries2 = append(entries2, e)
			}
		}
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
			bbs.logger.Info("os.Lstat() failed on fs.DirEntry",
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
// slash-paths (not os-specific).  In the scanning loop, it does not
// count directory entries.  COUNT counts files visited and it is used
// to check a start-index.  MEMO: A prefix should not have a
// preceeding delimiter.  A common-prefix has a trailing delimiter.
func (bbs *Bb_server) list_objects_flat(bucket string, index int, marker string, maxkeys int, delimiter string, prefix string) ([]object_list_entry, int, string, *Aws_s3_error) {
	var location = "/" + bucket
	var pool_path = "."
	var b = path.Join(pool_path, bucket)
	var bucket1 = os.DirFS(b)

	var entries []object_list_entry
	var nextindex int = 0
	var nextmarker string = ""
	var count int = 0
	var markerhit bool = false
	var commonprefix string = ""

	var err1 = fs.WalkDir(bucket1, ".", func(key1 string, e fs.DirEntry, err1 error) error {
		// Skip errors.

		if err1 != nil {
			bbs.logger.Info("os.DirFS() callbacks with error",
				"bucket", bucket, "path", key1, "error", err1)
			return nil
		}

		var name = e.Name()
		var stat, err2 = e.Info()
		if err2 != nil {
			bbs.logger.Info("os.Lstat() failed on fs.DirEntry",
				"direntry", e, "error", err2)
			// IGNORE ERRORS.
			return nil
		}

		{
			// Skip directories.  It totally skips contents of MPUL.
			// This should be before checking non-regular files.

			if e.IsDir() {
				if check_mpul_scratch_name(name) {
					return fs.SkipDir
				} else {
					return nil
				}
			}

			if check_special_file(stat) {
				return nil
			}
			if check_metainfo_name(name) {
				return nil
			}
		}

		// Skip unless the prefix matches.

		if !strings.HasPrefix(key1, prefix) {
			return nil
		}

		// Skip a common prefix when it is already encountered.

		var commonpart = check_common_prefix(key1, delimiter, prefix)
		if commonpart != "" {
			if commonprefix == commonpart {
				// Skip if it is the one encountered.
				return nil
			}
			commonprefix = commonpart
		}

		defer func() {
			count++
		}()

		// Check the start-marker first, then check the start-index.

		if marker != "" && !markerhit {
			if marker == key1 {
				markerhit = true
			} else {
				return nil
			}
		}
		if count < index {
			return nil
		}

		// Don't finish when fully collected.  It needs one extra
		// entry to check truncation.

		if len(entries) < maxkeys {
			entries = append(entries, object_list_entry{key1, stat})
			return nil
		} else {
			nextindex = count
			nextmarker = key1
			return fs.SkipAll
		}
	})
	if err1 != nil {
		return nil, 0, "", map_os_error(location, err1, nil)
	}

	return entries, nextindex, nextmarker, nil
}

// MAKE_LIST_OBJECTS_ENTRIES converts a list of objects to response
// entries.  It calculates MD5.
func (bbs *Bb_server) make_list_objects_entries(entries []object_list_entry, bucket string, delimiter string, prefix string, urlencode bool) ([]types.Object, []types.CommonPrefix, error) {
	var contents []types.Object
	var commonprefixes []types.CommonPrefix
	for _, e := range entries {
		var object = path.Join(bucket, e.key)
		var commonpart = check_common_prefix(e.key, delimiter, prefix)
		if commonpart == "" {
			var md5, _, err3 = bbs.calculate_csum2("", object, "")
			var etag string
			if err3 != nil {
				bbs.logger.Warn("MD5 calculation failed",
					"file", object, "error", err3)
				etag = ""
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
				// s : types.Object.
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
				ETag:         &etag,
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

func (bbs *Bb_server) list_mpuls_flat(bucket string, marker string, maxkeys int, delimiter string, prefix string, urlencode bool) ([]types.MultipartUpload, []types.CommonPrefix, string, *Aws_s3_error) {
	var location = "/" + bucket
	var pool_path = "."
	var b = path.Join(pool_path, bucket)
	var bucket1 = os.DirFS(b)

	var objects []types.MultipartUpload
	var commons []types.CommonPrefix
	var nextmarker string = ""
	var count int = 0
	var markerhit bool = false
	var commonprefix string = ""

	var err1 = fs.WalkDir(bucket1, ".", func(key1 string, e fs.DirEntry, err1 error) error {
		// Skip errors.

		if err1 != nil {
			bbs.logger.Info("os.DirFS() callbacks with error",
				"bucket", bucket, "path", key1, "error", err1)
			return nil
		}

		{
			// Skip non-directories as a store of MPUL is a directory.

			if !e.IsDir() {
				return nil
			}
			if !check_mpul_scratch_name(e.Name()) {
				return nil
			}
		}

		{
			var _, n = path.Split(key1)
			if n != e.Name() {
				log.Fatal("fs.WalkDir() returns an unexpected entry")
			}
		}

		// Fix the object-key: A scratch directory to an object name.

		var key2 = adjust_mpul_scratch_to_object_name(key1)

		// Check the prefix.

		if !strings.HasPrefix(key2, prefix) {
			return nil
		}

		// Check a common prefix, and check if already encountered.

		var commonpart = check_common_prefix(key2, delimiter, prefix)
		if commonpart != "" {
			if commonprefix == commonpart {
				// Skip if it is the one encountered.
				return nil
			}
			commonprefix = commonpart
		}

		defer func() {
			count++
		}()

		// Check the start-marker.

		if marker != "" && !markerhit {
			if marker == key2 {
				markerhit = true
			} else {
				return nil
			}
		}

		// Don't finish when fully collected.  It needs one extra
		// entry to check truncation.

		if len(objects) < maxkeys {
			if commonpart != "" {
				var object = path.Join(bucket, key2)
				var mpul, err4 = bbs.fetch_mpul_info(object)
				if err4 != nil {
					// IGNORE ERRORS.
					// Race among listing and others.
					bbs.logger.Info("Race in accessing MPUL,"+
						" listing and others",
						"func", "fetch_mpul_info", "error", err4)
					return nil
				}
				var fixedkey string
				if urlencode {
					fixedkey = url.QueryEscape(key2)
				} else {
					fixedkey = key2
				}
				var s = types.MultipartUpload{
					// s : types.MultipartUpload.
					// - ChecksumAlgorithm ChecksumAlgorithm
					// - ChecksumType ChecksumType
					// - Initiated *time.Time
					// - Initiator *Initiator
					// - Key *string
					// - Owner *Owner
					// - StorageClass StorageClass
					// - UploadId *string
					Key:               &fixedkey,
					UploadId:          &mpul.Upload_id,
					Initiated:         &mpul.Mtime,
					StorageClass:      types.StorageClassStandard,
					ChecksumAlgorithm: mpul.Checksum_algorithm,
					ChecksumType:      mpul.Checksum_type,
				}
				objects = append(objects, s)
			} else {
				var s = types.CommonPrefix{
					// s : types.CommonPrefix.
					// - Prefix *string
					Prefix: &commonpart,
				}
				commons = append(commons, s)
			}
			return nil
		} else {
			nextmarker = key2
			return fs.SkipAll
		}
	})
	if err1 != nil {
		return nil, nil, "", map_os_error(location, err1, nil)
	}

	return objects, commons, nextmarker, nil
}

// CHECK_BUCKET_EMPTY makes sure the emptiness of a bucket for
// deleting it.  It concerns only regular files, but excludes scratch
// files whose name begins with a dot.  Note a MPUL directory is named
// ".objectname@mpul".
func (bbs *Bb_server) check_bucket_empty(bucket string) *Aws_s3_error {
	var path1 = bbs.make_path_of_bucket(bucket)
	var err1 = bbs.check_directory_empty(bucket, path1)
	return err1
}

func (bbs *Bb_server) check_directory_empty(bucket string, path1 string) *Aws_s3_error {
	var location = "/" + bucket
	var filelist, err1 = os.ReadDir(path1)
	if err1 != nil {
		//if errors.Is(err1, fs.ErrNotExist)
		bbs.logger.Info("os.ReadDir() in a bucket failed",
			"path", path1, "error", err1)
		var errz = &Aws_s3_error{Code: InternalError,
			Message:  "Listing in a bucket failed.",
			Resource: location}
		return errz
	}
	for _, e := range filelist {
		var name = e.Name()
		var stat, err2 = e.Info()
		if err2 != nil {
			bbs.logger.Info("os.Lstat() failed on fs.DirEntry",
				"direntry", e, "error", err2)
			// IGNORE ERRORS.
			continue
		}

		if e.IsDir() {
			continue
		}
		if check_special_file(stat) {
			return nil
		}
		if check_metainfo_name(name) {
			continue
		}
		var errz = &Aws_s3_error{Code: BucketNotEmpty,
			Resource: location}
		return errz
	}
	for _, e := range filelist {
		if e.IsDir() {
			var name = e.Name()
			if name == "." || name == ".." {
				continue
			}
			if check_mpul_scratch_name(name) {
				var errz = &Aws_s3_error{Code: BucketNotEmpty,
					Resource: location}
				return errz
			}
			var path2 = filepath.Join(path1, name)
			var err3 = bbs.check_directory_empty(bucket, path2)
			if err3 != nil {
				return err3
			}
		}
	}
	return nil
}

func check_special_file(stat fs.FileInfo) bool {
	var dir = stat.IsDir()
	var mode = stat.Mode()
	var reg = (mode & fs.ModeType) == 0
	return !(dir || reg)
}

func check_metainfo_name(name string) bool {
	// Checks metainfo or scratch file name.
	return strings.HasPrefix(name, ".")
}

func check_mpul_scratch_name(name string) bool {
	var m, err1 = path.Match(mpul_scratch_pattern, name)
	if err1 != nil {
		log.Fatalf("BAD-IMPL: mpul_scratch_pattern")
	}
	return m
}
