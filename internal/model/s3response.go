// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

import (
	"encoding/xml"
	"time"
)

type ResponseError struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

type NotSatisfiableError struct {
	XMLName       xml.Name `xml:"Error"`
	Code          string   `xml:"Code"`
	Message       string   `xml:"Message"`
	Key           string   `xml:"Key"`
	Bucket        string   `xml:"BucketName"`
	Resource      string   `xml:"Resource"` // /{Bucket}/{Key}
	ContentLength int      `xml:"ActualObjectSize,omitempty"`
	ContentRange  string   `xml:"RangeRequested,omitempty"`
}

type CompleteMultipartUploadRequest struct {
	Part []PartsRequest `xml:"Part"`
}

type ChecksumFields struct {
	ChecksumCRC32     string `xml:"ChecksumCRC32,omitempty"`
	ChecksumCRC32C    string `xml:"ChecksumCRC32C,omitempty"`
	ChecksumCRC64NVME string `xml:"ChecksumCRC64NVME,omitempty"`
	ChecksumSHA1      string `xml:"ChecksumSHA1,omitempty"`
	ChecksumSHA256    string `xml:"ChecksumSHA256,omitempty"`
	ChecksumType      string `xml:"ChecksumType,omitempty"`
}

type PartsRequest struct {
	ChecksumFields
	ETag       string `xml:"ETag"`
	PartNumber string `xml:"PartNumber"`
}

type CompleteMultipartUploadResult struct {
	XMLName xml.Name `xml:"CompleteMultipartUploadResult"`
	Bucket  string   `xml:"Bucket"`
	Key     string   `xml:"Key"`
	ETag    string   `xml:"ETag"`
	ChecksumFields
	NotSatisfiableError `xml:"Error,omitempty"`
}

type CopyObjectResult struct {
	XMLName      xml.Name  `xml:"CopyObjectResult"`
	ETag         string    `xml:"ETag"`
	LastModified time.Time `xml:"LastModified"`
	ChecksumFields
	NotSatisfiableError `xml:"Error,omitempty"`
}

type CreateMultipartUploadResult struct {
	InitiateMultipartUploadResult PartList `xml:"InitiateMultipartUploadResult"`
	ChecksumAlgorithm             string   `xml:"ChecksumAlgorithm,omitempty"`
	ChecksumType                  string   `xml:"ChecksumType,omitempty"`
}

type PartList struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult" json:"-"`
	Bucket   string   `xml:"Bucket"                        json:"Bucket"`
	Key      string   `xml:"Key"                           json:"Key"`
	UploadID int      `xml:"UploadId"                      json:"UploadId"`
}

type DeleteRequest struct {
	Objects []ObjectKey `xml:"Object"`
	Quiet   bool        `xml:"Quiet,omitempty"`
}

type DeleteObjectsResult struct {
	Deleted []ObjectKey `xml:"Deleted"`
	Error   []ObjectKey `xml:"Error"`
}

type ObjectKey struct {
	Key string `xml:"Key"`
}

type ListBucketsResult struct {
	XMLName           xml.Name  `xml:"ListAllMyBucketsResult"`
	Buckets           []Buckets `xml:"Buckets"`
	Prefix            string    `xml:"Prefix,omitempty"`
	ContinuationToken string    `xml:"ContinuationToken,omitempty"`
}

type Buckets struct {
	Bucket []Bucket `xml:"Bucket"`
}

type Bucket struct {
	CreationDate time.Time `xml:"CreationDate"`
	Name         string    `xml:"Name"`
}

type Tagging struct {
	XMLName xml.Name `xml:"Tagging" json:"-"`
	TagSet  TagSet   `xml:"TagSet"  json:"TagSet"`
}

type TagSet struct {
	Tags []Tag `xml:"Tag" json:"Tag"`
}

type Tag struct {
	Key   string `xml:"Key"   json:"Key"`
	Value string `xml:"Value" json:"Value"`
}

type PutObjectResult struct {
	ETag string `xml:"ETag"`
	ChecksumFields
	NotSatisfiableError `       xml:"Error,omitempty"`
}

type ListObjectsResult struct {
	XMLName        xml.Name         `xml:"ListBucketResult"`
	Bucket         string           `xml:"Name"`
	Prefix         string           `xml:"Prefix"`
	Delimiter      string           `xml:"Delimiter"`
	Marker         string           `xml:"Marker"`
	NextMarker     string           `xml:"NextMarker"`
	MaxKeys        int              `xml:"MaxKeys"`
	IsTruncated    bool             `xml:"IsTruncated"`
	Contents       []Contents       `xml:"Contents"`
	CommonPrefixes []CommonPrefixes `xml:"CommonPrefixes,omitempty"`
	EncodingType   string           `xml:"EncodingType,omitempty"`
}

type ListObjectsV2Result struct {
	XMLName               xml.Name         `xml:"ListBucketResult"`
	XMLNS                 string           `xml:"xmlns,attr"`
	Bucket                string           `xml:"Name"`
	Prefix                string           `xml:"Prefix"`
	Delimiter             string           `xml:"Delimiter"`
	StartAfter            string           `xml:"StartAfter,omitempty"`
	KeyCount              int              `xml:"KeyCount"`
	MaxKeys               int              `xml:"MaxKeys"`
	IsTruncated           bool             `xml:"IsTruncated"`
	Contents              []Contents       `xml:"Contents"`
	CommonPrefixes        []CommonPrefixes `xml:"CommonPrefixes,omitempty"`
	ContinuationToken     string           `xml:"ContinuationToken,omitempty"`
	NextContinuationToken string           `xml:"NextContinuationToken,omitempty"`
	EncodingType          string           `xml:"EncodingType,omitempty"`
}

