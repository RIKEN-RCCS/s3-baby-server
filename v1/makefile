## makefile

s3-baby-server::
	go build ./cmd/s3-baby-server

get::
	go get github.com/aws/aws-sdk-go-v2/aws
	go get github.com/aws/aws-sdk-go-v2/service/s3
	go get github.com/riken-rccs/s3-baby-server/pkg/awss3aide
	go get github.com/riken-rccs/s3-baby-server/pkg/httpaide

check-linux-build::
	GOOS=linux GOARCH=amd64 go build ./server

check-unix-build::
	GOOS=solaris GOARCH=amd64 go build ./server

check-bsd-build::
	GOOS=freebsd GOARCH=amd64 go build ./server

check-windows-build::
	GOOS=windows GOARCH=amd64 go build -o s3-baby-server.exe ./server

tidy::
	go mod tidy

lint::
	golangci-lint run -v

golangci-lint-install:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0
