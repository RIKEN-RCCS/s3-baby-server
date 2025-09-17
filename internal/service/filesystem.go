// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package service

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"s3-baby-server/pkg/utils"
	"strconv"
	"strings"
)

type FileSystem struct {
	Logger   *slog.Logger
	RootPath string
	TmpPath  string
	MpPath   string
}

func (f *FileSystem) InitDir() {
	utils.RemoveAndMakeDir(f.getTmpPath(), f.Logger)
	utils.RemoveAndMakeDir(f.getMpPath(), f.Logger)
}

func (f *FileSystem) isFileExists(file string) bool {
	f.Logger.Debug("Path check", "path", f.getFullPath(file))
	_, err := os.Lstat(f.getFullPath(file))
	return !os.IsNotExist(err)
}

func (f *FileSystem) canCreateFile(path string) bool {
	file, err := os.Create(path)
	if err != nil {
		return false
	}
	utils.CloseFile(file, f.Logger)
	_ = os.Remove(path)
	return true
}

func (f *FileSystem) readFile(path string) []byte {
	file, err := os.ReadFile(f.getFullPath(path))
	if err != nil {
		return nil
	}
	return file
}

func (f *FileSystem) readDir() []os.DirEntry {
	es, err := os.ReadDir(f.RootPath)
	if err != nil {
		return nil
	}
	return es
}

func (f *FileSystem) createTmpFileFromFile(srcPath, dstPath string, offset, length int64) string {
	tmpAPath := f.getTmpFilePath(dstPath)
	tmpF, err := os.Create(tmpAPath)
	if err != nil {
		return ""
	}
	defer utils.CloseFile(tmpF, f.Logger)
	srcF, err := os.Open(f.getFullPath(srcPath))
	if err != nil {
		return ""
	}
	defer utils.CloseFile(srcF, f.Logger)
	if offset == 0 {
		if _, err = io.Copy(tmpF, srcF); err != nil {
			return ""
		}
	} else {
		sectionReader := io.NewSectionReader(srcF, offset, length)
		if _, err = io.Copy(tmpF, sectionReader); err != nil {
			return ""
		}
	}
	return tmpAPath
}

func (f *FileSystem) createTmpFileFromBody(src []byte, dstKey string) string {
	tmpAPath := f.getTmpFilePath(dstKey)
	tmpF, err := os.Create(tmpAPath)
	if err != nil {
		return ""
	}
	defer utils.CloseFile(tmpF, f.Logger)
	if _, err = io.Copy(tmpF, bytes.NewReader(src)); err != nil {
		return ""
	}
	return tmpAPath
}

func (f *FileSystem) createDir(dir string) error {
	return os.MkdirAll(f.getFullPath(dir), 0755)
}

func (f *FileSystem) moveFile(tmpAPath, dstPath string) bool {
	dstAPath := f.getFullPath(dstPath)
	f.Logger.Debug("tmpPath to dstPath", "tmpPath", tmpAPath, "dstPath", dstAPath)
	if err := os.MkdirAll(filepath.Dir(dstAPath), 0755); err != nil { // Keyに複数の階層が指定されたとき用にファイルの上位階層を作成
		return false
	}
	return os.Rename(tmpAPath, dstAPath) == nil // .TmpUploadから指定のパスに移動
}

func (f *FileSystem) copyFile(srcPath, dstPath string, offset, length int64) bool {
	tmpAPath := f.createTmpFileFromFile(srcPath, dstPath, offset, length)
	if tmpAPath == "" {
		return false
	}
	return f.moveFile(tmpAPath, dstPath)
}

func (f *FileSystem) uploadFile(src []byte, dstPath string) bool {
	f.Logger.Debug("受け取ったパス", "", dstPath)
	isDir := strings.HasSuffix(dstPath, "/")
	if isDir {
		f.Logger.Debug("ディレクトリ")
		dstAPath := f.getFullPath(dstPath)
		f.Logger.Debug("", "作成するディレクトリ", dstAPath)
		if err := os.MkdirAll(dstAPath, 0755); err != nil { // Keyに複数の階層が指定されたとき用にファイルの上位階層を作成
			f.Logger.Debug("ディレクトリ作成失敗")
			return false
		}
		f.Logger.Debug("ディレクトリ作成成功")
		return true
	}
	f.Logger.Debug("debug", "make to path", dstPath)
	tmpAPath := f.createTmpFileFromBody(src, dstPath)
	if tmpAPath == "" {
		return false
	}
	return f.moveFile(tmpAPath, dstPath)
}

