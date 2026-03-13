#!/bin/ksh

# This is a copy of "client-s3cmd.sh".

. ./cli-fn.sh

if [ -z ${EP} ] ; then
    EP=$(grep endpoint_url ~/.aws/config | sed -e 's/.*endpoint_url *= *//')
    if [ -z ${EP} ] ; then
	echo "It needs setting endpoint-url in shell variable EP"
    fi
fi

echo "Working on ENDPOINT = ${EP}"

flags="--endpoint-url ${EP} --no-verify-ssl"

CLILB="s5cmd ${flags} ls"
CLIMB="s5cmd ${flags} mb"
CLIRB="s5cmd ${flags} rb"

CLILS="s5cmd ${flags} ls"
CLIPUT="s5cmd ${flags} cp"
CLIGET="s5cmd ${flags} cp"
CLIRM="s5cmd ${flags} rm"
CLIMV="s5cmd ${flags} mv"

## EXEC_ECHO ${CLILB} s3://
EXEC_ECHO ${CLIMB} s3://mybucket1 || true

. ./copy-copy.sh

# ECHO "Test mv"

# EXEC_ECHO ${CLIPUT} data-01k.txt s3://mybucket1/object1.txt
# EXEC_ECHO ${CLIMV} s3://mybucket1/object1.txt s3://mybucket1/object2.txt
# EXEC_ECHO ${CLIGET} s3://mybucket1/object2.txt "zzz1"
# EXEC_ECHO cmp "zzz1" data-01k.txt
# EXEC_ECHO ${CLIRM} s3://mybucket1/data/object2.txt

EXEC_ECHO ${CLIRB} s3://mybucket1

ECHO_TEST_DONE
