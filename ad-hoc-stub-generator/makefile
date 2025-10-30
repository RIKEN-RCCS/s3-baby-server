# makefile

DUMPS='(dump-dispatcher "dispatcher.go") (dump-handlers "handlers.go") (dump-marshalers "marshalers.go") (dump-template "api-template.go")'

stubs::
	guile3.0 --r7rs -l stub.scm -c $(DUMPS)

# Download "s3.json" definition file.  It is for Smithy-2.0.  It is
# aws-sdk-go-v2 v1.37.2.

download-s3.json::
	wget https://github.com/aws/aws-sdk-go-v2/blob/711cf64d8ec7c903a234c1b84f63e94014de8622/codegen/sdk-codegen/aws-models/s3.json?raw=true -O s3.json
