// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

func (cl ListObjectsState) MakeListObjectsV2Result(
	res ListObjectsStateResult,
) *ListObjectsV2Result { // ListObjectsと同様の処理のため流用
	result := ListObjectsV2Result{
		Bucket:                cl.Bucket,
		KeyCount:              res.Cnt,
		NextContinuationToken: res.NextMarker,
		IsTruncated:           res.IsTruncated,
		Prefix:                cl.Prefix,
		Delimiter:             cl.Delimiter,
		MaxKeys:               cl.MaxKeys,
		ContinuationToken:     cl.ContinuationToken,
		StartAfter:            cl.StartAfter,
	}
	if cl.URLFlag {
		encodingType := "url"
		result.EncodingType = encodingType
	}
	for _, g := range res.Dirs {
		result.CommonPrefixes = append(result.CommonPrefixes, CommonPrefixes{Prefix: g})
	}
	return &result
}
