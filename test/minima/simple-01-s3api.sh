#!/bin/ksh -x

# Simple tests with AWS CLI.

# It runs AWS CLI s3api commands and expects the commands succeeds.
# It sets shell "-e" to stop on an error.  Start with nothing in the
# pool.  No "mybucket1" in particular.  It is tested with AWS CLI
# v2.31.13.
#
# It uses a temporary file "zzz" and "zzz.data1" and leaves them.

# Setting "pipefail" makes exit status consider all commands, not the
# rightmost one.

set -e
set -o pipefail

export AWS_EC2_METADATA_DISABLED=true

alias ECHO=:

#rm -f zzz

set -x

ECHO "*** Call list-buckets"

aws s3api list-buckets --no-cli-pager | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"Buckets": \[\],@ *"Owner": null,@ *"Prefix": null@}@' > /dev/null

ECHO "*** Call create-bucket."

aws s3api create-bucket --no-cli-pager --bucket "mybucket1" --object-ownership "BucketOwnerEnforced" | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"Location": "/mybucket1"@}@' > /dev/null

aws s3api create-bucket --no-cli-pager --bucket "mybucket2" --create-bucket-configuration 'LocationConstraint=us-west-1,Location={Type=LocalZone,Name=string},Bucket={DataRedundancy=SingleLocalZone},Tags=[{Key=string,Value=string},{Key=string,Value=string}]' | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"Location": "/mybucket2"@}@' > /dev/null

aws s3api create-bucket --no-cli-pager --bucket "mybucket3" --object-ownership "BAD-OWNERSHIP-TO_ERR" || true

ECHO "*** Call head-bucket."

aws s3api head-bucket --no-cli-pager --bucket "mybucket1"

aws s3api head-bucket --no-cli-pager --bucket "bucket-that-does-not-exist" || true

ECHO "*** Call list-buckets."

aws s3api list-buckets --no-cli-pager --max-buckets 7 | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"Buckets": \[@ *{@ *"Name": "mybucket1",@ *"CreationDate": "[-0-9T:+]*"@ *},@ *{@ *"Name": "mybucket2",@ *"CreationDate": "[-0-9T:+]*"@ *}@ *\]@}@' > /dev/null

aws s3api list-buckets --no-cli-pager --max-buckets 7 --prefix "my"

aws s3api list-buckets --no-cli-pager --max-buckets 7 --prefix "gomi"

ECHO "*** Call delete-bucket."

aws s3api delete-bucket --no-cli-pager --bucket "mybucket2"

ECHO "*** Call put-object."

aws s3api put-object --no-cli-pager --bucket "mybucket1" --key "object1.txt" --body data-01k.txt --cache-control no-cache | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"ETag": "\\"qkwTLse+ClningEv4pWrfw==\\"",@ *"ChecksumCRC64NVME": "Bhu12BI5T1s=",@ *"ChecksumType": "FULL_OBJECT"@}@'

aws s3api put-object --no-cli-pager --bucket "mybucket1" --key "object2.txt" --body data-01k.txt --tagging "mykey1=myvalue1&mykey2=myvalue2"

ECHO "*** Call head-object."

aws s3api head-object --no-cli-pager --bucket "mybucket1" --key "object1.txt"

echo "*** Call get-object."

aws s3api get-object --no-cli-pager --bucket "mybucket1" --key "object1.txt" "zzz.object1.txt"

cmp data-01k.txt "zzz.object1.txt"

ECHO "*** Call list-objects."

aws s3api list-objects --no-cli-pager --bucket "mybucket1"

ECHO "*** Call copy-object."

aws s3api copy-object --no-cli-pager --bucket "mybucket1" --key "object3.txt" --copy-source "mybucket1/object1.txt" --tagging-directive REPLACE --tagging "mykey5=myvalue5&mykey6=myvalue6" | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"CopyObjectResult": {@ *"ETag": "\\"qkwTLse+ClningEv4pWrfw==\\"",@ *"LastModified": "[-0-9T:+]*",@ *"ChecksumType": "FULL_OBJECT"@ *}@}@' > /dev/null

ECHO "*** Call list-objects-v2."

aws s3api list-objects-v2 --no-cli-pager --bucket "mybucket1" --prefix "obj" --max-keys 17

