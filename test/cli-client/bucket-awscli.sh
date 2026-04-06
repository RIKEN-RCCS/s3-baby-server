#!/bin/ksh

. ./cli-fn.sh
. ./cli-conf.sh

export AWS_EC2_METADATA_DISABLED=true
export PYTHONWARNINGS="ignore::InsecureRequestWarning"

CLIGET="aws s3 cp --no-verify-ssl --no-cli-pager --no-progress"
CLIPUT="aws s3 cp --no-verify-ssl --no-cli-pager --no-progress"
CLILS="aws s3 ls --no-verify-ssl --no-cli-pager"
CLIMV="aws s3 mv --no-verify-ssl --no-cli-pager"
CLIRM="aws s3 rm --no-verify-ssl --no-cli-pager"
CLIMB="aws s3 mb --no-verify-ssl --no-cli-pager"
CLIRB="aws s3 rb --no-verify-ssl --no-cli-pager"

EXEC_ECHO aws s3 ls --no-verify-ssl --no-cli-pager s3://

EXEC_ECHO ${CLIMB} s3://mybucket1

EXEC_ECHO ${CLIPUT} data-01k.txt s3://mybucket1/object1.txt
EXEC_ECHO ${CLIRM} s3://mybucket1/object1.txt

EXEC_ECHO ${CLIRB} s3://mybucket1

ECHO_TEST_DONE
