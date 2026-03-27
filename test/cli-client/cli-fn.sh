#!/bin/ksh

# Setting "-e" makes exit on errors.  Setting "pipefail" makes exit
# status consider all commands, not the rightmost one.  "-E" makes
# trap on ERR is inherited (it is bash only).

ECHO_FAIL() { echo "TEST FAIL."; }

trap 'ECHO_FAIL' ERR
set -e
set -o pipefail
# set -E

ECHO() { echo "$@" ; }

EXEC_ECHO() { trap 'ECHO_FAIL' ERR ; (echo ">> $@" 1>&2) ; "$@" ; }
EXEC_PASS() { trap 'ECHO_FAIL' ERR ; (echo "pass>> $@" 1>&2) ; "$@" ; CC=$? ; }
EXEC_FAIL() { (echo "fail>> $@" 1>&2) ; "$@" ; CC=$? ; }

EXPECT_PASS() { trap 'ECHO_FAIL' ERR ; EXEC_PASS "$@" ; }
EXPECT_FAIL() { trap 'ECHO_FAIL' ERR ; EXEC_FAIL "$@" || true ; [ $CC -ne 0 ] ; }

ECHO_TEST_DONE() { ECHO 'TEST DONE.' ; }

export AWS_EC2_METADATA_DISABLED=true
