## makefile

all::
	cd v1; make s3-baby-server

# Make a self-signed certificate pair for testing.  Or, use files that
# may be found in /etc/pki/tls/certs/ and /etc/pki/tls/private/.

crt::
	openssl genrsa -out crt.key 2048
	openssl ecparam -genkey -name secp384r1 -out crt.key
	openssl req -new -x509 -subj "/CN=s3bbs" -sha256 -key crt.key -out crt.crt -days 3650
