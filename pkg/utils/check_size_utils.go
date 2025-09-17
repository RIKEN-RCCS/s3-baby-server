// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package utils

import (
	"encoding/base64"
	"os"
)

func CheckPartNumber(partNumber string) bool {
	minThresh := 1
	maxThresh := 10000
	pNum := ToInt(partNumber)
	if pNum < minThresh || pNum > maxThresh {
		return false
	}
	return true
}

func LimitCheck(dirs []os.DirEntry) string {
	maxThresh := 10000
	if len(dirs) >= maxThresh {
		fn := []byte("continuation token: " + ToString(maxThresh-1))
		c := base64.StdEncoding.EncodeToString(fn)
		return c
	}
	return ""
}
