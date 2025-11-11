// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package service

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"s3-baby-server/internal/model"
	"s3-baby-server/internal/utils"
	"strings"
	"time"
)

func (s3 *S3Service) validateGetObjectOptions(option S3Options) *S3Error {
	if option.GetOption("Range") != "" && option.GetOption("partNumber") != "" {
		return InvalidRequest()
	}
	if !option.Validate(map[string]string{"Cache-Control": "no-cache"}) {
		return InvalidArgument()
	}
	return nil
}

func (s3 *S3Service) validateOptions(option S3Options) *S3Error {
	if !option.Validate(map[string]string{"Cache-Control": "no-cache"}) {
		return InvalidArgument()
	}
	if !option.Validate(map[string]string{"x-amz-storage-class": "STANDARD"}) {
		return InvalidStorageClass()
	}
	return nil
}

func (s3 *S3Service) validateOption(option S3Options) *S3Error {
	if !option.Validate(map[string]string{"objectOwnership": "BucketOwnerEnforced"}) {
		return BadRequest()
	}
	return nil
}

func (s3 *S3Service) validateGetObjectAttributesOptions(option S3Options, allowed []string) *S3Error {
	v := option.GetOption("x-amz-object-attributes")
	if v == "" {
		return InvalidArgument()
	}
	if !s3.FileSystem.allowedPhrases(v, allowed) {
		return InvalidArgument()
	}
	return nil
}

func (s3 *S3Service) getETag(path string) (string, *S3Error) {
	if isDir := strings.HasSuffix(path, "/"); isDir {
		return "", nil
	}
	etag, err := utils.CalcMD5(s3.FileSystem.getFullPath(path), s3.FileSystem.Logger)
	if err != nil {
		return "", InternalError()
	}
	return etag, nil
}

func (s3 *S3Service) compareChecksum(option S3Options, value, alg, multiAPath string, algFlg bool) (string, string, *S3Error) {
	s3.FileSystem.Logger.Debug("Checksum Algorithm", "value", alg)
	if alg == "" {
		return "", "", nil
	}
	alg = strings.ToUpper(alg)
	srcHash := value
	var err error
	if algFlg {
		if srcHash, err = s3.FileSystem.calcChecksum(alg, value, ""); err != nil {
			return "", "", BadRequest()
		}
	}
	dstHash, err := s3.FileSystem.calcChecksum(alg, option.GetPath(), multiAPath)
	if err != nil {
		s3.FileSystem.Logger.Error("", "error", err)
		return "", "", InternalError()
	}
	s3.FileSystem.Logger.Debug("Checksum to compare", "src", srcHash, "dst", dstHash)
	if dstHash != srcHash {
		return "", "", BadRequestChecksum()
	}
	return alg, dstHash, nil
}

func (s3 *S3Service) getChecksumMode(option S3Options, multiAPath string) (string, string, *S3Error) {
	var v string
	var errResp *S3Error
	alg := strings.ToUpper(option.GetOption("x-amz-sdk-checksum-algorithm")) // 値チェックのみ
	if !s3.FileSystem.validateChecksumAlgorithm(alg) {
		return "", "", InvalidArgument()
	}
	as := map[string]string{
		"CRC32":      "x-amz-checksum-crc32",
		"CRC32C":     "x-amz-checksum-crc32c",
		"CRC64NVME":  "x-amz-checksum-crc64nvme",
		"SHA1":       "x-amz-checksum-SHA1",
		"SHA256":     "x-amz-checksum-sha256",
		"rCRC32":     "ChecksumCRC32",
		"rCRC32C":    "ChecksumCRC32C",
		"rCRC64NVME": "ChecksumCRC64NVME",
		"rSHA1":      "ChecksumSHA1",
		"rSHA256":    "ChecksumSHA256",
	}
	for algorithm, optionKey := range as {
		v2 := option.GetOption(optionKey)
		if v2 != "" && multiAPath == "" {
			if alg, v, errResp = s3.compareChecksum(option, v2, algorithm, "", false); errResp != nil {
				s3.FileSystem.Logger.Debug("get checksum mode result", "algorithm", v2, "value", v)
				s3.FileSystem.Logger.Error("", "", errResp)
				return "", "", errResp
			}
		} else if v2 != "" && multiAPath != "" {
			if alg, v, errResp = s3.compareChecksum(option, v2, algorithm, multiAPath, false); errResp != nil {
				s3.FileSystem.Logger.Debug("get checksum mode result", "algorithm", alg, "multiPath", multiAPath)
				s3.FileSystem.Logger.Error("", "", errResp)
				return "", "", errResp
			}
		}
	}
	s3.FileSystem.Logger.Debug("checksum result", "algorithm", alg, "value", v)
	return alg, v, nil
}

