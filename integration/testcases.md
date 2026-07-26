# Integration Test Cases

## Goals

Test the core functionalites of the `nad-service-controller` controller using
KinD cluster as the test bed.

* For `Service` resources with the correct annotation and configuration, the
  controller correctly synchronizes the corresponding `EndpointSlice`
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

* Confirm that the `KUBECONFIG` environment variable is set to a kubeconfig file
* Connect to the K8s cluster using the endpoint and credentials found in the
  kubeconfig file
* Deploy `multus`, `whereabouts` and `nad` to the cluster:
`make multus whereabouts nad`
* Wait till all the components are up and running
* Build the controller image using `HEAD` commit from `main` branch and deploy it
  to the K8s cluster: `make apply`
* Wait for the controller to be ready
* Deploy test resources in the `testdata` directory: `make testdata`
  * See [Test Resources](#test-resources) section for details.

## Test Resources

The following test resources are needed in the `default` namespace for the integration
tests to run:

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
