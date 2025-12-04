// aws-s3-rule.go

package server

import (
	"regexp"
	"strings"
)

const list_buckets_limit = 10000
const list_objects_limit = 1000
const list_mpul_limit = 1000
const list_parts_limit = 1000

// [Amazon S3 multipart upload limits]
// https://docs.aws.amazon.com/AmazonS3/latest/userguide/qfacts.html

const max_object_size = 5 * 1024 * 1024 * 1024 * 1024
const max_part_number = 10000

// Part size is in range [5 MB, 5 GB] except for the last part.

const part_size_lb = 5 * 1024 * 1024
const part_size_ub = 5 * 1024 * 1024 * 1024

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
//
//    The object key name consists of a sequence of Unicode characters
//    encoded in UTF-8, with a maximum length of 1,024 bytes or
//    approximately 1,024 Latin characters.

const object_name_limit = 1000

// A list of avoidable characters in [Object key naming guidelines].

var object_naming_avoided_set = (`"#%<>[\]^{|}~` + "`")

// CHECK_OBJECT_NAMING checks the naming rules.  A passed name is a
// url path (passed as decoded) and expected as normalized.
func check_object_naming(name string) bool {
	if len(name) > object_name_limit {
		return false
	}
	if strings.ContainsAny(name, object_naming_avoided_set) {
		return false
	}
	var s1 = strings.Split(name, "/")
	if !(len(s1) >= 1 && s1[len(s1) - 1] != "") {
		return false
	}
	return true
}

// TAGGING

// - [Categorizing your objects using tags]
//   - https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-tagging.html

// MEMO: AWS-S3's limit of number of tags is 10, while EC2's limit is
// 50.  AWS-S3's limit of length is in Unicode characters, while EC2'
// limit is in utf-8.

const limit_of_number_of_tags = 10
const limit_of_tag_key_length = 128
const limit_of_tag_value_length = 256

const region_name_min_length = 0
const region_name_max_length = 20
