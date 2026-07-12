GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

KUBECTL ?= kubectl
KO ?= ko

GITHUB_TOKEN ?=

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o nad-service-controller main.go

test:
	go test -cover ./...

lint:
	golangci-lint run ./...

.PHONY: testdata
testdata:
	$(KUBECTL) delete -f testdata
	$(KUBECTL) apply -f testdata

image:
	KO_DOCKER_REPO=ghcr.io/ihcsim/nad-service-controller \
	GITHUB_TOKEN="$(GITHUB_TOKEN)" \
	$(KO) build --bare ./
