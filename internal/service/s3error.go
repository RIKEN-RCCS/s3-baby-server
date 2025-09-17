// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package service

import (
	"encoding/xml"
	"fmt"
	"net/http"
)

type S3Error struct {
	Status  int
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

func (e *S3Error) Error() string {
	return fmt.Sprintf("code: %d, message: %s", e.Status, e.Message)
}

func NotModified() *S3Error {
	return &S3Error{Status: http.StatusNotModified, Code: "NotModified", Message: "NotModified"}
}

func NoSuchBucket() *S3Error {
	return &S3Error{Status: http.StatusNotFound, Code: "NoSuchBucket", Message: "NoSuchBucket"}
}

func NoSuchKey() *S3Error {
	return &S3Error{Status: http.StatusNotFound, Code: "NoSuchKey", Message: "NoSuchKey"}
}

func NoSuchUpload() *S3Error {
	return &S3Error{Status: http.StatusNotFound, Code: "NoSuchUpload", Message: "NoSuchUpload"}
}

func BucketAlreadyExists() *S3Error {
	return &S3Error{Status: http.StatusConflict, Code: "BucketAlreadyExists", Message: "BucketAlreadyExists"}
}

func BucketAlreadyOwnedByYou() *S3Error {
	return &S3Error{Status: http.StatusConflict, Code: "BucketAlreadyOwnedByYou", Message: "BucketAlreadyOwnedByYou"}
}

func BucketNotEmpty() *S3Error {
	return &S3Error{Status: http.StatusConflict, Code: "BucketNotEmpty", Message: "BucketNotEmpty"}
}

func BadDigest() *S3Error {
	return &S3Error{Status: http.StatusBadRequest, Code: "BadDigest", Message: "BadDigest"}
}

func BadRequest() *S3Error {
	return &S3Error{Status: http.StatusBadRequest, Code: "BadRequest", Message: "BadRequest"}
}

func BadRequestChecksum() *S3Error {
	return &S3Error{
		Status:  http.StatusBadRequest,
		Code:    "XAmzContentChecksumMismatch",
		Message: "The provided x-amz-checksum header does not match what was computed.",
	}
}

func InvalidArgument() *S3Error {
	return &S3Error{Status: http.StatusBadRequest, Code: "InvalidArgument", Message: "InvalidArgument"}
}

func InvalidBucketName() *S3Error {
	return &S3Error{Status: http.StatusBadRequest, Code: "InvalidBucketName", Message: "InvalidBucketName"}
}

func InvalidDigest() *S3Error {
	return &S3Error{Status: http.StatusBadRequest, Code: "InvalidDigest", Message: "InvalidDigest"}
}

func InvalidStorageClass() *S3Error {
	return &S3Error{Status: http.StatusBadRequest, Code: "InvalidStorageClass", Message: "InvalidStorageClass"}
}

func InvalidTag() *S3Error {
	return &S3Error{Status: http.StatusBadRequest, Code: "InvalidTag", Message: "InvalidTag"}
}

func InvalidPart() *S3Error {
	return &S3Error{Status: http.StatusBadRequest, Code: "InvalidPart", Message: "InvalidPart"}
}

func InvalidPartOrder() *S3Error {
	return &S3Error{Status: http.StatusBadRequest, Code: "InvalidPartOrder", Message: "InvalidPartOrder"}
}

func KeyTooLongError() *S3Error {
	return &S3Error{Status: http.StatusBadRequest, Code: "KeyTooLongError", Message: "KeyTooLongError"}
}

func AccessDenied() *S3Error {
	return &S3Error{Status: http.StatusForbidden, Code: "AccessDenied", Message: "AccessDenied"}
}

func PreconditionFailed() *S3Error {
	return &S3Error{Status: http.StatusPreconditionFailed, Code: "PreconditionFailed", Message: "PreconditionFailed"}
}

func InternalError() *S3Error {
	return &S3Error{Status: http.StatusInternalServerError, Code: "InternalError", Message: "InternalError"}
}

func RangeNotSatisfiable() *S3Error {
	return &S3Error{
		Status:  http.StatusRequestedRangeNotSatisfiable,
		Code:    "InvalidRange",
		Message: "The requested range is not satisfiable",
	}
}

// func RangeNotSatisfiablePartNumber() *S3Error {
// 	return &S3Error{
// 		Status:  http.StatusRequestedRangeNotSatisfiable,
// 		Code:    "InvalidPartNumber",
// 		Message: "The requested partNumber is not satisfiable",
// 	}
// }

func InvalidRequest() *S3Error {
	return &S3Error{
		Status:  http.StatusBadRequest,
		Code:    "InvalidRequest",
		Message: "Cannot specify both Range header and partNumber query parameter",
	}
}

func NotImplemented() *S3Error {
	return &S3Error{
		Status:  http.StatusNotImplemented,
		Code:    "NotImplemented",
		Message: "A header you provided implies functionality that is not implemented.",
	}
}

func EntityTooSmallError() *S3Error {
	return &S3Error{Status: http.StatusNotImplemented, Code: "EntityTooSmallError", Message: "EntityTooSmallError"}
}
