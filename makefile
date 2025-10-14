# makefile

all::
	go build

tidy::
	go mod tidy

lint::
	golangci-lint run -v

golangci-lint-install:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0
