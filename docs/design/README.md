# README.md

## Error Responses

Error responses are defined in
[Error Responses](https://docs.aws.amazon.com/AmazonS3/latest/API/ErrorResponses.html)

Several errors related to S3 are defined in "types" in AWS-SDK:
https://github.com/aws/aws-sdk-go-v2/service/s3/types/errors.go

- Error Types
  - "BucketAlreadyExists"
  - "BucketAlreadyOwnedByYou"
  - "EncryptionTypeMismatch"
  - "IdempotencyParameterMismatch"
  - "InvalidObjectState"
  - "InvalidRequest"
  - "InvalidWriteOffset"
  - "NoSuchBucket"
  - "NoSuchKey"
  - "NoSuchUpload"
  - "NotFound"
  - "ObjectAlreadyInActiveTierError"
  - "ObjectNotInActiveTierError"
  - "TooManyParts"

These error types implement methods:
"Error()", "ErrorCode()", "ErrorFault()", "ErrorMessage()".  There are
some error related types: "s3/types.Error", "s3/types.ErrorDetails",
"s3/types.ErrorDocument".

These error types are listed in Smithy "s3.json", but their
definitions are empty.

There are many other error codes listed in "Error Responses" of the
API specification.  Those error codes appear in the comment in "types"
in AWS-SDK:
https://github.com/aws/aws-sdk-go-v2/service/s3/types/types.go

Errors are not defined in detail.  They are somewhat arbitrary (?).

### List of Error Codes

The file "aws-s3-error-codes.go" is taken from the "Error Responses"
section.

The "Error Responses" section in the API specification contains a list
of error codes.  It contains 88 total error entires, but distinct
codes are 80 -- it includes nine "InvalidRequest" duplicate entries.

The same list also exists as error codes (a table in XML) in a
document string in "s3.json".  They are in "shapes" /
"com.amazonaws.s3#Error" / "Code" / "traits" /
"smithy.api#documentation".

## Name Restrictions

[Naming Amazon S3 objects](https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-keys.html)
