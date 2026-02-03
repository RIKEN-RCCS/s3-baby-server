// aws-s3-aide.go

package server

import ()

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
