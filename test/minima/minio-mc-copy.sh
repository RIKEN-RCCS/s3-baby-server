#!/bin/ksh

# This script runs test with MinIO's "mc" command.  It assumes an
# alias is set up as "s3baby".

# mc alias set s3baby "http://localhost:9000" "abcdefghijklmnopqrstuvwxyz" "abcdefghijklmnopqrstuvwxyz" --api S3v4

. ./cli-fn.sh

EXEC_ECHO mc ls s3baby
EXEC_ECHO mc mb s3baby/mybucket1

EXEC_ECHO mc ls s3baby/mybucket1
EXEC_ECHO mc cp data-01k.txt s3baby/mybucket1/object1.txt

cmp "zzz1" data-01k.txt

rm -rf "zzz1"

ECHO 'TEST DONE.'
