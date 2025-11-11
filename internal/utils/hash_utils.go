// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package utils

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"hash"
	"log/slog"

	"encoding/base64"
	"encoding/hex"

	"hash/crc32"
	"hash/crc64"
	"io"
	"os"
)

const crc64NVME = 0x9a6c9329ac4bc9b5

func base64encode(h hash.Hash) []byte {
	value := h.Sum(nil)
	sum64 := make([]byte, base64.StdEncoding.EncodedLen(len(value)))
	base64.StdEncoding.Encode(sum64, value)
	return sum64
}

func CalcMD5(filePath string, logger *slog.Logger) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer CloseFile(file, logger)
	hash := md5.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func CalcMD5File(file *os.File) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func CalcMD5Byte(body []byte) (string, error) {
	sum := md5.Sum(body)
	return hex.EncodeToString(sum[:]), nil
}

func CalcContentMD5(etag string) (string, error) {
	bin, err := hex.DecodeString(etag)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bin), nil
}

func ChecksumCrc32(filePath string, logger *slog.Logger) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer CloseFile(file, logger)
	hash := crc32.NewIEEE()
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	return string(base64encode(hash)), nil
}

func ChecksumCrc32c(filePath string, logger *slog.Logger) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer CloseFile(file, logger)
	hash := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	return string(base64encode(hash)), nil
}

func ChecksumCrc64nvme(filePath string, logger *slog.Logger) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer CloseFile(file, logger)
	hash := crc64.New(crc64.MakeTable(crc64NVME))
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	return string(base64encode(hash)), nil
}

func ChecksumCrc64nvmeBody(content []byte) (string, error) {
	hash := crc64.New(crc64.MakeTable(crc64NVME))
	reader := bytes.NewReader(content)
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return string(base64encode(hash)), nil
}

func ChecksumSha1(filePath string, logger *slog.Logger) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer CloseFile(file, logger)
	hash := sha1.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	return string(base64encode(hash)), nil
}

func ChecksumSha256(filePath string, logger *slog.Logger) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer CloseFile(file, logger)
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	return string(base64encode(hash)), nil
}
