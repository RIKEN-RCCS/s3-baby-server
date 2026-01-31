// checksum.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Various parameters for CRC can be found (for example) at:
// https://reveng.sourceforge.io/crc-catalogue/all.htm

package server

import (
	//"bytes"
	//"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	//"errors"
	"hash"
	"hash/crc32"
	"hash/crc64"
	//"crypto/rand"
	//"fmt"
	//"io"
	"log"
	//"os"
	//"github.com/aws/aws-sdk-go-v2/service/s3"
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

func fill_checksum_record(checksum types.ChecksumAlgorithm, csum []byte) *types.Checksum {
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
