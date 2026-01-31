#!/bin/ksh

# Simple tests with AWS CLI.  This is for copying.

# Precondition: Start with an empty pool.
# Side-effects: Make temporary files "zzz*".

# Note command "jq -R" is used to quote-escape a string.  It is
# needed in passing ETags.

. ./cli-fn.sh

ECHO "Make a bucket for testing, assuming no buckets exists at the start."

EXEC_ECHO aws s3 ls --no-verify-ssl --no-cli-pager s3://

EXEC_ECHO aws s3 mb --no-verify-ssl --no-cli-pager s3://mybucket1 || true

EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-20m.txt s3://mybucket1/object1.txt

EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress s3://mybucket1/object1.txt "zzz1"

cmp "zzz1" data-20m.txt

ECHO "Clean up."

EXEC_ECHO aws s3 rm --no-cli-pager s3://mybucket1/object1.txt
EXEC_ECHO aws s3 rb --no-cli-pager s3://mybucket1

rm -f zzz zzz[123]

ECHO "TEST DONE."
