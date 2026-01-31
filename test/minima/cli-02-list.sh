#!/bin/ksh

# Simple tests with AWS CLI.

# Start with an empty pool.

. ./cli-fn.sh

ECHO "*** Make a bucket for testing, assuming no buckets at start."

EXEC_ECHO aws s3 ls --no-verify-ssl --no-cli-pager s3://

EXEC_ECHO aws s3 mb --no-verify-ssl --no-cli-pager s3://mybucket1 || true

ECHO "*** Copy files."

EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/akita.txt
EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/beagle.txt
EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/chihuahua.txt
EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/dachshund.txt
EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/entlebucher.txt
EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/eurasier.txt
EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/english/setter.txt
EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt s3://mybucket1/dog/english/terrier.txt

ECHO "*** List files."

EXEC_ECHO aws s3 ls --no-verify-ssl --no-cli-pager s3://mybucket1/dog | tee "zzz"

# OUTPUT:                            PRE dog/

cat "zzz" | tr '\n' '@' | grep -ae '^ *PRE dog/@' > /dev/null

EXEC_ECHO aws s3 ls --no-verify-ssl --no-cli-pager s3://mybucket1/dog/ | tee "zzz"

# OUTPUT:                            PRE english/
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 akita.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 beagle.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 chihuahua.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 dachshund.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 entlebucher.txt
# OUTPUT: yyyy-mm-dd hh:mm:ss       1299 eurasier.txt

cat "zzz" | tr '\n' '@' | grep -ae '^ *PRE english/@.*akita\.txt@.*beagle\.txt@.*chihuahua\.txt@.*dachshund\.txt@.*entlebucher\.txt@.*eurasier\.txt@' > /dev/null

EXEC_ECHO aws s3 ls --no-verify-ssl --no-cli-pager s3://mybucket1/dog/e | tee "zzz"

# OUTPUT:                            PRE english/
# OUTPUT: 2025-12-02 23:32:06       1299 entlebucher.txt
# OUTPUT: 2025-12-02 23:32:07       1299 eurasier.txt

cat "zzz" | tr '\n' '@' | grep -ae '^ *PRE english/@.*entlebucher\.txt@.*eurasier\.txt@' > /dev/null

ECHO "*** Remove files."

# EXEC_ECHO aws s3 rm s3://mybucket1/dog/akita.txt
# EXEC_ECHO aws s3 rm s3://mybucket1/dog/beagle.txt
# EXEC_ECHO aws s3 rm s3://mybucket1/dog/chihuahua.txt
# EXEC_ECHO aws s3 rm s3://mybucket1/dog/dachshund.txt
# EXEC_ECHO aws s3 rm s3://mybucket1/dog/entlebucher.txt
# EXEC_ECHO aws s3 rm s3://mybucket1/dog/eurasier.txt
# EXEC_ECHO aws s3 rm s3://mybucket1/dog/english/setter.txt
# EXEC_ECHO aws s3 rm s3://mybucket1/dog/english/terrier.txt

EXEC_ECHO aws s3 rm --no-verify-ssl --no-cli-pager --recursive s3://mybucket1/dog/

# OUTPUT: delete: s3://mybucket1/dog/akita.txt
# OUTPUT: delete: s3://mybucket1/dog/beagle.txt
# OUTPUT: delete: s3://mybucket1/dog/chihuahua.txt
# OUTPUT: delete: s3://mybucket1/dog/english/setter.txt
# OUTPUT: delete: s3://mybucket1/dog/dachshund.txt
# OUTPUT: delete: s3://mybucket1/dog/english/terrier.txt
# OUTPUT: delete: s3://mybucket1/dog/entlebucher.txt
# OUTPUT: delete: s3://mybucket1/dog/eurasier.txt

EXEC_ECHO aws s3 ls --no-verify-ssl --no-cli-pager s3://mybucket1

ECHO "*** Unicode file names."

EXEC_ECHO aws s3 cp --no-verify-ssl --no-cli-pager --no-progress data-01k.txt 's3://mybucket1/ファイル.txt'

EXEC_ECHO aws s3 ls --no-verify-ssl --no-cli-pager s3://mybucket1

EXEC_ECHO aws s3 rm 's3://mybucket1/ファイル.txt'

EXEC_ECHO aws s3 rb --no-verify-ssl --no-cli-pager s3://mybucket1

ECHO "TEST DONE."
