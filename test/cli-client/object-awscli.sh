#!/bin/ksh

. ./cli-fn.sh
. ./cli-conf.sh

export PYTHONWARNINGS="ignore::InsecureRequestWarning"

CLIGET="aws s3 cp --no-verify-ssl --no-cli-pager --no-progress"
CLIPUT="aws s3 cp --no-verify-ssl --no-cli-pager --no-progress"
CLILS="aws s3 ls --no-verify-ssl --no-cli-pager"
CLIMV="aws s3 mv --no-verify-ssl --no-cli-pager"
CLIRM="aws s3 rm --no-verify-ssl --no-cli-pager"
CLIMB="aws s3 mb --no-verify-ssl --no-cli-pager"
CLIRB="aws s3 rb --no-verify-ssl --no-cli-pager"

. ./copy-copy.sh

ECHO "Test mv"

EXEC_ECHO ${CLIPUT} data-01k.txt s3://${BKT}/object1.txt
EXEC_ECHO ${CLIMV} s3://${BKT}/object1.txt s3://${BKT}/object2.txt
EXEC_ECHO ${CLIGET} s3://${BKT}/object2.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-01k.txt
rm -f "zzz1"

ECHO "Clean up"

EXEC_ECHO ${CLIRM} s3://${BKT}/object2.txt

rm -f "zzz1"

ECHO_TEST_DONE
