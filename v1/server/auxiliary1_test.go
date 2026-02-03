package server

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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

var bucketconfig1 = `<CreateBucketConfiguration>
   <LocationConstraint>string</LocationConstraint>
   <Location>
      <Name>usw2-az1</Name>
      <Type>LocalZone</Type>
   </Location>
   <Bucket>
      <DataRedundancy>SingleLocalZone</DataRedundancy>
      <Type>Directory</Type>
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

var tagging1 = `<Tagging>
  <TagSet>
    <Tag>
      <Key>mykey1</Key>
      <Value>myvalue1</Value>
    </Tag>
    <Tag>
      <Key>mykey2</Key>
      <Value>myvalue2</Value>
    </Tag>
    <Tag>
      <Key>mykey3</Key>
      <Value>myvalue3</Value>
    </Tag>
  </TagSet>
</Tagging>`

var bucketsresult1 = `<ListAllMyBucketsResult>
  <Buckets>
    <Bucket>
      <BucketArn>bucket-arn1</BucketArn>
      <BucketRegion>bucket-region1</BucketRegion>
      <CreationDate>0001-01-01T00:00:00Z</CreationDate>
      <Name>bucket1</Name>
    </Bucket>
    <Bucket>
      <BucketArn>bucket-arn2</BucketArn>
      <BucketRegion>bucket-region2</BucketRegion>
      <CreationDate>0001-01-01T00:00:00Z</CreationDate>
      <Name>bucket2</Name>
    </Bucket>
    <Bucket>
      <BucketArn>bucket-arn3</BucketArn>
      <BucketRegion>bucket-region3</BucketRegion>
      <CreationDate>0001-01-01T00:00:00Z</CreationDate>
      <Name>bucket3</Name>
    </Bucket>
  </Buckets>
  <Owner>
    <DisplayName>name1</DisplayName>
    <ID>id1</ID>
  </Owner>
  <ContinuationToken>continuation1</ContinuationToken>
  <Prefix>prefix1</Prefix>
</ListAllMyBucketsResult>`

func TestXmlMarshal(t *testing.T) {
	fmt.Printf("Test XML Marshaling...\n")

	/*
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

			var target = strings.ReplaceAll(
				strings.ReplaceAll(buckets1, " ", ""), "\n", "")
			if bs.String() != target {
				log.Fatalf("RESULTS MISMATCH")
			}
		}
	*/

	{
		var o O_CreateBucketConfiguration
		var r = strings.NewReader("")
		var d = xml.NewDecoder(r)
		var err1 = d.Decode(&o)
		if err1 != nil {
			if err1 != io.EOF {
				log.Fatalf("Decode() error on empty stream: %v", err1)
			}
		}
		var _ = import_CreateBucketConfiguration(&o)
	}

	{
		var o O_CreateBucketConfiguration
		var r = strings.NewReader(bucketconfig1)
		var d = xml.NewDecoder(r)
		var err1 = d.Decode(&o)
		if err1 != nil {
			log.Fatalf("Decode() error: %v", err1)
		}
		var x = import_CreateBucketConfiguration(&o)
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
			log.Fatalf("RESULTS MISMATCH")
		}
	}

	{
		var o O_Tagging
		var r = strings.NewReader(tagging1)
		var d = xml.NewDecoder(r)
		var err1 = d.Decode(&o)
		if err1 != nil {
			log.Fatalf("Decode() error: %v", err1)
		}
		var x = import_Tagging(&o)
		//fmt.Printf("Tagging=%#v\n", x)

		var bs strings.Builder
		var e = xml.NewEncoder(&bs)
		var err2 = export_Tagging(e, x)
		if err2 != nil {
			log.Fatalf("Encode() error: %v", err2)
		}
		fmt.Printf("%v\n", bs.String())

		var target = strings.ReplaceAll(
			strings.ReplaceAll(tagging1, " ", ""), "\n", "")
		if bs.String() != target {
			log.Fatalf("RESULTS MISMATCH")
		}
	}

	{
		var o = s3.GetObjectTaggingOutput{
			TagSet: []types.Tag{
				types.Tag{
					Key:   h_thing_pointer("mykey1"),
					Value: h_thing_pointer("myvalue1"),
				},
				types.Tag{
					Key:   h_thing_pointer("mykey2"),
					Value: h_thing_pointer("myvalue2"),
				},
				types.Tag{
					Key:   h_thing_pointer("mykey3"),
					Value: h_thing_pointer("myvalue3"),
				},
			},
			VersionId: h_thing_pointer("version-id-is-not-in-xml"),
		}
		var x = O_GetObjectTaggingResponse(o)
		var bs strings.Builder
		var e = xml.NewEncoder(&bs)
		var err2 = e.Encode(x)
		if err2 != nil {
			log.Fatalf("Encode() error: %v", err2)
		}
		fmt.Printf("%v\n", bs.String())

		var target = strings.ReplaceAll(
			strings.ReplaceAll(tagging1, " ", ""), "\n", "")
		if bs.String() != target {
			log.Fatalf("RESULTS MISMATCH")
		}
	}

	{
		var o = s3.ListBucketsOutput{
			// ListAllMyBucketsResult
			Buckets: []types.Bucket{
				types.Bucket{
					BucketArn:    h_thing_pointer("bucket-arn1"),
					BucketRegion: h_thing_pointer("bucket-region1"),
					CreationDate: &time.Time{},
					Name:         h_thing_pointer("bucket1"),
				},
				types.Bucket{
					BucketArn:    h_thing_pointer("bucket-arn2"),
					BucketRegion: h_thing_pointer("bucket-region2"),
					CreationDate: &time.Time{},
					Name:         h_thing_pointer("bucket2"),
				},
				types.Bucket{
					BucketArn:    h_thing_pointer("bucket-arn3"),
					BucketRegion: h_thing_pointer("bucket-region3"),
					CreationDate: &time.Time{},
					Name:         h_thing_pointer("bucket3"),
				},
			},
			ContinuationToken: h_thing_pointer("continuation1"),
			Owner: &types.Owner{
				DisplayName: h_thing_pointer("name1"),
				ID:          h_thing_pointer("id1"),
			},
			Prefix: h_thing_pointer("prefix1"),
		}
		var x = O_ListBucketsResponse(o)
		var bs strings.Builder
		var e = xml.NewEncoder(&bs)
		var err2 = e.Encode(x)
		if err2 != nil {
			log.Fatalf("Encode() error: %v", err2)
		}
		fmt.Printf("%v\n", bs.String())

		var target = strings.ReplaceAll(
			strings.ReplaceAll(bucketsresult1, " ", ""), "\n", "")
		if bs.String() != target {
			log.Fatalf("RESULTS MISMATCH")
		}
	}
}

func export_CreateBucketConfiguration(e *xml.Encoder, i *types.CreateBucketConfiguration) error {
	var o = O_CreateBucketConfiguration{
		Bucket:             i.Bucket,
		Location:           i.Location,
		LocationConstraint: i.LocationConstraint,
		Tags: struct {
			Tag []types.Tag
		}{Tag: i.Tags},
	}
	var err1 = e.Encode(&o)
	if err1 != nil {
		return fmt.Errorf("CreateBucketConfiguration: %w", err1)
	}
	return nil
}

func export_Tagging(e *xml.Encoder, i *types.Tagging) error {
	var o = O_Tagging{
		TagSet: struct {
			Tag []types.Tag
		}{Tag: i.TagSet},
	}
	var err1 = e.Encode(&o)
	if err1 != nil {
		return fmt.Errorf("CreateBucketConfiguration: %w", err1)
	}
	return nil
}
