package endpointslice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	discv1 "k8s.io/api/discovery/v1"

	"k8s.io/apimachinery/pkg/types"

	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	discv1ac "k8s.io/client-go/applyconfigurations/discovery/v1"
	metav1ac "k8s.io/client-go/applyconfigurations/meta/v1"

	networkv1 "github.com/ihcsim/nad-service-controller/pkg/apis/k8s.cni.cncf.io/network/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrMsgInvalidPodReadinessCondition = errors.New("pod is both serving and terminating, which is an invalid state")
	ErrMsgEmptyPodNetworkIPs           = errors.New("pod does not have an IP address for the specified network")
)

// ApplyConfig creates an EndpointSliceApplyConfiguration for the endpoint slice of the given service.
// the service becomes the owner of the endpoint slice, and the endpoint slice will have the same name as the service with a suffix of "-<random-string>".
// the slice contains endpoints that are backed by the given pods.
func ApplyConfig(svc *corev1.Service, pods []corev1.Pod, network string, client client.Client) (*discv1ac.EndpointSliceApplyConfiguration, error) {
	// one pod address per endpoint apply configuration
	endpoints := []*discv1ac.EndpointApplyConfiguration{}
	for _, pod := range pods {
		nodeName := pod.Spec.NodeName
		node := &corev1.Node{}
		if err := client.Get(context.Background(), types.NamespacedName{Name: nodeName}, node); err != nil {
			return nil, fmt.Errorf("failed to get Node %s for Pod %s: %w", nodeName, pod.GetName(), err)
		}
		az := podTopology(node)

		ready, serving, terminating := readiness(&pod, svc)
		if serving && terminating {
			return nil, fmt.Errorf("%w. pod: %s", ErrMsgInvalidPodReadinessCondition, pod.GetName())
		}

		ipAddrs, err := podNetworkIPs(&pod, network)
		if err != nil {
			return nil, err
		}
		if len(ipAddrs) == 0 {
			return nil, fmt.Errorf("%w. pod:%s network:%s", ErrMsgEmptyPodNetworkIPs, pod.GetName(), network)
		}

		endpoints = append(endpoints, &discv1ac.EndpointApplyConfiguration{
			Addresses: ipAddrs,
			Conditions: &discv1ac.EndpointConditionsApplyConfiguration{
				Ready:       new(ready),
				Serving:     new(serving),
				Terminating: new(terminating),
			},
			NodeName: new(pod.Spec.NodeName),
			TargetRef: &corev1ac.ObjectReferenceApplyConfiguration{
				Kind:      new("Pod"),
				Namespace: new(pod.GetNamespace()),
				Name:      new(pod.GetName()),
				UID:       new(pod.GetUID()),
			},
			Zone: new(az),
		})
	}

	ports := []*discv1ac.EndpointPortApplyConfiguration{}
	for _, port := range svc.Spec.Ports {
		ports = append(ports, &discv1ac.EndpointPortApplyConfiguration{
			Name:     new(port.Name),
			Protocol: new(port.Protocol),
			Port:     new(port.Port),
		})
	}

	ownerRef := &metav1ac.OwnerReferenceApplyConfiguration{
		Kind:       new("Service"),
		APIVersion: new(svc.APIVersion),
		Name:       new(svc.GetName()),
		UID:        new(svc.GetUID()),
	}

	name := fmt.Sprintf("%s-%s", svc.GetName(), "slice")
	config := discv1ac.EndpointSlice(name, svc.GetNamespace()).
		WithEndpoints(endpoints...).
		WithPorts(ports...).
		WithOwnerReferences(ownerRef).
		WithAddressType(discv1.AddressTypeIPv4).
		WithLabels(map[string]string{
			discv1.LabelServiceName: svc.GetName(),
		})

	return config, nil
}

func podTopology(node *corev1.Node) string {
	for key, value := range node.GetLabels() {
		if key == corev1.LabelTopologyZone {
			return value
		}
	}
	return ""
}

// see https://github.com/kubernetes/kubernetes/blob/688614f24c44fe55eb5368171f8b669b9a7928f6/staging/src/k8s.io/endpointslice/utils.go#L39-L43
func readiness(pod *corev1.Pod, svc *corev1.Service) (serving, terminating, ready bool) {
	podReady := false
	podStatus := pod.Status
	for i := range podStatus.Conditions {
		if podStatus.Conditions[i].Type == corev1.PodReady && podStatus.Conditions[i].Status == corev1.ConditionTrue {
			podReady = true
		}
	}
	serving = podReady
	terminating = pod.DeletionTimestamp != nil
	ready = svc.Spec.PublishNotReadyAddresses || (serving && !terminating)
	return
}

// find the network IP address by examining the pod's NetworkSelectionElement object
func podNetworkIPs(pod *corev1.Pod, network string) ([]string, error) {
	annotation, exists := pod.GetAnnotations()[networkv1.NetworkStatusAnnot]
	if !exists {
		return nil, fmt.Errorf("pod %s does not have the network annotation %s", pod.GetName(), networkv1.NetworkStatusAnnot)
	}

	var networkSelectionElements []networkv1.NetworkSelectionElement
	raw := []byte(annotation)
	if !json.Valid(raw) {
		return nil, fmt.Errorf("invalid JSON in pod %s annotation %s", pod.GetName(), networkv1.NetworkStatusAnnot)
	}

	if err := json.Unmarshal(raw, &networkSelectionElements); err != nil {
		return nil, fmt.Errorf("failed to unmarshal network selection elements from pod %s annotation %s: %w", pod.GetName(), networkv1.NetworkStatusAnnot, err)
	}

	var ipAddrs []string
	for _, element := range networkSelectionElements {
		if element.Name == network {
			copy(ipAddrs, element.IPRequest)
			break
		}
	}

	return ipAddrs, nil
}
