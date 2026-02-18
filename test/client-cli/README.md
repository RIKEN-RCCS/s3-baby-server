# Test by S3 Clients

## AWS-CLI

### Running a Test

```
sh client-awscli.sh
```

Test stops on an error.

### Installing AWS-CLI

An installation guide can be found at:

https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html

The CLI can be installed in user-mode (non-root):

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

### MEMO on AWS-CLI vs. RCLONE Differences

- AWS-CLI uses http/1.1 while that RCLONE uses http/2.0.  There is
  likely no way to make AWS-CLI use http/2.0.

- AWS-CLI attaches `x-amz-checksum-crc64nvme` by default.  In
  contrast, RCLONE does not attach `Content-MD5` or
  `x-amz-checksum-crc64nvme` by default.  RCLONE checks the returned
  ETag as an MD5 sum.

### Note on Running AWS-CLI

AWS-CLI accesses "http://169.254.169.254/latest/api/token" for
metadata.  It slows tests.  To disable metadata service request, set
the enviroment variable:

```
export AWS_EC2_METADATA_DISABLED=true
```

## RCLONE

### Running a Test

Run a test by:

```
sh client-rclone.sh
```

### Installing RCLONE

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

### MEMO on RCLONE Behavior

- RCLONE assumes an ETag is an MD5 sum, and checks the checksum
  against an ETag.  This behavior can be skipped by
  "--ignore-checksum".

- RCLONE copies (not upload) an object, when it exists in the remote
  with a same ETag.

- RCLONE first checks the directory part (prior part of "/") of an
  object.  It sends a HEAD request on that part.

## Google Cloud CLI

### Running a Test

Run a test by:

```
sh client-gcloud.sh
```

### Installing gcloud (and gsutil)

https://docs.cloud.google.com/sdk/docs/install-sdk

```
$ gcloud config set storage/s3_endpoint_url https://localhost:9000
$ gcloud config set auth/disable_ssl_validation True
```

BELOW DOES NOT WORK:

```
cat <<EOF > cred
{
"accessKeyId": "abcdefghijklmnopqrstuvwxyz",
"secretAccessKey": "abcdefghijklmnopqrstuvwxyz"
}
EOF
gcloud secrets create 'test_secret' --data-file=cred
```

Configuration stored in "~/.boto":

```
[s3]
use-sigv4=True
[Credentials]
s3_host = localhost
s3_port = 9000
aws_access_key_id = abcdefghijklmnopqrstuvwxyz
aws_secret_access_key = abcdefghijklmnopqrstuvwxyz
```

Configuration stored in "~/.config/gcloud/configurations/config_default":

```
[auth]
disable_ssl_validation = True

[storage]
s3_endpoint_url = http://localhost:9000
```

### MEMO: gcloud-storage Command Usage

https://docs.cloud.google.com/sdk/gcloud/reference/storage

### MEMO: gcloud-storage logs

Logs are stored in "~/.config/gcloud/logs".

## MinIO Client MC

### Running a Test

Setup MC by assigning an alias, for example, "s3baby".  We assume an
alias name "s3baby" in the test script.

```
$ mc alias set "s3baby" "http://localhost:9000" "abcdefghijklmnopqrstuvwxyz" "abcdefghijklmnopqrstuvwxyz" --api S3v4
```

Run a test by:

```
sh client-minio-mc.sh
```

### Installing "mc"

```
wget https://dl.min.io/client/mc/release/linux-amd64/mc
```

Configuration of "mc" is stored in "~/.mc/config.json".

### References on "mc"

- https://github.com/minio/mc
- https://docs.min.io/enterprise/aistor-object-store/reference/cli/

## s3cmd

### Running a Test

Note s3cmd "mv" does not work in our test.  Uncertain, but, it seems
it needs some ACL definition retured from the server side.

```
s3cmd --configure
```

"~/.s3cfg" needs the following fields, at least.

```
host_base = localhost:9000
access_key = abcdefghijklmnopqrstuvwxyz
secret_key = abcdefghijklmnopqrstuvwxyz
access_token =
website_endpoint =
```

```
sh client-s3cmd.sh
```

#### Installing s3cmd

https://github.com/s3tools/s3cmd
(https://s3tools.org/s3cmd)

```
pip3 install --user s3cmd
```

Collecting s3cmd
  Downloading s3cmd-2.4.0-py2.py3-none-any.whl (164 kB)
Collecting python-magic
  Downloading python_magic-0.4.27-py2.py3-none-any.whl (13 kB)
Requirement already satisfied: python-dateutil in /home/users/m-matsuda/.local/lib/python3.9/site-packages (from s3cmd) (2.9.0.post0)
Requirement already satisfied: six>=1.5 in /home/users/m-matsuda/.local/lib/python3.9/site-packages (from python-dateutil->s3cmd) (1.16.0)
Installing collected packages: python-magic, s3cmd
Successfully installed python-magic-0.4.27 s3cmd-2.4.0

## s3fs-fuse

### Setup and Mount

Usage to mount the fs is described in:
"https://github.com/s3fs-fuse/s3fs-fuse"

```
echo abcdefghijklmnopqrstuvwxyz:abcdefghijklmnopqrstuvwxyz > ~/.passwd-s3fs
chmod 600 ~/.passwd-s3fs

mkdir ~/mnt
s3fs mybucket1 ~/mnt -o url=http://localhost:9000/ -o use_path_request_style -o passwd_file=~/.passwd-s3fs
```

### Installing s3fs-fuse

```
dnf install s3fs-fuse
```

"s3fs-fuse" is in EPEL.

### (Tests by WinSCP)
