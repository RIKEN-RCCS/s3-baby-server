#!/bin/ksh

. ./cli-fn.sh

EXEC_ECHO s3cmd mb s3://mybucket1 || true

CLI=s3cmd
CLIPUT="s3cmd put"
CLIGET="s3cmd get"
. ./client-copy.sh

ECHO 'TEST DONE.'