func (s3 *S3Service) checkChecksumMode(option S3Options, content []byte) (string, *S3Error) {
	if v := option.GetOption("x-amz-checksum-mode"); v != "ENABLED" {
		return "", nil
	}
	hash, err := utils.ChecksumCrc64nvmeBody(content)
	if err != nil {
		return "", InternalError()
	}
	return hash, nil
}

func (s3 *S3Service) checkObjectSize(option S3Options, path string) *S3Error {
	if v := option.GetOption("x-amz-mp-object-size"); v != "" {
		info, err := s3.getFileInfo(path)
		if err != nil {
			return InternalError()
		}
		size := utils.ToInt64(v)
		if size != info.Size() {
			s3.FileSystem.Logger.Error("", "指定サイズ", size, "実際のサイズ", info.Size())
			return InvalidRequest()
		}
	}
	return nil
}

func (s3 *S3Service) checkPartSize(id string, reqBody model.CompleteMultipartUploadRequest) bool {
	for i, part := range reqBody.Part {
		info, err := s3.getFileInfo(s3.FileSystem.getPartNumberPath(id, part.PartNumber))
		if err != nil {
			s3.FileSystem.Logger.Error("", "failed read file info", err)
		}
		if i < len(reqBody.Part)-1 && info.Size() < 5*1024*1024 {
			return false
		}
	}
	return true
}

func (s3 *S3Service) isBucketAndKeyExists(bucket, path string) *S3Error {
	if !s3.FileSystem.isFileExists(bucket) {
		return NoSuchBucket()
	}
	if !s3.FileSystem.isFileExists(path) {
		return NoSuchKey()
	}
	return nil
}

func (s3 *S3Service) getFileInfoAndETag(path string) (os.FileInfo, string, *S3Error) {
	var info os.FileInfo
	var etag string
	var err *S3Error
	if info, err = s3.getFileInfo(path); err != nil {
		return nil, "", err
	}
	if etag, err = s3.getETag(path); err != nil {
		return nil, "", err
	}
	return info, etag, nil
}

func (s3 *S3Service) getFileInfo(path string) (os.FileInfo, *S3Error) {
	info, err := os.Lstat(s3.FileSystem.getFullPath(path))
	if err != nil {
		return nil, InternalError()
	}
	return info, nil
}

func (s3 *S3Service) etagIfNeeded(path, v string) (string, *S3Error) {
	if !strings.Contains(v, "ETag") {
		return "", nil
	}
	etag, err := s3.getETag(path)
	if err != nil {
		return "", InternalError()
	}
	return etag, nil
}

func (s3 *S3Service) checksumIfNeeded(path, v string) (string, *S3Error) {
	if !strings.Contains(v, "Checksum") {
		return "", nil
	}
	hash, err := utils.ChecksumCrc64nvme(s3.FileSystem.getFullPath(path), s3.FileSystem.Logger)
	if err != nil {
		return "", InternalError()
	}
	return hash, nil
}

