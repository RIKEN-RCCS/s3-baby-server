#!/bin/ksh

# Simple tests using AWS-CLI.

# Start with an empty pool.

. ./cli-fn.sh

ECHO "Make a bucket for testing, assuming no buckets exist at the start."

EXEC_ECHO aws s3 ls --no-verify-ssl --no-cli-pager s3://

EXEC_ECHO aws s3 mb --no-verify-ssl --no-cli-pager s3://mybucket1 || true

ECHO "*** Copy files."

EXPECT_PASS aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/akita.txt
EXPECT_PASS aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/beagle.txt
EXPECT_PASS aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/chihuahua.txt
EXPECT_PASS aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/dachshund.txt
EXPECT_PASS aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/entlebucher.txt
EXPECT_PASS aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/eurasier.txt
EXPECT_PASS aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/english/setter.txt
EXPECT_PASS aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/english/terrier.txt

ECHO "*** List files."

EXPECT_PASS aws s3 ls --no-verify-ssl --no-cli-pager s3://mybucket1/dog | tee "zzz"

# OUTPUT:                            PRE dog/

cat "zzz" | tr '\n' '@' | grep -ae '^ *PRE dog/@' > /dev/null

EXPECT_PASS aws s3 ls --no-verify-ssl --no-cli-pager s3://mybucket1/dog/ | tee "zzz"

# OUTPUT:                            PRE english/
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 akita.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 beagle.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 chihuahua.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 dachshund.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 entlebucher.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 eurasier.txt

cat "zzz" | tr '\n' '@' | grep -ae '^ *PRE english/@.*akita\.txt@.*beagle\.txt@.*chihuahua\.txt@.*dachshund\.txt@.*entlebucher\.txt@.*eurasier\.txt@' > /dev/null

EXPECT_PASS aws s3 ls --no-verify-ssl --no-cli-pager s3://mybucket1/dog/e | tee "zzz"

# OUTPUT:                            PRE english/
# OUTPUT: 2025-12-02 23:32:06       1299 entlebucher.txt
# OUTPUT: 2025-12-02 23:32:07       1299 eurasier.txt

cat "zzz" | tr '\n' '@' | grep -ae '^ *PRE english/@.*entlebucher\.txt@.*eurasier\.txt@' > /dev/null

ECHO "*** Check HEAD on a directory.  This should fail."

EXPECT_FAIL aws s3api head-object --no-verify-ssl --no-cli-pager --bucket "mybucket1" --key "dog"

ECHO "*** Clean up files."

# EXPECT_PASS aws s3 rm s3://mybucket1/dog/akita.txt
# EXPECT_PASS aws s3 rm s3://mybucket1/dog/beagle.txt
# EXPECT_PASS aws s3 rm s3://mybucket1/dog/chihuahua.txt
# EXPECT_PASS aws s3 rm s3://mybucket1/dog/dachshund.txt
# EXPECT_PASS aws s3 rm s3://mybucket1/dog/entlebucher.txt
# EXPECT_PASS aws s3 rm s3://mybucket1/dog/eurasier.txt
# EXPECT_PASS aws s3 rm s3://mybucket1/dog/english/setter.txt
# EXPECT_PASS aws s3 rm s3://mybucket1/dog/english/terrier.txt

EXPECT_PASS aws s3 rm --no-verify-ssl --no-cli-pager --recursive s3://mybucket1/dog/

# OUTPUT: delete: s3://mybucket1/dog/akita.txt
# OUTPUT: delete: s3://mybucket1/dog/beagle.txt
# OUTPUT: delete: s3://mybucket1/dog/chihuahua.txt
# OUTPUT: delete: s3://mybucket1/dog/english/setter.txt
# OUTPUT: delete: s3://mybucket1/dog/dachshund.txt
# OUTPUT: delete: s3://mybucket1/dog/english/terrier.txt
# OUTPUT: delete: s3://mybucket1/dog/entlebucher.txt
# OUTPUT: delete: s3://mybucket1/dog/eurasier.txt

EXPECT_PASS aws s3 ls --no-verify-ssl --no-cli-pager s3://mybucket1

ECHO "*** Unicode file names."

EXPECT_PASS aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt 's3://mybucket1/ファイル.txt'

EXPECT_PASS aws s3 ls --no-verify-ssl --no-cli-pager s3://mybucket1

EXPECT_PASS aws s3 rm --no-verify-ssl 's3://mybucket1/ファイル.txt'

ECHO "Clean up."

EXPECT_PASS aws s3 rb --no-verify-ssl --no-cli-pager s3://mybucket1

ECHO_TEST_DONE