ECHO "*** Call put-object-tagging."

aws s3api put-object-tagging --no-cli-pager --bucket "mybucket1" --key "object2.txt" --tagging 'TagSet=[{Key=mykey1,Value=myvalue1},{Key=mykey2,Value=myvalue2},{Key=mykey3,Value=myvalue3}]'

ECHO "*** Call get-object-tagging."

aws s3api get-object-tagging --no-cli-pager --bucket "mybucket1" --key "object2.txt" | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"TagSet": \[@ *{@ *"Key": "mykey1",@ *"Value": "myvalue1"@ *},@ *{@ *"Key": "mykey2",@ *"Value": "myvalue2"@ *},@ *{@ *"Key": "mykey3",@ *"Value": "myvalue3"@ *}@ *\]@}@' > /dev/null

ECHO "*** Call delete-object-tagging."

aws s3api delete-object-tagging --no-cli-pager --bucket "mybucket1" --key "object2.txt"

aws s3api get-object-tagging --no-cli-pager --bucket "mybucket1" --key "object2.txt" | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"TagSet": \[\]@}@' > /dev/null

ECHO "*** Call get-object-attributes."

aws s3api get-object-attributes --no-cli-pager --bucket "mybucket1" --key "object2.txt" --object-attributes ETag Checksum ObjectParts StorageClass ObjectSize

ECHO "*** Call create-multipart-upload, upload-part, upload-part-copy."

aws s3api create-multipart-upload --no-cli-pager --bucket "mybucket1" --key "object4.txt" --tagging "mykey41=myvalue41&mykey42=myvalue42" | tee "zzz"
UPLOAD_ID=$(jq -r '.UploadId' < "zzz")

aws s3api upload-part --no-cli-pager --bucket "mybucket1" --key "object4.txt" --part-number 1 --body data-08k.txt --upload-id $UPLOAD_ID

aws s3api upload-part-copy --no-cli-pager --bucket "mybucket1" --key "object4.txt" --part-number 2 --copy-source "mybucket1"/"object2.txt" --upload-id $UPLOAD_ID

# Failing upload-part...

aws s3api upload-part --no-cli-pager --bucket "mybucket1" --key "object1.txt" --part-number 1 --body data-08k.txt --upload-id $UPLOAD_ID || true

echo "*** Call list-multipart-uploads."

aws s3api create-multipart-upload --no-cli-pager --bucket "mybucket1" --key "object5.txt"

aws s3api list-multipart-uploads --no-cli-pager --bucket "mybucket1"

echo "*** Call list-parts, complete-multipart-upload."

aws s3api list-parts --no-cli-pager --bucket "mybucket1" --key "object4.txt" --upload-id $UPLOAD_ID | tee "zzz"
ETAG1=$(jq '.Parts[0].ETag' < "zzz")
ETAG2=$(jq '.Parts[1].ETag' < "zzz")

aws s3api complete-multipart-upload --no-cli-pager --bucket "mybucket1" --key "object4.txt" --upload-id $UPLOAD_ID --multipart-upload "{\"Parts\":[{\"ETag\":$ETAG1,\"PartNumber\":1},{\"ETag\":$ETAG2,\"PartNumber\":2}]}"

ECHO "*** Call abort-multipart-upload."

aws s3api create-multipart-upload --no-cli-pager --bucket "mybucket1" --key "object5.txt" | tee "zzz"
UPLOAD_ID=$(jq -r '.UploadId' < "zzz")

aws s3api abort-multipart-upload --no-cli-pager --bucket "mybucket1" --key "object5.txt" --upload-id $UPLOAD_ID

ECHO "*** Call delete-object."

aws s3api delete-object --no-cli-pager --bucket "mybucket1" --key "object1.txt"

ECHO "*** Call delete-objects."

aws s3api delete-objects --no-cli-pager --bucket "mybucket1" --delete "{\"Objects\":[{\"Key\":\"object2.txt\"},{\"Key\":\"object4.txt\"}],\"Quiet\":false}"

ECHO "*** Call delete-bucket."

aws s3api delete-bucket --no-cli-pager --bucket "mybucket1"

set +x

echo "====================================="
echo "[Done] Test done successfully."
