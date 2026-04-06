// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Checksum with CRC64-NVME

// MEMO: Various parameters of CRC can be found (for example) at:
// https://reveng.sourceforge.io/crc-catalogue/all.htm

package server

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"hash"
	"hash/crc32"
	"hash/crc64"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Generator polynomial of NVME.  The value is defined as
// 0xad93d23594c93659 in "NVM Express; NVM Command Set Specification".
// Note polynomials in Golang hash/crc64 are bit reversed (in
// little-endian).

const polynomial_nvme = 0x9a6c9329ac4bc9b5

// CHECKSUM_ALGORITHM maps the name of checksum to a hash function.  It
// returns nil on an unknown algorithm name.
func checksum_algorithm(checksum types.ChecksumAlgorithm) hash.Hash {
	var hash2 hash.Hash
	switch checksum {
	case "":
		hash2 = nil
	case types.ChecksumAlgorithmCrc32:
		hash2 = crc32.NewIEEE()
	case types.ChecksumAlgorithmCrc32c:
		hash2 = crc32.New(crc32.MakeTable(crc32.Castagnoli))
	case types.ChecksumAlgorithmCrc64nvme:
		hash2 = crc64.New(crc64.MakeTable(polynomial_nvme))
	case types.ChecksumAlgorithmSha1:
		hash2 = sha1.New()
	case types.ChecksumAlgorithmSha256:
		hash2 = sha256.New()
	default:
		log.Fatalf("Bad s3/types.ChecksumAlgorithm: %s", checksum)
	}
	return hash2
}

// FILL_CHECKSUM_UNION fills types.Checksum which is a union of
// checksums.
func fill_checksum_union(checksum types.ChecksumAlgorithm, csum []byte) *types.Checksum {
	var csumset types.Checksum
	var cs = base64.StdEncoding.EncodeToString(csum)
	csumset.ChecksumType = types.ChecksumTypeFullObject
	switch checksum {
	case types.ChecksumAlgorithmCrc32:
		csumset.ChecksumCRC32 = &cs
	case types.ChecksumAlgorithmCrc32c:
		csumset.ChecksumCRC32C = &cs
	case types.ChecksumAlgorithmCrc64nvme:
		csumset.ChecksumCRC64NVME = &cs
	case types.ChecksumAlgorithmSha1:
		csumset.ChecksumSHA1 = &cs
	case types.ChecksumAlgorithmSha256:
		csumset.ChecksumSHA256 = &cs
	}
	return &csumset
}

// DECODE_CHECKSUM_UNION decodes types.Checksum.  It will return
// nothing silently when no checksum is given.
func (bbs *Bbs_server) decode_checksum_union(rid uint64, object string, csumset *types.Checksum) (types.ChecksumAlgorithm, []byte, *Aws_s3_error) {
	var location = "/" + object
	bbs_assert(csumset != nil)
	var checksumtype = csumset.ChecksumType
	if checksumtype != "" && checksumtype != types.ChecksumTypeFullObject {
		bbs.logger.Info("Checksum by composite-object unsupported",
			"rid", rid)
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message:  "Checksum by composite-object unsupported.",
			Resource: location}
		return "", nil, errz
	}
	var checksum types.ChecksumAlgorithm
	var csum1 *string
	var count = 0
	if csumset.ChecksumCRC32 != nil {
		checksum = types.ChecksumAlgorithmCrc32
		csum1 = csumset.ChecksumCRC32
		count++
	}
	if csumset.ChecksumCRC32C != nil {
		checksum = types.ChecksumAlgorithmCrc32c
		csum1 = csumset.ChecksumCRC32C
		count++
	}
	if csumset.ChecksumCRC64NVME != nil {
		checksum = types.ChecksumAlgorithmCrc64nvme
		csum1 = csumset.ChecksumCRC64NVME
		count++
	}
	if csumset.ChecksumSHA1 != nil {
		checksum = types.ChecksumAlgorithmSha1
		csum1 = csumset.ChecksumSHA1
		count++
	}
	if csumset.ChecksumSHA256 != nil {
		checksum = types.ChecksumAlgorithmSha256
		csum1 = csumset.ChecksumSHA256
		count++
	}
	if csum1 == nil {
		// No checksum is given.
		return "", nil, nil
	}
	if count >= 2 {
		bbs.logger.Info("Multiple checksum values are specified",
			"rid", rid)
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message:  "Multiple checksum values are specified.",
			Resource: location}
		return "", nil, errz
	}
	var csum, err5 = base64.StdEncoding.DecodeString(*csum1)
	if err5 != nil {
		bbs.logger.Info("Bad checksum encoding",
			"rid", rid, "checksum", *csum1, "error", err5)
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "Bad checksum encoding.",
			Resource: location}
		return "", nil, errz
	}
	return checksum, csum, nil
}

