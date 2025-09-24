#!/bin/bash

dd if=/dev/zero of=file_5mb.bin bs=1M count=5

dd if=/dev/zero of=file_8gb.bin bs=1G count=8

dd if=/dev/zero of=file_5gb.bin bs=1G count=5

dd if=/dev/zero of=file_5.1gb.bin bs=1M count=5224

ENDPOINT=http://localhost:7000
BUCKET=time-measurement-test-bucket

cp_5mb_total=0
cp_8gb_total=0
rm_5mb_total=0
rm_8gb_total=0
put_total=0
get_total=0
copy_total=0
put_tag_total=0
get_tag_total=0
delete_tag_total=0
count=10
tag_count=100

aws --endpoint-url $ENDPOINT s3 mb s3://$BUCKET

for i in $(seq 1 $count); do
    cp_5mb_start_time=$(date '+%s.%3N')
    aws --endpoint-url $ENDPOINT s3 cp "file_5mb.bin" s3://$BUCKET
    cp_5mb_end_time=$(date '+%s.%3N')
    cp_8gb_start_time=$(date '+%s.%3N')
    aws --endpoint-url $ENDPOINT s3 cp "file_8gb.bin" s3://$BUCKET
    cp_8gb_end_time=$(date '+%s.%3N')
    rm_5mb_start_time=$(date '+%s.%3N')
    aws --endpoint-url $ENDPOINT s3 rm s3://$BUCKET/"file_5mb.bin"
    rm_5mb_end_time=$(date '+%s.%3N')
    rm_8gb_start_time=$(date '+%s.%3N')
    aws --endpoint-url $ENDPOINT s3 rm s3://$BUCKET/"file_8gb.bin"
    rm_8gb_end_time=$(date '+%s.%3N')
    put_start_time=$(date '+%s.%3N')
    aws --endpoint-url $ENDPOINT s3api put-object --bucket $BUCKET --key object.txt --body "file_5mb.bin"
    put_end_time=$(date '+%s.%3N')
    get_start_time=$(date '+%s.%3N')
    aws --endpoint-url $ENDPOINT s3api get-object --bucket $BUCKET --key object.txt download.txt
    get_end_time=$(date '+%s.%3N')
    copy_start_time=$(date '+%s.%3N')
    aws --endpoint-url $ENDPOINT s3api copy-object --bucket $BUCKET --key copy.txt --copy-source $BUCKET/object.txt
    copy_end_time=$(date '+%s.%3N')

    cp_5mb_elapsed=$(perl -e "print $cp_5mb_end_time - $cp_5mb_start_time")
    cp_8gb_elapsed=$(perl -e "print $cp_8gb_end_time - $cp_8gb_start_time")
    rm_5mb_elapsed=$(perl -e "print $rm_5mb_end_time - $rm_5mb_start_time")
    rm_8gb_elapsed=$(perl -e "print $rm_8gb_end_time - $rm_8gb_start_time")
    put_elapsed=$(perl -e "print $put_end_time - $put_start_time")
    get_elapsed=$(perl -e "print $get_end_time - $get_start_time")
    copy_elapsed=$(perl -e "print $copy_end_time - $copy_start_time")

    echo "cp_5mb : $cp_5mb_elapsed 秒, cp_8gb : $cp_8gb_elapsed 秒, rm_5mb : $rm_5mb_elapsed 秒, rm_8gb : $rm_8gb_elapsed 秒,
        put : $put_elapsed 秒, get : $get_elapsed 秒 , copy : $copy_elapsed 秒"

    cp_5mb_total=$(perl -e "print $cp_5mb_total + $cp_5mb_elapsed")
    cp_8gb_total=$(perl -e "print $cp_8gb_total + $cp_8gb_elapsed")
    rm_5mb_total=$(perl -e "print $rm_5mb_total + $rm_5mb_elapsed")
    rm_8gb_total=$(perl -e "print $rm_8gb_total + $rm_8gb_elapsed")
    put_total=$(perl -e "print $put_total + $put_elapsed")
    get_total=$(perl -e "print $get_total + $get_elapsed")
    copy_total=$(perl -e "print $copy_total + $copy_elapsed")
done

aws --endpoint-url $ENDPOINT s3 rb s3://$BUCKET --force

echo "-----------------------------------------------------------------------------"
echo "[Test] cp 5MBのファイルパス"
avg=$(perl -e "print $cp_5mb_total / $count")
echo "Avg  : $avg 秒"
echo "-----------------------------------------------------------------------------"

echo "[Test] cp 8GBのファイルパス"
avg=$(perl -e "print $cp_8gb_total / $count")
echo "Avg  : $avg 秒"
echo "-----------------------------------------------------------------------------"

