// scanner.go

// Copyright 2025-2025 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Http header scanners.

// ETAG (IF-MATCH)
// https://www.rfc-editor.org/rfc/rfc7232#appendix-C

// RANGE
// https://www.rfc-editor.org/rfc/rfc9110.html#name-range

// MEMO: Http servers in Golang's stdlib (such as
// "net/http.ServeContent") has parsers for http headers.  But they
// are not public.  Parsing "if-match" or "if-none-match" used for
// Etags are rather non-trivial.

package httpaide

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
