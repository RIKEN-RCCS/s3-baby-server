#!/bin/ksh

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

# Note command "copy" works on directories.  Use "copyto" for files.

EXEC_ECHO rclone -v lsd s3bbs:

EXEC_ECHO rclone --no-check-certificate -v mkdir s3bbs:mybucket1
EXEC_ECHO rclone --no-check-certificate -v ls s3bbs:mybucket1
EXEC_ECHO rclone --no-check-certificate -v copyto data-20m.txt s3bbs:mybucket1/data

EXEC_ECHO rclone --no-check-certificate -v copy data-01g.txt s3bbs:mybucket1/data