echo "[Test] rm 5MBのファイルパス"
avg=$(perl -e "print $rm_5mb_total / $count")
echo "Avg  : $avg 秒"
echo "-----------------------------------------------------------------------------"

echo "[Test] rm 8GBのファイルパス"
avg=$(perl -e "print $rm_8gb_total / $count")
echo "Avg  : $avg 秒"
echo "-----------------------------------------------------------------------------"

echo "[Test] put-object 5MBのファイルパス"
avg=$(perl -e "print $put_total / $count")
echo "Avg  : $avg 秒"
echo "-----------------------------------------------------------------------------"

echo "[Test] get-object 5MBのファイルパス"
avg=$(perl -e "print $get_total / $count")
echo "Avg  : $avg 秒"
echo "-----------------------------------------------------------------------------"

echo "[Test] copy-object 5MBのファイルパス"
avg=$(perl -e "print $copy_total / $count")
echo "Avg  : $avg 秒"
echo "-----------------------------------------------------------------------------"

file_num=100

aws --endpoint-url $ENDPOINT s3 mb s3://$BUCKET

for i in $(seq 1 $file_num); do
    aws --endpoint-url $ENDPOINT s3api put-object --bucket $BUCKET --key object_$i.txt
done

for i in $(seq 1 $count); do
    ls_start_time=$(date '+%s.%3N')
    aws --endpoint-url $ENDPOINT s3 ls s3://$BUCKET
    ls_end_time=$(date '+%s.%3N')

    ls_elapsed=$(perl -e "print $ls_end_time - $ls_start_time")

    echo "ls : $ls_elapsed 秒"

    ls_total=$(perl -e "print $ls_total + $ls_elapsed")
done

echo "[Test] ls"
avg=$(perl -e "print $ls_total / $count")
echo "Avg  : $avg 秒"
echo "-----------------------------------------------------------------------------"

for i in $(seq 1 $tag_count); do
    aws --endpoint-url $ENDPOINT s3api put-object --bucket $BUCKET --key object.txt --body "file_5mb.bin"

    put_tag_start_time=$(date '+%s.%3N')
    aws --endpoint-url $ENDPOINT s3api put-object-tagging --bucket $BUCKET --key object.txt \
    --tagging 'TagSet=[{Key=K1,Value=V1}, {Key=K2,Value=V2}, {Key=K3,Value=V3}, {Key=K4,Value=V4}, {Key=K5,Value=V5}, {Key=K6,Value=V6}, {Key=K7,Value=V7}, {Key=K8,Value=V8}, {Key=K9,Value=V9}, {Key=K10,Value=V10}]'
    put_tag_end_time=$(date '+%s.%3N')
    get_tag_start_time=$(date '+%s.%3N')
    aws --endpoint-url $ENDPOINT s3api get-object-tagging --bucket $BUCKET --key object.txt
    get_tag_end_time=$(date '+%s.%3N')
    delete_tag_start_time=$(date '+%s.%3N')
    aws --endpoint-url $ENDPOINT s3api delete-object-tagging --bucket $BUCKET --key object.txt
    delete_tag_end_time=$(date '+%s.%3N')

    put_tag_elapsed=$(perl -e "print $put_tag_end_time - $put_tag_start_time")
    get_tag_elapsed=$(perl -e "print $get_tag_end_time - $get_tag_start_time")
    delete_tag_elapsed=$(perl -e "print $delete_tag_end_time - $delete_tag_start_time")

    echo "put tag : $put_tag_elapsed 秒, get tag : $get_tag_elapsed 秒, delete tag : $delete_tag_elapsed"

    put_tag_total=$(perl -e "print $put_tag_total + $put_tag_elapsed")
    get_tag_total=$(perl -e "print $get_tag_total + $get_tag_elapsed")
    delete_tag_total=$(perl -e "print $delete_tag_total + $delete_tag_elapsed")
done

aws --endpoint-url $ENDPOINT s3 rb s3://$BUCKET --force

echo "-----------------------------------------------------------------------------"
echo "[Test] put-object-tagging"
avg=$(perl -e "print $put_tag_total / $tag_count")
echo "Avg  : $avg 秒"
echo "-----------------------------------------------------------------------------"

echo "[Test] get-object-tagging"
avg=$(perl -e "print $get_tag_total / $tag_count")
echo "Avg  : $avg 秒"
echo "-----------------------------------------------------------------------------"

echo "[Test] delete-object-tagging"
avg=$(perl -e "print $delete_tag_total / $tag_count")
echo "Avg  : $avg 秒"
echo "-----------------------------------------------------------------------------"
