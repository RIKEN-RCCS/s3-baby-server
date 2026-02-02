# Design Memo of Baby-Server

## Overall Work

Baby-server implements the actions following the API defined by
AWS-SDK.  Whereas the SDK is defined for the client side, it is
identical for the server side.  Especially, it uses the functions and
their input/output structures unchanged.  The implementation of the
actions are in "api-action.go".

Baby-server uses an RPC stubs generator which generates the server
side stub from the definition in Smithy.  The stubs are in
"dispatcher.go", "handler.go", and "marshaler.go".  The stub generator
is "stub.scm" in "adhoc-stub-generator" directory.  "stub.scm" runs
with Gnu-Guile which is a dialect of Scheme language.

The main behaviors of the actions are in "copying.go", "listing.go",
and "deleting.go".

## Server Control

Baby-server handles POST calls on "/bbs.ctl/quit" and "/bbs.ctl/stat",
where "quit" stops the server, and "stat" dumps memory usage to logger
at level=INFO.  Since these commands are not AWS-S3 operations, it
cannot be requested by AWS-CLI.  See "control-client.go" code in
"test/minima" to issue the commands.

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

Uniqueness of upload-ids is not guaranteed.  Baby-server does not
check the ID's of the currently on-going MPUL, although they are
stored in files.  It is only guaranteed by probabilistically as
upload-ids are generated randomly.

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
is not updated.

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

Note multiple checksum algorithms can be specified in internal types
in: types.Object and types.ObjectVersion.

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

## Temporary "Scratch" Files

Baby-server creates temporary "scratch" files on copying or uploading.
An object is once created as a scratch file, and when copying
completes, it is renamed to the true name.  A name of a scratch file
is prefixed with "." and suffixed with "@random" to the object name.
A random is hex digits.  Scratch files are created without serializing
accesses and its life time is limited while processing a request.

Baby-server creates scratch files for part-files for MPUL, too.
Although a temporary directory for MPUL is created to store
part-files, scratch files are not stored in that directory.  Instead,
scratch files are stored in the same directory where the MPUL object
will be created.

The above mentioned placement of scratch files for MPUL is to allow
removal of the temporary directory.  On aborting MPUL, it is necessary
to remove the directory, but on-going copying would prevent removal of
the directory.  (Such prevention behavior is found on NFS).  Placing
scratch files outside the temporary directory will avoid that
behavior.

## Implementation Limitations

### No Handling of Trailer Headers

Baby-server does not handle http trailer headers in either requests or
responses.  It issues an error in log and ignores them when trailer
headers are received.

### Assuptions on http Server in Golang stdlib

Baby-server assumes key part is clean as a filesystem path, as
ServMux() handles it.

### No Request Timeout

Baby-server does not set a timeout for request handlers.

### Response with 304-Not-Modified

Baby-server only issues 304 among 3xx status codes.
respond_on_action_error() handles errors with status=304
exceptionally.  It returns a response with headers "ETag" and
"Last-Modified".

Note that a 304 response cannot have a content, and it is required to
have a header from {Content-Location, Date, ETag, Vary}.

https://www.rfc-editor.org/rfc/rfc9110#status.304

Baby-server returns header ""ETag" and "Last-Modified" on
412-Precondition-Failed, too.

## MEMO

### Header "x-amz-sdk-checksum-algorithm"

Baby-server ignores "x-amz-sdk-checksum-algorithm" (note it is with
"sdk").  This header is said to be a marker used in AWS-SDK.  Note
"x-amz-sdk-checksum-algorithm" is passed as param.ChecksumAlgorithm in
the following actions.

- DeleteObjects
- PutObject
- PutObjectTagging
- UploadPart

### I/O Error Handling

Baby-server does not check fully transferring data by io.Copy() in
GetObject.  Also, it does not check on concatenating part files of
MPUL.  It ignores the count.

### Cancellation in Service

Baby-server does not handle contexts by itself, and assumes the
libraries (stdlib and AWS-SDK) handle them.  In addition, Baby-server
would wait indefinitely, because the timeout values on the http server
are not changed from the default.

### Time Format

Times in http headers are parsed in Golang's time.RFC1123.  Note the
RFC-1123 uses three letter time-zone.

### (MEMO) Logging

Server logs from Golang's http library is printed at level=ERROR.

### Wait Time Limit of Serialization

Actions of modifying the filesystem are serialized.  Baby-server uses
100 msec as a limit of wait time for exclusion (The value can be found
in "Exclusion_wait").  During developing Baby-server, it was noticed
that the wait limit of 10 msec was not enough, although a serialized
part of an operation is small.

### Chunked Encoding

Baby-server uses "httputil.NewChunkedReader" for chunked transfer.  It
only checks Transfer-Encoding as it is a single entry "chunked".  This
restriction is like http.parseTransferEncoding().

### Time Format (time.RFC1123)

rclone does not accept "UTC" for "GMT" in time strings, although it
seems to try a couple of time formats.

## References

https://docs.aws.amazon.com/s3/

https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3
