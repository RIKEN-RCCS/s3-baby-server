// checksum.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Various parameters for CRC can be found (for example) at:
// https://reveng.sourceforge.io/crc-catalogue/all.htm

package server

import (
	//"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	//"errors"
	"hash"
	"hash/crc32"
	"hash/crc64"
	//"crypto/rand"
	//"fmt"
	"io"
	"log"
	"os"
	//"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Generator polynomial of NVME.  The value is defined as
// 0xad93d23594c93659 in "NVM Express; NVM Command Set Specification".
// Note polynomials in Golang hash/crc64 are bit reversed (in
// little-endian).

const poly_nvme = 0x9a6c9329ac4bc9b5

// Calculates two checksum, md5 and one requested.  It skips one when
// algorithm="".  An algorithm key is types.ChecksumAlgorithm, and one
// of {"CRC32", "CRC32C", "SHA1", "SHA256", "CRC64NVME"}.
func (bbs *Bb_server) calculate_csum2(algorithm types.ChecksumAlgorithm, object string, scratch string) ([]byte, []byte, *Aws_s3_error) {
	var location = "/" + object
	var name = bbs.make_path_of_object(object, scratch)

	var stat, err1 = os.Lstat(name)
	if err1 != nil {
		bbs.logger.Info("os.Lstat() failed in calculate_csum2",
			"file", name, "error", err1)
		return nil, nil, map_os_error(location, err1, nil)
	}
	var f1, err2 = os.Open(name)
	if err2 != nil {
		bbs.logger.Warn("os.Open() failed", "file", name, "error", err2)
		return nil, nil, map_os_error(location, err2, nil)
	}
	defer func() {
		var err3 = f1.Close()
		if err3 != nil {
			bbs.logger.Warn("os.Close() failed", "file", name, "error", err3)
		}
	}()

	var hash1 hash.Hash = md5.New()

	var hash2 hash.Hash
	switch algorithm {
	case "":
		hash2 = nil
	case types.ChecksumAlgorithmCrc32:
		//strings.EqualFold(algorithm, "CRC32"):
		hash2 = crc32.NewIEEE()
	case types.ChecksumAlgorithmCrc32c:
		//strings.EqualFold(algorithm, "CRC32C"):
		hash2 = crc32.New(crc32.MakeTable(crc32.Castagnoli))
	case types.ChecksumAlgorithmCrc64nvme:
		//strings.EqualFold(algorithm, "CRC64NVME"):
		hash2 = crc64.New(crc64.MakeTable(poly_nvme))
	case types.ChecksumAlgorithmSha1:
		//strings.EqualFold(algorithm, "SHA1"):
		hash2 = sha1.New()
	case types.ChecksumAlgorithmSha256:
		//strings.EqualFold(algorithm, "SHA256"):
		hash2 = sha256.New()
	default:
		log.Fatalf("Bad s3/types.ChecksumAlgorithm: %s", algorithm)
	}

	var writer io.Writer
	if hash2 != nil {
		writer = io.MultiWriter(hash1, hash2)
	} else {
		writer = hash1
	}
	var count, err4 = io.Copy(writer, f1)
	if err4 != nil {
		return nil, nil, map_os_error(location, err4, nil)
	}
	if count != stat.Size() {
		bbs.logger.Info("io.Copy() failed, bad copy size")
		var err5 = &Aws_s3_error{Code: InternalError,
			Message:  "io.Copy() failed, incomplete copy",
			Resource: location}
		return nil, nil, err5
	}

	//var sum []byte = hash1.Sum(nil)
	//var sum []byte = hash1.Sum(nil)
	//var s = hex.EncodeToString(sum)
	//var s = base64.StdEncoding.EncodeToString(sum)
	//return sum, nil

	if hash2 != nil {
		var sum1 []byte = hash1.Sum(nil)
		var sum2 []byte = hash2.Sum(nil)
		return sum1, sum2, nil
	} else {
		var sum1 []byte = hash1.Sum(nil)
		return sum1, nil, nil
	}
}

func fill_checksum_record(algorithm types.ChecksumAlgorithm, csum []byte) *types.Checksum {
	var cs = types.Checksum{
		ChecksumType: types.ChecksumTypeFullObject,
	}
	var csum1 = base64.StdEncoding.EncodeToString(csum)
	switch algorithm {
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
