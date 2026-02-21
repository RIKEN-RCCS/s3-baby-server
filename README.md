# README

S3 Baby-server is a file server of AWS-S3 protocol.  It is designed to
share existing files in a filesystem via S3.  In contrast, most
full-fledged servers store files in chunks (of manageable sizes) and
are not adequate for this purpose.  Baby-server is similar to "rclone
serve s3".  Baby-server can be used in combination with "Lens3" to run
multiple servers at a single http end-point.  See for Lens3
https://github.com/RIKEN-RCCS/lens3.

## Running the server

```
./s3-baby-server serve 127.0.0.1:9000 ~/pool --cred s3baby,s3baby
```

where "~/pool" specifies a pool directory where buckets are created.
Existing directories in the pool are considered as buckets.  "--cred"
specifies a credential pair separated by a comma (access-key and
secret-access-key).

## Build Procedure

Prepare Golang.  Then,

```
cd v1
make get
make
```

Or,

```
go install github.com/RIKEN-RCCS/s3-baby-server/v1@v1.2.1
```

## Restrictions

- Object names cannot begin with a dot (".").  They are hidden and
  internally used.

- Object names cannot be end with "/".

- Bucket names cannot include any dots (".").  It is more restrictive
  compared to other S3 servers.

- Object versions are not supported at all.

- Copying by "CopyObject" is only allowed inside a single bucket.

- Symbolic links in a filesystem are ignored; They are treated as not
  exist.  It is an error when an object name (a path) includes
  symbolic links.  It is to avoid a file being stored in an
  inaccessible path.

- Baby-server does not return owner information.  "Ower" in responses
  is always missing in ListObjects, etc.  The value of query
  "fetch-owner" is ignored.  Configuration on "accept_fetch_owner"
  changes "fetch-owner" to be an error.

- Tags are not supported on buckets.  Tags on a CreateBucket request
  are ignored.

- ETags are MD5.  ETags are always strong.  Baby-server may record an
  ETag as object's metainfo, when the file size is large.

- GetObjectAttributes returns no "ObjectParts" infomation.
  Baby-server does not retain parts information after finishing
  multi-part uploads.

- Badly formatted http-date in http-headers "if-modified-since",
  "if-unmodified-since", and "x-amz-if-match-last-modified-time"
  invokes an error, although they should be ignored.

- ContentType of a response is "binary/octet-stream".  We are not sure
  it is better be "application/octet-stream".

## Access Logs

- Baby-server stores access logs in a directory ".s3bbs/log" when it
  exists in a pool-directory.  It is checked at starting the server.
  The log file is ".s3bbs/log/access-log".  It is useful when outputs
  from the server are not accessible to the user.

## Terse Error Messages

- Errors returned to a client do not contain information from OS such
  as directory paths or user id, because they can be something that
  should not be disclosed to a client.

## Other Restrictions

### Restrictions on Bucket and Object Names

Restrictions on names for buckets and objects are stricter than
AWS-S3, Google GCS, or MinIO.

Characters of bucket names are from the set "[a-z0-9-]".  Bucket names
should not start or end with "-".  DOTS (".") ARE FORBIDDEN.

Characters of object names are from the set
"[a-zA-Z0-9!$&'()+,-./;=@_]" plus utf-8 characters.

### Restrictions on mtime

Baby-server only distinguishes between ctime and mtime on buckets.  It
uses mtime for objects.

The mtime of an object may not correct when copying.  As Baby-server
performs simple copying by making a hard-link of a file, mtime is not
updated.

### Security (IMPORTANT)

Baby-server does not check the message digest in signing.  It uses the
given hash value without checking it.

## Implemented API Actions

Baby-server is based on 2019-03-27 Release of AWS-S3 API.  Baby-server
implements the following list of actions.

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

## For Developers

An implementation note of Baby-server is
[design.md](./v1/doc/design.md)
