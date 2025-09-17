// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

import "os"

type GetObjectState struct {
	ETag              string
	Content           []byte
	Info              os.FileInfo
	TagCount          string
	MissingMeta       string
	ContentRange      string
	ResponseCrc64nvme string
}

func (cg GetObjectState) MakeGetObjectResult() *GetObjectResult {
	result := GetObjectResult{
		LastModified:  cg.Info.ModTime(),
		ETag:          cg.ETag,
		ContentLength: len(cg.Content),
		Content:       cg.Content,
	}
	if cg.ResponseCrc64nvme != "" {
		result.CRC64NVME = cg.ResponseCrc64nvme
		result.ChecksumType = checksumType // チェックサムの値がある場合に付与
	}
	result.TagCount = cg.TagCount
	if cg.MissingMeta != "0" { // メタデータ内にutf8以外の文字がある場合に出力
		result.MissingMeta = cg.MissingMeta
	}
	result.ContentRange = cg.ContentRange
	return &result
}
