#!/bin/ksh

# Setting "-e" makes exit on errors, and "-E" makes trap on ERR is
# inherited.  Setting "pipefail" makes exit status consider all
# commands, not the rightmost one.

trap 'echo "TEST FAIL."' ERR
set -eE
set -o pipefail

alias ECHO=echo
EXEC_ECHO() { (echo "$*" 1>&2) ; "$@" ; }

export AWS_EC2_METADATA_DISABLED=true
