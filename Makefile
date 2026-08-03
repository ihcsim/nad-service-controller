GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

KUBECTL ?= kubectl
HELM ?= helm
KO ?= ko
KIND ?= kind
KIND_CLUSTER_NAME ?= isim-dev

WHEREABOUTS_VERSION ?= 0.9.4

KO_DOCKER_REPO ?= ghcr.io/ihcsim/nad-service-controller
GITHUB_TOKEN ?=

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o nad-service-controller main.go

test:
	go test -v -cover ./...

lint:
	golangci-lint run ./...

run:
	go run -ldflags="-s -w -X main.Version=$(shell git rev-parse --short HEAD)" main.go

mod:
	go mod verify

.PHONY: kind
kind:
	$(KIND) create cluster --name $(KIND_CLUSTER_NAME) --config kind/kind.yaml

multus:
	$(KUBECTL) apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset-thick.yml
	$(KUBECTL) -n kube-system wait --for condition=Ready po -lapp=multus

nad:
	$(KUBECTL) apply -f kind/nad-macvlan.yaml

whereabouts:
	$(KUBECTL) apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/refs/heads/master/doc/crds/whereabouts.cni.cncf.io_ippools.yaml
	$(KUBECTL) apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/refs/heads/master/doc/crds/whereabouts.cni.cncf.io_nodeslicepools.yaml
	$(KUBECTL) apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/refs/heads/master/doc/crds/whereabouts.cni.cncf.io_overlappingrangeipreservations.yaml
	$(HELM) template whereabouts oci://ghcr.io/k8snetworkplumbingwg/whereabouts-chart --version $(WHEREABOUTS_VERSION) --set "nodeSelector.kubernetes\.io/os=linux" --set "nodeSelector.node-role\.kubernetes\.io/control-plane=" | $(KUBECTL) apply -f -
	$(KUBECTL) -n kube-system wait --timeout 180s --for condition=Ready po -lapp=whereabouts-chart

cluster: kind multus whereabouts nad

purge:
	$(KIND) delete cluster --name $(KIND_CLUSTER_NAME)

.PHONY: testdata
testdata:
	$(KUBECTL) delete --ignore-not-found -f testdata
	$(KUBECTL) apply -f testdata

image-local:
	 $(KO) build --local ./

image-release:
	KO_DOCKER_REPO="$(KO_DOCKER_REPO)" \
	GITHUB_TOKEN="$(GITHUB_TOKEN)" \
	$(KO) resolve --bare --sbom-dir=sbom -f deploy.yaml > release.yaml

apply-kind:
	KO_DOCKER_REPO=kind.local \
	KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) \
	$(KO) apply -f deploy.yaml

apply:
	KO_DOCKER_REPO="$(KO_DOCKER_REPO)" \
	$(KO) apply -f deploy.yaml

delete:
	$(KO) delete -f deploy.yaml

compile-aw:
	gh aw compile
