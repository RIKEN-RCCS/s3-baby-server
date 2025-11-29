module s3-baby-server

go 1.24

toolchain go1.24.6

replace github.com/riken-rccs/s3-baby-server/pkg/awss3aide => ../pkg/awss3aide

replace github.com/riken-rccs/s3-baby-server/pkg/httpaide => ../pkg/httpaide

require (
	github.com/aws/aws-sdk-go-v2 v1.40.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.14 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.14 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.92.1 // indirect
	github.com/aws/smithy-go v1.23.2 // indirect
	github.com/riken-rccs/s3-baby-server/pkg/awss3aide v0.0.0-00010101000000-000000000000 // indirect
	github.com/riken-rccs/s3-baby-server/pkg/httpaide v0.0.0-00010101000000-000000000000 // indirect
)
