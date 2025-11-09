// aws-s3-aid.go

package server

import (
	"log"
	"net/url"
	//"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var unsupported_header_list = []string{
	"fetch-owner",
	"versionId",
	"x-amz-acl",
	"x-amz-bucket-object-lock-enabled",
	"x-amz-bypass-governance-retention",
	"x-amz-copy-source-server-side-encryption-customer-algorithm",
	"x-amz-copy-source-server-side-encryption-customer-key",
	"x-amz-copy-source-server-side-encryption-customer-key-MD5",
	"x-amz-expected-bucket-owner",
	"x-amz-grant-full-control",
	"x-amz-grant-read",
	"x-amz-grant-read-acp",
	"x-amz-grant-write",
	"x-amz-grant-write-acp",
	"x-amz-if-match-initiated-time",
	"x-amz-if-match-last-modified-time",
	"x-amz-if-match-size",
	"x-amz-mfa",
	"x-amz-object-lock-legal-hold",
	"x-amz-object-lock-mode",
	"x-amz-object-lock-retain-until-date",
	"x-amz-server-side-encryption",
	"x-amz-server-side-encryption-aws-kms-key-id",
	"x-amz-server-side-encryption-bucket-key-enabled",
	"x-amz-server-side-encryption-context",
	"x-amz-server-side-encryption-customer-algorithm",
	"x-amz-server-side-encryption-customer-key",
	"x-amz-server-side-encryption-customer-key-MD5",
	"x-amz-source-expected-bucket-owner",
	"x-amz-write-offset-bytes",
}

// CHECK: I cannot find about nested tagging, while v1.1.1 code
// allowed nested tagging in values in the format
// 'TagSet=[{Key=<key>,Value=<value>}]'.

func parse_tags(s string) (types.Tagging, error) {
	var m, err1 = url.ParseQuery(s)
	if err1 != nil {
		return types.Tagging{}, err1
	}
	var tags = []types.Tag{}
	for k, v := range m {
		if len(v) != 1 {
			log.Printf("ignore multiple values in tags\n")
		}
		var value string
		if len(v) == 0 {
			value = ""
		} else {
			value = v[0]
		}
		tags = append(tags, types.Tag{Key: &k, Value: &value})
	}
	var tagging = types.Tagging{TagSet: tags}
	return tagging, nil
}
