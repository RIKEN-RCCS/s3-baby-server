# README

S3-Baby-server is a file server via AWS-S3 protocol.  It is designed
to share existing files in a usual filesystem via AWS-S3.  Note
full-fledged servers are not adequate for the purpose as they store
files in chunks (of manageable sizes).  It is similar to "rclone serve
s3".  Baby-server can be used in combination with "Lens3" to run
multiple servers at a single http end-point
(https://github.com/RIKEN-RCCS/lens3).

## Running a server

```
./s3-baby-server serve 127.0.0.1:9000 ~/pool --cred s3baby,s3baby
```

where "~/pool" specifies a pool directory where buckets are created.
Existing directories in the pool are considered as buckets.  "--cred"
specifies a credential pair separated by a comma.

## Restrictions

- Object names cannot begin with a dot (".").  They are hidden and
  internally used.

- Object names cannot be end with "/".

- Bucket names cannot include any dots (".").  It is restrictive
  compared to other servers.

- Object versions are not supported at all.

- Copying by "CopyObject" is only allowed inside a single bucket.

- Symbolic links in a filesystem are ignored; They are treated as not
  exist.  It is an error when an object name (a path) includes
  symbolic links.  It is to avoid a file being stored in an
  inaccessible path.  Baby-server explicitly checks it.

- Baby-server does not return owner information.  "Ower" in responses
  is always missing in ListObjects, etc.

- Tags are not supported on buckets.  Tags on a CreateBucket request
  are ignored.

- ETags are not MD5.  Baby-server generates an ETag from an inode
  number, mtime, and a size.  ETags are always strong.

- GetObjectAttributes returns no "ObjectParts" infomation.
  Baby-server does not retain parts information after finishing
  multi-part uploads.

- Badly formatted http-date in http-headers "if-modified-since",
  "if-unmodified-since", and "x-amz-if-match-last-modified-time"
  invokes an error, although they should be ignored.

- ContentType of a response is "binary/octet-stream".  I am not sure
  it is better be "application/octet-stream".

## Additional Features

- Baby-server stores access logs in a directory ".s3bbs/log" when it
  exists in a pool-direcotry.  It is checked at starting a server.
  The log file is ".s3bbs/log/access-log".

## Terse Error Messages

- Errors returned to a client do not contain information from OS such
  as "fs.PathError", because they may reveal the home path that should
  not be disclosed to a client.

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

### Security

Baby-server does not check the hash in signing.  It uses the given
hash without checking it.

Baby-server uses inode numbers to generate ETags.  There is a concern
that some may consider inode numbers are sensitive.

## Implemented API Actions

Baby-server is based on 2019-03-27 Release of AWS-S3 API.

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
