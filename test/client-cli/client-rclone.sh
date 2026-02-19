#!/bin/ksh

# Note command "copy" works on directories.  Use "copyto" for files
# instead.

# The list of commands of RCLONE likely workable with AWS-S3: {copy,
# copyto, delete, -deletefile (a single file), ls, lsd (list buckets),
# -lsf, -lsl, mkdir, -move, moveto, rmdir, -rmdirs, -size, -sync,
# -tree}.

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

ECHO ''
ECHO 'Make a bucket for testing, assuming no buckets exist at the start'

EXEC_ECHO rclone -v lsd s3bbs:

EXEC_ECHO rclone --no-check-certificate -v mkdir s3bbs:mybucket1 || true
EXEC_ECHO rclone --no-check-certificate -v ls s3bbs:mybucket1

ECHO ''
ECHO "Test cp"

EXEC_ECHO rclone --no-check-certificate -v copyto data-01k.txt s3bbs:mybucket1/data/object1.txt
EXEC_ECHO rclone --no-check-certificate -v copyto s3bbs:mybucket1/data/object1.txt "zzz1"
cmp "zzz1" data-01k.txt

EXEC_ECHO rclone --no-check-certificate -v copyto data-08k.txt s3bbs:mybucket1/data/object2.txt
EXEC_ECHO rclone --no-check-certificate -v copyto s3bbs:mybucket1/data/object2.txt "zzz1"
cmp "zzz1" data-08k.txt

EXEC_ECHO rclone --no-check-certificate -v copyto data-04m.txt s3bbs:mybucket1/data/object3.txt
EXEC_ECHO rclone --no-check-certificate -v copyto s3bbs:mybucket1/data/object3.txt "zzz1"
cmp "zzz1" data-04m.txt

EXEC_ECHO rclone --no-check-certificate -v copyto data-20m.txt s3bbs:mybucket1/data/object4.txt
EXEC_ECHO rclone --no-check-certificate -v copyto s3bbs:mybucket1/data/object4.txt "zzz1"
cmp "zzz1" data-20m.txt

EXEC_ECHO rclone --no-check-certificate -v copyto data-01g.txt s3bbs:mybucket1/data/object5.txt
EXEC_ECHO rclone --no-check-certificate -v copyto s3bbs:mybucket1/data/object5.txt "zzz1"
cmp "zzz1" data-01g.txt

ECHO "Clean up"

EXEC_ECHO rclone --no-check-certificate -v delete s3bbs:mybucket1/data/object1.txt
EXEC_ECHO rclone --no-check-certificate -v delete s3bbs:mybucket1/data/object2.txt
EXEC_ECHO rclone --no-check-certificate -v delete s3bbs:mybucket1/data/object3.txt
EXEC_ECHO rclone --no-check-certificate -v delete s3bbs:mybucket1/data/object4.txt
EXEC_ECHO rclone --no-check-certificate -v delete s3bbs:mybucket1/data/object5.txt

ECHO ''
ECHO '*** Test uploading/downloading'

EXEC_ECHO rclone --no-check-certificate -v copyto data-01k.txt s3bbs:mybucket1/object1.txt
EXEC_ECHO rclone --no-check-certificate -v copyto s3bbs:mybucket1/object1.txt "zzz1"

cmp "zzz1" data-01k.txt

EXEC_ECHO rclone --no-check-certificate -v copyto data-20m.txt s3bbs:mybucket1/object2.txt
EXEC_ECHO rclone --no-check-certificate -v copyto s3bbs:mybucket1/object2.txt "zzz1"

cmp "zzz1" data-20m.txt

EXEC_ECHO rclone --no-check-certificate -v delete s3bbs:mybucket1/object1.txt
EXEC_ECHO rclone --no-check-certificate -v delete s3bbs:mybucket1/object2.txt

ECHO ''
ECHO '*** Test copying directory contents'

rm -rf datafiles
mkdir datafiles
for i in `seq 1 4` ; do cp -p data-08k.txt datafiles/data0$i.txt ; done

EXEC_ECHO rclone --no-check-certificate -v copy datafiles s3bbs:mybucket1/data/

EXEC_ECHO rclone --no-check-certificate -v ls s3bbs:mybucket1/data/

EXEC_ECHO rclone --no-check-certificate -v copyto s3bbs:mybucket1/data/data01.txt "zzz1"

cmp "zzz1" data-08k.txt

EXEC_ECHO rclone --no-check-certificate -v delete s3bbs:mybucket1/data/

ECHO ''
ECHO '*** Test uploading/downloading, again'

EXEC_ECHO rclone --no-check-certificate -v copyto data-01g.txt s3bbs:mybucket1/object1.txt

EXEC_ECHO rclone --no-check-certificate -v copyto s3bbs:mybucket1/object1.txt "zzz1"

cmp "zzz1" data-01g.txt

EXEC_ECHO rclone --no-check-certificate -v delete s3bbs:mybucket1/object1.txt

ECHO "*** Test move."

EXEC_ECHO rclone --no-check-certificate -v copyto data-01k.txt s3bbs:mybucket1/object1.txt

EXEC_ECHO rclone --no-check-certificate -v moveto s3bbs:mybucket1/object1.txt s3bbs:mybucket1/object2.txt

EXEC_ECHO rclone --no-check-certificate -v copyto s3bbs:mybucket1/object2.txt "zzz1"

cmp "zzz1" data-01k.txt

EXEC_ECHO rclone --no-check-certificate -v delete s3bbs:mybucket1/object2.txt

ECHO ''
ECHO '*** Clean up files'

EXEC_ECHO rclone --no-check-certificate -v rmdir s3bbs:mybucket1

rm -rf datafiles
rm -rf "zzz1"

ECHO_TEST_DONE
