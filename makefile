# makefile

all::
	go build

get::
	go get github.com/aws/aws-sdk-go-v2/aws
	go get github.com/aws/aws-sdk-go-v2/service/s3

check-linux-build::
	GOOS=linux GOARCH=386 go build

check-windows-build::
	GOOS=windows GOARCH=386 go build -o s3-baby-server.exe

check-unix-build::
	GOOS=freebsd GOARCH=386 go build

tidy::
	go mod tidy

lint::
	golangci-lint run -v

golangci-lint-install:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0
