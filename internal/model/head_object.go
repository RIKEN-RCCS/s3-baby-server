// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

func (cg GetObjectState) MakeHeadObjectResult() *GetObjectResult { // GetObjectと同様の処理のため流用
	result := GetObjectResult{
		LastModified:  cg.Info.ModTime(),
		ETag:          cg.ETag,
		ContentLength: len(cg.Content),
	}
	if cg.ResponseCrc64nvme != "" {
		result.CRC64NVME = cg.ResponseCrc64nvme
		result.ChecksumType = checksumType // チェックサムの値がある場合に付与
	}
	result.ContentRange = cg.ContentRange
	return &result
}
