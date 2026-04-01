#!/bin/ksh

# s3cmd commands are: mb, rb, ls, la (buckets), put, get, del, (rm=del), mv.

. ./cli-fn.sh
. ./cli-conf.sh

#flags=--no-ssl

CLILB="s3cmd ${flags} la"
CLIMB="s3cmd ${flags} mb"
CLIRB="s3cmd ${flags} rb"

CLILS="s3cmd ${flags} ls"
CLIPUT="s3cmd ${flags} put"
CLIGET="s3cmd ${flags} get"
CLIRM="s3cmd ${flags} del"
CLIMV="s3cmd ${flags} mv"

## EXEC_ECHO ${CLILB} s3://
## EXEC_ECHO ${CLIMB} s3://${BKT} || true

. ./copy-copy.sh

# ECHO "Test mv"

# EXEC_ECHO ${CLIPUT} data-01k.txt s3://${BKT}/object1.txt
# EXEC_ECHO ${CLIMV} s3://${BKT}/object1.txt s3://${BKT}/object2.txt
# EXEC_ECHO ${CLIGET} s3://${BKT}/object2.txt "zzz1"
# EXEC_ECHO cmp "zzz1" data-01k.txt
# EXEC_ECHO ${CLIRM} s3://${BKT}/data/object2.txt

## EXEC_ECHO ${CLIRB} s3://${BKT}

ECHO_TEST_DONE
