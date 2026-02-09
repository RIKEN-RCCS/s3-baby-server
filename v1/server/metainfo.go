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
// information.  ETag stores an MD5 sum.  Checksum stores a checksum.
// Headers stores "x-amz-meta-".  Tags stores tagging tags.
type Meta_info struct {
	Entity_key         string                  `json:"entity-key"`
	ETag               string                  `json:"etag"`
	Checksum_algorithm types.ChecksumAlgorithm `json:"checksum_algorithm"`
	Checksum           string                  `json:"checksum"`
	Headers            map[string]string       `json:"headers"`
	Tags               *types.Tagging          `json:"tags"`

	//ContentDisposition *string
	//ContentEncoding    *string
	//ContentLanguage    *string
	//ContentType        *string
	//Expires            *time.Time
}

// MPUL-Information.  It is stored as a file "info" in an MPUL
// temporary directory.  It corresponds to the fields of
// "types.MultipartUpload", where used ones are: {UploadId, Initiated,
// ChecksumAlgorithm, ChecksumType}.  Metainfo keeps a record which
// will be restored at MPUL completion.  Metainfo is only partially
// filled (missing ETag and entity-key slots).
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
	Checksum      types.ChecksumAlgorithm `json:"checksum"`
	Checksum_type types.ChecksumType      `json:"checksum-type"`
	Metainfo      *Meta_info              `json:"metainfo"`
}

// MPUL-Catalog.  It is stored as a file "list" in an MPUL temporary
// directory.
type Mpul_catalog struct {
	Parts []Mpul_part `json:"parts"`
}

// (types.CopyObjectResult, CopyPartResult)
type Mpul_part struct {
	Entity_key string    `json:"entity-key"`
	Size       int64     `json:"size"`
	ETag       string    `json:"etag"`
	Checksum   string    `json:"checksum"`
	Mtime      time.Time `json:"mtime"`
}