func (s3 *S3Service) compareMd5(option S3Options, value []byte, etag string) *S3Error {
	v := option.GetOption("content-md5")
	if v == "" {
		return nil
	}
	var base64MD5 string
	var err error
	if value != nil {
		base64MD5, err = utils.CalcMD5Byte(value)
	} else {
		base64MD5, err = utils.CalcContentMD5(etag)
	}
	s3.FileSystem.Logger.Debug("content-md5", "specified：", v, "calc：", base64MD5)
	if err != nil {
		return InternalError()
	}
	if v != base64MD5 {
		return BadDigest()
	}
	return nil
}

func (s3 *S3Service) listObjects(s model.ListObjectsState) (*[]model.Contents, *model.ListObjectsStateResult) {
	var allPaths []string
	err := filepath.WalkDir(s.BucketAPath, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			s3.FileSystem.Logger.Error("", "error", err)
			return err
		}
		relPath, err := filepath.Rel(s.BucketAPath, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		if relPath == "." {
			return nil
		}
		if strings.HasPrefix(relPath, s.Prefix) {
			trimPath := strings.TrimPrefix(relPath, s.Prefix)
			if strings.Contains(trimPath, s.Delimiter) && s.Delimiter != "" {
				return nil
			}
			allPaths = append(allPaths, relPath)
		}
		return nil
	})
	if err != nil {
		return nil, nil
	}
	startIndex := 0
	if s.Marker != "" || s.StartAfter != "" || s.Target != 0 {
		for i, p := range allPaths {
			if s.Marker != "" && p != s.Marker {
				continue
			}
			if s.StartAfter != "" && p != s.StartAfter {
				continue
			}
			if s.Target != 0 && i < s.Target {
				continue
			}
			startIndex = i + 1
			break
		}
	}
	endIndex := startIndex + s.MaxKeys
	if endIndex > len(allPaths) {
		endIndex = len(allPaths)
	}
	selectedPaths := allPaths[startIndex:endIndex]
	responseResult := model.ListObjectsStateResult{}
	responseResult.IsTruncated = endIndex < len(allPaths)
	if s.MaxKeys == 0 {
		responseResult.IsTruncated = false
	}
	var result []model.Contents
	for _, relPath := range selectedPaths {
		fullPath := filepath.Join(s.BucketAPath, relPath)
		fi, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}
		etag, err := s3.getETag(filepath.Join(s.Bucket, relPath))
		if errors.Is(err, InternalError()) {
			continue
		}
		if s.URLFlag {
			relPath = url.QueryEscape(relPath)
		}
		if fi.IsDir() {
			relPath += string(filepath.Separator)
			responseResult.Dirs = append(responseResult.Dirs, relPath)
		}
		result = append(result, model.Contents{
			Key:          relPath,
			LastModified: fi.ModTime(),
			ETag:         `"` + etag + `"`,
			Size:         fi.Size(),
			StorageClass: "STANDARD",
		})
	}
	if responseResult.IsTruncated {
		lastKey := selectedPaths[len(selectedPaths)-1]
		if s.V2Flg {
			fn := []byte("continuation token: " + utils.ToString(endIndex-1))
			responseResult.NextMarker = base64.StdEncoding.EncodeToString(fn)
		} else {
			responseResult.NextMarker = lastKey
		}
	}
	responseResult.Cnt = len(result)
	return &result, &responseResult
}

