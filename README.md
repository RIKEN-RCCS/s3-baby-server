# README

## Running a server

./s3-baby-server serve 127.0.0.1:9000 ~/pool-s3bbs --cred s3baby,s3baby

## NEED FIX

- CHECK concurrent multipart uploads (with the same arguments).  A 2nd
  start of multipart upload (with the same arguments) to cancel the
  1st one.  Look at the directory ".S3BabyServer/MultipartUpload"

- Increment upload id always.

- File ower.  "Ower" is missing in ListObjects.

- ContentType (maybe) better be "binary/octet-stream"
than "application/octet-stream".

- https support, with oreore-cert creation.

- control support, under a dummy bucket "s3bbs.ctl".

## Restrictions

- Object names cannot begin with a dot (".").  They are hidden, and
  internally used.

- Object names cannot be end with "/".

- No tags are allowed on buckets.  Tags on a request are ignored.

- Symbolic links in the local filesystem are ignored.  Putting an
  object is an error when the path includes symbolic links.  It is to
  avoid a file being stored in an inaccessible path.  Baby-server
  explicitly checks it.

- No owner information is returned.

- Copying by "CopyObject" is only inside a single bucket.

- Baby-server returns an error on bad format of http-date.  It should
  be ignored in "if-modified-since" and "if-unmodified-since".

- ETags of Baby-server is always strong.

- Buckets cannot have tags.

- GetObjectAttributes returns no "ObjectParts" infomation.
  Baby-server does not retain parts information.

## Terse Error Messages

- Errors returned to a client do not contain information from OS such
  as "fs.PathError", because they can show the home path that should
  not be disclosed to a client.

## Additional Features

- Baby-server stores logs in a directory ".s3bbs/access-log" or
  ".s3bbs/server-log" when it exists in a pool-direcotry.  It is
  checked at starting a server.

## Restrictions

### Restrictions on Bucket and Object Names

Restrictions on names for buckets and objects are stricter than
AWS-S3, Google GCS, or MinIO.

Characters of bucket names are from the set "[a-z0-9-]".  DOTS (".") ARE
FORBIDDEN.  Bucket names should not start or end with "-".

Characters of object names are from the set
"[a-zA-Z0-9!$&'()+,-./;=@_]" plus utf-8 characters.

### Restrictions on mtime

The mtime of an object is not correct by copying.  Baby-server
performs simple copying by linking a file.

Baby-server only distinguishes between ctime and mtime on buckets.  It
uses mtime for objects.

## golangci-lint

$ go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0

$TOP/.golangci.yml

It is based on "Golden config for golangci-lint v2.3.0"

https://gist.githubusercontent.com/maratori/47a4d00457a92aa426dbd48a18776322/raw/a4976d0afdc490e34a4c4c7b6221bb12c673a04d/.golangci.yml

--- golangci.yml-golded-2.3.0	2025-10-14 10:15:52.000000000 +0900
+++ ../.golangci.yml	2025-09-24 18:17:22.000000000 +0900
@@ -55,7 +55,7 @@
     - depguard # checks if package imports are in a list of acceptable packages
     - dupl # tool for code clone detection
     - durationcheck # checks for two durations multiplied together
-    - embeddedstructfieldcheck # checks embedded types in structs
+    # - embeddedstructfieldcheck # checks embedded types in structs
     - errcheck # checking for unchecked errors, these unchecked errors can be critical bugs in some cases
     - errname # checks that sentinel errors are prefixed with the Err and error types are suffixed with the Error
     - errorlint # finds code that will cause problems with the error wrapping scheme introduced in Go 1.13

----------------------------------------------------------------

## Unsupported

- Versions
- POST Object

## Implemented API

- AbortMultipartUpload
- CompleteMultipartUpload
- CopyObject
- CreateBucket
- CreateMultipartUpload
- DeleteBucket
- DeleteObject
- DeleteObjects
- DeleteObjectTagging
- GetObject
- GetObjectAttributes
- GetObjectTagging
- HeadBucket
- HeadObject
- ListBuckets
- ListMultipartUploads
- ListObjects
- ListObjectsV2
- ListParts
- PutObject
- PutObjectTagging
- UploadPart
- UploadPartCopy
