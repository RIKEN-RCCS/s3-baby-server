# test/minima

## Tests by AWS-CLI

### Note on Running AWS CLI

AWS-CLI accesses "http://169.254.169.254/latest/api/token" for
metadata.  It slows tests.  To disable metadata service request, set
the enviroment variable:

```
export AWS_EC2_METADATA_DISABLED=true
```

## Tests by RCLONE

RCLONE assumes an ETag is an MD5 sum.  However, an ETag in Baby-server
is calculated from an inode number, a date, and a size.  Using RCLONE
needs "--ignore-checksum", otherwise it always fails to check a
checksum.

Note that rclone uses http/2.0, while AWS-CLI uses http/1.1.  I cannot
find a way to make AWS-CLI use http/2.0.

~/.config/rclone/rclone.conf

## Tests by Google Cloud CLI

https://docs.cloud.google.com/sdk/docs/install-sdk

## Tests by bbs-ctl

"bbs-ctl" is an AWS-S3 client using AWS-SDK-GO-V2.  It is to stress
the server.

----------------

## Other Tests

This uses GNU-Guile (Scheme language), requiring guile-3.0.9 or later,
as it uses "spawn" to run subprocesses.

## artifact-bottom.json

- Testing the "bottom" set needs to start with an empty bucket-pool.
- Bucket-pool may contain dot files (e.g., ".something").

## Note

In AWC CLI, the "s3" command returns a non-json string, while the
"s3api" command returns json.  Note "--output json" on "s3" command
does not work.

## Tools

- "http-snoop-proxy.sh": It runs a proxy that dumps http traffic:
port=9001 (client side) to port=9000 (server side).

## TODO: CHECK ERROR CASES

- CompleteMultipartUpload operation: "EntityTooSmallError"

----------------

## Miscellaneous Memo

### MEMO: json Pattern Matching

Values are one of the following data types in json:

- string
- number
- object
- array
- boolean
- null

### MEMO

Bucket owner should be something like

```
"Owner": {
    "DisplayName": "minio",
    "ID": "02d6176db174dc93cb1b899f7c6078f08654445fe8cf1b6ce98d8855f66bdbf4"
}
```