func (s3 *S3Service) listMpUploads(s model.ListMultipartUploadsState) (*[]model.Uploads, *model.ListMultipartUploadsStateResult) {
	var result, allUploads []model.Uploads
	responseResult := model.ListMultipartUploadsStateResult{}
	for _, entry := range s.Dirs {
		info, err := entry.Info()
		if err != nil {
			s3.FileSystem.Logger.Error("", "error", err)
			continue
		}
		i := utils.ToInt(info.Name())
		if i == 0 {
			continue
		}
		file, err := os.Open(s3.FileSystem.getMpUploadMetaPath(i))
		if err != nil {
			s3.FileSystem.Logger.Error("", "error", err)
			continue
		}
		var multipart model.PartList
		if err = json.NewDecoder(file).Decode(&multipart); err != nil {
			s3.FileSystem.Logger.Error("failed read meta file", "error", err)
			utils.CloseFile(file, s3.FileSystem.Logger)
			continue
		}
		utils.CloseFile(file, s3.FileSystem.Logger)
		if multipart.Bucket != s.Bucket {
			continue
		}
		if !strings.HasPrefix(multipart.Key, filepath.FromSlash(s.Prefix)) {
			continue
		}
		if multipart.Key <= s.KeyMarker {
			continue
		}
		if s.Target >= i {
			continue
		}
		allUploads = append(allUploads, model.Uploads{
			Initiated:    info.ModTime(),
			Key:          multipart.Key,
			StorageClass: "STANDARD",
			UploadID:     info.Name(),
		})
	}
	endIndex := s.MaxUploads
	if endIndex > len(allUploads) {
		endIndex = len(allUploads)
	}
	result = allUploads[:endIndex]
	responseResult.IsTruncated = endIndex < len(allUploads)
	if responseResult.IsTruncated {
		last := result[len(result)-1]
		responseResult.NextKeyMarker = last.Key
		i := utils.ToInt(last.UploadID)
		if i > 0 {
			fn := []byte("continuation token: " + utils.ToString(i))
			responseResult.NextUploadIDMarker = base64.StdEncoding.EncodeToString(fn)
		}
	}
	return &result, &responseResult
}

