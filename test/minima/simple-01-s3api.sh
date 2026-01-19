#!/bin/ksh

# Simple tests with AWS CLI.

# It runs AWS CLI s3api commands and expects the commands succeeds.
# It sets shell "-e" to stop on an error.  Start with nothing in the
# pool.  No "mybucket1" in particular.  It is tested with AWS CLI
# v2.31.13.
#
# It uses a temporary file "zzz" and "zzz.data1" and leaves them.

# Setting "-e" makes exit on errors, and "-E" makes trap on ERR is
# inherited.  Setting "pipefail" makes exit status consider all
# commands, not the rightmost one.

trap 'echo "TEST FAIL."' ERR
set -eE
set -o pipefail

alias ECHO=echo
EXEC_ECHO() { (echo "$*" 1>&2) ; "$@" ; }

rm -f zzz

export AWS_EC2_METADATA_DISABLED=true

ECHO "*** Test list-buckets"

EXEC_ECHO aws s3api list-buckets --no-cli-pager | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"Buckets": \[\],@ *"Owner": null,@ *"Prefix": null@}@' > /dev/null

ECHO "*** Test create-bucket"

EXEC_ECHO aws s3api create-bucket --no-cli-pager --bucket "mybucket1" --object-ownership "BucketOwnerEnforced" | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"Location": "/mybucket1"@}@' > /dev/null

EXEC_ECHO aws s3api create-bucket --no-cli-pager --bucket "mybucket2" --create-bucket-configuration 'LocationConstraint=us-west-1,Location={Type=LocalZone,Name=string},Bucket={DataRedundancy=SingleLocalZone},Tags=[{Key=string,Value=string},{Key=string,Value=string}]' | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"Location": "/mybucket2"@}@' > /dev/null

ECHO 'aws s3api create-bucket --no-cli-pager --bucket "mybucket3" --object-ownership "BAD-OWNERSHIP-TO-ERR"'

aws s3api create-bucket --no-cli-pager --bucket "mybucket3" --object-ownership "BAD-OWNERSHIP-TO_ERR" || true

ECHO "*** Test head-bucket"

EXEC_ECHO aws s3api head-bucket --no-cli-pager --bucket "mybucket1"

EXEC_ECHO aws s3api head-bucket --no-cli-pager --bucket "bucket-that-does-not-exist" || true

ECHO "*** Test list-buckets"

EXEC_ECHO aws s3api list-buckets --no-cli-pager --max-buckets 7 | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"Buckets": \[@ *{@ *"Name": "mybucket1",@ *"CreationDate": "[-0-9T:+]*"@ *},@ *{@ *"Name": "mybucket2",@ *"CreationDate": "[-0-9T:+]*"@ *}@ *\]@}@' > /dev/null

EXEC_ECHO aws s3api list-buckets --no-cli-pager --max-buckets 7 --prefix "my"

EXEC_ECHO aws s3api list-buckets --no-cli-pager --max-buckets 7 --prefix "gomi"

ECHO "*** Test delete-bucket"

EXEC_ECHO aws s3api delete-bucket --no-cli-pager --bucket "mybucket2"

ECHO "*** Test put-object"

EXEC_ECHO aws s3api put-object --no-cli-pager --bucket "mybucket1" --key "object1.txt" --body data-01k.txt | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"ETag": "\\"[a-zA-Z0-9+/=]*\\"",@ *"ChecksumCRC64NVME": "Bhu12BI5T1s=",@ *"ChecksumType": "FULL_OBJECT",@ *"Size": 1299@}@' >/dev/null

EXEC_ECHO aws s3api put-object --no-cli-pager --bucket "mybucket1" --key "object2.txt" --body data-01k.txt --tagging "mykey1=myvalue1&mykey2=myvalue2"

ECHO "*** Test head-object"

EXEC_ECHO aws s3api head-object --no-cli-pager --bucket "mybucket1" --key "object1.txt"

ECHO "*** Test get-object"

EXEC_ECHO aws s3api get-object --no-cli-pager --bucket "mybucket1" --key "object1.txt" "zzz"

cmp data-01k.txt "zzz"

ECHO "*** Test list-objects"

EXEC_ECHO aws s3api list-objects --no-cli-pager --bucket "mybucket1"

ECHO "*** Test copy-object"

EXEC_ECHO aws s3api copy-object --no-cli-pager --bucket "mybucket1" --key "object3.txt" --copy-source "mybucket1/object1.txt" --tagging-directive REPLACE --tagging "mykey5=myvalue5&mykey6=myvalue6" | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"CopyObjectResult": {@ *"ETag": "\\"[a-zA-Z0-9+/=]*\\"",@ *"LastModified": "[-0-9T:+]*",@ *"ChecksumType": "FULL_OBJECT",@ *"ChecksumCRC64NVME": "[0-9a-zA-Z+/=]*"@ *}@}@' > /dev/null

ECHO "*** Test list-objects-v2"

