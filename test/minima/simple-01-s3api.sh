#!/bin/ksh -x

# Simple tests with AWS CLI.

# It runs AWS CLI s3api commands and expects the commands succeeds.
# It sets shell "-e" to stop on an error.  Start with nothing in the
# pool.  No "mybucket1" in particular.  It is tested with AWS CLI
# v2.31.13.
#
# It uses a temporary file "zzz.json", "zzz.data1" and leaves them.

set -e

export AWS_EC2_METADATA_DISABLED=true

echo "====================================="
echo "Making a test file when not exists."

#rm -f zzz.json

set -x

echo "Call list-buckets"

aws s3api list-buckets --no-cli-pager

echo "Call create-bucket."

aws s3api create-bucket --no-cli-pager --bucket "mybucket1" --object-ownership "BAD-KEYWORD-FOR-OWNERSHIP" || true

aws s3api create-bucket --no-cli-pager --bucket "mybucket1" --object-ownership "BucketOwnerEnforced"

echo "Call head-bucket."

aws s3api head-bucket --no-cli-pager --bucket "bucket-that-should-not-exist" || true

aws s3api head-bucket --no-cli-pager --bucket "mybucket1"

echo "Call list-buckets."

aws s3api list-buckets --no-cli-pager --max-buckets 2 --prefix my

echo "Call put-object."

aws s3api put-object --no-cli-pager --bucket "mybucket1" --key "dataobject1.txt" --body data-04m.txt --tagging "mytag1=medium&mytag2=median" --cache-control no-cache

aws s3api put-object --no-cli-pager --bucket "mybucket1" --key "dataobject2.txt" --body data-20m.txt --tagging "mytag1=supreme&mytag2=remarkable" --cache-control no-cache

echo "Call head-object."

aws s3api head-object --no-cli-pager --bucket "mybucket1" --key "dataobject1.txt"

echo "Call get-object."

aws s3api get-object --no-cli-pager --bucket "mybucket1" --key "dataobject1.txt" "zzz.data1"

echo "Call list-objects."

aws s3api list-objects --no-cli-pager --bucket "mybucket1"

echo "Call copy-object."

aws s3api copy-object --no-cli-pager --bucket "mybucket1" --key "dataobject2.txt" --copy-source ""mybucket1"/"dataobject1.txt"" --tagging-directive REPLACE --tagging "testTag=testProject&Tag=testPJ"

echo "Call list-objects-v2."

aws s3api list-objects-v2 --no-cli-pager --bucket "mybucket1" --prefix "data" --max-keys 2

echo "Call put-object-tagging."

aws s3api put-object-tagging --no-cli-pager --bucket "mybucket1" --key "dataobject2.txt" --tagging 'TagSet=[{Key=mytag1,Value=myvalue1},{Key=mytag2,Value=myvalue2},{Key=mytag3,Value=myvalue3}]'

echo "Call get-object-tagging."

aws s3api get-object-tagging --no-cli-pager --bucket "mybucket1" --key "dataobject2.txt"

echo "Call delete-object-tagging."

aws s3api delete-object-tagging --no-cli-pager --bucket "mybucket1" --key "dataobject2.txt"

echo "Call get-object-attributes."

aws s3api get-object-attributes --no-cli-pager --bucket "mybucket1" --key "dataobject2.txt" --object-attributes ETag Checksum ObjectParts StorageClass ObjectSize

echo "Call create-multipart-upload."

#UPLOAD_ID=$(aws s3api create-multipart-upload --no-cli-pager --bucket "mybucket1" --key "dataobject3.txt" --tagging "testTag=testMultipartUploadProject&Tag=testMultipartUploadProjectPJ" | jq -r '.UploadId')

aws s3api create-multipart-upload --no-cli-pager --bucket "mybucket1" --key "dataobject3.txt" --tagging "testTag=testMultipartUploadProject&Tag=testMultipartUploadProjectPJ" | tee zzz.json
UPLOAD_ID=$(jq -r '.UploadId' < zzz.json)

echo "Call upload-part."

aws s3api upload-part --no-cli-pager --bucket "mybucket1" --key "dataobject3.txt" --part-number 1 --body data-20m.txt --upload-id $UPLOAD_ID

echo "Call upload-part-copy."

aws s3api upload-part-copy --no-cli-pager --bucket "mybucket1" --key "dataobject3.txt" --part-number 2 --copy-source ""mybucket1"/"dataobject2.txt"" --upload-id $UPLOAD_ID

echo "Call list-parts."

aws s3api list-parts --no-cli-pager --bucket "mybucket1" --key "dataobject3.txt" --upload-id $UPLOAD_ID

echo "Call list-multipart-uploads."

aws s3api list-multipart-uploads --no-cli-pager --bucket "mybucket1"

echo "Call complete-multipart-upload."

#ETAG1=$(aws s3api list-parts --no-cli-pager --bucket "mybucket1" --key "dataobject3.txt" --upload-id $UPLOAD_ID | jq '.Parts[0].ETag')
#ETAG2=$(aws s3api list-parts --no-cli-pager --bucket "mybucket1" --key "dataobject3.txt" --upload-id $UPLOAD_ID | jq '.Parts[1].ETag')

aws s3api list-parts --no-cli-pager --bucket "mybucket1" --key "dataobject3.txt" --upload-id $UPLOAD_ID | tee zzz.json
ETAG1=$(jq '.Parts[0].ETag' < zzz.json)
aws s3api list-parts --no-cli-pager --bucket "mybucket1" --key "dataobject3.txt" --upload-id $UPLOAD_ID | tee zzz.json
ETAG2=$(jq '.Parts[1].ETag' < zzz.json)

aws s3api complete-multipart-upload --no-cli-pager --bucket "mybucket1" --key "dataobject3.txt" --upload-id $UPLOAD_ID --multipart-upload "{\"Parts\":[{\"ETag\":$ETAG1,\"PartNumber\":1},{\"ETag\":$ETAG2,\"PartNumber\":2}]}"

echo "Call delete-object."

aws s3api delete-object --no-cli-pager --bucket "mybucket1" --key "dataobject1.txt"

echo "Call delete-objects."

aws s3api delete-objects --no-cli-pager --bucket "mybucket1" --delete "{\"Objects\":[{\"Key\":\"dataobject2.txt\"},{\"Key\":\"dataobject3.txt\"}],\"Quiet\":false}"

echo "Call delete-bucket."

aws s3api delete-bucket --no-cli-pager --bucket "mybucket1"

set +x

echo "====================================="
echo "[Done] Test done successfully."
