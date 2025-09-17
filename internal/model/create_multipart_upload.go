// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

type CreateMultipartUploadState struct {
	Bucket   string
	Key      string
	UploadID int
}

func (crm CreateMultipartUploadState) MakeCreateMultipartUploadResult() *CreateMultipartUploadResult {
	result := CreateMultipartUploadResult{
		InitiateMultipartUploadResult: PartList{
			Bucket:   crm.Bucket,
			Key:      crm.Key,
			UploadID: crm.UploadID,
		},
	}
	return &result
}
