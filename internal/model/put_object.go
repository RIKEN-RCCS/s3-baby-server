// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

type PutObjectState struct {
	ETag              string
	ChecksumAlgorithm string
	ChecksumValue     string
}

func (po PutObjectState) MakePutObjectResult() *PutObjectResult {
	result := PutObjectResult{
		ETag: po.ETag,
	}
	setChecksumFields(po.ChecksumAlgorithm, po.ChecksumValue, &result.ChecksumFields)
	return &result
}
