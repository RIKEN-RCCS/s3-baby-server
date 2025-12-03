# makefile

all::
	@echo "Usage: make download-srfi-180"

data-files: data-01k.txt data-08k.txt data-04m.txt data-20m.txt

data-01k.txt:
	cp ../../LICENSE data-01k.txt

data-08k.txt:
	touch data-08k.txt
	shred -n 1 -s 8K data-08k.txt

data-04m.txt:
	touch data-04m.txt
	shred -n 1 -s 4M data-04m.txt

data-20m.txt:
	touch data-20m.txt
	shred -n 1 -s 20M data-20m.txt

# Download "srfi-180" (json library) which is not avaialble in
# Guile-3.0.10 yet.  We use the latest one commit="725372e" on
# 2023-11-19.  commit="725372ebe5dff447188656754cf84c7ba5155cc7".  See
# https://github.com/scheme-requests-for-implementation/srfi-180

download-srfi-180::
	wget https://raw.githubusercontent.com/scheme-requests-for-implementation/srfi-180/725372ebe5dff447188656754cf84c7ba5155cc7/srfi/180/body.scm -O srfi-180-body.scm
