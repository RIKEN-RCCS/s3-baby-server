#!/bin/ksh

# mc alias set ANON "http://localhost:9000" "abcdefghijklmnopqrstuvwxyz" "abcdefghijklmnopqrstuvwxyz"

. ./cli-fn.sh

EXEC_ECHO gcloud storage buckets create s3://mybucket1 || true

EXEC_ECHO gcloud storage cp data-01k.txt s3://mybucket1/object1.txt
EXEC_ECHO gcloud storage ls s3://mybucket1
EXEC_ECHO gcloud storage mv s3://mybucket1/object1.txt s3://mybucket1/object2.txt
EXEC_ECHO gcloud storage cp s3://mybucket1/object2.txt zzz1

cmp "zzz1" data-01k.txt

EXEC_ECHO gcloud storage rm s3://mybucket1/object2.txt
EXEC_ECHO gcloud storage buckets delete s3://mybucket1

rm -rf "zzz1"

ECHO 'TEST DONE.'