type Contents struct {
	Key          string    `xml:"Key"`
	LastModified time.Time `xml:"LastModified"`
	ETag         string    `xml:"ETag"`
	Size         int64     `xml:"Size"`
	StorageClass string    `xml:"StorageClass"`
}

type ListMultipartUploadsResult struct {
	XMLName            xml.Name         `xml:"ListMultipartUploadsResult"`
	Bucket             string           `xml:"Bucket"`
	KeyMarker          string           `xml:"KeyMarker"`
	UploadIDMarker     string           `xml:"UploadIdMarker"`
	NextKeyMarker      string           `xml:"NextKeyMarker"`
	NextUploadIDMarker string           `xml:"NextUploadIdMarker"`
	Prefix             string           `xml:"Prefix"`
	Delimiter          string           `xml:"Delimiter"`
	MaxUploads         int              `xml:"MaxUploads"`
	IsTruncated        bool             `xml:"IsTruncated"`
	Upload             []Uploads        `xml:"Upload"`
	CommonPrefixes     []CommonPrefixes `xml:"CommonPrefixes,omitempty"`
	EncodingType       string           `xml:"EncodingType,omitempty"`
}

type CommonPrefixes struct {
	Prefix string `xml:"Prefix"`
}

type Uploads struct {
	ChecksumAlgorithm string    `xml:"ChecksumAlgorithm,omitempty"`
	ChecksumType      string    `xml:"ChecksumType,omitempty"`
	Initiated         time.Time `xml:"Initiated"`
	Key               string    `xml:"Key"`
	StorageClass      string    `xml:"StorageClass"`
	UploadID          string    `xml:"UploadId"`
}

type ListPartsResult struct {
	XMLName              xml.Name `xml:"ListPartsResult"`
	Bucket               string   `xml:"Bucket"`
	Key                  string   `xml:"Key"`
	UploadID             string   `xml:"UploadId"`
	PartNumberMarker     int      `xml:"PartNumberMarker"`
	NextPartNumberMarker int      `xml:"NextPartNumberMarker"`
	MaxParts             int      `xml:"MaxParts"`
	IsTruncated          bool     `xml:"IsTruncated"`
	Part                 []Parts  `xml:"Part"`
	StorageClass         string   `xml:"StorageClass"`
	ChecksumAlgorithm    string   `xml:"ChecksumAlgorithm,omitempty"`
	ChecksumType         string   `xml:"ChecksumType,omitempty"`
}

type Parts struct {
	ChecksumCRC64NVME string    `xml:"ChecksumCRC64NVME,omitempty"`
	ETag              string    `xml:"ETag,omitempty"`
	LastModified      time.Time `xml:"LastModified,omitempty"`
	PartNumber        string    `xml:"PartNumber,omitempty"`
	Size              int64     `xml:"Size,omitempty"`
}

type GetObjectResult struct {
	LastModified       time.Time `xml:"LastModified"`
	ETag               string    `xml:"ETag"`
	ContentLength      int       `xml:"ContentLength"`
	ContentRange       string    `xml:"ContentRange,omitempty"`
	Content            []byte    `xml:"Content,omitempty"`
	ContentDisposition string    `xml:"ContentDisposition,omitempty"`
	ContentEncoding    string    `xml:"ContentEncoding,omitempty"`
	ContentLanguage    string    `xml:"ContentLanguage,omitempty"`
	ContentType        string    `xml:"ContentType,omitempty"`
	CRC64NVME          string    `xml:"CRC64NVME,omitempty"`
	TagCount           string    `xml:"TagCount,omitempty"`
	MissingMeta        string    `xml:"MissingMeta,omitempty"`
	ChecksumType       string    `xml:"ChecksumType,omitempty"`
}

type GetObjectAttributesResult struct {
	GetObjectAttributesContents GetContents `xml:"getObjectAttributesResponse"`
	LastModified                time.Time   `xml:"LastModified"`
}

type GetContents struct {
	XMLName      xml.Name     `xml:"getObjectAttributesResponse"`
	ETag         string       `xml:"ETag,omitempty"`
	Checksum     *Checksum    `xml:"Checksum,omitempty"`
	ObjectParts  *ObjectParts `xml:"ObjectParts,omitempty"`
	StorageClass string       `xml:"StorageClass,omitempty"`
	ObjectSize   int64        `xml:"ObjectSize,omitempty"`
}

type Checksum struct {
	ChecksumCRC64NVME string `xml:"ChecksumCRC64NVME,omitempty"`
}

type ObjectParts struct {
	IsTruncated          bool   `xml:"IsTruncated"`
	MaxParts             int    `xml:"MaxParts,omitempty"`
	NextPartNumberMarker int    `xml:"NextPartNumberMarker,omitempty"`
	PartNumberMarker     int    `xml:"PartNumberMarker,omitempty"`
	Parts                *Parts `xml:"Part,omitempty"`
	PartsCount           int    `xml:"PartsCount,omitempty"`
}
