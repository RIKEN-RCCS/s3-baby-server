// scanner_test.go

package httpaide

import (
	"fmt"
	"testing"
	"time"
)

func TestScanRfc9110Ranges(t *testing.T) {
	fmt.Printf("Test Scan_Rfc9110_Ranges...\n")
	var v [][2]int64
	var err error

	v, err = Scan_rfc9110_ranges(" bytes = 0 - , 4500 - 5499 , - 1000 ")
	fmt.Printf("v=%v err=%v\n", v, err)

	time.Sleep(1 * time.Second)
	fmt.Printf("DONE\n")
}