func (f *FileSystem) deleteFile(file string) bool {
	if err := os.Remove(f.getFullPath(file)); err != nil {
		f.Logger.Error("Failed to delete file", "fileName", f.getFullPath(file))
		return false
	}
	metaPath := filepath.Join(filepath.Dir(file), f.getFileName(file)+"_meta.json")
	if f.isFileExists(metaPath) {
		if err := os.Remove(f.getFullPath(metaPath)); err != nil {
			f.Logger.Error("Failed to delete meta file", "fileName", f.getFullPath(metaPath))
			return false
		}
	}
	return true
}

func (f *FileSystem) deleteMetaFile(file string) bool {
	metaPath := filepath.Join(filepath.Dir(file), f.getFileName(file)+"_meta.json")
	if err := os.Remove(f.getFullPath(metaPath)); err != nil {
		f.Logger.Error("Failed to force delete directory", "path", f.getFullPath(metaPath), "error", err)
		return false
	}
	return true
}

func (f *FileSystem) forceDeleteDir(id string) bool {
	path := f.getUploadIDPath(id)
	if err := os.RemoveAll(f.getFullPath(path)); err != nil {
		f.Logger.Error("Failed to force delete directory", "path", f.getFullPath(path), "error", err)
		return false
	}
	return true
}

func (f *FileSystem) deleteFiles(b string, ks []string) ([]string, []string) {
	var deleted, errors []string
	for _, obj := range ks {
		if !f.isFileExists(filepath.Join(b, obj)) {
			errors = append(errors, obj)
			continue
		}
		if !f.deleteFile(filepath.Join(b, obj)) {
			errors = append(errors, obj)
		} else {
			deleted = append(deleted, obj)
		}
	}
	return deleted, errors
}

func (f *FileSystem) calcChecksum(algorithm, value, multiAPath string) (string, error) {
	var dstAPath string
	if multiAPath == "" {
		dstAPath = f.getFullPath(value)
	} else {
		dstAPath = f.getFullPath(multiAPath)
	}
	switch algorithm {
	case "CRC32", "rCRC32":
		return utils.ChecksumCrc32(dstAPath, f.Logger)
	case "CRC32C", "rCRC32C":
		return utils.ChecksumCrc32c(dstAPath, f.Logger)
	case "CRC64NVME", "rCRC64NVME":
		return utils.ChecksumCrc64nvme(dstAPath, f.Logger)
	case "SHA1", "rSHA1":
		return utils.ChecksumSha1(dstAPath, f.Logger)
	case "SHA256", "rSHA256":
		return utils.ChecksumSha256(dstAPath, f.Logger)
	case "":
		return "", nil
	default:
		return "", BadRequest()
	}
}

func (f *FileSystem) getFullPath(path string) string {
	isDir := strings.HasSuffix(path, "/")
	path = filepath.Join(f.RootPath, filepath.Clean(path)) // ルートパスの追加、パスの正規化
	if isDir {
		path += string(filepath.Separator)
	}
	return path
}

func (f *FileSystem) getRandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}
	return string(result)
}

func (f *FileSystem) getTmpFilePath(key string) string {
	fn := filepath.Base(key)
	ext := filepath.Ext(fn)
	name := strings.TrimSuffix(fn, ext) + f.getRandomString(8) // add random string to file names
	bytes := md5.Sum([]byte(name))
	tmpFName := hex.EncodeToString(bytes[:]) + ".tmp"
	return filepath.Join(f.getTmpPath(), tmpFName)
}

func (f *FileSystem) getTmpPath() string {
	return f.getFullPath(f.TmpPath)
}

