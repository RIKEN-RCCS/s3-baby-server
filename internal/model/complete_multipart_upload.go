// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

type CompleteMultipartUploadState struct {
	Bucket            string
	Key               string
	ETag              string
	ChecksumAlgorithm string
	ChecksumValue     string
	DstPath           string
}

func (com CompleteMultipartUploadState) MakeCompleteMultipartUploadResult() *CompleteMultipartUploadResult {
	result := CompleteMultipartUploadResult{
		Bucket: com.Bucket,
		Key:    com.Key,
		ETag:   com.ETag,
	}
	setChecksumFields(com.ChecksumAlgorithm, com.ChecksumValue, &result.ChecksumFields)
	return &result
}
