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
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
)

// CHUNKED_READER is an io.Reader.  It wraps an io.Reader by
// bufio.Reader as it peeks the stream to check the existence of a
// chunk-header.  CONTENT_LENGTH is the total size including chunk
// headers.  PAYLOAD_LENGTH is the size of data.  CHUNKS holds the
// signatures read so far.  CHUNK_LENGTH=0 means hits an EOF.
type chunked_reader struct {
	r              *bufio.Reader
	chunked        chunked_type
	hunting_header bool
	content_length int64
	payload_length int64
	content_n      int64
	payload_n      int64
	chunk_length   int64
	chunk_n        int64
	chunks         []chunk_record
}

type chunked_type int

const (
	NOT_CHUNKED chunked_type = iota
	CHUNKED_HTTP1
	CHUNKED_AWSS3
)

type chunk_record struct {
	chunk_offset int64
	chunk_length int64
	signature    string
}

// A chunk-header consists of length + ";" + "chunk-signature=" +
// signature + "cr+lf" + "cr+lf".  A signature is (assmued) a SHA256
// in hex string (length=64).
const chunk_header_peek int = 160

const content_sha256_key = "STREAMING-AWS4-HMAC-SHA256-PAYLOAD"

// MAKE_CHUNKED_READER returns a modified io.Reader of the body.
// Should use the returned stream for later operation.  It is because
// it may consume the underlying stream in order to check the
// chunked-ness.  This check is partly needed because http.Server's
// document says it handles an http1-chunked stream (but might not).
// (A returned error is sometimes just a marker and meaningless).
func make_chunked_reader__(q *http.Request, body io.Reader) (io.Reader, chunked_type, error) {
	if check_aws_chunked(q) {
		// Check aws-chuncked.
		var h = q.Header
		if !(h.Get("X-Amz-Decoded-Content-Length") != "" &&
			q.ContentLength != -1) {
			return body, CHUNKED_AWSS3, io.ErrUnexpectedEOF
		}
		var r2, ok = body.(*bufio.Reader)
		if !ok {
			r2 = bufio.NewReader(body)
		}
		var length, _, sig, _ = lookat_chunk_header(r2)
		if length == 0 || sig == "" {
			// Transfer-encoding is chunked but without a chunk header.
			return r2, CHUNKED_AWSS3, io.ErrUnexpectedEOF
		} else {
			// Transfer-encoding for aws-chunked.
			var len1 = q.ContentLength
			var s1 = h.Get("X-Amz-Decoded-Content-Length")
			var len2, err1 = strconv.ParseInt(s1, 10, 64)
			if err1 != nil {
				return r2, CHUNKED_AWSS3, err1
			}
			var r3 = &chunked_reader{
				r:              r2,
				chunked:        CHUNKED_AWSS3,
				hunting_header: true,
				content_length: len1,
				payload_length: len2,
				//chunks: make([]chunk_record),
			}
			return r3, CHUNKED_AWSS3, nil
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
			return r2, CHUNKED_HTTP1, io.ErrUnexpectedEOF
		} else {
			// Transfer-Encoding http1-chunked.
			return httputil.NewChunkedReader(r2), CHUNKED_HTTP1, nil
		}
	} else {
		return body, NOT_CHUNKED, nil
	}
}

// READ reads the body (io.Reader interface).
func (r *chunked_reader) Read(b []byte) (int, error) {
	if r.chunked == NOT_CHUNKED {
		return r.r.Read(b)
	}
	if r.chunked == CHUNKED_HTTP1 {
		return r.r.Read(b)
	}
	if r.hunting_header {
		var err1 = r.read_chunk_header()
		if err1 != nil {
			return 0, err1
		}
	}
	if r.chunk_length == 0 {
		// End of stream.
		return 0, io.EOF
	}
	var n, err2 = r.read_chunk_body(b)
	return n, err2
}

func (r *chunked_reader) read_chunk_header() error {
	var length, size, sig, err1 = lookat_chunk_header(r.r)
	if err1 != nil {
		return err1
	}
	// Consume the bytes of a chunk header.  It was peeked and a
	// single read can consume all.
	var b = make([]byte, length)
	var n, err2 = r.r.Read(b)
	if err2 != nil {
		return err2
	}
	if n != length {
		return io.ErrUnexpectedEOF
	}
	r.chunks = append(r.chunks, chunk_record{
		chunk_offset: r.payload_n,
		chunk_length: size,
		signature:    sig,
	})
	r.content_n += int64(length)
	r.chunk_length = size
	r.chunk_n = 0
	r.hunting_header = false
	// CHUNK_LENGTH=0 means got an end of stream.
	if size == 0 {
		if !(r.content_length == r.content_n &&
			r.payload_length == r.payload_n) {
			Printf("chunked_reader EOF badly; content_length=%v content_n=%v content_length=%v content_n=%v\n",
				r.content_length, r.content_n,
				r.payload_length, r.payload_n)
		}
	}
	return nil
}

func (r *chunked_reader) read_chunk_body(b []byte) (int, error) {
	var n1 = min(int64(len(b)), (r.chunk_length - r.chunk_n))
	var n2, err1 = r.r.Read(b[:n1])
	r.chunk_n += int64(n2)
	bb_assert(r.chunk_n <= r.chunk_length)
	if r.chunk_length == r.chunk_n {
		r.hunting_header = true
		r.content_n += r.chunk_length
		r.payload_n += r.chunk_length
	}
	return n2, err1
}

func check_http1_chunked(q *http.Request) bool {
	var enc = q.TransferEncoding
	return len(enc) == 1 && strings.EqualFold(enc[0], "chunked")
}

// CHECK_AWS_CHUNKED check the request is chunked.  The condition is
// NOT a conjunction BUT a disjunction of the header entries
// "Content-Encoding" or "X-Amz-Content-Sha256".
func check_aws_chunked(q *http.Request) bool {
	var h = q.Header
	return (h.Get("Content-Encoding") == "aws-chunked" ||
		h.Get("X-Amz-Content-Sha256") == content_sha256_key)
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
		if err2 != nil {
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
