ENDPOINT=http://localhost:7000
BUCKET=bucket0
KEY=test_object.txt
COPY_KEY=object.txt
MULTIPART_KEY=test_multipart_upload.txt
LOCAL_FILE=test.txt

# 初期化
echo "[Init] S3 API簡易自動テスト開始"
echo "====================================="
echo "[Info] ENDPOINT=$ENDPOINT"
echo "[Info] BUCKET=$BUCKET"
echo "====================================="

echo "[Test] list-buckets"
aws --endpoint-url $ENDPOINT s3api list-buckets

echo "[Test] create-bucket"
aws --endpoint-url $ENDPOINT s3api create-bucket --bucket $BUCKET --object-ownership BucketOwnerEnforced

echo "[Test] head-bucket"
aws --endpoint-url $ENDPOINT s3api head-bucket --bucket $BUCKET

echo "[Test] list-buckets"
aws --endpoint-url $ENDPOINT s3api list-buckets --max-buckets 2 --prefix bucket

echo "Hello" > $LOCAL_FILE
echo "[Test] put-object"
aws --endpoint-url $ENDPOINT s3api put-object --bucket $BUCKET --key $KEY --body $LOCAL_FILE \
  --tagging "testTag=testProject&testT=testP" --cache-control no-cache

echo "[Test] head-object"
aws --endpoint-url $ENDPOINT s3api head-object --bucket $BUCKET --key $KEY

echo "[Test] get-object"
aws --endpoint-url $ENDPOINT s3api get-object --bucket $BUCKET --key $KEY download.txt

echo "[Test] list-objects"
aws --endpoint-url $ENDPOINT s3api list-objects --bucket $BUCKET

echo "[Test] copy-object"
aws --endpoint-url $ENDPOINT s3api copy-object --bucket $BUCKET --key $COPY_KEY \
  --copy-source "$BUCKET/$KEY" --tagging-directive REPLACE \
  --tagging "testTag=testProject&Tag=testPJ"

echo "[Test] list-objects-v2"
aws --endpoint-url $ENDPOINT s3api list-objects-v2 --bucket $BUCKET --prefix "object" --max-keys 2

echo "[Test] put-object-tagging"
aws --endpoint-url $ENDPOINT s3api put-object-tagging --bucket $BUCKET --key $COPY_KEY \
  --tagging 'TagSet=[{Key=Environment,Value=Dev}, {Key=E,Value=D}]'

echo "[Test] get-object-tagging"
aws --endpoint-url $ENDPOINT s3api get-object-tagging --bucket $BUCKET --key $COPY_KEY

echo "[Test] delete-object-tagging"
aws --endpoint-url $ENDPOINT s3api delete-object-tagging --bucket $BUCKET --key $COPY_KEY

echo "[Test] get-object-attributes"
aws --endpoint-url $ENDPOINT s3api get-object-attributes --bucket $BUCKET --key $COPY_KEY \
  --object-attributes "ETag,Checksum,ObjectParts,StorageClass,ObjectSize"

echo "[Test] create-multipart-upload"
UPLOAD_ID=$(aws --endpoint-url $ENDPOINT s3api create-multipart-upload --bucket $BUCKET --key $MULTIPART_KEY \
  --tagging "testTag=testMultipartUploadProject&Tag=testMultipartUploadProjectPJ" \
  | jq -r '.UploadId')

echo "[Info] 5MB以上のテストファイルを作成"
dd if=/dev/zero of=$LOCAL_FILE bs=1M count=5

echo "[Test] upload-part"
aws --endpoint-url $ENDPOINT s3api upload-part --bucket $BUCKET --key $MULTIPART_KEY --part-number 1 \
  --body $LOCAL_FILE --upload-id $UPLOAD_ID

echo "[Test] upload-part-copy"
aws --endpoint-url $ENDPOINT s3api upload-part-copy --bucket $BUCKET --key $MULTIPART_KEY --part-number 2 \
  --copy-source "$BUCKET/$COPY_KEY" --upload-id $UPLOAD_ID

echo "[Test] list-parts"
aws --endpoint-url $ENDPOINT s3api list-parts --bucket $BUCKET --key $MULTIPART_KEY --upload-id $UPLOAD_ID

echo "[Test] list-multipart-uploads"
aws --endpoint-url $ENDPOINT s3api list-multipart-uploads --bucket $BUCKET

echo "[Test] complete-multipart-upload"
ETAG1=$(aws --endpoint-url $ENDPOINT s3api list-parts --bucket $BUCKET --key $MULTIPART_KEY --upload-id $UPLOAD_ID \
  | jq '.Parts[0].ETag')
ETAG2=$(aws --endpoint-url $ENDPOINT s3api list-parts --bucket $BUCKET --key $MULTIPART_KEY --upload-id $UPLOAD_ID \
  | jq '.Parts[1].ETag')
aws --endpoint-url $ENDPOINT s3api complete-multipart-upload --bucket $BUCKET --key $MULTIPART_KEY --upload-id $UPLOAD_ID \
  --multipart-upload "{\"Parts\":[{\"ETag\":$ETAG1,\"PartNumber\":1},{\"ETag\":$ETAG2,\"PartNumber\":2}]}"

echo "[Test] delete-object"
aws --endpoint-url $ENDPOINT s3api delete-object --bucket $BUCKET --key $KEY

echo "[Test] delete-objects"
aws --endpoint-url $ENDPOINT s3api delete-objects --bucket $BUCKET \
  --delete "{\"Objects\":[{\"Key\":\"$COPY_KEY\"},{\"Key\":\"$MULTIPART_KEY\"}],\"Quiet\":false}"

echo "[Test] delete-bucket"
aws --endpoint-url $ENDPOINT s3api delete-bucket --bucket $BUCKET

echo "====================================="
echo "[Done] S3 API簡易自動テスト完了"