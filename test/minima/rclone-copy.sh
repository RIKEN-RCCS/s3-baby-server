#!/bin/ksh

# ~/.config/rclone/rclone.conf
#
# [s3bbs]
# type = s3
# provider = Other
# env_auth = true
# access_key_id = s3baby
# secret_access_key = s3baby
# endpoint = https://localhost:9000
# acl = private

. ./cli-fn.sh

EXEC_ECHO rclone --no-check-certificate -v lsd s3bbs:

# --s3-use-multipart-etag=false

EXEC_ECHO rclone --no-check-certificate -v mkdir s3bbs:mybucket1
EXEC_ECHO rclone --no-check-certificate --ignore-checksum -v copy data-20m.txt s3bbs:mybucket1/data

EXEC_ECHO rclone --no-check-certificate -v copy data-01g.txt s3bbs:mybucket1/data
