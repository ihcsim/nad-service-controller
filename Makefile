GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

KUBECTL ?= kubectl

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o nad-service-controller main.go

test:
	go test ./...

lint:
	golangci-lint run ./...

.PHONY: testdata
testdata:
	$(KUBECTL) delete -f testdata
	$(KUBECTL) apply -f testdata
