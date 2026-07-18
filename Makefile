GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

KUBECTL ?= kubectl
KO ?= ko
KIND ?= kind
KIND_CLUSTER_NAME ?= isim-dev

GITHUB_TOKEN ?=

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o nad-service-controller main.go

test:
	go test -cover ./...

lint:
	golangci-lint run ./...

run:
	go run -ldflags="-s -w -X main.Version=$(shell git rev-parse --short HEAD)" main.go

.PHONY: kind
kind:
	$(KIND) create cluster --name $(KIND_CLUSTER_NAME) --config kind/kind.yaml
	$(KUBECTL) apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset-thick.yml
	$(KUBECTL) -n kube-system wait --for condition=Ready po -lapp=multus
	$(KUBECTL) apply -f kind/nad-macvlan.yaml

purge:
	$(KIND) delete cluster --name $(KIND_CLUSTER_NAME)

.PHONY: testdata
testdata:
	$(KUBECTL) delete --ignore-not-found -f testdata
	$(KUBECTL) apply -f testdata

image-local:
	 $(KO) build --local ./

image:
	KO_DOCKER_REPO=ghcr.io/ihcsim/nad-service-controller \
	GITHUB_TOKEN="$(GITHUB_TOKEN)" \
	$(KO) build --bare ./

apply:
	$(KO) apply -f deploy.yaml

delete:
	$(KO) delete -f deploy.yaml

release:
	$(KO) resolve -f deploy.yaml > release.yaml
