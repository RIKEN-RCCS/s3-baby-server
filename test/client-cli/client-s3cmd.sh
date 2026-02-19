#!/bin/ksh

. ./cli-fn.sh

CLILS="s3cmd ls"
CLIPUT="s3cmd put"
CLIGET="s3cmd get"
CLIRM="s3cmd rm"

EXEC_ECHO s3cmd mb s3://mybucket1 || true

. ./copy-copy.sh

ECHO "Test mv"

EXEC_ECHO s3cmd cp data-01k.txt s3://mybucket1/object1.txt
EXEC_ECHO s3cmd mv s3://mybucket1/object1.txt s3://mybucket1/object2.txt
EXEC_ECHO s3cmd cp s3://mybucket1/object2.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-01k.txt

EXEC_ECHO s3cmd rm s3://mybucket1/data/object2.txt

ECHO_TEST_DONE
