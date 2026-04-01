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

Prepare `cli-conf.sh`, which specifies the bucket name and the
access-key.  For example,

```
BKT=lenticularis-oddity-x1
export AWS_ACCESS_KEY_ID=abcdefghijkl
export AWS_SECRET_ACCESS_KEY=abcdefghijklmnopqrstuvwxyz
```

Run the test by

```
bash object-awscli.sh
```

Test stops on an error.

### Installing AWS-CLI

An installation guide can be found at:

https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html

The CLI can be installed in user-mode (non-root):

`./aws/install -i ~/opt/aws-cli -b ~/bin`

An access-key can be stored either in environment variables or in
setting `~/.aws/config`.  Setting of `~/.aws/config` may look like:

```
[default]
s3 =
     signature_version = s3v4
ec2_metadata_disabled = true
endpoint_url = http://127.0.0.1:9000
# aws_access_key_id = abcdefghijkl
# aws_secret_access_key = abcdefghijklmnopqrstuvwxyz
```

### Note on Running AWS-CLI

AWS-CLI accesses "http://169.254.169.254/latest/api/token" for
metadata.  It slows tests.  To disable metadata service request, set
the enviroment variable:

```
export AWS_EC2_METADATA_DISABLED=true
```

## Google Cloud CLI

### Running a Test

Prepare `cli-conf.sh`, which specifies the bucket name and the
access-key.  For example,

```
BKT=lenticularis-oddity-x1
export AWS_ACCESS_KEY_ID=abcdefghijkl
export AWS_SECRET_ACCESS_KEY=abcdefghijklmnopqrstuvwxyz
```

Run the test by

```
bash object-gcloud.sh
```

### Installing gcloud

https://docs.cloud.google.com/sdk/docs/install-sdk

"gcloud" can be installed from google's repository:

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

### MEMO: Store Access-key in "~/.boto"

Configuration can be stored in "~/.boto":

```
[s3]
use-sigv4=True
[Credentials]
s3_host = localhost
s3_port = 9000
aws_access_key_id = abcdefghijkl
aws_secret_access_key = abcdefghijklmnopqrstuvwxyz
```

An access-key can be stored in environment variables or in boto3
setting "~/.boto".

## RCLONE

### Running a Test

Prepare `cli-conf.sh`, which specifies the bucket name.  For example,

```
BKT=lenticularis-oddity-x1
```

Set the access-key in `~/.config/rclone/rclone.conf`.  Environment
variables do not work.  See below for configuration.

Run the test by

```
bash object-rclone.sh
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
access_key_id = abcdefghijkl
secret_access_key = abcdefghijklmnopqrstuvwxyz
endpoint = https://localhost:9000
acl = private
```

## MinIO Client MC

### Running a Test

Set up MC by assigning an alias with an arbitrary name, "s3baby", for
example.  The test script assumes the alias as "s3baby".

```
$ mc alias set --api S3v4 --insecure "s3baby" "https://localhost:9000" "abcdefghijkl" "abcdefghijklmnopqrstuvwxyz"
```

Configuration of "mc" is stored in `~/.mc/config.json`.

Run the test by

```
bash object-minio-mc.sh
```

### Installing "mc"

```
wget https://dl.min.io/client/mc/release/linux-amd64/mc
```

### References on "mc"

  - https://github.com/minio/mc
  - https://docs.min.io/enterprise/aistor-object-store/reference/cli/

## s3cmd

https://github.com/s3tools/s3cmd

### Running a Test

Prepare `cli-conf.sh`, which specifies the bucket name.  For example,

```
BKT=lenticularis-oddity-x1
```

Set the access-key in `~/.s3cfg`.  Environment variables do not work.
See below for configuration.

Run the test by

```
bash object-s3cmd.sh
```

Add `--no-ssl` for http access, or drop it for https access.  Add
`s3cmd --debug` for tracing operation of s3cmd.

### Installing s3cmd

https://github.com/s3tools/s3cmd
(https://s3tools.org/s3cmd)

```
pip3 install --user s3cmd
```

It installs: python-magic, s3cmd, python-dateutil.

### Configuring s3cmd

```
s3cmd --configure
```

`~/.s3cfg` needs the following fields, at least.  Empty "host_bucket"
uses path-style bucket naming.

```
host_base = localhost:9000
host_bucket =
access_key = abcdefghijkl
secret_key = abcdefghijklmnopqrstuvwxyz
access_token =
website_endpoint =
```

### MEMO on s3cmd

s3cmd "mv" seems not work in our test environment.  Uncertain, but, it
seems it needs some ACL definition returned from the server side.

## s4cmd

https://github.com/bloomreach/s4cmd

### Running a Test

Prepare `cli-conf.sh`, which specifies the bucket name and the
access-key.  For example,

```
BKT=lenticularis-oddity-x1
export AWS_ACCESS_KEY_ID=abcdefghijkl
export AWS_SECRET_ACCESS_KEY=abcdefghijklmnopqrstuvwxyz
```

s4cmd shares the configuration of AWS-CLI.

```
bash object-s4cmd.sh
```

### Installing s4cmd

s4cmd is in Python.

```
pip3 install --user s4cmd
```

Or, the client in Ubuntu can be installed by apt.

```
apt install s4cmd
```

## s5cmd

https://github.com/peak/s5cmd

### Running a Test

Prepare `cli-conf.sh`, which specifies the bucket name and the
access-key.  For example,

```
BKT=lenticularis-oddity-x1
export AWS_ACCESS_KEY_ID=abcdefghijkl
export AWS_SECRET_ACCESS_KEY=abcdefghijklmnopqrstuvwxyz
```

s5cmd shares the configuration of AWS-CLI.  However, s5cmd needs
"--endpoint-url" on the command line.  The test script will extract an
EP entry from "endpoint-url" in `~/.aws/config`.

```
bash object-s5cmd.sh
```

### Installing s5cmd

s5cmd binary can be downloaded from

https://github.com/peak/s5cmd/releases

## (WinSCP)

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
