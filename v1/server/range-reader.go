// range-reader.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Range_reader is a reader of a file in the given range.  It is type
// io.ReadCloser.  io.ReadCloser extends io.Reader and io.Closer.
// io.LimitedReader and io.SectionReader are similar but they cannot
// directly be used because it is not io.Closer.

package server

import (
	"io"
	//"log"
	"os"
)

type Range_reader struct {
	*io.SectionReader
	f *os.File
}

func (r *Range_reader) Close() error {
	return r.f.Close()
}

// NEW_RANGE_READER makes a range reader.  A range should be within
// the file size.  It does not close the underlying os.File on errors.
func New_range_reader(f *os.File, extent *[2]int64) io.ReadCloser {
	if extent == nil {
		return f
	} else {
		var r = io.NewSectionReader(f, extent[0], extent[1]-extent[0])
		return &Range_reader{SectionReader: r, f: f}
	}
}
