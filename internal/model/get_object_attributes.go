// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

import "os"

type GetObjectAttributesState struct {
	Allowed      []string
	ETag         string
	Checksum     string
	ObjectParts  bool
	StorageClass string
	ObjectSize   int64
	MaxParts     string
	Marker       int
	Info         os.FileInfo
}

const (
	maxParts = 10000
	partsNum = 1
)

func (g GetObjectAttributesState) MakeGetObjectAttributesResult() *GetObjectAttributesResult {
	result := GetObjectAttributesResult{
		LastModified: g.Info.ModTime(),
	}
	if g.ETag != "" {
		result.GetObjectAttributesContents.ETag = g.ETag
	}
	if g.Checksum != "" {
		result.GetObjectAttributesContents.Checksum = &Checksum{ChecksumCRC64NVME: g.Checksum}
	}
	if g.ObjectParts && g.Marker <= 1 {
		result.GetObjectAttributesContents.ObjectParts = &ObjectParts{
			IsTruncated:          false,
			MaxParts:             maxParts,
			NextPartNumberMarker: 1,
			PartNumberMarker:     g.Marker,
			PartsCount:           partsNum,
			Parts: &Parts{
				PartNumber: "1", Size: g.Info.Size()}}
	}
	if g.ObjectParts && g.Marker > 1 {
		result.GetObjectAttributesContents.ObjectParts = &ObjectParts{
			IsTruncated:          true,
			MaxParts:             maxParts,
			NextPartNumberMarker: 0,
			PartNumberMarker:     g.Marker,
			PartsCount:           partsNum,
			Parts:                nil,
		}
	}
	if g.StorageClass != "" {
		result.GetObjectAttributesContents.StorageClass = g.StorageClass
	}
	if g.ObjectSize != 0 {
		result.GetObjectAttributesContents.ObjectSize = g.ObjectSize
	}
	return &result
}