EXEC_ECHO aws s3api list-objects-v2 --no-cli-pager --bucket "mybucket1" --prefix "obj" --max-keys 17

ECHO "*** Test put-object-tagging"

EXEC_ECHO aws s3api put-object-tagging --no-cli-pager --bucket "mybucket1" --key "object2.txt" --tagging 'TagSet=[{Key=mykey1,Value=myvalue1},{Key=mykey2,Value=myvalue2},{Key=mykey3,Value=myvalue3}]'

ECHO "*** Test get-object-tagging"

EXEC_ECHO aws s3api get-object-tagging --no-cli-pager --bucket "mybucket1" --key "object2.txt" | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"TagSet": \[@ *{@ *"Key": "mykey1",@ *"Value": "myvalue1"@ *},@ *{@ *"Key": "mykey2",@ *"Value": "myvalue2"@ *},@ *{@ *"Key": "mykey3",@ *"Value": "myvalue3"@ *}@ *\]@}@' > /dev/null

ECHO "*** Test delete-object-tagging"

EXEC_ECHO aws s3api delete-object-tagging --no-cli-pager --bucket "mybucket1" --key "object2.txt"

EXEC_ECHO aws s3api get-object-tagging --no-cli-pager --bucket "mybucket1" --key "object2.txt" | tee "zzz"

cat "zzz" | tr '\n' '@' | grep -ae '{@ *"TagSet": \[\]@}@' > /dev/null

ECHO "*** Test get-object-attributes"

EXEC_ECHO aws s3api get-object-attributes --no-cli-pager --bucket "mybucket1" --key "object2.txt" --object-attributes ETag Checksum ObjectParts StorageClass ObjectSize

ECHO "*** Test create-multipart-upload, upload-part, upload-part-copy"

EXEC_ECHO aws s3api create-multipart-upload --no-cli-pager --bucket "mybucket1" --key "object4.txt" --tagging "mykey41=myvalue41&mykey42=myvalue42" | tee "zzz"

UPLOAD_ID=$(jq -r '.UploadId' < "zzz")

EXEC_ECHO aws s3api upload-part --no-cli-pager --bucket "mybucket1" --key "object4.txt" --part-number 1 --body data-08k.txt --upload-id $UPLOAD_ID

EXEC_ECHO aws s3api upload-part-copy --no-cli-pager --bucket "mybucket1" --key "object4.txt" --part-number 2 --copy-source "mybucket1"/"object2.txt" --upload-id $UPLOAD_ID

# Failing upload-part (using a bad key)...

EXEC_ECHO aws s3api upload-part --no-cli-pager --bucket "mybucket1" --key "bad-object.txt" --part-number 1 --body data-08k.txt --upload-id $UPLOAD_ID || true

ECHO "*** Test list-multipart-uploads"

EXEC_ECHO aws s3api create-multipart-upload --no-cli-pager --bucket "mybucket1" --key "object5.txt"

EXEC_ECHO aws s3api list-multipart-uploads --no-cli-pager --bucket "mybucket1"

ECHO "*** Test list-parts, complete-multipart-upload"

EXEC_ECHO aws s3api list-parts --no-cli-pager --bucket "mybucket1" --key "object4.txt" --upload-id $UPLOAD_ID | tee "zzz"

ETAG1=$(jq '.Parts[0].ETag' < "zzz")
ETAG2=$(jq '.Parts[1].ETag' < "zzz")

EXEC_ECHO aws s3api complete-multipart-upload --no-cli-pager --bucket "mybucket1" --key "object4.txt" --upload-id $UPLOAD_ID --multipart-upload "{\"Parts\":[{\"ETag\":$ETAG1,\"PartNumber\":1},{\"ETag\":$ETAG2,\"PartNumber\":2}]}"

ECHO "*** Test abort-multipart-upload"

EXEC_ECHO aws s3api create-multipart-upload --no-cli-pager --bucket "mybucket1" --key "object5.txt" | tee "zzz"

UPLOAD_ID=$(jq -r '.UploadId' < "zzz")

EXEC_ECHO aws s3api abort-multipart-upload --no-cli-pager --bucket "mybucket1" --key "object5.txt" --upload-id $UPLOAD_ID

ECHO "*** Test delete-object"

EXEC_ECHO aws s3api delete-object --no-cli-pager --bucket "mybucket1" --key "object1.txt"

ECHO "*** Test delete-objects"

EXEC_ECHO aws s3api delete-objects --no-cli-pager --bucket "mybucket1" --delete "{\"Objects\":[{\"Key\":\"object2.txt\"},{\"Key\":\"object3.txt\"},{\"Key\":\"object4.txt\"},{\"Key\":\"object5.txt\"}],\"Quiet\":false}"

ECHO "*** Test delete-bucket"

EXEC_ECHO aws s3api delete-bucket --no-cli-pager --bucket "mybucket1"

ECHO "TEST DONE."
