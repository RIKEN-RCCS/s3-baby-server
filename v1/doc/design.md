# Design Memo of Baby-server


## Overall Implementation

Baby-server implements the actions following the API defined by
AWS-SDK.  Whereas AWS-SDK defines the client side, it can be identical
to the server side.  Especially, SDK's function signatures of the
actions and their input/output records are likely to be reused.
Baby-server uses those function signatures. The implementations of
them are provided in "api-action.go".

The main behaviors of the actions are defined in "copying.go",
"listing.go", and "deleting.go".

Baby-server uses an RPC (remote procedure call) stub generator which
generates the server side stubs from the description in Smithy IDL
(interface definition language).  The generated stubs are in
"dispatcher.go", "handler.go", and "marshaler.go".  The stub generator
is "stub.scm" in "adhoc-stub-generator" directory.  It is written in a
dialect of Scheme language and uses Gnu-Guile (version 3) to run the
generator.


## Server Control (via http)

Baby-server specially handles POST calls on "/bbs.ctl/quit" and
"/bbs.ctl/stat", where "quit" stops the server and "stat" dumps memory
usage to the logger at level=INFO.  Since these commands are not
AWS-S3 operations, it cannot be requested by AWS-CLI.  See
"control-client.go" code in "test/control" to issue these commands.


## Metainfo File

Baby-server stores meta-information in a metainfo file named
"." + object-name + "@meta".  This file records:

  - an MD5 value for an ETag, to avoid costly recalculations
  - a checksum value, when an object is uploaded with a checksum
  - tags or metadata headers, when an object has them

An object usually has a metainfo file because it is uploaded with a
checksum value.

A metainfo file is usually created at uploading.  However, a metainfo
file (for an existing file) will be created when an access occurs on
the object (including HEAD/GET accesses).

Baby-server keeps the association of a metainfo file and an object
file by recoding an entity-key (an inode number) in metainfo.


## Temporary Scratch Files

Baby-server creates temporary scratch files on copying or uploading.
An object is once created as a scratch file, and then, it is renamed
to the true name.  A name of a scratch file is named
"." + objectname + "@random".  A random is hex digits.

Scratch files are also created for metainfo files, and for part-files
for MPUL.

Scratch files are created without serializing accesses and its life
time is limited to request processing.


## Exclusion (Serialization)

### Exclusion Overview

Baby-server only excludes modifications on the filesystem.  That is,
listing and downloading are not exclusive with uploading and copying.

In most cases, operations are prepared outside of exclusion and
continue with final renaming in exclusion.  Operations performed with
exclusion are (1) updating its metainfo file, and (2) renaming a
scratch file to an actual object.  Other operations are outside of
exclusion.

Wait time of exclusion has a limit, and a timeout aborts a request by
a RequestTimeout error.  The limit should be large because requests
are queued for exclusion.  The limit is set to 5000ms.  It seems too
large, but we found the work in exclusion took rather long in our test
environment (with Lustre via NFS translator).  Configuration
"Exclusion_wait" can control the time.

Baby-server keeps consistency of identity of a file with an
"entity-key".  An entity-key is an identity of a file, similar to an
ETag, but it is based on an inode number and an mtime to calculate it
fast.  Baby-server often performs a recheck of an entity-key after
exclusion to detect a race condtion.

### Access Order of Metainfo File and Object File

Baby-server stores metainfo in another file, and there is an access
order restriction between a metainfo file and an object file.  Writing
is exclusive, and Baby-server stores metainfo then an object in this
order.

Baby-server keeps the association of metainfo to an object by storing
the entity-key (an inode number) of an object in metainfo.  Reading is
not exclusive, but, to be consistent, metainfo should be read before
an object.

In short, both reading and writing should respect the access order:
metainfo then an object.  To make this restriction work, both metainfo
and an object are stored atomically.  Files are created once in a
scratch file, them renamed to an actual file.

### Exclusion Details

- Accesses to an object file and a metainfo file are serialized by an
  object name.

- Deletion of an object file and its metainfo file is not atomic.
  Baby-server performs a deletion of a metainfo file first.  An errro
  in a deletion of an object may leave an object without metainfo.

- Listing is performed without serialization.  Listing parts of MPUL
  is loose as well as listing of objects.

