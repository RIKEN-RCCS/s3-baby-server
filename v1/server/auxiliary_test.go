package server

import (
	//"encoding/xml"
	"fmt"
	"log"
	"strings"
	"testing"
)

// <Tagging xmlns="http://s3.amazonaws.com/doc/2006-03-01/">

var tags1 = `<Tagging>
  <TagSet>
    <Tag>
      <Key>mytag3</Key>
      <Value>prominent</Value>
    </Tag>
    <Tag>
      <Key>mytag4</Key>
      <Value>distinguished</Value>
    </Tag>
  </TagSet>
</Tagging>`

func TestXmlMarshal(t *testing.T) {
	fmt.Printf("Test XML Marshaling...\n")

	var r = strings.NewReader(tags1)
	var x, err1 = intern_Tagging(r)
	if err1 != nil {
		log.Fatalf("Decode() error: %v", err1)
	}
	// var bs, _ = xml.MarshalIndent(x, "", "  ")
	//fmt.Printf("Tagging x=%#v\n", string(bs))

	var bs strings.Builder
	var err2 = extern_Tagging(x, &bs)
	if err2 != nil {
		log.Fatalf("Encode() error: %v", err2)
	}
	fmt.Printf("%v\n", bs.String())

	var tags2 = strings.ReplaceAll(
		strings.ReplaceAll(tags1, " ", ""), "\n", "")
	if bs.String() != tags2 {
		log.Fatalf("results mismatch")
	}

	fmt.Printf("DONE\n")
}
