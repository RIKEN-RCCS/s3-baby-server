#!/bin/ksh

# This creates files needed to run the "s3cli" and "s3api" tests.

# (1) Make a file of 8KB, 4MB, 10MB.

rm -f data-08k.txt
touch data-08k.txt
shred -n 1 -s 8K data-08k.txt

rm -f data-04m.txt
touch data-04m.txt
shred -n 1 -s 4M data-04m.txt

rm -f data-10m.txt
touch data-10m.txt
shred -n 1 -s 10M data-10m.txt

if [ ! -d ./datafiles ]; then
    mkdir ./datafiles
fi
cp data-10m.txt ./datafiles/data-001.txt
cp data-10m.txt ./datafiles/data-002.txt
cp data-10m.txt ./datafiles/data-003.txt
cp data-10m.txt ./datafiles/data-004.txt
