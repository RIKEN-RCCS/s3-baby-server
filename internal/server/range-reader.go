// range-reader.go
// Copyright 2025-2025 RIKEN R-CCS.
// SPDX-License-Identifier: BSD-2-Clause

// Range_reader is a reader of a file in the given range.  It is type
// io.ReadCloser.  io.ReadCloser extends io.Reader and io.Closer.
// io.LimitedReader cannot be used because it is not io.Closer.
//
// type ReadCloser interface {Reader; Closer}
// - Read(p []byte) (n int, err error)
// - Close() error

package server

import (
	//"bytes"
	//"context"
	//"crypto/md5"
	//"crypto/sha1"
	//"crypto/sha256"
	//"encoding/base64"
	//"errors"
	//"hash"
	//"hash/crc32"
	//"hash/crc64"
	//"log/slog"
	//"crypto/rand"
	//"encoding/json"
	//"fmt"
	"io"
	//"io/fs"
	"log"
	"os"
	//"path"
	//"path/filepath"
	//"github.com/aws/aws-sdk-go-v2/service/s3"
	//"github.com/aws/aws-sdk-go-v2/service/s3/types"
	//"regexp"
	//"s3-baby-server/pkg/utils"
	//"strconv"
	//"strings"
)

type Range_reader struct {
	f *os.File
	extent [2]int64
	pos int64
}

func (s *Range_reader) Read(p []byte) (n int, err error) {
	if s.pos < s.extent[0] {
		// The file is in an unexpected state.
		return 0, io.ErrUnexpectedEOF
	}
	var lim int64 = min(s.extent[1] - s.pos, int64(len(p)))
	if lim <= 0 {
		return 0, io.EOF
	}
	var n1, err1 = s.f.Read(p)
	if err1 != nil {
		return n1, err1
	}
	s.pos += int64(n1)
	return n1, nil
}

func (s *Range_reader) Close() error {
	var err1 = s.f.Close()
	return err1
}

// NEW_RANGE_READER makes a range reader.  A range should be within
// the file size.  It does not close the underlying os.File on errors.
func New_range_reader(f *os.File, extent [2]int64) (*Range_reader, error) {
	var pos, err1 = f.Seek(extent[0], 0)
	if err1 != nil {
		return nil, err1
	}
	if pos < extent[0] {
		log.Fatalf("os.Seek returned incomplete")
		return nil, io.ErrUnexpectedEOF
	}
	return &Range_reader{f, extent, pos}, nil
}
