// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Checksum with CRC64-NVME

// MEMO: Various parameters of CRC can be found (for example) at:
// https://reveng.sourceforge.io/crc-catalogue/all.htm

package server

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"hash"
	"hash/crc32"
	"hash/crc64"
	"log"

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
func (bbs *Bb_server) decode_checksum_union(rid uint64, object string, csumset *types.Checksum) (types.ChecksumAlgorithm, []byte, *Aws_s3_error) {
	var location = "/" + object
	bb_assert(csumset != nil)
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
			"rid", rid, "checksum-value", *csum1)
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
func (bbs *Bb_server) reject_composite_checksum(rid uint64, object string, checksumtype types.ChecksumType) *Aws_s3_error {
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
