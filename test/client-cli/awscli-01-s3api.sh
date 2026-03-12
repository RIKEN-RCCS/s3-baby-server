#!/bin/ksh

# Low level tests with AWS-CLI s3api commands.

# It runs AWS-CLI s3api commands and expects the commands succeeds.
# It stops on an error as it sets shell "-e".  Start with nothing in
# the pool.  No "mybucket1" in particular.  It is tested with AWS-CLI
# v2.31.13.

# Precondition: Start with an empty pool.
# Side-effects: Make temporary files "zzz*".

# It uses a temporary file "zzz" and leaves it.

# Note command "jq -R" is used to quote-escape a string.  It is needed
# in passing ETags.

. ./cli-fn.sh

rm -f zzz

ECHO ''
ECHO '*** Test list-buckets'

EXPECT_PASS aws s3api list-buckets --no-verify-ssl --no-cli-pager | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"Buckets": \[\],@ *"Owner": null,@ *"Prefix": null@}@' > /dev/null

ECHO ''
ECHO '*** Test create-bucket'

EXPECT_PASS aws s3api create-bucket --no-verify-ssl --no-cli-pager --bucket "mybucket1" --object-ownership "BucketOwnerEnforced" | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"Location": "/mybucket1"@}@' > /dev/null

EXPECT_PASS aws s3api create-bucket --no-verify-ssl --no-cli-pager --bucket "mybucket2" --create-bucket-configuration 'LocationConstraint=us-west-1,Location={Type=LocalZone,Name=string},Bucket={DataRedundancy=SingleLocalZone},Tags=[{Key=string,Value=string},{Key=string,Value=string}]' | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"Location": "/mybucket2"@}@' > /dev/null

# ECHO 'aws s3api create-bucket --no-verify-ssl --no-cli-pager --bucket "mybucket3" --object-ownership "BAD-OWNERSHIP-TO-ERR"'

EXPECT_FAIL aws s3api create-bucket --no-verify-ssl --no-cli-pager --bucket "mybucket3" --object-ownership "BAD-OWNERSHIP-TO-ERR"

ECHO ''
ECHO '*** Test head-bucket'

EXPECT_PASS aws s3api head-bucket --no-verify-ssl --no-cli-pager --bucket "mybucket1"

EXPECT_FAIL aws s3api head-bucket --no-verify-ssl --no-cli-pager --bucket "bucket-that-does-not-exist"

ECHO ''
ECHO '*** Test list-buckets'

EXPECT_PASS aws s3api list-buckets --no-verify-ssl --no-cli-pager --max-buckets 7 | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"Buckets": \[@ *{@ *"Name": "mybucket1",@ *"CreationDate": "[-0-9.T:+]*"@ *},@ *{@ *"Name": "mybucket2",@ *"CreationDate": "[-0-9.T:+]*"@ *}@ *\]@}@' > /dev/null

EXPECT_PASS aws s3api list-buckets --no-verify-ssl --no-cli-pager --max-buckets 7 --prefix "my"

EXPECT_PASS aws s3api list-buckets --no-verify-ssl --no-cli-pager --max-buckets 7 --prefix "gomi"

ECHO ''
ECHO '*** Test delete-bucket'

EXPECT_PASS aws s3api delete-bucket --no-verify-ssl --no-cli-pager --bucket "mybucket2"

ECHO ''
ECHO '*** Test put-object'

EXPECT_PASS aws s3api put-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object1.txt" --body data-01k.txt | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"ETag": "\\"[a-zA-Z0-9+/=]*\\"",@ *"ChecksumCRC64NVME": "Bhu12BI5T1s=",@ *"ChecksumType": "FULL_OBJECT",@ *"Size": 1299@}@' >/dev/null

EXPECT_PASS aws s3api put-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object2.txt" --body data-01k.txt --tagging "mykey1=myvalue1&mykey2=myvalue2"

ECHO ''
ECHO '*** Test head-object'

EXPECT_PASS aws s3api head-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object1.txt"

EXPECT_FAIL aws s3api head-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object-that-does-not-exist"

ECHO ''
ECHO '*** Test get-object'

EXPECT_PASS aws s3api get-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object1.txt" "zzz"

cmp data-01k.txt "zzz"

ECHO ''
ECHO '*** Test list-objects'

EXPECT_PASS aws s3api list-objects --no-verify-ssl --no-cli-pager --bucket "mybucket1"

ECHO ''
ECHO '*** Test copy-object'

EXPECT_PASS aws s3api copy-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object3.txt" --copy-source "mybucket1/object1.txt" --tagging-directive REPLACE --tagging "mykey5=myvalue5&mykey6=myvalue6" | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"CopyObjectResult": {@ *"ETag": "\\"[a-zA-Z0-9+/=]*\\"",@ *"LastModified": "[-0-9.T:+]*",@ *"ChecksumType": "FULL_OBJECT",@ *"ChecksumCRC64NVME": "[0-9a-zA-Z+/=]*"@ *}@}@' > /dev/null

ECHO ''
ECHO '*** Test list-objects-v2'

EXPECT_PASS aws s3api list-objects-v2 --no-verify-ssl --no-cli-pager --bucket "mybucket1" --prefix "obj" --max-keys 17

ECHO ''
ECHO '*** Test put-object-tagging'

