package endpointslice

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	discv1 "k8s.io/api/discovery/v1"

	"k8s.io/apimachinery/pkg/types"

	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	discv1ac "k8s.io/client-go/applyconfigurations/discovery/v1"
	metav1ac "k8s.io/client-go/applyconfigurations/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyConfig creates an EndpointSliceApplyConfiguration for the endpoint slice of the given service.
// the service becomes the owner of the endpoint slice, and the endpoint slice will have the same name as the service with a suffix of "-<random-string>".
// the slice contains endpoints that are backed by the given pods.
func ApplyConfig(svc *corev1.Service, pods []corev1.Pod, client client.Client) (*discv1ac.EndpointSliceApplyConfiguration, error) {
	endpoints := []*discv1ac.EndpointApplyConfiguration{}
	for _, pod := range pods {
		nodeName := pod.Spec.NodeName
		node := &corev1.Node{}
		if err := client.Get(context.Background(), types.NamespacedName{Name: nodeName}, node); err != nil {
			return nil, fmt.Errorf("failed to get Node %s for Pod %s: %w", nodeName, pod.GetName(), err)
		}

		var az string
		for key, value := range node.GetLabels() {
			if key == corev1.LabelTopologyZone {
				az = value
				break
			}
		}

		endpoints = append(endpoints, &discv1ac.EndpointApplyConfiguration{
			Addresses: []string{pod.Status.PodIP},
			NodeName:  new(pod.Spec.NodeName),
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
