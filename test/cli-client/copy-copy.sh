#!/bin/ksh

# Template of copying test.  Set variable CLI and call.

ECHO "Test cp"

rm -f "zzz1"

EXEC_ECHO ${CLIPUT} data-01k.txt s3://mybucket1/data/object1.txt
EXEC_ECHO ${CLILS} s3://mybucket1
EXEC_ECHO ${CLIGET} s3://mybucket1/data/object1.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-01k.txt
rm -f "zzz1"

EXEC_ECHO ${CLIPUT} data-08k.txt s3://mybucket1/data/object2.txt
EXEC_ECHO ${CLIGET} s3://mybucket1/data/object2.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-08k.txt
rm -f "zzz1"

EXEC_ECHO ${CLIPUT} data-04m.txt s3://mybucket1/data/object3.txt
EXEC_ECHO ${CLIGET} s3://mybucket1/data/object3.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-04m.txt
rm -f "zzz1"

EXEC_ECHO ${CLIPUT} data-20m.txt s3://mybucket1/data/object4.txt
EXEC_ECHO ${CLIGET} s3://mybucket1/data/object4.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-20m.txt
rm -f "zzz1"

EXEC_ECHO ${CLIPUT} data-01g.txt s3://mybucket1/data/object5.txt
EXEC_ECHO ${CLIGET} s3://mybucket1/data/object5.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-01g.txt
rm -f "zzz1"

ECHO "Clean up"

EXEC_ECHO ${CLIRM} s3://mybucket1/data/object1.txt
EXEC_ECHO ${CLIRM} s3://mybucket1/data/object2.txt
EXEC_ECHO ${CLIRM} s3://mybucket1/data/object3.txt
EXEC_ECHO ${CLIRM} s3://mybucket1/data/object4.txt
EXEC_ECHO ${CLIRM} s3://mybucket1/data/object5.txt
