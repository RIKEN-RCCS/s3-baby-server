// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Chunked Reader

// This implements a stream for Content-Encoding: "aws-chunked".  THIS
// CHUNKED_READER DOES NOT CALCULATE SIGNATURES.
//
// It assumes Golang's http server does not handle
// Transfer-Encoding=chunked.  Since AWS-S3's chunked-encoding seems
// not to nest with http's chunked-transfer, handling by Golang may
// corrupt the body stream.
//
// This implements "X-Amz-Content-Sha256" keyword values:
//  - UNSIGNED-PAYLOAD
//  - STREAMING-AWS4-HMAC-SHA256-PAYLOAD
//  - STREAMING-UNSIGNED-PAYLOAD-TRAILER

// "aws-chunked" is described in AWS-S3 signature v4 documents.  See
// https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-streaming.html

// NOTE: This chunked-reader checks for aws-chunked by looking at
// headers "X-Amz-Content-Sha256" and "X-Amz-Decoded-Content-Length".
// "Content-Encoding" seems unreliable as it may be missing.  (Note
// MinIO MC client lacks a Content-Encoding header).
//
// Headers for "aws-chunked" look like the following
//
//   Content-Encoding: aws-chunked
//   X-Amz-Decoded-Content-Length: nnnn
//   X-Amz-Content-Sha256: STREAMING-AWS4-HMAC-SHA256-PAYLOAD
//   Content-Length: nnnn
//
// Each chunk has a header line which looks like:
//
// 513;chunk-signature=be348984ab8e284170b29e2bb5a424370e4203c81c981d5eb5e8534912ac1c5d

// MEMO: string to sign = previous-signature + hash("") +
// hash(current-chunk-data)

package server

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
)

// CHUNKED_READER is an io.Reader.  It wraps an io.Reader by
// bufio.Reader as it peeks the stream to check the existence of a
// chunk-header.  CONTENT_LENGTH is the total size including chunk
// headers.  It can be -1.  PAYLOAD_LENGTH is the size of data.
// CHUNKS holds the signatures read so far.  CHUNK_LENGTH=0 means hits
// an EOF.
type Chunked_reader struct {
	r                      *bufio.Reader
	chunked                Chunked_type
	trailer                bool
	hunting_header         bool
	content_length         int64
	content_n              int64
	payload_length         int64
	payload_n              int64
	chunk_length           int64
	chunk_n                int64
	rid                    uint64
	logger                 *slog.Logger
	forbid_last_chunk_crlf bool
	chunks                 []chunk_record
}

type Chunked_type int

const (
	Chunked_NO Chunked_type = iota
	Chunked_HTTP1
	Chunked_AWSS3
)

// A CHUNK_RECORD is record of chunks stored in a chuck_reader.  A
// CHUNK_OFFSET is an offset in payload data (not in a raw stream).
type chunk_record struct {
	chunk_offset int64
	chunk_length int64
	signature    string
}

// Keys (some from defined) stored in "x-amz-content-sha256" besides
// an actual hash value.  The value "actual-value" is a dummy.
const (
	sha256_actual_value         string = "actual-value"
	sha256_key_unsigned         string = "UNSIGNED-PAYLOAD"
	sha256_key_unsigned_trailer string = "STREAMING-UNSIGNED-PAYLOAD-TRAILER"
	sha256_key_hmac             string = "STREAMING-AWS4-HMAC-SHA256-PAYLOAD"
	sha256_key_hmac_trailer     string = "STREAMING-AWS4-HMAC-SHA256-PAYLOAD-TRAILER"
)

// A chunk-header consists of length + ";" + "chunk-signature=" +
// signature + "cr+lf" + "cr+lf".  A signature is (assmued) a SHA256
// in hex string (length=64).
const chunk_header_peek int = 160

type Chunked_reader_error struct {
	Msg    string
	Reader *Chunked_reader
	Nested error
}

func (e *Chunked_reader_error) Error() string {
	return fmt.Sprintf("%s; error=%v", e.Msg, e.Nested)
}

func (e *Chunked_reader_error) Unwrap() error {
	return e.Nested
}

// Error description.  They are stored as a message string in errors.
const (
	unknown_sha256_key             = "Unknown x-amz-content-sha256 key"
	inconsitent_headers            = "Inconsitent headers"
	missing_decoded_content_length = "Missing decoded-content-length in chunk"
	missing_cr_lf                  = "Missing cr+lf in chunk"
	bad_chunk_header               = "Bad chunk header"
	bad_decoded_content_length     = "Bad decoded-content-length in chunk"
	peek_error_in_chunk_header     = "Peek error in chunk header"
	read_error_in_chunk_header     = "Read error in chunk header"
	read_error_in_chunk_body       = "Read error in chunk body"
	read_truncated_in_chunk_header = "Read truncated in chunk header"
)

