// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

import (
	"strings"
)

type ListObjectsState struct {
	BucketAPath, MarkerAPath string
	Prefix, Delimiter        string
	Bucket                   string
	StartAfter               string
	Marker                   string
	ContinuationToken        string
	Target                   int
	MaxKeys                  int
	URLFlag                  bool
	V2Flg                    bool
}

type ListObjectsStateResult struct {
	Cnt         int
	NextMarker  string
	IsTruncated bool
	Dirs        []string
}

func (cl ListObjectsState) MakeListObjectsResult(res ListObjectsStateResult) *ListObjectsResult {
	result := ListObjectsResult{
		Bucket:      cl.Bucket,
		NextMarker:  strings.ReplaceAll(res.NextMarker, "\\", "/"),
		IsTruncated: res.IsTruncated,
		Prefix:      cl.Prefix,
		Delimiter:   cl.Delimiter,
		MaxKeys:     cl.MaxKeys,
	}
	if cl.URLFlag {
		encodingType := "url"
		result.EncodingType = encodingType
	}
	if cl.Marker != "" {
		cl.Marker = strings.Trim(cl.Marker, "\\")
		result.Marker = strings.ReplaceAll(cl.Marker, "\\", "/")
	}
	for _, g := range res.Dirs {
		result.CommonPrefixes = append(result.CommonPrefixes, CommonPrefixes{Prefix: g})
	}
	return &result
}
