package server

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var body1 = `<CompleteMultipartUpload xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Part>
    <ETag>"etag1"</ETag>
    <PartNumber>1</PartNumber>
  </Part>
  <Part>
    <ETag>"etag2"</ETag>
    <PartNumber>2</PartNumber>
  </Part>
</CompleteMultipartUpload>`

var body2 = `<Delete xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Object>
    <ETag>etag1</ETag>
    <Key>key1</Key>
    <LastModifiedTime>2006-01-02T15:04:05Z</LastModifiedTime>
    <Size>4096</Size>
    <VersionId></VersionId>
  </Object>
  <Object>
    <ETag>etag2</ETag>
    <Key>key2</Key>
    <LastModifiedTime>2006-01-02T15:04:05Z</LastModifiedTime>
    <Size>1024</Size>
    <VersionId></VersionId>
  </Object>
  <Quiet>false</Quiet>
</Delete>`

func TestXmlMarshal2(t *testing.T) {
	fmt.Printf("Test XML Marshaling 2...\n")
	{
		var r = strings.NewReader(body1)

		var o O_CompletedMultipartUpload
		var err1 = h_decode_body(&o, r, nil)
		if err1 != nil {
			log.Fatalf("Decode() error: %v", err1)
		}
		var x *types.CompletedMultipartUpload = import_CompletedMultipartUpload(&o)
		fmt.Printf("Body x=%#v\n", x)
	}

	{
		var r = strings.NewReader(body2)

		var o O_Delete
		var err1 = h_decode_body(&o, r, nil)
		if err1 != nil {
			log.Fatalf("Decode() error: %v", err1)
		}
		var x *types.Delete = import_Delete(&o)
		fmt.Printf("Body x=%#v\n", x)
	}
}
