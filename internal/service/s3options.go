// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package service

type S3Options interface {
	GetBucket() string
	GetKey() string
	GetPath() string
	GetBody() []byte
	GetOption(string) string
	HeaderQueryCheck([]string) bool
	Validate(map[string]string) bool
	CheckErrorHeader() bool
}
