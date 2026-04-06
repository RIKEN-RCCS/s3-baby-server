// access-logging.go

// Copyright 2022-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Apache HTTPD like logging.  It formats a common access-log entry.
// It also provides "ResponseWriter2" which may be used to replace
// Golang's http.ResponseWriter to record status-code and
// content-length.

// MEMO: Apache httpd access log format:
//
//   LogFormat %h %l %u %t "%r" %>s %b "%{Referer}i" "%{User-Agent}i" combined
//
// https://en.wikipedia.org/wiki/Common_Log_Format
//
// EXAMPLE:
//   192.168.2.2 - - [02/Jan/2006:15:04:05 -0700] "GET /... HTTP/1.1"
//   200 333 "-" "aws-cli/1.18.156 Python/3.6.8
//   Linux/4.18.0-513.18.1.el8_9.x86_64 botocore/1.18.15"

// MEMO: Golang's ResponseWriter typically is an instance of
// "response" defined in "net/http/server.go".  It implements Flusher,
// Hijacker and methods:
//
//   - Flush()
//   - FlushError() error // alternative Flush returning an error
//   - Hijack() (net.Conn, *bufio.ReadWriter, error)
//   - SetReadDeadline(deadline time.Time) error
//   - SetWriteDeadline(deadline time.Time) error
//   - EnableFullDuplex() error

package httpaide

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const common_log_time_layout = "02/Jan/2006:15:04:05 -0700"

// ResponseWriter2 is ResponseWriter but records status-code and
// content-length.  It is suggested in Stackoverflow:
// https://stackoverflow.com/questions/66528234/.  Content-length of
// the request side is stored in http.Request.  MEMO: The type
// "response", a true type of ResponseWriter (defined in
// "net/http/server.go"), contains "status" and "written" fields but
// they are not visible.
type ResponseWriter2 struct {
	http.ResponseWriter
	Status_code    int
	Content_length int64
}

func (w *ResponseWriter2) WriteHeader(code int) {
	w.Status_code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *ResponseWriter2) Write(s []byte) (int, error) {
	w.Content_length += int64(len(s))
	return w.ResponseWriter.Write(s)
}

// LOG_ACCESS formats an access log entry.  It generates a line
// without a newline.  It takes the client-host from the header
// "X-Forwarded-For", or r.RemoteAddr if the header is not set.  USER
// will be an access-key in S3-Baby-server.
func Log_access(request *http.Request, code int, length int64, user string) string {
	var uid string
	if user != "" {
		uid = user
	} else {
		uid = "-"
	}

	// l: RFC 1413 client identity by identd
	// u: user
	// rf: Referer

	// Choose the first entry of "X-Forwarded-For".

	var h1 = request.Header.Get("X-Forwarded-For")
	var h = strings.Split(h1, ",")[0]
	if h == "" {
		h = request.RemoteAddr
	}

	var l = "-"
	var u = uid
	var t = time.Now().Format(common_log_time_layout)
	var r = fmt.Sprintf("%s %s %s", request.Method, request.URL, request.Proto)
	var s = fmt.Sprintf("%d", code)
	var b = fmt.Sprintf("%d", length)
	var rf = "-"
	var ua = request.Header.Get("User-Agent")

	var msg1 string
	msg1 = fmt.Sprintf(
		("%s %s %s [%s] %q" + " %s %s %q %q"),
		h, l, u, t, r,
		s, b, rf, ua)
	return msg1
}
