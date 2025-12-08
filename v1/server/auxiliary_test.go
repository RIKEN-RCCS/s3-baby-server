package server

import (
	"encoding/xml"
	"fmt"
	"log"
	"strings"
	"testing"
)

// <Tagging xmlns="http://s3.amazonaws.com/doc/2006-03-01/">

var buckets1 = `<Buckets>
  <Bucket>
    <CreationDate>2019-12-11T23:32:47Z</CreationDate>
    <Name>amzn-s3-demo-bucket</Name>
  </Bucket>
  <Bucket>
    <CreationDate>2019-11-10T23:32:13Z</CreationDate>
    <Name>amzn-s3-demo-bucket1</Name>
  </Bucket>
</Buckets>`

var tagging1 = `<Tagging>
  <TagSet>
    <Tag>
      <Key>mytag1</Key>
      <Value>myvalue1</Value>
    </Tag>
    <Tag>
      <Key>mytag2</Key>
      <Value>myvalue2</Value>
    </Tag>
    <Tag>
      <Key>mytag3</Key>
      <Value>myvalue3</Value>
    </Tag>
  </TagSet>
</Tagging>`

var bucketconfig1 = `<CreateBucketConfiguration>
   <LocationConstraint>string</LocationConstraint>
   <Location>
      <Name>usw2-az1</Name>
      <Type>LocalZone</Type>
   </Location>
   <Bucket>
      <DataRedundancy>SingleLocalZone</DataRedundancy>
   </Bucket>
   <Tags>
      <Tag>
         <Key>key1</Key>
         <Value>value1</Value>
      </Tag>
      <Tag>
         <Key>key2</Key>
         <Value>value2</Value>
      </Tag>
      <Tag>
         <Key>key3</Key>
         <Value>value3</Value>
      </Tag>
   </Tags>
</CreateBucketConfiguration>`

func TestXmlMarshal(t *testing.T) {
	fmt.Printf("Test XML Marshaling...\n")

	{
		var r = strings.NewReader(buckets1)
		var d = xml.NewDecoder(r)
		var x, err1 = import_Buckets(d)
		if err1 != nil {
			log.Fatalf("Decode() error: %v", err1)
		}
		// var bs, _ = xml.MarshalIndent(x, "", "  ")
		//fmt.Printf("Buckets x=%#v\n", string(bs))

		var bs strings.Builder
		var e = xml.NewEncoder(&bs)
		var err2 = export_Buckets(e, x)
		if err2 != nil {
			log.Fatalf("Encode() error: %v", err2)
		}
		fmt.Printf("%v\n", bs.String())

		var buckets2 = strings.ReplaceAll(
			strings.ReplaceAll(buckets1, " ", ""), "\n", "")
		if bs.String() != buckets2 {
			log.Fatalf("results mismatch")
		}
	}

	{
		var r = strings.NewReader(tagging1)
		var d = xml.NewDecoder(r)
		var x, err1 = import_Tagging(d)
		if err1 != nil {
			log.Fatalf("Decode() error: %v", err1)
		}
		//fmt.Printf("Tagging=%#v\n", x)

		var bs strings.Builder
		var e = xml.NewEncoder(&bs)
		var err2 = export_Tagging(e, x)
		if err2 != nil {
			log.Fatalf("Encode() error: %v", err2)
		}
		fmt.Printf("%v\n", bs.String())

		var tags2 = strings.ReplaceAll(
			strings.ReplaceAll(tagging1, " ", ""), "\n", "")
		if bs.String() != tags2 {
			log.Fatalf("results mismatch")
		}
	}

	{
		var r = strings.NewReader(bucketconfig1)
		var d = xml.NewDecoder(r)
		var x, err1 = import_CreateBucketConfiguration(d)
		if err1 != nil {
			log.Fatalf("Decode() error: %v", err1)
		}
		// var bs, _ = xml.MarshalIndent(x, "", "  ")
		//fmt.Printf("Buckets x=%#v\n", string(bs))

		var bs strings.Builder
		var e = xml.NewEncoder(&bs)
		var err2 = export_CreateBucketConfiguration(e, x)
		if err2 != nil {
			log.Fatalf("Encode() error: %v", err2)
		}
		fmt.Printf("%v\n", bs.String())

		var bucketconfig2 = strings.ReplaceAll(
			strings.ReplaceAll(bucketconfig1, " ", ""), "\n", "")
		if bs.String() != bucketconfig2 {
			log.Fatalf("results mismatch")
		}
	}
}
