#!/bin/ksh

# This script runs test with MinIO's "mc" command.  It assumes an
# alias is set up as "s3baby".
#
# mc alias set --api S3v4 --insecure "s3baby" "https://localhost:9000" "abcdefghijkl" "abcdefghijklmnopqrstuvwxyz"

. ./cli-fn.sh
. ./cli-conf.sh

ECHO "Test cp"

rm -f "zzz1"

## EXEC_ECHO mc ls --insecure --disable-pager s3baby
## EXEC_ECHO mc mb --insecure s3baby/mybucket1 || true

EXEC_ECHO mc cp --insecure data-01k.txt s3baby/${BKT}/data/object1.txt
EXEC_ECHO mc ls --insecure --disable-pager s3baby/${BKT}
EXEC_ECHO mc cp --insecure s3baby/${BKT}/data/object1.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-01k.txt

EXEC_ECHO mc cp --insecure data-08k.txt s3baby/${BKT}/data/object2.txt
EXEC_ECHO mc cp --insecure s3baby/${BKT}/data/object2.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-08k.txt

EXEC_ECHO mc cp --insecure data-04m.txt s3baby/${BKT}/data/object3.txt
EXEC_ECHO mc cp --insecure s3baby/${BKT}/data/object3.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-04m.txt

EXEC_ECHO mc cp --insecure data-20m.txt s3baby/${BKT}/data/object4.txt
EXEC_ECHO mc cp --insecure s3baby/${BKT}/data/object4.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-20m.txt

EXEC_ECHO mc cp --insecure data-01g.txt s3baby/${BKT}/data/object5.txt
EXEC_ECHO mc cp --insecure s3baby/${BKT}/data/object5.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-01g.txt

ECHO "Clean up"

EXEC_ECHO mc rm --insecure s3baby/${BKT}/data/object1.txt
EXEC_ECHO mc rm --insecure s3baby/${BKT}/data/object2.txt
EXEC_ECHO mc rm --insecure s3baby/${BKT}/data/object3.txt
EXEC_ECHO mc rm --insecure s3baby/${BKT}/data/object4.txt
EXEC_ECHO mc rm --insecure s3baby/${BKT}/data/object5.txt

ECHO "Test mv"

EXEC_ECHO mc cp --insecure data-01k.txt s3baby/${BKT}/object1.txt
EXEC_ECHO mc mv --insecure s3baby/${BKT}/object1.txt s3baby/${BKT}/object2.txt
EXEC_ECHO mc cp --insecure s3baby/${BKT}/object2.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-01k.txt

ECHO "Clean up"

EXEC_ECHO mc rm --insecure s3baby/${BKT}/object2.txt

## EXEC_ECHO mc rb --insecure s3baby/${BKT}

rm -rf "zzz1"

ECHO_TEST_DONE
