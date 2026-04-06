#!/bin/ksh

# Listing buckets does not work with an error related to a word
# 'soft_deleted'.
#  - gcloud storage ls s3://
#  - gcloud storage buckets list s3://

# ~/.config/gcloud/configurations/config_default
#
# [auth]
# disable_ssl_validation = True
# [storage]
# s3_endpoint_url = https://localhost:9000

# ~/.boto
#
# [s3]
# use-sigv4=True
# [Credentials]
# s3_host = localhost
# s3_port = 9000
# aws_access_key_id = abcdefghijklmnopqrstuvwxyz
# aws_secret_access_key = abcdefghijklmnopqrstuvwxyz

. ./cli-fn.sh
. ./cli-conf.sh

## EXEC_ECHO gcloud storage buckets list s3://
## EXEC_ECHO gcloud storage buckets create s3://${BKT} || true

CLI="gcloud storage"
CLIGET="gcloud storage cp"
CLIPUT="gcloud storage cp"
CLILS="gcloud storage ls"
CLIRM="gcloud storage rm"

. ./copy-copy.sh

ECHO "Test mv"

EXEC_ECHO ${CLIPUT} data-01k.txt s3://${BKT}/object1.txt
EXEC_ECHO ${CLI} mv s3://${BKT}/object1.txt s3://${BKT}/object2.txt
EXEC_ECHO ${CLIGET} s3://${BKT}/object2.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-01k.txt
rm -f "zzz1"

ECHO "Test mv"

EXEC_ECHO gcloud storage cp data-01k.txt s3://${BKT}/object1.txt
EXEC_ECHO gcloud storage mv s3://${BKT}/object1.txt s3://${BKT}/object2.txt
EXEC_ECHO gcloud storage cp s3://${BKT}/object2.txt "zzz1"
EXEC_ECHO cmp "zzz1" data-01k.txt

ECHO "Clean up"

EXEC_ECHO gcloud storage rm s3://${BKT}/object2.txt
## EXEC_ECHO gcloud storage buckets delete s3://${BKT}

rm -f "zzz1"

ECHO_TEST_DONE
