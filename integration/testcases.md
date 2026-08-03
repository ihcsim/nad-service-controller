# Integration Test Cases

## Goals

Test the core functionalites of the `nad-service-controller` controller using
a KinD cluster as the test bed.

* For `Service` resources with the correct annotation and configuration, the
  controller correctly synchronizes the corresponding `EndpointSlice` to point
  to the pods' secondary IP addresses from the `macvlan` secondary network
* For `Service` resources without the correct annotation or configuration, the
  controller does not create any `EndpointSlice`

## Required Tools

* kubectl
* make
* ko

Can't proceed if tools are missing.

## Setup

Hint: All the `make` commands are defined in the `Makefile` and can be run from
the root of the repository.

* Confirm that KinD has created a kubeconfig file for the test cluster:

```sh
export KIND_CLUSTER_NAME=<cluster-name>

kind get kubeconfig --name "${KIND_CLUSTER_NAME}" 
```

* Connect to the K8s cluster using the endpoint and credentials found in the
  kubeconfig file:

```sh
kubectl cluster-info
```

* Deploy `multus`, `whereabouts` and `nad` to the cluster and wait till all the
  components are up and running

```sh
make multus whereabouts nad
```

* Build the controller image using `HEAD` commit from `main` branch and deploy it
  to the K8s cluster. Wait for the controller to be ready:

```sh
make apply-kind
```

Proceed to the [Test Resources](#test-resources) section to create the test
resources in the cluster.

## Test Resources

Use the `make testdata` command to create the following test resources
in the `default` namespace:

Resource Type | Resource Name         | Namespace | Test file
------------- | --------------------- | --------- | -------------------------
Service       | nginx-basic           | default   | testdata/basic.yaml
Deployment    | nginx-basic           | default   | testdata/basic.yaml
Service       | nginx-nad             | default   | testdata/nad.yaml
Deployment    | nginx-nad             | default   | testdata/nad.yaml
Deployment    | netutils-base         | default   | testdata/netutils.yaml
Deployment    | netutils-nad          | default   | testdata/netutils.yaml
Service       | unsupported-clusterip | default   | testdata/unsupported.yaml
Service       | unsupported-selector  | default   | testdata/unsupported.yaml

Once the test resources are ready, proceed to run the test cases.

## Test Case 1 - Check RBAC

Expect these RBAC resources to be deployed when the controller is started:

Resource Type      | Resource Name          | Scope      | Namespace (if applicable)
 ----------------- | ---------------------- | ---------- | -------------------------
ServiceAccount     | nad-service-controller | Namespace  | default
ClusterRole        | nad-service-controller | Cluster    | N/A
ClusterRoleBinding | nad-service-controller | Cluster    | N/A

The `nad-service-controller` service account should have the following
permissions to resources in all namespaces:

Resource Type  | Permissions
 ------------- | -----------
pods           | get, list, watch
services       | get, list, watch
nodes          | get, list, watch
endpointslices | all
leases         | get, create, update

## Test Case 2 - Service with the correct annotation and configuration

* Namespace: `default`
* Required test resources:
  * Service: `nginx-nad`
  * Deployment: `nginx-nad`
* Expect controller to create an `EndpointSlice`:
  * Resource name prefix: `nginx-nad-`
  * Owner: Service `nginx-nad`
  * Endpoints: Contain the pods' secondary IP addresses from the `macvlan`
  secondary network
* Expect the `netutil-nad-*` pods can send HTTP requests to the `nginx-nad`
service DNS name
  * Expect the Nginx server to respond with HTTP 200 OK status code
* Expect the `netutil-base-*` pods to not be able to send HTTP requests to the
`nginx-nad` service because they are not connected to the `macvlan` network
  * Expect the connection attempt to either be refused or timeout

## Test Case 3 - Nginx Pods Restart

* Namespace: `default`
* Required test resources:
  * Service: `nginx-nad`
  * Deployment: `nginx-nad`
* Restart the `nginx-nad` Deployment
* Wait for the new pods to be ready
* Expect controller to update the `EndpointSlice` to:
  * contain the new pods' secondary IP addresses
  * remove the old pods' secondary IP addresses

## Test Case 4 - Service without the annotation

* Namespace: `default`
* Required test resources:
  * Service: `nginx-basic`
  * Deployment: `nginx-basic`
* Expect controller to not create any `EndpointSlice`
* Note K8s would still create the default `EndpointSlice` that contains the pods'
primary IP addresses

## Test Case 5 - Service with `spec.clusterIP != none`

* Namespace: `default`
* Required test resources:
  * Service: `unsupported-clusterip`
* Expect controller to not create any `EndpointSlice`

## Test Case 6 - Service with `spec.selector != nil`

* Namespace: `default`
* Required test resources:
  * Service: `unsupported-selector`
* Expect controller to not create any `EndpointSlice`

## Clean up

Once all the test cases finished their execution, delete the controller:
`make delete`