- Operations between buckets and objects are not serialized.
  Exclusion is based on a bucket or on an object.  A bucket can be
  removed while operations on objects are in progress.


## Multipart-Upload (MPUL)

### MPUL Implementation

Baby-server creates a temporary directory (named
"." + objectname + "@mpul") and stores files "info", "list", and
"partNNNNN".  The array of parts saved in the "list" file are indexed
in zero origin (part - 1).

Although a temporary directory for MPUL is created to store
part-files, scratch files are not stored in that directory.  Instead,
scratch files are stored in the same directory where the object will
be created.

Such placement of scratch files for MPUL is to allow removal of the
temporary directory.  On aborting MPUL, it is necessary to remove the
directory, but on-going copying would prevent removal of the
directory.  (Such prevention behavior is found on NFS).  Placing
scratch files outside the temporary directory will avoid that
behavior.

GetObject with "?partNumber=" is an error in Baby-server (excpet
"?partNumber=1" which is legal).  An object uploaded by
multipart-upload is concatenated at completion and its parts are lost.
Note it is not a legal operation in AWS-S3 to download a part while
multipart-upload is in-progress.

ListMultipartUploads never returns NextUploadIdMarker in the output.


## Chunked-Encoding

### Implementation of Chunked-Transfer

Baby-server uses its own reader for chunked-transfer and does not use
"httputil.NewChunkedReader".  It only checks Transfer-Encoding has a
single entry "chunked".  This restriction is like
http.parseTransferEncoding().

### Prioritize Chunked-Encoding to Chunked-Transfer

Chunked streams do not nest (i.e., use both) in AWS-S3, when both
"transfer-encoding=chunked" and "content-encoding=aws-chunked" are
specified.  Baby-server ignores "transfer-encoding=chunked" when both
are specified.  Note that AWS-CLI may occasionally specify both.

### Chunked-Encoding with STREAMING-UNSIGNED-PAYLOAD-TRAILER

X-Amz-Content-Sha256=STREAMING-UNSIGNED-PAYLOAD-TRAILER is used for
TLS connections.  In this case, it omits chunk-signatures in chunk
headers.  It means it is the same as usual http's chunked stream:
Transfer-Encoding=chunked.  In this case, "X-Amz-Trailer" is used
instead of usual "Trailer" header.

A sample of aws-chunked header from AWS-CLI is:

```
Accept-Encoding: gzip
Expect: 100-continue
Transfer-Encoding: chunked
Content-Length: -1
Content-Encoding: aws-chunked
Trailer:
X-Amz-Content-Sha256: STREAMING-UNSIGNED-PAYLOAD-TRAILER
X-Amz-Date: 20260307T142429Z
X-Amz-Decoded-Content-Length: 8388608
X-Amz-Sdk-Checksum-Algorithm: CRC64NVME
X-Amz-Trailer: x-amz-checksum-crc64nvme
```

"x-amz-content-sha256" values:

  - Actual payload checksum value
  - UNSIGNED-PAYLOAD
  - STREAMING-UNSIGNED-PAYLOAD-TRAILER
  - STREAMING-AWS4-HMAC-SHA256-PAYLOAD
  - STREAMING-AWS4-HMAC-SHA256-PAYLOAD-TRAILER
  - STREAMING-AWS4-ECDSA-P256-SHA256-PAYLOAD
  - STREAMING-AWS4-ECDSA-P256-SHA256-PAYLOAD-TRAILER

Here, actual values and UNSIGNED-PAYLOAD are not chunked.

They are listed in:

  - https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-auth-using-authorization-header.html

Other hints related to chunked streams are:

https://git.deuxfleurs.fr/Deuxfleurs/garage/issues/824

### MEMO: Chunked-Encoding

Transfer-Encoding≠identity ⇔ Content-Length=-1 (omitted)


## Peculiar Processing

### HEAD Request on a Directory Response

Baby-server returns NoSuchKey, on a HEAD request on a directory.
Baby-server ignores all non-regular files in a bucket.

### Ignoring a Trailing-Slash in URL

Baby-server ignores (multiple) trailing-slashes on a path part of a
URL.  It rewrites URL's path and drops trailing-slashes before passing
it to http.ServeMux.  Configuration "Keep_trailing_slash" will disable
this behavior.

