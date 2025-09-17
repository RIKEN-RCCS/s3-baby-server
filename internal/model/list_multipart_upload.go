// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

import "os"

type ListMultipartUploadsState struct {
	Bucket                    string
	MaxUploads                int
	Prefix, Delimiter         string
	UploadIDMarker, KeyMarker string
	Target                    int
	Dirs                      []os.DirEntry
	ContinuationToken         string
	URLFlag                   bool
}

type ListMultipartUploadsStateResult struct {
	NextUploadIDMarker string
	IsTruncated        bool
	NextKeyMarker      string
	Paths              []string
}

func (lm ListMultipartUploadsState) MakeListMultipartUploadsResult(
	res ListMultipartUploadsStateResult,
) *ListMultipartUploadsResult {
	result := ListMultipartUploadsResult{
		Bucket:             lm.Bucket,
		KeyMarker:          lm.KeyMarker,
		UploadIDMarker:     lm.UploadIDMarker,
		MaxUploads:         lm.MaxUploads,
		Prefix:             lm.Prefix,
		Delimiter:          lm.Delimiter,
		NextKeyMarker:      res.NextKeyMarker,
		NextUploadIDMarker: res.NextUploadIDMarker,
		IsTruncated:        res.IsTruncated,
	}
	if lm.URLFlag {
		encodingType := "url"
		result.EncodingType = encodingType
	}
	for _, g := range res.Paths {
		result.CommonPrefixes = append(result.CommonPrefixes, CommonPrefixes{Prefix: g})
	}
	return &result
}
