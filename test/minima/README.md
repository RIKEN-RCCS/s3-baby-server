# test/minima

## Tests by S3 Clients

### Tests by AWS-CLI

#### Installing AWS-CLI

A guide of installation can be found at:

https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html

It can be installed in user-mode (non-root):

`./aws/install -i ~/opt/aws-cli -b ~/bin`

Setting of `~/.aws/config` may look like:

```
[default]
s3 =
     signature_version = s3v4
ec2_metadata_disabled = true
endpoint_url = http://127.0.0.1:9000
aws_access_key_id = abcdefghijklmnopqrstuvwxyz
aws_secret_access_key = abcdefghijklmnopqrstuvwxyz
```

#### MEMO on AWS-CLI vs. RCLONE Differences

- AWS-CLI uses http/1.1 while that RCLONE uses http/2.0.  I cannot
  find a way to make AWS-CLI use http/2.0.

- AWS-CLI attaches `x-amz-checksum-crc64nvme` by default.  In
  contrast, RCLONE does not attach `Content-MD5` or
  `x-amz-checksum-crc64nvme` by default.  RCLONE checks the returned
  ETag as an MD5 sum.

#### Note on Running AWS-CLI

AWS-CLI accesses "http://169.254.169.254/latest/api/token" for
metadata.  It slows tests.  To disable metadata service request, set
the enviroment variable:

```
export AWS_EC2_METADATA_DISABLED=true
```

### Tests by RCLONE

#### Installing RCLONE

RCLONE can be installed by `dnf info rclone` on Redhat/Rocky.

Setting for RCLONE can be found in `~/.config/rclone/rclone.conf`.
The content may look like:

```
[s3bbs]
type = s3
provider = Other
env_auth = false
access_key_id = abcdefghijklmnopqrstuvwxyz
secret_access_key = abcdefghijklmnopqrstuvwxyz
endpoint = https://localhost:9000
acl = private
```

#### MEMO on RCLONE Behavior

- RCLONE assumes an ETag is an MD5 sum, and checks the checksum
  against an ETag.  This behavior can be skipped by
  "--ignore-checksum".

- RCLONE copies (not upload) an object, when it exists in the remote
  with a same ETag.

- RCLONE first checks the directory part (prior part of "/") of an
  object.  It sends a HEAD request on that part.

### Tests by Google Cloud CLI

gsutil
or
gcloud storage

#### Installing gcloud (and gsutil)

https://docs.cloud.google.com/sdk/docs/install-sdk

$ gcloud config set storage/s3_endpoint_url https://localhost:9000
$ gcloud config set auth/disable_ssl_validation True

#### gcloud-storage Command Usage

https://docs.cloud.google.com/sdk/gcloud/reference/storage

### (Tests by WinSCP)

### Tests by s3cmd

#### Installing s3cmd

  pip install s3cmd

### Tests by MinIO Client (mc)

#### Installing "mc"

  wget https://dl.min.io/client/mc/release/linux-amd64/mc

### Tests by s3fs-fuse

#### Installing s3fs-fuse

  apt install s3fs

----------------

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
