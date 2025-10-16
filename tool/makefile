# makefile

# Download "s3.json" definition file.  It is for Smithy-2.0.

download-golang-s3.json::
	wget https://github.com/aws/aws-sdk-go-v2/blob/711cf64d8ec7c903a234c1b84f63e94014de8622/codegen/sdk-codegen/aws-models/s3.json?raw=true -O s3.json

# There are many "s3.json" files but they are virtually identical.
# For instance, other "s3.json" files can be found in Smithy-rust and
# Smithy-java.
# https://github.com/smithy-lang/smithy-rs/blob/main/aws/sdk/aws-models/s3.json
# https://github.com/smithy-lang/smithy-java/blob/main/aws/client/aws-client-rulesengine/src/shared-resources/software/amazon/smithy/java/aws/client/rulesengine/s3.json