// REJECT_COMPOSITE_CHECKSUM rejects other than the full-object
// checksum type.  Baby-server can only handle
// "types.ChecksumTypeFullObject".  The returned checksum is always
// for full-object.
func (bbs *Bbs_server) reject_composite_checksum(rid uint64, object string, checksumtype types.ChecksumType) *Aws_s3_error {
	var location = "/" + object
	if checksumtype != "" && checksumtype != types.ChecksumTypeFullObject {
		bbs.logger.Info("Checksum by not full-object unsupported",
			"rid", rid, "checksum-type", checksumtype)
		var errz = &Aws_s3_error{Code: NotImplemented,
			Message:  "Checksum by not full-object unsupported.",
			Resource: location}
		return errz
	}
	return nil
}

func encode_base64(object string, csum []byte) (string, *Aws_s3_error) {
	var s = base64.StdEncoding.EncodeToString(csum)
	return s, nil
}

func decode_base64(object string, csum *string) ([]byte, *Aws_s3_error) {
	if csum == nil {
		return nil, nil
	} else {
		var location = "/" + object
		var csum2, err5 = base64.StdEncoding.DecodeString(*csum)
		if err5 != nil {
			var errz = &Aws_s3_error{Code: InvalidArgument,
				Message:  "Bad base64 (MD5) encoding.",
				Resource: location}
			return nil, errz
		}
		return csum2, nil
	}
}

func (bbs *Bbs_server) check_trailer_checksum(ctx context.Context, rid uint64, object string) (types.ChecksumAlgorithm, *Aws_s3_error) {
	var location = "/" + object
	var _, r = get_handler_arguments(ctx)
	var h = r.Header
	var keys = h["X-Amz-Trailer"]
	if len(keys) == 0 {
		return "", nil
	}
	var acc []types.ChecksumAlgorithm
	for _, k := range keys {
		var checksum = map_header_name_to_checksum_algorithm(k)
		if checksum != "" {
			acc = append(acc, checksum)
		}
	}
	if len(acc) == 0 {
		return "", nil
	} else if len(acc) == 1 {
		return acc[0], nil
	} else {
		bbs.logger.Info("Multiple checksum headers in trailer",
			"rid", rid, "object", object, "trailer", keys)
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "Multiple checksum headers in trailer.",
			Resource: location}
		return "", errz
	}
}

func (bbs *Bbs_server) extract_trailer_checksum(ctx context.Context, rid uint64, object string, checksum types.ChecksumAlgorithm) ([]byte, *Aws_s3_error) {
	var location = "/" + object
	var _, r = get_handler_arguments(ctx)
	var h = r.Header
	var k = map_checksum_algorithm_to_header_name(checksum)
	if k == "" {
		return nil, nil
	}
	var v = h.Get(k)
	if v == "" {
		bbs.logger.Info("Specified trailer checksum missing",
			"rid", rid, "trailer", k)
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "Specified trailer checksum missing.",
			Resource: location}
		return nil, errz
	}
	var csum, err2 = base64.StdEncoding.DecodeString(v)
	if err2 != nil {
		bbs.logger.Info("Bad checksum encoding",
			"rid", rid, "checksum", v, "error", err2)
		var errz = &Aws_s3_error{Code: InvalidArgument,
			Message:  "Bad checksum encoding.",
			Resource: location}
		return nil, errz
	}
	return csum, nil
}

func map_header_name_to_checksum_algorithm(s string) types.ChecksumAlgorithm {
	var k = strings.ToLower(s)
	switch k {
	case "x-amz-checksum-crc32":
		return types.ChecksumAlgorithmCrc32
	case "x-amz-checksum-crc32c":
		return types.ChecksumAlgorithmCrc32c
	case "x-amz-checksum-crc64nvme":
		return types.ChecksumAlgorithmCrc64nvme
	case "x-amz-checksum-sha1":
		return types.ChecksumAlgorithmSha1
	case "x-amz-checksum-sha256":
		return types.ChecksumAlgorithmSha256
	default:
		return ""
	}
}

func map_checksum_algorithm_to_header_name(k types.ChecksumAlgorithm) string {
	switch k {
	case types.ChecksumAlgorithmCrc32:
		return "x-amz-checksum-crc32"
	case types.ChecksumAlgorithmCrc32c:
		return "x-amz-checksum-crc32c"
	case types.ChecksumAlgorithmCrc64nvme:
		return "x-amz-checksum-crc64nvme"
	case types.ChecksumAlgorithmSha1:
		return "x-amz-checksum-sha1"
	case types.ChecksumAlgorithmSha256:
		return "x-amz-checksum-sha256"
	default:
		return ""
	}
}