// NEW_CHUNKED_READER returns a modified io.Reader of the body.  It
// also returns the payload length or -1 (not content-length).  User
// should use the returned stream in later operation even when it the
// stream is not chucked, because it may consume the underlying stream
// to verify the chunked-ness.  Note it is possible both http1-chunked
// and aws-chunked simultaneously.
func New_chunked_reader(w http.ResponseWriter, q *http.Request, body io.Reader, rid uint64, forbid_last_chunk_crlf bool, logger *slog.Logger) (io.Reader, Chunked_type, int64, error) {
	var chunked, trailer, _, err1 = check_aws_chunked(q)
	if err1 != nil {
		return body, Chunked_NO, -1, err1
	}
	switch chunked {
	case Chunked_NO:
		fallthrough
	default:
		return body, Chunked_NO, -1, nil

	case Chunked_HTTP1:
		// Make http1-chuncked.
		// Transfer-Encoding http1-chunked.

		var body2 = make_io_reader_bufio_reader(body)
		var length, _, _, err1 = lookat_chunk_header(nil, body2)
		if length == 0 {
			// Transfer-Encoding chunked but without a chunk header.
			return body2, chunked, -1, err1
		}
		var body3 = httputil.NewChunkedReader(body2)
		return body3, chunked, -1, nil

	case Chunked_AWSS3:
		// Make aws-chuncked.
		// Content-encoding aws-chunked.

		// Note q.ContentLength can be -1.

		var body2 = make_io_reader_bufio_reader(body)
		var h = q.Header
		var decoded_content_length = h.Get("X-Amz-Decoded-Content-Length")
		if decoded_content_length == "" {
			return body2, chunked, -1, &Chunked_reader_error{
				Msg: missing_decoded_content_length, Reader: nil, Nested: nil}
		}
		var length, _, sig, err1 = lookat_chunk_header(nil, body2)
		if length == 0 || sig == "" {
			// Transfer-encoding is chunked but without a chunk header.
			return body2, chunked, -1, err1
		}
		var len1 = q.ContentLength
		var payload, err2 = strconv.ParseInt(decoded_content_length, 10, 64)
		if err2 != nil {
			return body2, chunked, -1, &Chunked_reader_error{
				Msg: bad_decoded_content_length, Reader: nil, Nested: err2}
		}
		var body3 = &Chunked_reader{
			r:                      body2,
			chunked:                Chunked_AWSS3,
			trailer:                trailer,
			hunting_header:         true,
			content_length:         len1,
			payload_length:         payload,
			rid:                    rid,
			logger:                 logger,
			forbid_last_chunk_crlf: forbid_last_chunk_crlf,
			//chunks: make([]chunk_record),
		}
		return body3, chunked, payload, nil
	}
}

func make_io_reader_bufio_reader(r1 io.Reader) *bufio.Reader {
	var r2, ok = r1.(*bufio.Reader)
	if ok {
		return r2
	} else {
		return bufio.NewReader(r1)
	}
}

// RESPOND_CONTINUE_WHEN_EXPECTED sends 100-Continue.  We are not sure
// but Golang's http server fails to send it when reading the body by
// bufio.Reader.Peek().  (Why?)
func respond_continue_when_expected(w http.ResponseWriter, q *http.Request) {
	if q.Header.Get("Expect") == "100-continue" {
		w.WriteHeader(http.StatusContinue)
	}
}