It is a bit tedious to ignore a trailing-slash using patterns of
http.ServeMux (go-1.25).  http.ServeMux's pattern matcher treats
"/{bucket}/" as "/{bucket}/{key...}".  That is, the pattern
"/{bucket}/" wouldn't match both "/{bucket}" and "/{bucket}/" as we
hoped for.  The pattern "/{bucket}/{$}" wouldn't work either.

### Fixing an ETag Quoting

Baby-server may attach double-qoutes to an ETag when it misses qoutes.
Configuration "Strict_etag_quoting" will disable this behavior.

Note "s3cmd" passes ETags without qoutes for a part list of a
multipart-upload.

### Extra cr+lf at the End of Chunked Streams

Baby-server accepts the existence of (empty) trailers, when no
trailers are expected.  That is, it accepts one extra cr+lf at the end
of chunks, which ends trailers.  Configuration
"Forbid_last_chunk_crlf" will disable this behavior.

Note MinIO client "mc" may attach an extra cr+lf, for example.

### PartNumber

A part-number can be specified in downloading to select a particular
part of an object that is uploaded by MPUL.  Baby-server does not
support "by-part" downloading, because the file is concatenated to a
single file after uploading and it does not remember the part
information of MPUL.

Specifying a part-number except for MPUL actions is an error.  But, as
an exception, Baby-server treats part-number=1 as a whole object.
Returning an error to a request with a part-number=1 does not work as
some clients give up downloading (errors such as "NoSuchUpload"
404-Not-Found).

### Error Code on a Race Condition

Baby-server returns "InternalError" (500-Internal-Server-Error) on a
race condition, expecting client retries.  Races in Baby-server occur
in modifying files during API actions, which are usually recovered by
retries.

### Name Restrictions

Baby-server follows the naming rules but a bit stricter.

See
[Naming Amazon S3 objects](https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-keys.html)

### Copying a File by Linking

Baby-server copies a file by linking a file.  The mtime of a new file
is not updated in this case.

### Timestamps of objects

AWS-S3 only manages timestamps of objects in mtime.  Only buckets
needs ctime and mtime.  An object's mtime is amended when it is
created with multipart upload.

### Checksums

Baby-server can only handle "types.ChecksumTypeFullObject".  A request
of a checksum is ignored when it is not the case.  A returned checksum
is always for a full object.

Note multiple checksum algorithms can be specified in internal types
in: types.Object and types.ObjectVersion.

### Ignore Header "x-amz-sdk-checksum-algorithm"

Baby-server ignores "x-amz-sdk-checksum-algorithm" (note it is with
"sdk").  This header is said to be a marker used in AWS-SDK.  Note
"x-amz-sdk-checksum-algorithm" is passed as param.ChecksumAlgorithm in
the following actions.

- DeleteObjects
- PutObject
- PutObjectTagging
- UploadPart

### Fix Checksum Algorithm

Baby-server fixes the checksum algorithm in CreateMultipartUpload to
CRC64NVME when none is given.


## Implementation Limitations

### No Handling of Trailer Headers

Baby-server does not handle http trailer headers in requests or
responses.  It issues an error in log when it ignores them.

### Fetch-Owner

Baby-server does not handle owners and just ignores a query of
"fetch-owner" (although it should be an error).  Note RCLONE requests
it.

### Request Checks

Baby-server does not check properness of enumerators in XML payload,
while it checks properness in headers.  Baby-server uses the standard
unmarshaler and it does not know about enumerators.

### No Request Timeout

Baby-server does not set a timeout for request handlers.

### I/O Error Handling

Baby-server does not check fully transferring data by io.Copy() in
GetObject.  It ignores the count.

### Response with 304-Not-Modified

Baby-server only issues 304 among 3xx status codes.
respond_on_action_error() exceptionally handles status=304 errors.  It
will return a response with setting headers "ETag" and
"Last-Modified".

Note that a status=304 response cannot have a content, and it is
required to have headers from {Content-Location, Date, ETag, Vary}.
See https://www.rfc-editor.org/rfc/rfc9110#status.304

Baby-server returns headers "ETag" and "Last-Modified" on
412-Precondition-Failed, too.

### Responses do not have "xmlns"

Baby-server does not add "xmlns".  It should be something like:

    <XXXX xmlns="http://s3.amazonaws.com/doc/2006-03-01/">

