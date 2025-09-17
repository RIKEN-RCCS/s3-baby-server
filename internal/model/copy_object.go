// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

import "os"

type CopyObjectState struct {
	Bucket            string
	Key               string
	ETag              string
	ChecksumAlgorithm string
	ChecksumValue     string
	SrcPath           string
	DstPath           string
	Info              os.FileInfo
	Offset            int64
	Length            int64
}

func (co CopyObjectState) MakeCopyObjectResult() *CopyObjectResult {
	result := CopyObjectResult{
		ETag:         co.ETag,
		LastModified: co.Info.ModTime(),
	}
	setChecksumFields(co.ChecksumAlgorithm, co.ChecksumValue, &result.ChecksumFields)
	return &result
}
