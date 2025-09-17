// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

const checksumType = "FULL_OBJECT"

func setChecksumFields(algorithm, value string, result *ChecksumFields) {
	checksums := map[string]*string{
		"CRC32":     &result.ChecksumCRC32,
		"CRC32C":    &result.ChecksumCRC32C,
		"CRC64NVME": &result.ChecksumCRC64NVME,
		"SHA1":      &result.ChecksumSHA1,
		"SHA256":    &result.ChecksumSHA256,
	}
	if field, ok := checksums[algorithm]; ok {
		*field = value
		result.ChecksumType = checksumType // チェックサムの値がある場合に付与
	}
}