Following lines are needed to add "xmlns" in the source code, for type
definitions,

    Xmlns string `xml:"xmlns,attr"

and for data,

    Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/"

### Inode Numbers in WINDOWS

Baby-server uses an inode number for creating an "entity-key", which
is an indicator of an identity of a file.  Baby-server reopens a file
to obtain an inode number of a file in WINDOWS implementation.

The code in stdlib's os.SameFile() takes an inode number from
fs.FileInfo, but it is not straightforward to implement.  It uses
sameFile() and loadFileId() in "src/os/types_windows.go".  They use
internals of fs.FileInfo and cannot be easily imitated from outside of
the package.

  - https://pkg.go.dev/syscall?GOOS=windows

### File Concatenation in MPUL

Baby-server concatenates parts of MPUL by copying.  There may be a
faster method to concatenates.

### No Timeout Cancellation in Service

Baby-server does not handle contexts by itself, and leaves for stdlib
libraries to handle them.  Thus, Baby-server would wait indefinitely
in service, because timeouts on are set in the http server.

Serving has no timeout limits, but wait time for exclusion has a
limit.  Actions of modifying the filesystem are serialized.
Baby-server uses 5000msec as a limit of wait time for exclusion (The
value can be found in configuration "Exclusion_wait").  During
developing Baby-server, it was noticed that the wait limit of 100msec
was not enough, although a serialized part of an operation is small.

### Time Format

Times in http headers are parsed in Golang's time.RFC1123.  Note the
RFC-1123 uses three letter time-zone.


### Logging from http Library

Logs from Golang's http library is printed at level=ERROR.


## Error Codes

### List of Error Codes

The file "aws-s3-errors.go" lists the errors taken from the "Error
Responses" section.  The "Error Responses" section in the API
specification contains a list of error codes.  It contains 88 total
error entires, but distinct codes are 80 -- it includes nine
"InvalidRequest" duplicate entries.

The same list also exists as error codes (a table in XML) in a
document string in "s3.json".  They are in "shapes" /
"com.amazonaws.s3#Error" / "Code" / "traits" /
"smithy.api#documentation".

### Error Types

AWS-SDK defines errors, but are not defined in detail.  They are
somewhat arbitrary.  Baby-server does not utilize the defined error
types.

Error responses are defined in
[Error Responses](https://docs.aws.amazon.com/AmazonS3/latest/API/ErrorResponses.html)

Several S3-related errors are defined in "types" in AWS-SDK:
https://github.com/aws/aws-sdk-go-v2/service/s3/types/errors.go

Error Types:

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

These error types implement methods: "Error()", "ErrorCode()",
"ErrorFault()", "ErrorMessage()".

These error types are listed in Smithy "s3.json", but their
definitions are empty.

There are some error related (abstract) types: "s3/types.Error",
"s3/types.ErrorDetails", "s3/types.ErrorDocument".

There are many other error codes listed in "Error Responses" of the
API specification.  Those error codes appear in the comment in "types"
in AWS-SDK:
https://github.com/aws/aws-sdk-go-v2/service/s3/types/types.go


## Golang's http Sever

### Body Stream

The body stream is "http.expectContinueReader" (an "io.ReadCloser"),
which implements http 100-continue.  It is defined in
"net/http/server.go".

"http.expectContinueReader" embeds "http.body", and is defined in
"net/http/transfer.go".  The comment says it is to implement an
"io.ReadCloser".

### Some Headers Moved to Structure Fields

Headers "Content-Length", "Transfer-Encoding", "Trailer", and "Host"
are moved from the headers to an http.Request structure:
r.ContentLength, r.TransferEncoding, r.Trailer and r.Host.

r.ContentLength=-1 when "Content-Length" is missing.

The documentation of r.TransferEncoding says it handles "chunked", but
it does not.  If it had handled chunked, implementing aws-chunked got
harder.

### Assuptions on http Server

Baby-server assumes object-key part in an URL is clean as a filesystem
path, as ServMux() handles it.

### Golang's bufio

The default buffer size (32KB) may be small.


## Client Oddities

### Trailing Slashes

Many S3 clients attach a slash (or multiple slashes) at the end of a
URL path of a bucket or an object name.  MinIO client "mc", "s3cmd",
and "s3fs-fuse" do, for example.

### MC

MinIO client "mc" attaches an extra cr+lf in a chucked stream.

### RCLONE

RCLONE requires ETags are MD5 values.  RCLONE checks the returned ETag
against the MD5 sum.  This behavior can be skipped by
"--ignore-checksum".  Note that RCLONE does not attach the header
"Content-MD5" by default.

### S3CMD

s3cmd passes ETags without qoutes for a part list of a
multipart-upload.

### S5CMD

s5cmd specifies at downloads the range larger than an object.  It
attaches a range "bytes=0-52428799" (50MB) even for small files.

### AWS-CLI

AWS-CLI uploads data by a chunked stream when via https.  In that
case, CLI erroneously specifies both http1's transfer-encoding=chunked
and AWS's content-encoding=aws-chunked.  Baby-server silently ignores
http1's chunked when both are specified.  The CLI version is
aws-cli/2.33.20.

### MEMO on AWS-CLI Behavior

  - AWS-CLI uses http/1.1.  There is likely no way to make AWS-CLI use
    http/2.0.

  - AWS-CLI attaches "x-amz-checksum-crc64nvme" by default.

### MEMO on RCLONE Behavior

  - RCLONE first checks the directory part (prior part of "/") of an
    object.  It sends a HEAD request on that part.  Baby-server
    responds to it with an error (invalied argument), because a
    directory is a non-object.

  - RCLONE copies (not upload) an object, when it exists in the remote
    with a same ETag.

  - RCLONE "lsd" (list buckets) does not work with https (???).
    RCLONE is rclone v1.73.0.

  - RCLONE uses http/2.0.

  - RCLONE does not attach "Content-MD5" nor
    "x-amz-checksum-crc64nvme" by default.

  - RCLONE requests "fetch-owner" in query.

  - RCLONE does not accept "UTC" for "GMT" in time strings, although
    it seems to try a couple of time formats.


## Clearification of Checksums in AWS-S3 Definition

### Multipart Upload

In CreateMultipartUpload, the checksum algorithm of a target object is
specified by "x-amz-checksum-algorithm" and "x-amz-checksum-type".
The algorithm and the type is returned in the response.

In CompleteMultipartUpload, the checksum algorithm of a target object
is specified by "x-amz-checksum-xxx" and "x-amz-checksum-type".  The
checksum value is returned in the response.

The checksum type should be equal in CreateMultipartUpload and
CompleteMultipartUpload.

### UploadPart

In UploadPart, the checksum algorithm is specified by
"x-amz-checksum-xxx".  The checksum value is returned in the response.
("x-amz-sdk-checksum-algorithm" is ignored).

### UploadPartCopy

In UploadPartCopy, the checksum algorithm is not explicitly described.
By the guess from CopyObject, the checksum algorithm is the one
specified at CreateMultipartUpload.  (The checksum algorithm is
required in CreateMultipartUpload, and thus, the checksum is never
copied from the source).  The checksum value is returned in the
response.

### PutObject

In PutObject, the checksum algorithm is specified by "x-amz-checksum-".
The checksum value is returned in the response.

### CopyObject

In CopyObject, the checksum algorithm is specified by
"x-amz-checksum-algorithm", or otherwise, it is copied from the
source.  The checksum value is returned in the response.

### The Default Checksum Algorithm

The default is "CRC64NVME".  It is described in the User Guide.

Also, the API document says CRC64NVME checksum is added when an object
uploaded without a checkusm, in sections CopyObject and PutObject.

### Required or Optional Headers

Required headers are described in Section "Checksums with multipart
upload operations" in the User Guide.

"x-amz-checksum-algorithm" is requied in CreateMultipartUpload and
CompleteMultipartUpoad, while it is sometimes optional in UploadPart.


## CODING RULES (Naming Variables)

### Variable names of os-Dependent Paths

In Golang, os-dependent paths are the results of filepath.Clean() or
filepath.Join().  In the source code, os-dependent paths prefer names
including "path", while os-independent paths (/-paths) are given names
avoiding "path".  Variables are usually given names such as "object",
"source", or "target" for /-paths.

Note os-dependence of paths by stdlib packages.

  - os-independent: "path", "io/fs"
  - os-dependent: "path/filepath", "os"

----------------

## References

https://docs.aws.amazon.com/s3/

https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3