func (s3 *S3Service) listParts(s model.ListPartsState) (*[]model.Parts, *model.ListPartsStateResult) {
	var result, allParts []model.Parts
	responseResult := model.ListPartsStateResult{}
	err := filepath.WalkDir(s.BucketAPath, func(_ string, info fs.DirEntry, err error) error {
		if err != nil {
			s3.FileSystem.Logger.Error("", "error", err)
			return err
		}
		if info.IsDir() || strings.HasSuffix(info.Name(), "_meta.json") {
			return nil
		}
		fi, err := info.Info()
		if err != nil {
			return nil
		}
		pNum := utils.ToInt(fi.Name())
		if s.Target != 0 && pNum <= s.Target {
			return nil
		}
		crc64, err := utils.ChecksumCrc64nvme(
			s3.FileSystem.getFullPath(filepath.Join(s3.FileSystem.MpPath, s.UploadID, fi.Name())),
			s3.FileSystem.Logger,
		)
		if err != nil {
			return nil
		}
		etag, err := s3.getETag(filepath.Join(s3.FileSystem.MpPath, s.UploadID, fi.Name()))
		if errors.Is(err, InternalError()) {
			return nil
		}
		allParts = append(allParts, model.Parts{
			ChecksumCRC64NVME: crc64,
			ETag:              etag,
			LastModified:      fi.ModTime(),
			PartNumber:        fi.Name(),
			Size:              fi.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, nil
	}
	endIndex := s.MaxParts
	if endIndex > len(allParts) {
		endIndex = len(allParts)
	}
	result = allParts[:endIndex]
	responseResult.IsTruncated = endIndex < len(allParts)
	if responseResult.IsTruncated {
		responseResult.NextMarker = utils.ToInt(allParts[endIndex-1].PartNumber)
	}
	return &result, &responseResult
}

func (s3 *S3Service) needsCopy(option S3Options, copySource string) bool {
	full := s3.FileSystem.getFullPath(copySource)
	actualMd5, e1 := utils.CalcMD5(full, s3.FileSystem.Logger)
	match := option.GetOption("x-amz-copy-source-if-match")
	nonMatch := option.GetOption("x-amz-copy-source-if-none-match")
	if e1 != nil || (match != "" && actualMd5 != match) || (nonMatch != "" && actualMd5 == nonMatch) {
		s3.FileSystem.Logger.Error("ETag verification result error")
		return false
	}
	stat, e2 := os.Lstat(full)
	actualMod := stat.ModTime()
	mod, e3 := time.Parse(time.RFC1123, option.GetOption("x-amz-copy-source-if-modified-since"))
	unMod, e4 := time.Parse(time.RFC1123, option.GetOption("x-amz-copy-source-if-unmodified-since"))
	return e2 == nil && (e3 != nil || !actualMod.Before(mod)) && (e4 != nil || !actualMod.After(unMod))
}

func (s3 *S3Service) validateETagAndTime(option S3Options) *S3Error {
	aPath := s3.FileSystem.getFullPath(option.GetPath())
	checkMeta := filepath.Base(aPath)
	if strings.Contains(checkMeta, "_meta.json") { // WinSCP対策（ファイルが存在する状態でバケット削除）
		return nil
	}
	actualMd5, e1 := utils.CalcMD5(aPath, s3.FileSystem.Logger)
	stat, e2 := os.Lstat(aPath)
	if e1 != nil || e2 != nil {
		return InternalError()
	}
	match := option.GetOption("if-match")
	actualMod := stat.ModTime()
	unMod, e4 := time.Parse(time.RFC1123, option.GetOption("If-Unmodified-Since"))
	if (match != "" && actualMd5 != match) || (e4 == nil || !actualMod.After(unMod)) {
		s3.FileSystem.Logger.Debug("", "match", match)
		s3.FileSystem.Logger.Debug("", "actualMd5", actualMd5)
		s3.FileSystem.Logger.Debug("", "e4", e4)
		s3.FileSystem.Logger.Debug("", "unMod", unMod)
		s3.FileSystem.Logger.Debug("", "1", (match != "" && actualMd5 != match), "2", (e4 == nil || actualMod.After(unMod)))
		return PreconditionFailed()
	}
	nonMatch := option.GetOption("if-none-match")
	mod, e3 := time.Parse(time.RFC1123, option.GetOption("If-Modified-Since"))
	if (nonMatch != "" && actualMd5 == nonMatch) || (e3 == nil || actualMod.Before(mod)) {
		s3.FileSystem.Logger.Debug("", "nonMatch", nonMatch)
		s3.FileSystem.Logger.Debug("", "actualMd5", actualMd5)
		s3.FileSystem.Logger.Debug("", "e3", e3)
		s3.FileSystem.Logger.Debug("", "mod", mod)
		s3.FileSystem.Logger.Debug("", "1", (nonMatch != "" && actualMd5 == nonMatch), "2", (e3 == nil || actualMod.Before(mod)))
		return NotModified()
	}
	return nil
}

func (s3 *S3Service) checkEncodingType(option S3Options) (bool, *S3Error) {
	if v := option.GetOption("User-Agent"); strings.Contains(v, "aws-cli") {
		return false, nil // クライアントがAWS CLIの場合処理なし
	}
	if v := option.GetOption("encoding-type"); v != "" && v != "url" {
		return false, InvalidArgument() // エンコーディングタイプがurl以外の場合はエラー
	} else if v != "" && v == "url" {
		return true, nil
	}
	return false, nil
}

func (s3 *S3Service) getRangeValue(option S3Options) (int64, int64) {
	v := option.GetOption("Range")
	if v == "" {
		return 0, 0
	}
	v = strings.TrimPrefix(v, "bytes=")
	val := strings.Split(v, "-")
	s := utils.ToInt(val[0])
	e := utils.ToInt(val[1])
	if s > e || (s == 0 && e == 0) {
		return -1, -1
	}
	return int64(s), int64(e)
}

func (s3 *S3Service) getPartNumberContent(option S3Options) bool {
	if v := option.GetOption("partNumber"); v != "" {
		if v != "1" { // 単一のファイルとして扱うため、1以外のリクエストは受付しない
			return false
		}
	}
	return true
}

func (s3 *S3Service) getRangeContent(option S3Options, content []byte) (string, []byte, *S3Error) {
	if v := option.GetOption("Range"); v != "" {
		contentRange, rangeContent := s3.FileSystem.getRange(v, content)
		if rangeContent == nil {
			return "", nil, RangeNotSatisfiable()
		}
		return contentRange, rangeContent, nil
	}
	return "", content, nil
}
