// quotedstring_test.go

package quotedstring

import (
	"fmt"
	"testing"
)

func TestSlog(t *testing.T) {
	fmt.Printf("Test Slog_parse...\n")

	var s string
	var v [][2]string
	var err error

	s = `key=value`
	v, err = Slog_parse(s)
	fmt.Printf("%#v; err=%v\n", v, err)

	s = `key=value  `
	v, err = Slog_parse(s)
	fmt.Printf("%#v; err=%v\n", v, err)

	s = `=`
	v, err = Slog_parse(s)
	fmt.Printf("%#v; err=%v\n", v, err)

	s = `key=`
	v, err = Slog_parse(s)
	fmt.Printf("%#v; err=%v\n", v, err)

	s = `=value`
	v, err = Slog_parse(s)
	fmt.Printf("%#v; err=%v\n", v, err)

	s = `key=""`
	v, err = Slog_parse(s)
	fmt.Printf("%#v; err=%v\n", v, err)

	s = `""=value`
	v, err = Slog_parse(s)
	fmt.Printf("%#v; err=%v\n", v, err)

	s = `time=2026-02-16T07:01:25.800Z level=INFO msg="Handling time" rid=1771225285799519 request="GET /mybucket1/?object-lock=" request-length=0 code=200 response-length=155 elapse=611.983µs`
	v, err = Slog_parse(s)
	fmt.Printf("%#v; err=%v\n", v, err)

	fmt.Printf("DONE\n")
}

func TestScan(t *testing.T) {
	fmt.Printf("Test Scan_quoted...\n")

	var s string
	var v string
	var i int
	var err error

	s = `token=value`
	v, i, err = Scan_quoted(s, 0)
	fmt.Printf("%#v + %q; i=%v err=%v\n", v, s[i:], i, err)

	s = `token value`
	v, i, err = Scan_quoted(s, 0)
	fmt.Printf("%#v + %q; i=%v err=%v\n", v, s[i:], i, err)

	s = "token\nvalue"
	v, i, err = Scan_quoted(s, 0)
	fmt.Printf("%#v + %q; i=%v err=%v\n", v, s[i:], i, err)

	s = "tokenstring"
	v, i, err = Scan_quoted(s, 0)
	fmt.Printf("%#v + %q; i=%v err=%v\n", v, s[i:], i, err)

	s = "=empty-token"
	v, i, err = Scan_quoted(s, 0)
	fmt.Printf("%#v + %q; i=%v err=%v\n", v, s[i:], i, err)

	s = `""=quoted-empty-token`
	v, i, err = Scan_quoted(s, 0)
	fmt.Printf("%#v + %q; i=%v err=%v\n", v, s[i:], i, err)

	s = `"t o k e n"remaining`
	v, i, err = Scan_quoted(s, 0)
	fmt.Printf("%#v + %q; i=%v err=%v\n", v, s[i:], i, err)

	s = `"Hello, 世界"`
	v, i, err = Scan_quoted(s, 0)
	fmt.Printf("%#v + %q; i=%v err=%v\n", v, s[i:], i, err)

	s = `"Hello, \u4E16\u754C"`
	v, i, err = Scan_quoted(s, 0)
	fmt.Printf("%#v + %q; i=%v err=%v\n", v, s[i:], i, err)

	s = `"unclosed`
	v, i, err = Scan_quoted(s, 0)
	fmt.Printf("%#v + %q; i=%v err=%v\n", v, s[i:], i, err)

	s = `"backslash-at-tail\`
	v, i, err = Scan_quoted(s, 0)
	fmt.Printf("%#v + %q; i=%v err=%v\n", v, s[i:], i, err)

	fmt.Printf("DONE\n")
}
