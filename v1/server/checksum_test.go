package server

import (
	"bytes"
	"fmt"
	"hash"
	"hash/crc64"
	"io"
	"testing"
)

//const poly_nvme uint64 = 0x9a6c9329ac4bc9b5

func TestCRC64NVME(t *testing.T) {
	fmt.Printf("Check CRC64NVME with known value...\n")

	var data1 = bytes.Repeat([]byte{0x00}, 4096)
	var crc1 = []byte{0x64, 0x82, 0xd3, 0x67, 0xeb, 0x22, 0xb6, 0x4e}
	var data2 = bytes.Repeat([]byte{0xff}, 4096)
	var crc2 = []byte{0xc0, 0xdd, 0xba, 0x73, 0x02, 0xec, 0xa3, 0xac}

	var dataset = [][]byte{data1, data2}
	var crcset = [][]byte{crc1, crc2}
	for i := range dataset {
		var hash1 hash.Hash = crc64.New(crc64.MakeTable(poly_nvme))
		var dati = dataset[i]
		var crci = crcset[i]
		var bc = bytes.NewReader(dati)
		var count, err4 = io.Copy(hash1, bc)
		var sum1 []byte = hash1.Sum(nil)
		fmt.Printf("err=%d\n", err4)
		fmt.Printf("count=%d\n", count)
		fmt.Printf("sum=%#v\n", sum1)
		fmt.Printf("crc=%#v\n", crci)
	}
}
