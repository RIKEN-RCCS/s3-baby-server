// metainfo.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Meta-information associated to an object.

package server

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Meta-information associated to an object.  It is stored in a hidden
// file in json.  Entity_key is used to check the validity of
// information.  ETag stores an MD5 sum.  Csum stores a checksum.
// Headers stores "x-amz-meta-".  Tags stores tagging tags.  The part
// of content-headers is stored but not used.
type Meta_info struct {
	Entity_key         string                  `json:"entity-key"`
	ETag               string                  `json:"etag"`
	Checksum           types.ChecksumAlgorithm `json:"checksum"`
	Csum               string                  `json:"csum"`
	Headers            map[string]string       `json:"headers"`
	Tags               *types.Tagging          `json:"tags"`
	CacheControl       *string                 `json:"cache-control"`
	ContentDisposition *string                 `json:"content-disposition"`
	ContentEncoding    *string                 `json:"content-encoding"`
	ContentLanguage    *string                 `json:"content-language"`
	ContentType        *string                 `json:"content-type"`
	Expires            *time.Time              `json:"expires"`
}

// MPUL-Information.  It is stored as a file "info" in an MPUL
// temporary directory.  It corresponds to some of the slots of
// "types.MultipartUpload".  Metainfo records what will be stored as
// metainfo at MPUL completion.  Note metainfo is only partially
// filled (missing entity-key and ETag slots).
//
// The "types.MultipartUpload" has fields:
//   - ChecksumAlgorithm ChecksumAlgorithm
//   - ChecksumType ChecksumType
//   - Initiated *time.Time
//   - Initiator *Initiator
//   - Key *string
//   - Owner *Owner
//   - StorageClass StorageClass
//   - UploadId *string
type Mpul_info struct {
	Upload_id     *string                 `json:"upload-id"`
	Initiate_time *time.Time              `json:"initiate-time"`
	Checksum_type types.ChecksumType      `json:"checksum-type"`
	Checksum      types.ChecksumAlgorithm `json:"checksum"`
	Metainfo      *Meta_info              `json:"metainfo"`
}

// MPUL-Catalog.  It is stored as a file "list" in an MPUL temporary
// directory.
type Mpul_catalog struct {
	Parts []Mpul_part `json:"parts"`
}

// (types.CopyObjectResult, CopyPartResult)
type Mpul_part struct {
	Entity_key string                  `json:"entity-key"`
	ETag       string                  `json:"etag"`
	Size       int64                   `json:"size"`
	Mtime      time.Time               `json:"mtime"`
	Checksum   types.ChecksumAlgorithm `json:"checksum"`
	Csum       string                  `json:"csum"`
}
