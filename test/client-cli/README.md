# Test by S3 Clients

## Prerequisite

### Run s3-baby-server

```
./s3-baby-server serve 127.0.0.1:9000 ~/pool -cred s3baby,s3babybaby -log debug -log-access -prof 6060
```

The access-key-id is "s3baby" and the secret-access-key is
"s3babybaby" in this README.

Attaching options may be helpful.  "-log" sets the logger level,
"-log-access" enables printing access logs, and "-prof 6060" lets
Golang's pprof mechanism to listen on port=6060.

https://pkg.go.dev/net/http/pprof

Use "-https-crt ssl-certificate-file" and "-https-key
ssl-certificate-key-file" to run the server with https.  For https
testing, a self-signed certificate pair can be created with openssl:

```
make crt
```

### Make data files

Tests needs files of sizes 1k, 8k, 4m, 20m, and 1g.  They are prepared
with:

```
make data-files
```

### Notes

The test scripts don't work with "dash" ("sh" in Ubuntu).  dash lacks
"-o filefail".

## AWS-CLI

### Running a Test

```
bash client-awscli.sh
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
aws_access_key_id = s3baby
aws_secret_access_key = s3babybaby
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
bash client-rclone.sh
```

### Installing RCLONE

RCLONE can be installed by "dnf" from Redhat/Rocky EPEL.

```
dnf install rclone
```

Optionally, enable EPEL first.

```
dnf install epel-release
```

Setting for RCLONE can be found in `~/.config/rclone/rclone.conf`.
The content may look like:

```
[s3bbs]
type = s3
provider = Other
env_auth = false
access_key_id = s3baby
secret_access_key = s3babybaby
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

- RCLONE "lsd" (list buckets) does not work with https (???).  RCLONE
  is rclone v1.73.0.
## Google Cloud CLI

### Running a Test

Run a test by:

```
bash client-gcloud.sh
```

### Installing gcloud (and gsutil)

https://docs.cloud.google.com/sdk/docs/install-sdk

It says "gcloud" can be installed from google's repository:

  - Copy the text to "/etc/yum.repos.d/google-cloud-sdk.repo".
  - Install SDK by DNF.

```
sudo dnf install libxcrypt-compat.x86_64
sudo dnf install google-cloud-cli
```

Minimal (?) configuration is set by:

```
gcloud config set storage/s3_endpoint_url https://localhost:9000
gcloud config set auth/disable_ssl_validation True
```

Configuration is stored in
"~/.config/gcloud/configurations/config_default":

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

### MEMO: Below configuration does not work:

```
cat <<EOF > cred
{
"accessKeyId": "s3baby",
"secretAccessKey": "s3babybaby"
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
aws_access_key_id = s3baby
aws_secret_access_key = s3babybaby
```

## MinIO Client MC

### Running a Test

Setup MC by assigning an alias, for example, "s3baby".  We assume an
alias name "s3baby" in the test script.

```
$ mc alias set "s3baby" "http://localhost:9000" "s3baby" "s3babybaby" --api S3v4
```

Run a test by:

```
bash client-minio-mc.sh
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

Note: s3cmd "mv" does not work in our test.  Uncertain, but, it seems
it needs some ACL definition retured from the server side.

```
s3cmd --configure
```

"~/.s3cfg" needs the following fields, at least.  Specifying
"host_bucket" uses path-style bucket naming.

```
host_base = localhost:9000
host_bucket = localhost:9000
access_key = s3baby
secret_key = s3babybaby
access_token =
website_endpoint =
```

```
bash client-s3cmd.sh
```

Add `--no-ssl` for http access, or drop it for https access.  Add
`s3cmd --debug` for tracing s3cmd.

### Installing s3cmd

https://github.com/s3tools/s3cmd
(https://s3tools.org/s3cmd)

```
pip3 install --user s3cmd
```

It installs: python-magic, s3cmd, python-dateutil.

## s3fs-fuse

### Setup and Mount

Usage to mount the fs is described in:
"https://github.com/s3fs-fuse/s3fs-fuse"

```
echo s3baby:s3babybaby > ~/.passwd-s3fs
chmod 600 ~/.passwd-s3fs

mkdir ~/mnt
s3fs mybucket1 ~/mnt -o url=http://localhost:9000/ -o use_path_request_style -o passwd_file=~/.passwd-s3fs
```

### Installing s3fs-fuse

"s3fs-fuse" is in EPEL.

```
dnf install s3fs-fuse
```

Optionally, enable EPEL first.

```
dnf install epel-release
```

## s4cmd

https://github.com/bloomreach/s4cmd

### Running a Test

s4cmd shares the configuration of AWS-CLI.

```
bash client-s4cmd.sh
```

### Installing s4cmd

s4cmd is in Python.  We used the client in Ubuntu, in this case.

```
apt install s4cmd
```

## (s5cmd)

https://github.com/peak/s5cmd

Download from

https://github.com/peak/s5cmd/releases

## (WinSCP)
