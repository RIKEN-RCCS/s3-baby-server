#!/bin/ksh

# Simple tests with AWS CLI.

# Start with an empty pool.

set -e

alias ECHO=:

export AWS_EC2_METADATA_DISABLED=true
bucket=mybucket1

set -x

ECHO "Make a bucket for testing, assuming no buckets at start."

aws s3 ls --no-cli-pager s3://

set +e
aws s3 mb --no-cli-pager s3://mybucket1
set -e

ECHO "Copy files."

aws s3 cp --no-progress data-01k.txt s3://mybucket1/dog/akita.txt
aws s3 cp --no-progress data-01k.txt s3://mybucket1/dog/beagle.txt
aws s3 cp --no-progress data-01k.txt s3://mybucket1/dog/chihuahua.txt
aws s3 cp --no-progress data-01k.txt s3://mybucket1/dog/dachshund.txt
aws s3 cp --no-progress data-01k.txt s3://mybucket1/dog/entlebucher.txt
aws s3 cp --no-progress data-01k.txt s3://mybucket1/dog/eurasier.txt
aws s3 cp --no-progress data-01k.txt s3://mybucket1/dog/english/setter.txt
aws s3 cp --no-progress data-01k.txt s3://mybucket1/dog/english/terrier.txt

ECHO "List files."

aws s3 ls --no-cli-pager s3://mybucket1/dog | tee zzz

# OUTPUT:                            PRE dog/

cat zzz | tr '\n' '@' | grep -ae '^ *PRE dog/@' > /dev/null

aws s3 ls --no-cli-pager s3://mybucket1/dog/ | tee zzz

# OUTPUT:                            PRE english/
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 akita.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 beagle.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 chihuahua.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 dachshund.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 entlebucher.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 eurasier.txt

cat zzz | tr '\n' '@' | grep -ae '^ *PRE english/@.*akita\.txt@.*beagle\.txt@.*chihuahua\.txt@.*dachshund\.txt@.*entlebucher\.txt@.*eurasier\.txt@' > /dev/null

aws s3 ls --no-cli-pager s3://mybucket1/dog/e | tee zzz

# OUTPUT:                            PRE english/
# OUTPUT: 2025-12-02 23:32:06       1299 entlebucher.txt
# OUTPUT: 2025-12-02 23:32:07       1299 eurasier.txt

cat zzz | tr '\n' '@' | grep -ae '^ *PRE english/@.*entlebucher\.txt@.*eurasier\.txt@' > /dev/null

ECHO "Remove files."

#aws s3 rm s3://mybucket1/dog/akita.txt
#aws s3 rm s3://mybucket1/dog/beagle.txt
#aws s3 rm s3://mybucket1/dog/chihuahua.txt
#aws s3 rm s3://mybucket1/dog/dachshund.txt
#aws s3 rm s3://mybucket1/dog/entlebucher.txt
#aws s3 rm s3://mybucket1/dog/eurasier.txt
#aws s3 rm s3://mybucket1/dog/english/setter.txt
#aws s3 rm s3://mybucket1/dog/english/terrier.txt

aws s3 rm --recursive s3://mybucket1/dog/

# OUTPUT: delete: s3://mybucket1/dog/akita.txt
# OUTPUT: delete: s3://mybucket1/dog/beagle.txt
# OUTPUT: delete: s3://mybucket1/dog/chihuahua.txt
# OUTPUT: delete: s3://mybucket1/dog/english/setter.txt
# OUTPUT: delete: s3://mybucket1/dog/dachshund.txt
# OUTPUT: delete: s3://mybucket1/dog/english/terrier.txt
# OUTPUT: delete: s3://mybucket1/dog/entlebucher.txt
# OUTPUT: delete: s3://mybucket1/dog/eurasier.txt

aws s3 ls s3://mybucket1

ECHO "Done."
