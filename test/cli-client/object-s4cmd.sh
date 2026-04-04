#!/bin/ksh

# s4cmd commands: ls, put, get, cp, mv, sync, del, du.  No
# delete-bucket command is provided.

. ./cli-fn.sh
. ./cli-conf.sh

# It seems "--ignore-certificate" is not readly yet.
# FLG="--ignore-certificate"

CLIMB="s4cmd ${FLG} mb"
CLIRB="false"
CLILS="s4cmd ${FLG} ls"
CLIPUT="s4cmd ${FLG} put"
CLIGET="s4cmd ${FLG} get"
CLIRM="s4cmd ${FLG} del"
CLIMV="s4cmd ${FLG} mv"

## EXEC_ECHO ${CLIMB} s3://${BKT} || true

. ./copy-copy.sh

ECHO "Test mv"

EXEC_ECHO ${CLIPUT} data-01k.txt s3://${BKT}/object1.txt
EXEC_ECHO ${CLIMV} s3://${BKT}/object1.txt s3://${BKT}/object2.txt
EXEC_ECHO ${CLIGET} s3://${BKT}/object2.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-01k.txt

EXEC_ECHO ${CLIRM} s3://${BKT}/object2.txt

ECHO "Test leaves a bucket ${BKT}."

ECHO_TEST_DONE
