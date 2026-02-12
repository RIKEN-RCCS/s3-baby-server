#!/bin/ksh

# This script runs test with MinIO's "mc" command.  It assumes an
# alias is set up as "s3baby".

# mc alias set s3baby "http://localhost:9000" "abcdefghijklmnopqrstuvwxyz" "abcdefghijklmnopqrstuvwxyz" --api S3v4

. ./cli-fn.sh

EXEC_ECHO mc ls --insecure --disable-pager s3baby
EXEC_ECHO mc mb --insecure s3baby/mybucket1

EXEC_ECHO mc ls --insecure --disable-pager s3baby/mybucket1
EXEC_ECHO mc cp --insecure data-01k.txt s3baby/mybucket1/object1.txt
EXEC_ECHO mc cp --insecure data-01k.txt s3baby/mybucket1/object2.txt

EXEC_ECHO mc cp --insecure s3baby/mybucket1/object1.txt zzz1

cmp "zzz1" data-01k.txt

EXEC_ECHO mc cp --insecure data-01k.txt s3baby/mybucket1/object2.txt


rm -rf "zzz1"

ECHO 'TEST DONE.'
