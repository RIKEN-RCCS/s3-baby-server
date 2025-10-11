#!/bin/ksh -x

# Simple tests with AWS CLI (2025-09-25).

# It runs AWS CLI s3api commands and expects the commands succeeds.
# It sets shell "-e" to stop on an error.  Start with nothing in the
# pool.  No $BUCKET exists, in particular.  It is tested with AWS CLI
# v2.31.13.

# It uses a temporary file "zzz.json" and leaves it.

set -e

BUCKET=mybucket1
KEY1=test_object.txt
KEY2=object.txt
KEY3=test_multipart_upload.txt
TESTFILE=data-10m.txt
RESULT=zzz.json

echo "[Init] Test starts."

echo "====================================="
echo "[Info] BUCKET=$BUCKET"
echo "[Info] KEY1=$KEY1"
echo "[Info] KEY2=$KEY2"
echo "[Info] KEY3=$KEY3"
echo "[Info] TESTFILE=$TESTFILE"
echo "[Info] RESULT=$RESULT"
echo "====================================="

echo "[Info] Making a test file when not exists"

rm -f $RESULT

if [ ! -e $TESTFILE ]; then
    touch $TESTFILE
    shred -n 1 -s 10M $TESTFILE
fi

echo "[Test] list-buckets"

aws s3api list-buckets --no-cli-pager

echo "[Test] create-bucket"

aws s3api create-bucket --no-cli-pager --bucket $BUCKET --object-ownership BucketOwnerEnforced

echo "[Test] head-bucket"

aws s3api head-bucket --no-cli-pager --bucket $BUCKET

echo "[Test] list-buckets"

aws s3api list-buckets --no-cli-pager --max-buckets 2 --prefix bucket

echo "[Test] put-object"

aws s3api put-object --no-cli-pager --bucket $BUCKET --key $KEY1 --body $TESTFILE --tagging "testTag=testProject&testT=testP" --cache-control no-cache

echo "[Test] head-object"

aws s3api head-object --no-cli-pager --bucket $BUCKET --key $KEY1

echo "[Test] get-object"

aws s3api get-object --no-cli-pager --bucket $BUCKET --key $KEY1 download.txt

echo "[Test] list-objects"

aws s3api list-objects --no-cli-pager --bucket $BUCKET

echo "[Test] copy-object"

aws s3api copy-object --no-cli-pager --bucket $BUCKET --key $KEY2 --copy-source "$BUCKET/$KEY1" --tagging-directive REPLACE --tagging "testTag=testProject&Tag=testPJ"

echo "[Test] list-objects-v2"

aws s3api list-objects-v2 --no-cli-pager --bucket $BUCKET --prefix "object" --max-keys 2

echo "[Test] put-object-tagging"

aws s3api put-object-tagging --no-cli-pager --bucket $BUCKET --key $KEY2 --tagging 'TagSet=[{Key=Environment,Value=Dev}, {Key=E,Value=D}]'

echo "[Test] get-object-tagging"

aws s3api get-object-tagging --no-cli-pager --bucket $BUCKET --key $KEY2

echo "[Test] delete-object-tagging"

aws s3api delete-object-tagging --no-cli-pager --bucket $BUCKET --key $KEY2

echo "[Test] get-object-attributes"

aws s3api get-object-attributes --no-cli-pager --bucket $BUCKET --key $KEY2 --object-attributes ETag Checksum ObjectParts StorageClass ObjectSize

echo "[Test] create-multipart-upload"

#UPLOAD_ID=$(aws s3api create-multipart-upload --no-cli-pager --bucket $BUCKET --key $KEY3 --tagging "testTag=testMultipartUploadProject&Tag=testMultipartUploadProjectPJ" | jq -r '.UploadId')

aws s3api create-multipart-upload --no-cli-pager --bucket $BUCKET --key $KEY3 --tagging "testTag=testMultipartUploadProject&Tag=testMultipartUploadProjectPJ" | tee $RESULT
UPLOAD_ID=$(jq -r '.UploadId' < $RESULT)

echo "[Test] upload-part"

aws s3api upload-part --no-cli-pager --bucket $BUCKET --key $KEY3 --part-number 1 --body $TESTFILE --upload-id $UPLOAD_ID

echo "[Test] upload-part-copy"

aws s3api upload-part-copy --no-cli-pager --bucket $BUCKET --key $KEY3 --part-number 2 --copy-source "$BUCKET/$KEY2" --upload-id $UPLOAD_ID

echo "[Test] list-parts"

aws s3api list-parts --no-cli-pager --bucket $BUCKET --key $KEY3 --upload-id $UPLOAD_ID

echo "[Test] list-multipart-uploads"

aws s3api list-multipart-uploads --no-cli-pager --bucket $BUCKET

echo "[Test] complete-multipart-upload"

#ETAG1=$(aws s3api list-parts --no-cli-pager --bucket $BUCKET --key $KEY3 --upload-id $UPLOAD_ID | jq '.Parts[0].ETag')
#ETAG2=$(aws s3api list-parts --no-cli-pager --bucket $BUCKET --key $KEY3 --upload-id $UPLOAD_ID | jq '.Parts[1].ETag')

aws s3api list-parts --no-cli-pager --bucket $BUCKET --key $KEY3 --upload-id $UPLOAD_ID | tee $RESULT
ETAG1=$(jq '.Parts[0].ETag' < $RESULT)
aws s3api list-parts --no-cli-pager --bucket $BUCKET --key $KEY3 --upload-id $UPLOAD_ID | tee $RESULT
ETAG2=$(jq '.Parts[1].ETag' < $RESULT)

aws s3api complete-multipart-upload --no-cli-pager --bucket $BUCKET --key $KEY3 --upload-id $UPLOAD_ID --multipart-upload "{\"Parts\":[{\"ETag\":$ETAG1,\"PartNumber\":1},{\"ETag\":$ETAG2,\"PartNumber\":2}]}"

echo "[Test] delete-object"

aws s3api delete-object --no-cli-pager --bucket $BUCKET --key $KEY1

echo "[Test] delete-objects"

aws s3api delete-objects --no-cli-pager --bucket $BUCKET --delete "{\"Objects\":[{\"Key\":\"$KEY2\"},{\"Key\":\"$KEY3\"}],\"Quiet\":false}"

echo "[Test] delete-bucket"

aws s3api delete-bucket --no-cli-pager --bucket $BUCKET

echo "====================================="
echo "[Done] Test done successfully."
