package indexer

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	networkv1 "github.com/ihcsim/nad-service-controller/pkg/apis/k8s.cni.cncf.io/network/v1"
)

var log = ctrllog.FromContext(context.Background()).WithName("indexer")

// ServiceByNetworkFunc is an index function that indexes services by their associated network.
// The network name is derived from the service's annotation "isim.dev/network".
// If the service does not have the annotation, it will not be indexed.
func ServiceByNetworkFunc(obj client.Object) []string {
	log := log.WithValues("service", fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName()))
	svc := obj.(*corev1.Service)
	network, exists := svc.GetAnnotations()[ServiceNetworkAnnotation]
	if !exists {
		return nil
	}

	// TODO make sure NAD exists and is valid, otherwise return nil to avoid indexing invalid network names

	indexValue := fmt.Sprintf("%s/%s", svc.GetNamespace(), network)
	log.Info("indexing service", "indexValues", indexValue)
	return []string{indexValue}
}

// PodByNetworkFunc is an index function that indexes pods by their associated
// networks. The network names are derived from the pod's annotation
// "k8s.v1.cni.cncf.io/networks-status", which contains a JSON array of
// NetworkSelectionElement objects.
//
// If a pod belongs to multiple networks, it will be indexed under each network
// name. "equality" in the field selector means that at least one key matches
// the value.
//
// If the pod does not have the annotation or if the annotation is invalid, it will not be indexed.
func PodByNetworkFunc(obj client.Object) []string {
	log := log.WithValues("pod", fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName()))
	pod := obj.(*corev1.Pod)
	annotation, exists := pod.GetAnnotations()[networkv1.NetworkStatusAnnot]
	if !exists {
		return nil
	}

	var networkSelectionElements []networkv1.NetworkSelectionElement
	raw := []byte(annotation)
	if !json.Valid(raw) {
		log.Error(fmt.Errorf("invalid JSON"), "failed to unmarshal network selection elements from pod annotations", "annotations", annotation)
		return nil
	}

	if err := json.Unmarshal(raw, &networkSelectionElements); err != nil {
		log.Error(err, "failed to unmarshal network selection elements from pod annotations", "annotations", annotation)
		return nil
	}

	var indexValues []string
	for _, element := range networkSelectionElements {
		indexValues = append(indexValues, element.Name)
	}

	log.Info("indexing pod", "indexValues", indexValues)
	return indexValues
}
