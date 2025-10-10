#!/bin/ksh

# This creates files needed to run the "s3cli" and "s3api" tests.

# (1) Make a file of 4MB, 10MB.

rm -f data-04m.txt
touch data-04m.txt
shred -n 1 -s 4M data-04m.txt

rm -f data-10m.txt
touch data-10m.txt
shred -n 1 -s 10M data-10m.txt

if [ ! -d files ]; then
    mkdir ./files
fi
cp data-10m.txt ./testfiles/data-001.txt
cp data-10m.txt ./testfiles/data-002.txt
cp data-10m.txt ./testfiles/data-003.txt
cp data-10m.txt ./testfiles/data-004.txt
