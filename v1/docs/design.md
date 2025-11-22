# README.md

## Error Responses

Error responses are defined in
[Error Responses](https://docs.aws.amazon.com/AmazonS3/latest/API/ErrorResponses.html)

Several errors related to S3 are defined in "types" in AWS-SDK:
https://github.com/aws/aws-sdk-go-v2/service/s3/types/errors.go

- Error Types
  - "BucketAlreadyExists"
  - "BucketAlreadyOwnedByYou"
  - "EncryptionTypeMismatch" (no correponding error code)
  - "IdempotencyParameterMismatch" (no correponding error code)
  - "InvalidObjectState"
  - "InvalidRequest"
  - "InvalidWriteOffset" (no correponding error code)
  - "NoSuchBucket"
  - "NoSuchKey"
  - "NoSuchUpload"
  - "NotFound" (no correponding error code)
  - "ObjectAlreadyInActiveTierError" (no correponding error code)
  - "ObjectNotInActiveTierError" (no correponding error code)
  - "TooManyParts" (no correponding error code)

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

The file "aws-s3-errors.go" is taken from the "Error Responses"
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

## Assuptions on Golang stdlib

- It assumes key part is clean as a filesystem path, as ServMux()
  handles it.

## Serialization (Criticals)

- Accesses to an object file and a meta-info file are serialized by an
  object name.  Without a serialization, an uploaded file and its tags
  could mismatch.

## Upload ID

Uniqueness of upload-id is not guaranteed.  Baby-server does not
record upload-id for restarting the server.

## Multipart Upload (MPUL)

It creates a temporary directory (named "."+filename+"@mpul") and
stores files "info", "list", "partNNNNN".

## ???

v1.1.1 code allowed nested tagging in values in the format
'TagSet=[{Key=<key>,Value=<value>}]'.  (I cannot find about nested
tagging).


## Timestamp of objects

AWS-S3 only manages timestamps of objects in mtime.  Only buckets
needs ctime and mtime.  An object's mtime is amended when it is
created with multipart upload.

## Checksum

Baby-server can only handle "types.ChecksumTypeFullObject".  A
checksum in a request is ignored when it is not the case.  A returned
checksum is always for full object.
