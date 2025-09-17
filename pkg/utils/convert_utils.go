// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package utils

import (
	"encoding/binary"
	"strconv"
)

func ToString(num int) string {
	return strconv.Itoa(num)
}

func ToBytes(n int) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(n))
	return b
}

func ToInt(text string) int {
	num, err := strconv.Atoi(text)
	if err != nil {
		// slog.Error("failed string to int")
		return 0
	}
	return num
}

func ToInt64(text string) int64 {
	num, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		// slog.Error("failed string to int64")
		return 0
	}
	return num
}
