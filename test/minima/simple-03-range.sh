#!/bin/ksh

# Simple tests with AWS CLI.

# Start with an empty pool.

# Setting "-e" makes exit on errors, and "-E" makes trap on ERR is
# inherited.  Setting "pipefail" makes exit status consider all
# commands, not the rightmost one.

trap 'echo "TEST FAIL."' ERR
set -eE
set -o pipefail

alias ECHO=echo
EXEC_ECHO() { (echo "$*" 1>&2) ; "$@" ; }

export AWS_EC2_METADATA_DISABLED=true

ECHO "Make a bucket for testing, assuming no buckets at start."

EXEC_ECHO aws s3 ls --no-cli-pager s3://

set +e
aws s3 mb --no-cli-pager s3://mybucket1 || true
set -e

ECHO "Upload a file."

aws s3 cp --no-cli-pager --no-progress data-04m.txt s3://mybucket1/data-04m.txt

ECHO "Download a range of a file 1MB at 1MB offset."

EXEC_ECHO aws s3api get-object --no-cli-pager --bucket "mybucket1" --key "data-04m.txt" --range "bytes=1048576-2097151" "zzz1"

dd if="data-04m.txt" of="zzz2" bs=1M skip=1 count=1
cmp "zzz1" "zzz2"

EXEC_ECHO aws s3 rm --no-cli-pager s3://mybucket1/data-04m.txt
EXEC_ECHO aws s3 rb --no-cli-pager s3://mybucket1

ECHO "TEST DONE."
