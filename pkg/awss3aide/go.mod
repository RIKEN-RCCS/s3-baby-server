module github.com/riken-rccs/s3-baby-server/pkg/awss3aide

go 1.25

require github.com/aws/aws-sdk-go-v2 v1.41.2

require github.com/aws/smithy-go v1.24.1 // indirect

replace github.com/riken-rccs/s3-baby-server/pkg/awss3aide => ./awss3aide
