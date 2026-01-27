module server-control

go 1.25

replace github.com/riken-rccs/s3-baby-server/pkg/awss3aide => ../../pkg/awss3aide

require github.com/riken-rccs/s3-baby-server/pkg/awss3aide v0.0.0-00010101000000-000000000000

require (
	github.com/aws/aws-sdk-go-v2 v1.41.1 // indirect
	github.com/aws/smithy-go v1.24.0 // indirect
)
