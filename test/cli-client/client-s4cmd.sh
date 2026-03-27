#!/bin/ksh

# s4cmd commands: ls, put, get, cp, mv, sync, del, du.  No
# delete-bucket command is provided.

. ./cli-fn.sh

CLIMB="s4cmd mb"
CLIRB="false"
CLILS="s4cmd ls"
CLIPUT="s4cmd put"
CLIGET="s4cmd get"
CLIRM="s4cmd del"
CLIMV="s4cmd mv"

EXEC_ECHO ${CLIMB} s3://mybucket1 || true

. ./copy-copy.sh

ECHO "Test mv"

EXEC_ECHO ${CLIPUT} data-01k.txt s3://mybucket1/object1.txt
EXEC_ECHO ${CLIMV} s3://mybucket1/object1.txt s3://mybucket1/object2.txt
EXEC_ECHO ${CLIGET} s3://mybucket1/object2.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-01k.txt

EXEC_ECHO ${CLIRM} s3://mybucket1/object2.txt

ECHO "Test leaves a bucket mybucket1."

ECHO_TEST_DONE
