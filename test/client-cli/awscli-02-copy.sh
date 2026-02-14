#!/bin/ksh

# Simple tests with AWS CLI.  This is for copying.

# Precondition: Start with an empty pool.
# Side-effects: Make temporary files "zzz*".

. ./cli-fn.sh

ECHO "Make a bucket for testing, assuming no buckets exist at the start."

EXEC_ECHO aws s3 ls --no-verify-ssl --no-cli-pager s3://
EXEC_ECHO aws s3 mb --no-verify-ssl --no-cli-pager s3://mybucket1 || true

ECHO "*** Test uploading/downloading."

EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/object1.txt
EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress s3://mybucket1/object1.txt "zzz1"
cmp "zzz1" data-01k.txt

EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-08k.txt s3://mybucket1/object2.txt
EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress s3://mybucket1/object2.txt "zzz1"
cmp "zzz1" data-08k.txt

EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-04m.txt s3://mybucket1/object3.txt
EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress s3://mybucket1/object3.txt "zzz1"
cmp "zzz1" data-04m.txt

EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-20m.txt s3://mybucket1/object4.txt
EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress s3://mybucket1/object4.txt "zzz1"
cmp "zzz1" data-20m.txt

EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01g.txt s3://mybucket1/object5.txt
EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress s3://mybucket1/object5.txt "zzz1"
cmp "zzz1" data-01g.txt

ECHO "Clean up."

EXEC_ECHO aws s3 rm --no-verify-ssl --no-cli-pager s3://mybucket1/object1.txt
EXEC_ECHO aws s3 rm --no-verify-ssl --no-cli-pager s3://mybucket1/object2.txt
EXEC_ECHO aws s3 rm --no-verify-ssl --no-cli-pager s3://mybucket1/object3.txt
EXEC_ECHO aws s3 rm --no-verify-ssl --no-cli-pager s3://mybucket1/object4.txt
EXEC_ECHO aws s3 rm --no-verify-ssl --no-cli-pager s3://mybucket1/object5.txt
EXEC_ECHO aws s3 rb --no-verify-ssl --no-cli-pager s3://mybucket1

rm -f zzz1

ECHO "TEST DONE."
