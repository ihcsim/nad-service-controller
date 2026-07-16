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

run:
	go run -ldflags="-s -w -X main.Version=$(shell git rev-parse --short HEAD)" main.go

.PHONY: testdata
testdata:
	$(KUBECTL) delete -f testdata
	$(KUBECTL) apply -f testdata

image-local:
	 ko build --local ./

image:
	KO_DOCKER_REPO=ghcr.io/ihcsim/nad-service-controller \
	GITHUB_TOKEN="$(GITHUB_TOKEN)" \
	$(KO) build --bare ./

apply:
	ko apply -f deploy.yaml

delete:
	ko delete -f deploy.yaml

release:
	ko resolve -f deploy.yaml > release.yaml
