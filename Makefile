GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o nad-service-controller main.go

test:
	go test ./...

lint:
	golangci-lint run ./...
