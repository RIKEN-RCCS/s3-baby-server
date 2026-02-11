#!/bin/ksh

# Note command "copy" works on directories.  Use "copyto" for files.

# Configuration File: ~/.config/rclone/rclone.conf
#
# [s3bbs]
# type = s3
# provider = Other
# env_auth = false
# access_key_id = abcdefghijklmnopqrstuvwxyz
# secret_access_key = abcdefghijklmnopqrstuvwxyz
# endpoint = https://localhost:9000
# acl = private

. ./cli-fn.sh

# --no-check-certificate
# --ignore-checksum
# --s3-use-multipart-etag=false

ECHO "Make a bucket for testing, assuming no buckets exist at the start."

EXEC_ECHO rclone -v lsd s3bbs:

EXEC_ECHO rclone --no-check-certificate -v mkdir s3bbs:mybucket1
EXEC_ECHO rclone --no-check-certificate -v ls s3bbs:mybucket1

ECHO "*** Test uploading/downloading."

EXEC_ECHO rclone --no-check-certificate -v copyto data-01k.txt s3bbs:mybucket1/object1.txt

EXEC_ECHO rclone --no-check-certificate -v copyto s3bbs:mybucket1/object1.txt "zzz1"

cmp "zzz1" data-01k.txt

EXEC_ECHO rclone --no-check-certificate -v copyto data-20m.txt s3bbs:mybucket1/object2.txt

EXEC_ECHO rclone --no-check-certificate -v copyto s3bbs:mybucket1/object2.txt "zzz1"

cmp "zzz1" data-20m.txt

ECHO "*** Test copying directory contents."

rm -rf datafiles
mkdir datafiles
for i in `seq 1 4` ; do cp -p data-08k.txt datafiles/data0$i.txt ; done

EXEC_ECHO rclone --no-check-certificate -v copy datafiles s3bbs:mybucket1/data/

EXEC_ECHO rclone --no-check-certificate -v ls s3bbs:mybucket1/data/

EXEC_ECHO rclone --no-check-certificate -v copyto s3bbs:mybucket1/data/data01.txt "zzz1"

cmp "zzz1" data-08k.txt

ECHO "*** Clean up files."

EXEC_ECHO rclone --no-check-certificate -v delete s3bbs:mybucket1/object1.txt
EXEC_ECHO rclone --no-check-certificate -v delete s3bbs:mybucket1/object2.txt

EXEC_ECHO rclone --no-check-certificate -v delete s3bbs:mybucket1/data/
EXEC_ECHO rclone --no-check-certificate -v rmdir s3bbs:mybucket1

rm -rf datafiles
rm -rf "zzz1"

ECHO "TEST DONE."