func (f *FileSystem) getMpPath() string {
	return f.getFullPath(f.MpPath)
}

func (f *FileSystem) getUploadIDPath(id string) string {
	return filepath.Join(f.MpPath, id)
}

func (f *FileSystem) getPartNumberPath(id, partNumber string) string {
	return filepath.Join(f.getUploadIDPath(id), partNumber)
}

func (f *FileSystem) getMpUploadMetaPath(uploadID int) string {
	id := strconv.Itoa(uploadID)
	return f.getFullPath(filepath.Join(f.MpPath, id, id+"_meta.json"))
}

func (f *FileSystem) checkBucketName(bucket string) bool {
	if strings.Contains(bucket, "..") { // バケット名にピリオドの連続が含まれている場合エラー
		return false
	}
	if len(bucket) < 3 || len(bucket) > 63 { // バケット名の長さチェック
		return false
	}
	re := regexp.MustCompile(`^[A-Za-z0-9].*[A-Za-z0-9-]$`) // バケット名のチェック(開始：A-Z, a-z, 0-9、終了：A-Z, a-z, 0-9, -)
	return re.MatchString(bucket)
}

func (f *FileSystem) checkKeyName(key string) bool {
	return len(key) <= 1024 // キーの長さチェック
}

func (f *FileSystem) validateChecksumAlgorithm(value string) bool {
	switch strings.ToUpper(value) {
	case "CRC32", "CRC32C", "CRC64NVME", "SHA1", "SHA256", "":
		return true
	default:
		return false
	}
}

func (f *FileSystem) checkBuckets(v string, dirs []os.DirEntry, maxThresh int) ([]os.DirEntry, string) {
	minThresh := 1
	e := utils.ToInt(v)
	if minThresh > e || e > maxThresh {
		return nil, ""
	}
	if e > len(dirs) { // max-bucketsよりも少ない場合はすべて表示
		return dirs, ""
	}
	dirs = dirs[0:e]
	fn := []byte("continuation token: " + utils.ToString(e-1)) // 何番目のディレクトリまで読んだか
	c := base64.StdEncoding.EncodeToString(fn)
	return dirs, c
}

func (f *FileSystem) decodeContinuationToken(v string) (string, int) {
	token, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return "", 0
	}
	t := strings.TrimPrefix(string(token), "continuation token: ")
	f.Logger.Debug("", "ターゲット", t)
	target := utils.ToInt(t)
	if target == 0 {
		return "", 0
	}
	return v, target
}

func (f *FileSystem) continuationBucket(v string, dirs []os.DirEntry) []os.DirEntry {
	_, target := f.decodeContinuationToken(v)
	if len(dirs) <= target || target == 0 {
		return nil
	}
	dirs = dirs[target:]
	return dirs
}

func (f *FileSystem) prefixBucket(v string, dirs []os.DirEntry) []os.DirEntry {
	tmp := []fs.DirEntry{}
	for _, entry := range dirs {
		if strings.HasPrefix(entry.Name(), v) {
			tmp = append(tmp, entry)
		}
	}
	dirs = tmp
	return dirs
}

func (f *FileSystem) allowedPhrases(s string, allowed []string) bool {
	pattern := "^(" + strings.Join(allowed, "|") + ")+$"
	re := regexp.MustCompile(pattern)
	return re.MatchString(s)
}

func (f *FileSystem) getFileName(path string) string {
	return filepath.Base(path[:len(path)-len(filepath.Ext(path))])
}

func (f *FileSystem) getMetaFileName(path string) string {
	fn := f.getFileName(path)
	return filepath.Join(filepath.Dir(path), fn+"_meta.json")
}

func (f *FileSystem) getRange(value string, content []byte) (string, []byte) {
	v := strings.TrimPrefix(value, "bytes=")
	val := strings.Split(v, "-")
	start := utils.ToInt(val[0])
	end := utils.ToInt(val[1])
	if (start > len(content)) || (end > len(content)) || (start == 0 && end == 0) {
		return "", nil
	}
	r := fmt.Sprintf("bytes %s/%d", v, len(content))
	content = content[start : end+1]
	return r, content
}
