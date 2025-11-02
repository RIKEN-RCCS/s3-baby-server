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

- File names cannot begin with a dot ("."), they are hidden.

- No tags are allowed on buckets.  Tags on a request are ignored.

## Additional Features

- s3bbs stores access logs if a ".access-log" directory exists.

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

# API

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
