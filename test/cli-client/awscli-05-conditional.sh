#!/bin/ksh

# Simple tests with AWS CLI.  This is for conditionals.

# Conditionals can be specified in many actions:
# CompleteMultipartUpload, CopyObject, DeleteObject, DeleteObjects,
# GetObject, HeadObject, PutObject, UploadPart, and UploadPartCopy.

# Precondition: Start with an empty pool.
# Side-effects: Makes "mybucket1". Makes files "zzz*".

# [--if-match <value>]
# [--if-none-match <value>]
# [--if-modified-since <value>]
# [--if-unmodified-since <value>]

. ./cli-fn.sh

ECHO "Make a bucket for testing, assuming no buckets exist at the start."

EXEC_ECHO aws s3 ls --no-verify-ssl --no-cli-pager s3://

EXEC_ECHO aws s3 mb --no-verify-ssl --no-cli-pager s3://mybucket1 || true

EXPECT_PASS aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-04m.txt s3://mybucket1/object1.txt

## Test with "--if-match" and "--if-none-match".

EXPECT_PASS aws s3api list-objects --no-verify-ssl --no-cli-pager --bucket "mybucket1" | tee "zzz"

ETAG1=$(jq -r '.Contents[0].ETag' < "zzz")

ECHO "Download a file when conditionals match."

EXPECT_FAIL aws s3api get-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object1.txt" --if-match 'ILL-FORMED-ETAG' "zzz1" 2>&1 | tee "zzz"

grep 'InvalidArgument' "zzz" > /dev/null

EXPECT_FAIL aws s3api get-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object1.txt" --if-match '"BAD-ETAG"' "zzz1" 2>&1 | tee "zzz"

grep 'PreconditionFailed' "zzz" > /dev/null

EXPECT_FAIL aws s3api get-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object1.txt" --if-none-match $ETAG1 "zzz1" 2>&1 | tee "zzz"

grep '304' "zzz" > /dev/null

EXPECT_FAIL aws s3api get-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object1.txt" --if-none-match '*' "zzz1" 2>&1 | tee "zzz"

grep '304' "zzz" > /dev/null

# (finally, successful case).

EXPECT_PASS aws s3api get-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object1.txt" --if-match '"BAD-ETAG1"',$ETAG1,'"BAD-ETAG2"' "zzz1" | tee "zzz"

cmp data-04m.txt "zzz1"
rm "zzz1"

## Test with "--if-modified-since" and "--if-unmodified-since".

# date -R (for RFC-1123 or RFC-5322-date)
# date -d "+10 days" (for specified date)

DATEBF=$(date -R -d "-3 days")
DATEAF=$(date -R -d "+3 days")

EXPECT_FAIL aws s3api get-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object1.txt" --if-modified-since "$DATEAF" "zzz1" 2>&1 | tee "zzz"

grep '304' "zzz" > /dev/null

EXPECT_FAIL aws s3api get-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object1.txt" --if-unmodified-since "$DATEBF" "zzz1" 2>&1 | tee "zzz"

grep 'PreconditionFailed' "zzz" > /dev/null

# (finally, successful case).

EXPECT_PASS aws s3api get-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object1.txt" --if-modified-since "$DATEBF" --if-unmodified-since "$DATEAF" "zzz1" | tee "zzz"

cmp data-04m.txt "zzz1"
rm "zzz1"

ECHO "Clean up."

EXPECT_PASS aws s3 rm --no-verify-ssl --no-cli-pager s3://mybucket1/object1.txt
EXPECT_PASS aws s3 rb --no-verify-ssl --no-cli-pager s3://mybucket1

rm -f zzz zzz[123]

ECHO_TEST_DONE
