# Integration Test Cases

This document describes the integration test cases for `nad-service-controller`.
These tests verify end-to-end behavior of the controller in a live Kubernetes cluster.

## Prerequisites

The following tools must be available:

- `kubectl` — Kubernetes CLI
- `kind` — local Kubernetes cluster using Docker
- `ko` — build and deploy Go containers
- `helm` — Kubernetes package manager
- `make` — build automation

## Test Cases

### TC-001: Cluster Setup

**Description**: Verify that a KinD cluster can be created with all required components
(Multus CNI, Whereabouts IPAM, and the macvlan NetworkAttachmentDefinition).

**Steps**:
1. Run `make kind` to create a KinD cluster.
2. Run `make multus` and wait for Multus DaemonSet pods to be ready:
   `kubectl -n kube-system wait --for condition=Ready po -lapp=multus --timeout=120s`
3. Run `make whereabouts` and wait for Whereabouts pods to be ready:
   `kubectl -n kube-system wait --for condition=Ready po -lapp=whereabouts-chart --timeout=120s`
4. Run `make nad` to apply the macvlan NetworkAttachmentDefinition.

**Expected**:
- KinD cluster is running and accessible via `kubectl`
- Multus pods are `Ready` in the `kube-system` namespace
- Whereabouts pods are `Ready` in the `kube-system` namespace
- NetworkAttachmentDefinition `macvlan` exists in the default namespace:
  `kubectl get network-attachment-definitions macvlan`

---

### TC-002: Controller Deployment

**Description**: Verify that the `nad-service-controller` can be built with `ko` and
deployed to the cluster.

**Steps**:
1. Run `make apply-kind` to build the controller image and apply the deployment manifest.
2. Wait for the controller pods to be ready:
   `kubectl wait --for condition=Ready po -lapp.kubernetes.io/name=nad-service-controller --timeout=120s`

**Expected**:
- At least one controller pod is `Running` and `Ready` in the `default` namespace

---

### TC-003: EndpointSlice Created for Secondary Network Service

**Description**: Verify that the controller creates an `EndpointSlice` for a `Service`
annotated with `isim.dev/network: macvlan`, targeting the pods' secondary network IPs.

**Steps**:
1. Run `make testdata` to deploy the test workloads (`nginx-basic`, `nginx-nad`,
   `netutils-base`, `netutils-nad`).
2. Wait for the `nginx-nad` pods to be ready:
   `kubectl wait --for condition=Ready po -lapp=nginx-nad --timeout=120s`
3. Check that an `EndpointSlice` exists for the `nginx-nad` service:
   `kubectl get endpointslices -lkubernetes.io/service-name=nginx-nad`
4. Retrieve the EndpointSlice addresses:
   `kubectl get endpointslices -lkubernetes.io/service-name=nginx-nad -ojsonpath='{.items[*].endpoints[*].addresses}'`
5. Retrieve the secondary network IPs from pod annotations:
   `kubectl get po -lapp=nginx-nad -ojson | jq -cr '.items[].metadata.annotations["k8s.v1.cni.cncf.io/network-status"] | fromjson | .[1].ips'`

**Expected**:
- An `EndpointSlice` named `nginx-nad-slice` exists
- The `EndpointSlice` `ADDRESSTYPE` is `IPv4`
- The IP addresses in the `EndpointSlice` match the secondary network IPs from the
  `k8s.v1.cni.cncf.io/network-status` annotation on the `nginx-nad` pods

---

### TC-004: No EndpointSlice for Unannotated Service

**Description**: Verify that the controller does NOT create an `EndpointSlice` for a
`Service` that lacks the `isim.dev/network` annotation (`nginx-basic`).

**Steps**:
1. Check for EndpointSlices for the `nginx-basic` service:
   `kubectl get endpointslices -lkubernetes.io/service-name=nginx-basic`

**Expected**:
- No EndpointSlice with label `kubernetes.io/service-name=nginx-basic` is created by
  the controller (there may be a default Kubernetes-managed EndpointSlice, but its
  `addressType` should be based on the primary pod network only, and no custom slice
  from `nad-service-controller` should be present)

---

### TC-005: Network Connectivity from NAD-Attached Pod

**Description**: Verify that a pod with secondary network access (`netutils-nad`) can
reach both the primary network service (`nginx-basic`) and the secondary network
service (`nginx-nad`).

**Steps**:
1. Wait for `netutils-nad` pods to be ready:
   `kubectl wait --for condition=Ready po -lapp=netutils-nad --timeout=120s`
2. Get the name of a `netutils-nad` pod:
   `kubectl get po -lapp=netutils-nad -o jsonpath='{.items[0].metadata.name}'`
3. Send an HTTP request to `nginx-nad` from that pod:
   `kubectl exec <pod-name> -- curl -s -o /dev/null -w "%{http_code}" nginx-nad`
4. Send an HTTP request to `nginx-basic` from that pod:
   `kubectl exec <pod-name> -- curl -s -o /dev/null -w "%{http_code}" nginx-basic`

**Expected**:
- Both requests return HTTP status code `200`

---

### TC-006: Network Isolation for Base Pod

**Description**: Verify that a pod without secondary network access (`netutils-base`)
can reach the primary network service but cannot reach the secondary network service.

**Steps**:
1. Wait for `netutils-base` pods to be ready:
   `kubectl wait --for condition=Ready po -lapp=netutils-base --timeout=120s`
2. Get the name of a `netutils-base` pod:
   `kubectl get po -lapp=netutils-base -o jsonpath='{.items[0].metadata.name}'`
3. Send an HTTP request to `nginx-basic` from that pod:
   `kubectl exec <pod-name> -- curl -s -o /dev/null -w "%{http_code}" nginx-basic`
4. Attempt to connect to `nginx-nad` with a timeout:
   `kubectl exec <pod-name> -- curl -s -o /dev/null -w "%{http_code}" --connect-timeout 10 nginx-nad`

**Expected**:
- Request to `nginx-basic` returns HTTP status code `200`
- Request to `nginx-nad` times out with exit code `28` or HTTP code `000`
