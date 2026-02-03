#!/bin/ksh

# ~/.boto
# 
# [s3]
# use-sigv4=True
# [Credentials]
# s3_host = localhost
# s3_port = 9000
# aws_access_key_id = abcdefghijklmnopqrstuvwxyz
# aws_secret_access_key = abcdefghijklmnopqrstuvwxyz

# ~/.config/gcloud/configurations/config_default
#
# [auth]
# disable_ssl_validation = True
# [storage]
# s3_endpoint_url = https://localhost:9000

gcloud storage ls s3://mybucket1
gcloud storage cp data-01k.txt s3://mybucket1
gcloud storage cp s3://mybucket1/data-01k.txt zzz1

#gsutil ls s3://mybucket1
#gsutil rsync -d -r gs://my-gs-bucket s3://my-s3-bucket
