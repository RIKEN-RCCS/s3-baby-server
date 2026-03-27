# makefile

all::
	@echo "Usage: data-files clean-data"

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

# Make a self-signed certificate pair for testing.  Or, use files that
# may be found in /etc/pki/tls/certs/ and /etc/pki/tls/private/.

crt::
	openssl genrsa -out crt.key 2048
	openssl ecparam -genkey -name secp384r1 -out crt.key
	openssl req -new -x509 -subj "/CN=s3bbs" -sha256 -key crt.key -out crt.crt -days 3650
