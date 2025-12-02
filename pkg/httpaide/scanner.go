// scanner.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Http header scanners for ETag condtionals and byte-ranges.

// ETAG (IF-MATCH)
// https://www.rfc-editor.org/rfc/rfc7232#appendix-C

// RANGE
// https://www.rfc-editor.org/rfc/rfc9110.html#name-range

// HTTP-date
// https://www.rfc-editor.org/rfc/rfc7231#section-7.1.1.1

// MEMO: Http servers in Golang's stdlib (such as
// "net/http.ServeContent") has parsers for http headers including
// "if-match" or "if-none-match" elements.  But they are not public.

package httpaide

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var rfc9110_range_unit_re = regexp.MustCompile(`^bytes *=`)
var rfc9110_range_spec_re = regexp.MustCompile(`^([0-9]+)? *- *([0-9]+)?$`)

// SCAN_RFC9110_RANGES scans a string of ranges in RFC-9110.  A
// missing upper bound is -1.  Ranges are defined as "1#range-spec".
// Note that '#' in BNF means a comma-separated list.  It only parses
// simple ones like "_bytes_=_0_-_,_4500_-_5499_,_-_1000_".
func Scan_rfc9110_ranges(s string) ([][2]int64, error) {
	var s1 = strings.TrimSpace(s)
	var m1 = rfc9110_range_unit_re.FindStringSubmatch(s1)
	if len(m1) != 1 {
		var err = fmt.Errorf("bad rfc9110 ranges without a unit in %s",
			strconv.Quote(s))
		return [][2]int64{}, err
	}
	var s2 = s1[len(m1[0]):]
	var v [][2]int64
	var list = strings.Split(s2, ",")
	for i, r1 := range list {
		var r2 = strings.TrimSpace(r1)
		var m2 = rfc9110_range_spec_re.FindStringSubmatch(r2)
		if len(m2) != 3 {
			//fmt.Printf("r2=%v m2=%#v\n", r2, m2)
			var err = fmt.Errorf("bad rfc9110 ranges at %d-th in %s", i,
				strconv.Quote(s))
			return [][2]int64{}, err
		}
		var b int64
		if m2[1] == "" {
			b = 0
		} else {
			var n, err2 = strconv.ParseInt(m2[1], 10, 64)
			if err2 != nil {
				var err = fmt.Errorf("bad rfc9110 ranges at %d-th in %s", i,
					strconv.Quote(s))
				return [][2]int64{}, err
			}
			b = n
		}
		var e int64
		if m2[2] == "" {
			e = -1
		} else {
			var n, err3 = strconv.ParseInt(m2[2], 10, 64)
			if err3 != nil {
				var err = fmt.Errorf("bad rfc9110 ranges at %d-th in %s", i,
					strconv.Quote(s))
				return [][2]int64{}, err
			}
			e = n
		}
		v = append(v, [2]int64{b, e})
	}
	return v, nil
}

// Match condition is either "*" or a list of ETags.  ETag can contain
// characters except: double-quote, space, delete, and control
// characters.
//
//  - If-Match = "*" / ( *( "," WS ) entity-tag *( WS "," [ WS entity-tag ] ) )
//  - If-None-Match = If-Match
//  - entity-tag = [ "W/" ] DQUOTE *etagc DQUOTE
//  - etagc = "!" / "#"-"~" / %x80-FF
//  - WS = <optional whitespace>

var rfc7273_etag_re = regexp.MustCompile(`^(W/)?"[!#-~\x80-\xff]*"$`)

// SCAN_RFC7232_ETAGS scans conditionals of "if-match" and
// "if-none-match" headers.  It is looser than the defition as it
// calls trim-spaces.
func Scan_rfc7232_etags(s string) ([]string, error) {
	var s1 = strings.TrimSpace(s)
	if s1 == "*" {
		return []string{"*"}, nil
	}
	var etags []string
	var etaglist = strings.Split(s1, ",")
	for i, e := range etaglist {
		var t1 = strings.TrimSpace(e)
		if t1 == "" {
			continue
		}
		var m1 = rfc9110_range_spec_re.FindStringSubmatch(t1)
		if len(m1) == 0 {
			//fmt.Printf("t1=%v m1=%#v\n", t1, m1)
			var err = fmt.Errorf("bad rfc7232 etag at %d-th in %s", i,
				strconv.Quote(s))
			return nil, err
		}
		etags = append(etags, m1[0])
	}
	return etags, nil
}

// MEMO:
//   - time.RFC1123="Mon,_02_Jan_2006_15:04:05_MST"
//   - net/http.TimeFormat="Mon,_02_Jan_2006_15:04:05_GMT"

// SCAN_RFC5322_DATE scans a http-date of "if-modified-since" and
// "if-unmodified-since" headers.  IMF-fixdate is like
// "Sun,_06_Nov_1994_08:49:37_GMT".
func Scan_rfc5322_date(s string) (time.Time, error) {
	var t1, err1 = time.Parse(http.TimeFormat, s)
	return t1, err1
}
