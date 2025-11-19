// aws-s3-rule.go
// Copyright 2025-2025 RIKEN R-CCS.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"regexp"
)

const list_buckets_limit = 10000
const list_objects_limit = 1000

// - [General purpose bucket naming rules]
//   - https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html
// - [Bucket naming guidelines]
//   - https://cloud.google.com/storage/docs/naming-buckets)

var bucket_naming_good_re = regexp.MustCompile(`^[a-z0-9-]{3,63}$`)

var bucket_naming_forbidden_re = regexp.MustCompile(
	`^[0-9.]*$` +
		// Begin and end with a letter or number:
		`|^[-.].*$` +
		`|^.*[-.]$` +
		// No two adjacent periods:
		`|\\.\\.` +
		// Bad prefixes:
		`|^xn--.*$` +
		`|^sthree-.*$` +
		`|^amzn-s3-demo-.*$` +
		// Bad suffixes:
		`|^.*-s3alias$` +
		`|^.*--ol-s3$` +
		`|^.*\\.mrap$` +
		`|^.*--x-s3$` +
		`|^.*--table-s3$` +
		// baby-server's additional rules:
		`|^.*-$` +
		`|^aws$` +
		`|^amazon$` +
		`|^goog.*$` +
		`|^g00g.*$` +
		`|^minio$`)

// CHECK_BUCKET_NAMING checks bucket naming restrictions.  Note
// s3-baby-server forbits any DOTS, "aws", "amazon", "goog*", "g00g*",
// and "minio", in addition to AWS rules.
func check_bucket_naming(name string) bool {
	return (3 <= len(name) && len(name) <= 63 &&
		bucket_naming_good_re.MatchString(name) &&
		!bucket_naming_forbidden_re.MatchString(name))
}

// - [Naming Amazon S3 objects]
//   - https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-keys.html

func check_object_naming(name string) bool {
	return true
}

// TAGGING

// - [Categorizing your objects using tags]
//   - https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-tagging.html

// MEMO: AWS-S3's limit of number of tags is 10, while EC2's limit is
// 50.  AWS-S3's limit of length is in Unicode characters, while EC2'
// limit is in utf-8.

const limit_of_number_of_tags_1 = 10
const limit_of_tag_key_length_ = 128
const limit_of_tag_value_length_ = 256
