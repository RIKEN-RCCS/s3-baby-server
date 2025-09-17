// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package service

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"s3-baby-server/internal/model"
	"s3-baby-server/pkg/utils"
	"strconv"
	"sync"
)

type MultiPart struct {
	FileSystem *FileSystem
	mu         sync.Mutex
}

func (m *MultiPart) uploadIDExists(id string) bool {
	return m.FileSystem.isFileExists(m.FileSystem.getUploadIDPath(id))
}

func (m *MultiPart) getMpDirInfo() []os.DirEntry {
	es, err := os.ReadDir(m.FileSystem.getMpPath())
	if err != nil {
		return nil
	}
	return es
}

func (m *MultiPart) makeUploadID() int { // ディレクトリ内の最大アップロードID+1した値を返す
	m.mu.Lock()
	defer m.mu.Unlock() // 排他処理
	dirs := m.getMpDirInfo()
	if dirs == nil {
		return -1
	}
	maxID := 0
	for _, dir := range dirs {
		if dir.IsDir() {
			id, err := strconv.Atoi(dir.Name())
			if err == nil && id > maxID {
				maxID = id
			}
		}
	}
	multiAPath := m.FileSystem.getMpUploadMetaPath(maxID + 1)
	if err := os.MkdirAll(filepath.Dir(multiAPath), 0755); err != nil {
		return -1
	}
	return maxID + 1
}

func (m *MultiPart) createMpUploadMeta(res model.CreateMultipartUploadResult) bool {
	multiAPath := m.FileSystem.getMpUploadMetaPath(res.InitiateMultipartUploadResult.UploadID)
	if err := os.MkdirAll(filepath.Dir(multiAPath), 0755); err != nil {
		return false
	}
	file, err := os.Create(multiAPath)
	if err != nil {
		return false
	}
	defer utils.CloseFile(file, m.FileSystem.Logger)
	data, err := json.MarshalIndent(res.InitiateMultipartUploadResult, " ", " ")
	if err != nil {
		return false
	}
	if _, err = file.Write(data); err != nil {
		return false
	}
	return true
}

func (m *MultiPart) completeMpUpload(key, id, dstPath string, reqBody model.CompleteMultipartUploadRequest) *S3Error {
	tmpAPath := m.FileSystem.getTmpFilePath(key)
	out, err := os.Create(tmpAPath)
	if err != nil {
		m.FileSystem.Logger.Error("", "error", err)
	}
	var beforePNum int
	for _, part := range reqBody.Part {
		pNum := utils.ToInt(part.PartNumber)
		if beforePNum >= pNum {
			return InvalidPartOrder()
		}
		beforePNum = pNum
		in, err := os.Open(filepath.Join(m.FileSystem.getMpPath(), id, part.PartNumber))
		if err != nil {
			m.FileSystem.Logger.Error("", "error", err)
			return InvalidPart()
		}
		etag, err := utils.CalcMD5File(in)
		if err != nil {
			return InternalError()
		}
		if etag != part.ETag {
			return BadDigest()
		}
		if _, err = in.Seek(0, io.SeekStart); err != nil {
			m.FileSystem.Logger.Error("", "error", err)
		}
		if _, err = io.Copy(out, in); err != nil {
			m.FileSystem.Logger.Error("", "error", err)
		}
		utils.CloseFile(in, m.FileSystem.Logger)
	}
	utils.CloseFile(out, m.FileSystem.Logger)
	m.FileSystem.moveFile(tmpAPath, dstPath)
	return nil
}

func (m *MultiPart) getMpUploadMeta(key, id string) string {
	i := utils.ToInt(id)
	if i == 0 {
		return ""
	}
	file, err := os.Open(m.FileSystem.getMpUploadMetaPath(i))
	if err != nil {
		m.FileSystem.Logger.Error("", "error", err)
	}
	defer utils.CloseFile(file, m.FileSystem.Logger)
	multipart := model.PartList{}
	if err = json.NewDecoder(file).Decode(&multipart); err != nil {
		m.FileSystem.Logger.Error("failed read meta file", "error", err)
	}
	m.FileSystem.Logger.Debug("", "", multipart)
	if key != multipart.Key {
		m.FileSystem.Logger.Error("", "error", err)
		return ""
	}
	return multipart.Key
}
