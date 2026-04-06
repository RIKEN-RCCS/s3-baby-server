#!/bin/ksh

# This is a copy of "client-s3cmd.sh".

. ./cli-fn.sh
. ./cli-conf.sh

if [ -z ${EP} ] ; then
    EP=$(grep -v "^#" ~/.aws/config | grep "endpoint_url" | sed -e 's/.*endpoint_url *= *//')
    if [ -z ${EP} ] ; then
	echo "It needs setting endpoint-url in shell variable EP"
    fi
fi

echo "Working on ENDPOINT = ${EP}"

FLG="--endpoint-url ${EP} --no-verify-ssl"

CLILB="s5cmd ${FLG} ls"
CLIMB="s5cmd ${FLG} mb"
CLIRB="s5cmd ${FLG} rb"

CLILS="s5cmd ${FLG} ls"
CLIPUT="s5cmd ${FLG} cp"
CLIGET="s5cmd ${FLG} cp"
CLIRM="s5cmd ${FLG} rm"
CLIMV="s5cmd ${FLG} mv"

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
