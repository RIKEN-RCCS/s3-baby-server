// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

type ListPartsState struct {
	BucketAPath, MarkerAPath string
	Prefix, Delimiter        string
	Bucket, Key              string
	KeyMarker                string
	UploadID                 string
	Target                   int
	MaxParts                 int
	URLFlag                  bool
}

type ListPartsStateResult struct {
	NextMarker  int
	IsTruncated bool
}

func (lp ListPartsState) MakeListPartsResult(res ListPartsStateResult) *ListPartsResult {
	result := ListPartsResult{
		Bucket:               lp.Bucket,
		Key:                  lp.Key,
		UploadID:             lp.UploadID,
		PartNumberMarker:     lp.Target,
		MaxParts:             lp.MaxParts,
		StorageClass:         "STANDARD", // STANDARDのみ
		ChecksumAlgorithm:    "",
		ChecksumType:         "",
		NextPartNumberMarker: res.NextMarker,
		IsTruncated:          res.IsTruncated,
	}
	return &result
}
