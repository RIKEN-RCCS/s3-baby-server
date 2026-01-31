// scanner_test.go

package httpaide

import (
	"fmt"
	"testing"
	"time"
)

func TestScanRfc9110Range(t *testing.T) {
	fmt.Printf("Test Scan Rfc9110 Range...\n")
	var v [][2]int64
	var err error

	v, err = Scan_rfc9110_range(" bytes = 0 - , 4500 - 5499 , - 1000 ")
	fmt.Printf("v=%v err=%v\n", v, err)

	time.Sleep(1 * time.Second)
	fmt.Printf("DONE\n")
}