EXPECT_PASS aws s3api put-object-tagging --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object2.txt" --tagging 'TagSet=[{Key=mykey1,Value=myvalue1},{Key=mykey2,Value=myvalue2},{Key=mykey3,Value=myvalue3}]'

ECHO ''
ECHO '*** Test get-object-tagging'

EXPECT_PASS aws s3api get-object-tagging --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object2.txt" | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"TagSet": \[@ *{@ *"Key": "mykey1",@ *"Value": "myvalue1"@ *},@ *{@ *"Key": "mykey2",@ *"Value": "myvalue2"@ *},@ *{@ *"Key": "mykey3",@ *"Value": "myvalue3"@ *}@ *\]@}@' > /dev/null

ECHO ''
ECHO '*** Test delete-object-tagging'

EXPECT_PASS aws s3api delete-object-tagging --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object2.txt"

EXPECT_PASS aws s3api get-object-tagging --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object2.txt" | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"TagSet": \[\]@}@' > /dev/null

ECHO ''
ECHO '*** Test get-object-attributes'

EXPECT_PASS aws s3api get-object-attributes --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object2.txt" --object-attributes ETag Checksum ObjectParts StorageClass ObjectSize

ECHO ''
ECHO '*** Test create-multipart-upload + upload-part + upload-part-copy'

EXPECT_PASS aws s3api create-multipart-upload --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object4.txt" --tagging "mykey41=myvalue41&mykey42=myvalue42" | tee "zzz"

UPLOADID=$(jq -r '.UploadId' < "zzz")

EXPECT_PASS aws s3api upload-part --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object4.txt" --upload-id $UPLOADID --part-number 1 --body data-08k.txt

EXPECT_PASS aws s3api upload-part-copy --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object4.txt" --upload-id $UPLOADID --part-number 2 --copy-source "mybucket1"/"object2.txt"

# Failing upload-part (using a bad key)...

EXPECT_FAIL aws s3api upload-part --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "bad-object.txt" --upload-id $UPLOADID --part-number 1 --body data-08k.txt

ECHO ''
ECHO '*** Test list-multipart-uploads'

EXPECT_PASS aws s3api create-multipart-upload --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object5.txt"

EXPECT_PASS aws s3api list-multipart-uploads --no-verify-ssl --no-cli-pager --bucket "mybucket1"

ECHO ''
ECHO '*** Test list-parts + complete-multipart-upload'

EXPECT_PASS aws s3api list-parts --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object4.txt" --upload-id $UPLOADID | tee "zzz"

ETAG1=$(jq -r '.Parts[0].ETag' < "zzz")
ETAG2=$(jq -r '.Parts[1].ETag' < "zzz")
QETAG1=$(echo $ETAG1 | jq -R '.')
QETAG2=$(echo $ETAG2 | jq -R '.')

EXPECT_PASS aws s3api complete-multipart-upload --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object4.txt" --upload-id $UPLOADID --multipart-upload "{\"Parts\":[{\"ETag\":$QETAG1,\"PartNumber\":1},{\"ETag\":$QETAG2,\"PartNumber\":2}]}"

ECHO ''
ECHO '*** Test abort-multipart-upload'

EXPECT_PASS aws s3api create-multipart-upload --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object5.txt" | tee "zzz"

UPLOADID=$(jq -r '.UploadId' < "zzz")

EXPECT_PASS aws s3api abort-multipart-upload --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object5.txt" --upload-id $UPLOADID

ECHO ''
ECHO '*** Test delete-object'

EXPECT_PASS aws s3api delete-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object1.txt"

ECHO ''
ECHO '*** Test delete-objects'

EXPECT_PASS aws s3api delete-objects --no-verify-ssl --no-cli-pager --bucket "mybucket1" --delete "{\"Objects\":[{\"Key\":\"object2.txt\"},{\"Key\":\"object3.txt\"},{\"Key\":\"object4.txt\"},{\"Key\":\"object5.txt\"}],\"Quiet\":false}"

ECHO ''
ECHO '*** Test Content-MD5 verification is working'

## Note bad MD5 sum "aLMp2piT40CZx9itXLnJQA==" is for the empty file.

MD5SUM=$(cat data-01k.txt | openssl dgst -md5 -binary | openssl enc -base64)

EXPECT_PASS aws s3api put-object --no-verify-ssl --no-cli-pager --content-md5 "$MD5SUM" --bucket "mybucket1" --key "object6.txt" --body data-01k.txt

EXPECT_FAIL aws s3api put-object --no-verify-ssl --no-cli-pager --content-md5 "aLMp2piT40CZx9itXLnJQA==" --bucket "mybucket1" --key "object6.txt" --body data-01k.txt

EXPECT_FAIL aws s3api put-object --no-verify-ssl --no-cli-pager --checksum-crc64-nvme "aLMp2piT40CZx9itXLnJQA==" --bucket "mybucket1" --key "object6.txt" --body data-01k.txt

ECHO ''
ECHO '*** Test copying to the same file'

EXPECT_PASS aws s3api copy-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object6.txt" --copy-source "mybucket1/object6.txt"

## Delete "object6.txt".

EXPECT_PASS aws s3api delete-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "object6.txt"

ECHO ''
ECHO '*** Test delete-bucket'

EXPECT_PASS aws s3api delete-bucket --no-verify-ssl --no-cli-pager --bucket "mybucket1"

ECHO_TEST_DONE
