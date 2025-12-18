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
  object name.  It is needed to keep correspondence between an object
  and its meta-info.  Deletion of an object and its meta-info is not
  atomic.  Baby-server performs a deletion of a meta-info first and a
  failure of a deletion of an object may leave an object without
  meta-info.

- Downloading of an object is not excluded by other operations such as
  deletion.  Thus, an object can be truncated while downloading.

- Deletion of buckets/objects are slack.  Deletion is serialized after
  checking conditions.  ETag calculation takes time and it is placed
  out size of serialization.

- Listing of objects and parts (of multipart uploads) are slack.
  Listing is performed without serialization.

- Operations between buckets and objects are not serialized.
  Exclusion is based on a bucket or on an object.  A bucket can be
  removed while operations on objects are in progress.

## Upload-ID

Uniqueness of an upload-id is not guaranteed.  Baby-server does not
check the records of currently on-going MPUL's, although they are
stored in files.  It is guaranteed by probability.

## Multipart-Upload (MPUL)

It creates a temporary directory (named "."+filename+"@mpul") and
stores files "info", "list", "partNNNNN".

GetObject with "?partNumber=" is an error in Baby-server.  An object
uploaded by multipart-upload is concatenated at completion and its
parts are lost.  Note it is not a legal operation in AWS-S3 to
download a part while multipart-upload is in-progress.

(* DeleteBucket does not remove a bucket in existences of some MPUL's
that are in-progress. *)

ListMultipartUploads never returns NextUploadIdMarker in the output.

The saved array of parts are indexed in zero origin (part - 1).

## Copying a file

Baby-server copies a file by linking a file.  The mtime of a new file
is not updated.  Note AWS-S3 never updates files and linking is safe.

## Timestamps of objects

AWS-S3 only manages timestamps of objects in mtime.  Only buckets
needs ctime and mtime.  An object's mtime is amended when it is
created with multipart upload.

## Checksums

Baby-server can only handle "types.ChecksumTypeFullObject".  A request
of a checksum is ignored when it is not the case.  A returned checksum
is always for full object.

Baby-server records minimal metadata.  It discards metadata except for
explicitly provided tags and headers.  Especially, it does not record
checksums or checksum algorithms, too.  It returns checksums of
CRC64NVME in spite of an algorithm specified at "PutObject".

Baby-server ignores checking with "x-amz-sdk-checksum-algorithm" (note
it is with "sdk").  This header is used to check the algorithm matches
the stored one.  Baby-server does not store the checksum algorithm.
Note "x-amz-sdk-checksum-algorithm" is used in {DeleteObjects,
PutObject, PutObjectTagging, UploadPart}.

Note multiple checksum algorithms can be specified in internally
defined types in: types.Object and types.ObjectVersion.

## Responses

Baby-server does not add "xmlns".  It should be something like:
  <XXXX xmlns="http://s3.amazonaws.com/doc/2006-03-01/">

Following lines are needed to add "xmlns", in type definition,
  Xmlns string `xml:"xmlns,attr"`
and in data,
  Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/"

## Request Checks

Baby-server does not check properness of enumerators in XML payload,
while it checks that in headers.  Baby-server uses the standard
unmarshaler and it does not know about enumerators.