// READ reads the body (io.Reader interface).
func (r *Chunked_reader) Read(b []byte) (int, error) {
	if r.chunked == Chunked_NO {
		return r.r.Read(b)
	}
	if r.chunked == Chunked_HTTP1 {
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

func (r *Chunked_reader) read_chunk_header() error {
	var length, size, sig, err1 = lookat_chunk_header(r, r.r)
	if err1 != nil {
		return err1
	}
	// Consume the bytes of a chunk header.  It was peeked and a
	// single read can consume all.
	var drain = make([]byte, length)
	var n, err2 = r.r.Read(drain)
	if err2 != nil {
		return &Chunked_reader_error{
			Msg: read_error_in_chunk_header, Reader: r, Nested: err2}
	}
	if n != length {
		return &Chunked_reader_error{
			Msg: read_truncated_in_chunk_header, Reader: r, Nested: nil}
	}
	if string(drain[max(0, len(drain)-2):]) != "\r\n" {
		return &Chunked_reader_error{
			Msg: missing_cr_lf, Reader: r, Nested: nil}
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
		if r.content_length != -1 {
			if r.content_length == r.content_n+2 {
				if !r.forbid_last_chunk_crlf {
					// It allows extra cr+lf at the end.
					var err2 = r.consume_cr_lf()
					if err2 != nil {
						r.logger.Warn("Chunk-reader hits EOF badly",
							"rid", r.rid, "error", err2)
					}
					r.content_n += 2
				}
			}
		}

		if !((r.content_length == -1 || r.content_length == r.content_n) &&
			r.payload_length == r.payload_n) {
			r.logger.Warn("Chunk-reader hits EOF badly",
				"rid", r.rid,
				"content_length", r.content_length,
				"content_n", r.content_n,
				"payload_length", r.payload_length,
				"payload_n", r.payload_n)
		}
	}
	return nil
}

func (r *Chunked_reader) read_chunk_body(b []byte) (int, error) {
	var n1 = min(int64(len(b)), (r.chunk_length - r.chunk_n))
	var n2, err1 = r.r.Read(b[:n1])
	if err1 != nil {
		return n2, &Chunked_reader_error{
			Msg: read_error_in_chunk_body, Reader: r, Nested: err1}
	}
	r.chunk_n += int64(n2)
	bb_assert(r.chunk_n <= r.chunk_length)
	if r.chunk_length == r.chunk_n {
		var err2 = r.consume_cr_lf()
		if err2 != nil {
			return 0, err2
		}
		r.hunting_header = true
		r.content_n += (r.chunk_length + 2)
		r.payload_n += r.chunk_length
	}
	return n2, nil
}

// CONSUME_CR_LF consumes cr+lf.
func (r *Chunked_reader) consume_cr_lf() error {
	var drain = make([]byte, 2)
	var n, err3 = r.r.Read(drain)
	if err3 != nil {
		return &Chunked_reader_error{
			Msg: missing_cr_lf, Reader: r, Nested: err3}
	}
	if n != 2 || string(drain) != "\r\n" {
		// No "cr+lf" part.
		return &Chunked_reader_error{
			Msg: missing_cr_lf, Reader: r, Nested: nil}
	}
	return nil
}

func check_http1_chunked(q *http.Request) bool {
	var enc = q.TransferEncoding
	return len(enc) == 1 && strings.EqualFold(enc[0], "chunked")
}

// CHECK_AWS_CHUNKED checks the request is chunked.  It returns a
// chunked-type, a trailer existence, and a sha256-key.  It ignores
// the transfer-encoding when aws-chunked is used.  It errs on some of
// inconsistencies in "Content-Encoding" and "X-Amz-Content-Sha256".
func check_aws_chunked(q *http.Request) (Chunked_type, bool, string, error) {
	var http1_chunked = check_http1_chunked(q)
	var h = q.Header
	var aws_chunked = (h.Get("Content-Encoding") == "aws-chunked")
	var sha = h.Get("X-Amz-Content-Sha256")
	var _, err1 = hex.DecodeString(sha)
	var actualvalue = (err1 == nil)
	if actualvalue {
		if aws_chunked {
			var errx = fmt.Errorf("aws-chunked with content-sha256=%s", sha)
			return Chunked_NO, false, "", &Chunked_reader_error{
				Msg: inconsitent_headers, Reader: nil, Nested: errx}
		} else if http1_chunked {
			return Chunked_HTTP1, false, sha256_actual_value, nil
		} else {
			return Chunked_NO, false, sha256_actual_value, nil
		}
	} else {
		var errx1 = fmt.Errorf("aws-chunked with content-sha256=%s", sha)
		var errx2 = fmt.Errorf("Not aws-chunked with content-sha256=%s", sha)
		switch sha {
		case sha256_key_unsigned:
			if aws_chunked {
				return Chunked_NO, false, "", &Chunked_reader_error{
					Msg: inconsitent_headers, Reader: nil, Nested: errx1}
			}
			return Chunked_NO, false, sha256_key_unsigned, nil
		case sha256_key_unsigned_trailer:
			if !aws_chunked {
				return Chunked_NO, false, "", &Chunked_reader_error{
					Msg: inconsitent_headers, Reader: nil, Nested: errx2}
			}
			return Chunked_HTTP1, true, sha256_key_unsigned_trailer, nil
		case sha256_key_hmac:
			if !aws_chunked {
				return Chunked_NO, false, "", &Chunked_reader_error{
					Msg: inconsitent_headers, Reader: nil, Nested: errx2}
			}
			return Chunked_AWSS3, false, sha256_key_hmac, nil
		case sha256_key_hmac_trailer:
			if !aws_chunked {
				return Chunked_NO, false, "", &Chunked_reader_error{
					Msg: inconsitent_headers, Reader: nil, Nested: errx2}
			}
			return Chunked_AWSS3, true, sha256_key_hmac, nil
		default:
			var errx = fmt.Errorf("content-sha256=%s", sha)
			return Chunked_NO, false, "", &Chunked_reader_error{
				Msg: unknown_sha256_key, Reader: nil, Nested: errx}
		}
	}
}

// LOOKAT_CHUNK_HEADER checks the chunked header.  The READER argument
// is used as error information, as it can be used before marking a
// chunked_reader.  It does not consume the stream.  It returns a
// header length, a chunk size, a signatue, and an error.  It returns
// zero as a header length when it does not find a chunk header.  The
// returned length includes "cr+lf".  A non-empty signature string
// means aws-chucked, otherwise it is http1-chucked.  A returned error
// is either io.ErrUnexpectedEOF or one returned from
// bufio.Reader.Peek().  Note r.Peek() returns a slice from a
// temporary buffer and its lifetime is short.
func lookat_chunk_header(reader *Chunked_reader, r *bufio.Reader) (int, int64, string, error) {
	const chunk_signature_key = "chunk-signature="
	var b1, err1 = r.Peek(chunk_header_peek)
	if err1 != nil {
		if err1 == bufio.ErrBufferFull {
			log.Fatalf("BAD-IMPL: bufio.Reader error=%#v", err1)
		} else if err1 == io.ErrUnexpectedEOF || err1 == io.EOF {
			// It is usual -- Short peek of data.
			// IGNORE-ERRORS.
		} else {
			return 0, 0, "", &Chunked_reader_error{
				Msg: peek_error_in_chunk_header, Reader: reader, Nested: err1}
		}
	}

	fmt.Printf("AHO chuck peek; len=%d peek=%q\n", len(b1), string(b1))

	var x1 = bytes.IndexAny(b1, "\n")
	if x1 == -1 || x1 <= 1 {
		var peeks = string(b1)
		var errx = fmt.Errorf("Bad header; header=%s", peeks)
		return 0, 0, "", &Chunked_reader_error{
			Msg: bad_chunk_header, Reader: reader, Nested: errx}
	} else if b1[x1-1] != '\r' {
		// No "cr+lf".
		var peeks = string(b1)
		var errx = fmt.Errorf("Bad header; header=%s", peeks)
		return 0, 0, "", &Chunked_reader_error{
			Msg: bad_chunk_header, Reader: reader, Nested: errx}
	}
	var length = (x1 + 1)
	// Check up to excluding "cr+lf".
	var b2 = b1[:x1-1]
	var x2 = bytes.IndexAny(b2, ";")
	if x2 == -1 {
		// Check for http1-chunked header.
		var size, err2 = strconv.ParseInt(string(b2), 16, 64)
		if err2 != nil {
			return 0, 0, "", &Chunked_reader_error{
				Msg: bad_chunk_header, Reader: reader, Nested: err2}
		}
		return length, size, "", nil
	} else {
		// Check for aws-chunked header.
		if !bytes.HasPrefix(b2[(x2+1):], []byte(chunk_signature_key)) {
			// Not contain "chunk-signature=".
			var peeks = string(b2)
			var errx = fmt.Errorf("No chunk-signature; header=%s", peeks)
			return 0, 0, "", &Chunked_reader_error{
				Msg: bad_chunk_header, Reader: nil, Nested: errx}
		}
		var sig = string(b2[x2+1+len(chunk_signature_key):])
		if len(sig) == 0 {
			var peeks = string(b2)
			var errx = fmt.Errorf("Empty chunk-signature; header=%s", peeks)
			return 0, 0, "", &Chunked_reader_error{
				Msg: bad_chunk_header, Reader: reader, Nested: errx}
		}
		var size, err2 = strconv.ParseInt(string(b2[:x2]), 16, 64)
		if err2 != nil {
			return 0, 0, "", &Chunked_reader_error{
				Msg: bad_chunk_header, Reader: reader, Nested: err2}
		}
		return length, size, sig, nil
	}
}
