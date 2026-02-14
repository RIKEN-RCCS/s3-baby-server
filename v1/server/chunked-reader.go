// chunked-reader.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// This implements a stream for Content-Encoding: "aws-chunked".
// CHUNKED_READER DOES NOT CALCULATE SIGNATURES.

// See https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-streaming.html

// This chunked-reader checks for aws-chunked by looking at headers
// "X-Amz-Content-Sha256" (and "X-Amz-Decoded-Content-Length").
// "Content-Encoding" is unreliable as it may be missing.  (Note MinIO
// MC client misses Content-Encoding header).
//
// Headers for aws-chunked look like the following
//
//   Content-Encoding: aws-chunked
//   X-Amz-Decoded-Content-Length: nnnn
//   X-Amz-Content-Sha256: STREAMING-AWS4-HMAC-SHA256-PAYLOAD
//   Content-Length: nnnn
//
// Each chunk has a "aws-chunked" header which looks like:
//
// 513;chunk-signature=be348984ab8e284170b29e2bb5a424370e4203c81c981d5eb5e8534912ac1c5d

// string to sign = previous-signature + hash("") + hash(current-chunk-data)

package server

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
)

// CHUNKED_READER is an io.Reader.  It wraps an io.Reader by
// bufio.Reader as it peeks the stream to check the existence of a
// chunk-header.  CHUNKS holds the signatures read so far.
type chunked_reader struct {
	r              *bufio.Reader
	chunked        chunked_type
	content_length int64
	content_n      int64
	payload_length int64
	payload_n      int64
	chunk_length   int64
	chunk_n        int64
	chunks         []chunk_record
}

type chunked_type int

const (
	NOT_CHUNKED chunked_type = iota
	CHUNKED_HTTP1
	CHUNKED_AWS
)

type chunk_record struct {
	chunk_length int64
	chunk_offset int64
	signature    string
}

// A chunk-header consists of length + ";" + "chunk-signature=" +
// signature + "cr+lf" + "cr+lf".  A signature is (assmued) a SHA256
// in hex string (length=64).
const chunk_header_peek int = 160

// MAKE_CHUNKED_READER returns a modified reader of the body.  Use the
// returned stream for later operation.  It may consume the underlying
// stream in order to check the chunked-ness.  This check is needed
// because http.Server's document says it handles an http1-chunked
// stream (but might not).
func (bbs *Bb_server) make_chunked_reader(ctx context.Context, stream io.Reader, length int64, body io.Reader, q *http.Request) (io.Reader, chunked_type, error) {
	if check_aws_chunked(q) {
		// Check aws-chuncked.
		var r2, ok = body.(*bufio.Reader)
		if !ok {
			r2 = bufio.NewReader(body)
		}
		var length, _, sig, _ = lookat_chunk_header(r2)
		if length == 0 || sig == "" {
			// Transfer-encoding is chunked but without a chunk header.
			return r2, CHUNKED_AWS, io.ErrUnexpectedEOF
		} else {
			// Transfer-encoding for aws-chunked.
			var r3 = &chunked_reader{r: r2, chunked: CHUNKED_AWS}
			return r3, CHUNKED_AWS, nil
		}
	} else if check_http1_chunked(q) {
		// Check http1-chuncked.
		var r2, ok = body.(*bufio.Reader)
		if !ok {
			r2 = bufio.NewReader(body)
		}
		var length, _, _, _ = lookat_chunk_header(r2)
		if length == 0 {
			// Transfer-Encoding chunked but without a chunk header.
			return r2, NOT_CHUNKED, nil
		} else {
			// Transfer-Encoding http1-chunked.
			return httputil.NewChunkedReader(r2), CHUNKED_HTTP1, nil
		}
	} else {
		if length != -1 {
			return &io.LimitedReader{R: body, N: length}, NOT_CHUNKED, nil
		} else {
			return body, NOT_CHUNKED, nil
		}
	}
}

func check_http1_chunked(q *http.Request) bool {
	var enc = q.TransferEncoding
	return len(enc) == 1 && strings.EqualFold(enc[0], "chunked")
}

func check_aws_chunked(q *http.Request) bool {
	//   Content-Encoding: aws-chunked
	//   X-Amz-Decoded-Content-Length: nnnn
	//   X-Amz-Content-Sha256: STREAMING-AWS4-HMAC-SHA256-PAYLOAD
	//   Content-Length: nnnn
	//var enc = q.TransferEncoding
	return true
}

// (io.Reader interface).
func (r *chunked_reader) Read(b []byte) (n int, err error) {
	if r.chunked == NOT_CHUNKED {
		return r.r.Read(b)
	}
	if r.chunk_length == r.chunk_n {
		r.read_chunk_header()
	}
	r.read_chunk_body(b)
	return 0, nil
}

func (r *chunked_reader) read_chunk_header() error {
	return nil
}

func (r *chunked_reader) read_chunk_body(b []byte) (int, error) {
	var toread = r.content_length - r.content_n
	var bx []byte
	if int64(len(b)) > toread {
		bx = b[:toread]
	} else {
		bx = b
	}
	var n, err1 = r.r.Read(bx)
	r.content_n += int64(n)
	return n, err1
}

// LOOKAT_CHUNK_HEADER checks the chunked header.  It does not consume
// the stream.  It returns the header length or zero when it finds no
// chunk header.  The returned length includes "cr+lf".  A non-empty
// signature string means aws-chucked, otherwise it is http1-chucked.
// Note r.Peek() returns a slice from a temporary buffer and its
// lifetime is short.
func lookat_chunk_header(r *bufio.Reader) (int, int64, string, error) {
	const chunk_signature_key = "chunk-signature="
	var b1, err1 = r.Peek(chunk_header_peek)
	if err1 != nil {
		if err1 == bufio.ErrBufferFull {
			// It is usual.
			// IGNORE-ERRORS.
		} else {
			return 0, 0, "", err1
		}
	}
	var x1 = bytes.IndexAny(b1, "\n")
	if x1 == -1 {
		return 0, 0, "", io.ErrUnexpectedEOF
	} else if x1 <= 1 {
		return 0, 0, "", io.ErrUnexpectedEOF
	} else if b1[x1-1] != '\r' {
		// No "cr+lf" part.
		return 0, 0, "", io.ErrUnexpectedEOF
	}
	// Check up to excluding "cr+lf".
	var length = x1
	var b2 = b1[:x1-1]
	var x2 = bytes.IndexAny(b2, ";")
	if x2 != -1 {
		// Check for aws-chunked header.
		if !bytes.HasPrefix(b2[(x2+1):], []byte(chunk_signature_key)) {
			// Not contain "chunk-signature=".
			return 0, 0, "", io.ErrUnexpectedEOF
		}
		var size, err2 = strconv.ParseInt(string(b2[:x2]), 16, 64)
		if err2 == nil {
			return 0, 0, "", err2
		}
		var sig = string(b2[x2+1+len(chunk_signature_key):])
		if len(sig) == 0 {
			return 0, 0, "", io.ErrUnexpectedEOF
		}
		return length, size, sig, nil
	} else {
		// Check for http1-chunked header.
		var size, err2 = strconv.ParseInt(string(b2), 16, 64)
		if err2 == nil {
			return 0, 0, "", err2
		}
		return length, size, "", nil
	}
}
