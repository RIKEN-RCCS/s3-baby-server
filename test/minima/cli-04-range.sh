#!/bin/ksh

# Simple tests with AWS CLI.  This is for file ranges.  A range can be
# specified in actions GetObject, HeadObject, and UploadPartCopy.

# Precondition: Start with an empty pool.
# Side-effects: Make temporary files "zzz*".

# Note command "jq -R" is used to quote-escape a string.  It is
# needed in passing ETags.

. ./cli-fn.sh

ECHO "Make a bucket for testing, assuming no buckets exists at the start."

EXEC_ECHO aws s3 ls --no-verify-ssl --no-cli-pager s3://

EXEC_ECHO aws s3 mb --no-verify-ssl --no-cli-pager s3://mybucket1 || true

EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-04m.txt s3://mybucket1/object1.txt

dd if="data-04m.txt" of="zzz1" bs=1M skip=1 count=1

ECHO "Download a range of a file 1MB length at 1MB offset."

EXEC_ECHO aws s3api get-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object1.txt" --range "bytes=1048576-2097151" "zzz2"

cmp "zzz1" "zzz2"

ECHO "Copy a range of a file 1MB at 1MB offset."

EXEC_ECHO aws s3api create-multipart-upload --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object2.txt" | tee "zzz"

UPLOADID=$(jq -r '.UploadId' < "zzz")

EXEC_ECHO aws s3api upload-part-copy --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object2.txt" --upload-id $UPLOADID --part-number 1 --copy-source "mybucket1"/"object1.txt" --copy-source-range "bytes=1048576-2097151" | tee "zzz"

ETAG1=$(jq -r '.CopyPartResult.ETag' < "zzz")
QETAG1=$(echo $ETAG1 | jq -R '.')

EXEC_ECHO aws s3api complete-multipart-upload --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object2.txt" --upload-id $UPLOADID --multipart-upload "{\"Parts\":[{\"ETag\":$QETAG1,\"PartNumber\":1}]}"

EXEC_ECHO aws s3api get-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object2.txt" "zzz3"

cmp "zzz1" "zzz3"

ECHO "Clean up."

EXEC_ECHO aws s3 rm --no-verify-ssl --no-cli-pager s3://mybucket1/object1.txt
EXEC_ECHO aws s3 rm --no-verify-ssl --no-cli-pager s3://mybucket1/object2.txt
EXEC_ECHO aws s3 rb --no-verify-ssl --no-cli-pager s3://mybucket1

rm -f zzz zzz[123]

ECHO "TEST DONE."
