# makefile

all::
	@echo "Usage: make bbs-ctl or data-files"

bbs-ctl::
	go build .

get::
	go get

data-files: data-01k.txt data-08k.txt data-04m.txt data-20m.txt data-01g.txt

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

data-01g.txt:
	touch data-01g.txt
	shred -n 1 -s 1G data-01g.txt

clean-data::
	rm -f data-01k.txt data-08k.txt data-04m.txt data-20m.txt data-01g.txt

# Download "srfi-180" (json library) which is not avaialble in
# Guile-3.0.10 yet.  We use the latest one commit="725372e" on
# 2023-11-19.  commit="725372ebe5dff447188656754cf84c7ba5155cc7".  See
# https://github.com/scheme-requests-for-implementation/srfi-180

download-srfi-180::
	wget https://raw.githubusercontent.com/scheme-requests-for-implementation/srfi-180/725372ebe5dff447188656754cf84c7ba5155cc7/srfi/180/body.scm -O srfi-180-body.scm

# Make self-signed certificate pairs for testing.  Or, use files that
# may be found in /etc/pki/tls/certs/ and /etc/pki/tls/private/.

crt::
	openssl genrsa -out crt.key 2048
	openssl ecparam -genkey -name secp384r1 -out crt.key
	openssl req -new -x509 -subj "/CN=s3bbs" -sha256 -key crt.key -out crt.crt -days 3650
