// range-reader_test.go

package server

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
)

func TestRangeReader(t *testing.T) {
	fmt.Printf("Test Range_reader...\n")

	var f, err1 = os.CreateTemp("", "gomi")
	if err1 != nil {
		log.Fatal(err1)
	}
	defer os.Remove(f.Name())
	var _, err2 = f.Write([]byte("content01/content02/content03/content04"))
	if err2 != nil {
		log.Fatal(err2)
	}
	var _, err3 = f.Seek(0, 0)
	if err3 != nil {
		log.Fatal(err3)
	}

	var extent = [2]int64{10, 19}
	var r = New_range_reader(f, &extent)

	var bs, err4 = io.ReadAll(r)
	if err4 != nil {
		log.Fatal(err4)
	}
	var err5 = r.Close()
	if err5 != nil {
		log.Fatal(err5)
	}

	fmt.Printf("range-reader output: '%s'\n", bs)
	if bytes.Compare(bs, []byte("content02")) != 0 {
		log.Fatal("bad result")
	}
}
