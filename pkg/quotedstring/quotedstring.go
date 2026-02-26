// quotedstring.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// This is to parse an slog output.

// time=2026-02-16T07:01:25.800Z level=INFO msg="Handling time"
// rid=1771225285799519 request="GET /mybucket1/?object-lock="
// request-length=0 code=200 response-length=155 elapse=611.983µs

package quotedstring

import (
	"io"
	"strconv"
	"strings"
	"unicode"
)

// SLOG_PARSE returns "key=value" pairs of an slog message.  Keys and
// values of slog are escaped by strconv.Quote().
func Slog_parse(s string) ([][2]string, error) {
	var acc [][2]string
	var i = 0
	for i < len(s) {
		var k, j1, err1 = Scan_quoted(s, i)
		if err1 != nil {
			return acc, err1
		}
		if s[j1] != '=' {
			return acc, io.EOF
		}
		var v, j2, err2 = Scan_quoted(s, j1+1)
		if err2 != nil {
			return acc, err2
		}
		acc = append(acc, [2]string{k, v})
		var c = scan_spaces(s[j2:])
		if !((j2 + c) > i) {
			panic("(j2 + c) > i")
		}
		i = j2 + c
	}
	return acc, nil
}

func scan_spaces(s string) int {
	for i, r := range s {
		if r != ' ' {
			return i
		}
	}
	return len(s)
}

// SCAN_QUOTED scans a maybe-quoted string, token or "token".  It
// returns a token and its end position.  It accepts the empty string
// as a token.  For an unquoted token, it scans for a terminator
// {white-space, non-printable, '"', '='}.  Occurrence of some
// characters are error {'\'}.  For a quoted token, it simply scans
// for an closing quote, while skipping backslash-quote pairs.  A
// returned token is unquoted by strconv.Unquote().  There are similar
// functions in stdlib, but they do not meet the purpose: go/token,
// go/scanner, text/scanner, strconv.Unquote.
func Scan_quoted(s string, a0 int) (string, int, error) {
	if len(s) <= a0 {
		return "", a0, nil
	}
	var quoted bool
	var a1 int
	if s[a0] == '"' {
		quoted = true
		a1 = a0 + 1
	} else {
		quoted = false
		a1 = a0
	}
	if len(s) <= a1 {
		return "", a1, io.EOF
	}
	var backslash = false
	var i int
	var r rune
	for i, r = range s[a1:] {
		if quoted {
			if backslash == true {
				backslash = false
				// Add any character.
			} else if r == '\\' {
				backslash = true
				// Add a backslash character.
			} else if r == '"' {
				var token = s[a0 : a1+i+1]
				var v, err1 = strconv.Unquote(token)
				return v, (a1 + i + 1), err1
			} else {
				// Add one character.
			}
		} else {
			if unicode.IsSpace(r) || !unicode.IsPrint(r) ||
				strings.ContainsRune(`"=`, r) {
				return s[a1 : a1+i], (a1 + i), nil
			}
		}
	}
	if quoted == true {
		// Quote unclosed.
		return s, (a1 + i + 1), io.EOF
	} else if backslash == true {
		// Backslash at the tail.  (This case never happens).
		return s, (a1 + i + 1), io.EOF
	} else {
		return s[a1 : a1+i+1], (a1 + i + 1), nil
	}
}
