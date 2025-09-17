// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

import "os"

type ListBucketsState struct {
	MaxBuckets        int
	Prefix            string
	Dirs              []os.DirEntry
	ContinuationToken string
}

func (lb ListBucketsState) MakeListBucketsResult() *ListBucketsResult {
	res := Buckets{}
	for _, entry := range lb.Dirs {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.IsDir() {
			res.Bucket = append(res.Bucket, Bucket{
				Name:         info.Name(),
				CreationDate: info.ModTime(),
			})
		}
	}
	result := ListBucketsResult{}
	result.Buckets = append(result.Buckets, Buckets{Bucket: res.Bucket})
	if lb.Prefix != "" {
		result.Prefix = lb.Prefix
	}
	if lb.ContinuationToken != "" {
		result.ContinuationToken = lb.ContinuationToken
	}
	return &result
}
