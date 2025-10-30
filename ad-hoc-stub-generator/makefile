# makefile

DUMPCOMMAND='(dump-dispatcher "dispatcher.go") (dump-handlers "handlers.go") (dump-marshalers "marshalers.go") (dump-template "api-template.go")'

all::
	@echo "Usage: make stubs"
	@echo "Usage: make download-s3.json"
	@echo "Usage: make download-srfi-180"

stubs::
	guile3.0 --r7rs -l stub.scm -c $(DUMPCOMMAND)

# Download "s3.json" definition file.  It is for Smithy-2.0.  It is
# aws-sdk-go-v2 v1.37.2.

download-s3.json::
	wget https://github.com/aws/aws-sdk-go-v2/blob/711cf64d8ec7c903a234c1b84f63e94014de8622/codegen/sdk-codegen/aws-models/s3.json?raw=true -O s3.json

# Download "srfi-180-body.scm", a json module.  It is not yet bundled
# in Guile-3.0.10.

download-srfi-180::
	wget https://raw.githubusercontent.com/scheme-requests-for-implementation/srfi-180/725372ebe5dff447188656754cf84c7ba5155cc7/srfi/180/body.scm -O srfi-180-body.scm
