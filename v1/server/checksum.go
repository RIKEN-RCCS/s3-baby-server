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
	var cs = types.Checksum{
		ChecksumType: types.ChecksumTypeFullObject,
	}
	var csum1 = base64.StdEncoding.EncodeToString(csum)
	switch checksum {
	case types.ChecksumAlgorithmCrc32:
		cs.ChecksumCRC32 = &csum1
	case types.ChecksumAlgorithmCrc32c:
		cs.ChecksumCRC32C = &csum1
	case types.ChecksumAlgorithmCrc64nvme:
		cs.ChecksumCRC64NVME = &csum1
	case types.ChecksumAlgorithmSha1:
		cs.ChecksumSHA1 = &csum1
	case types.ChecksumAlgorithmSha256:
		cs.ChecksumSHA256 = &csum1
	}
	return &cs
}
