package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	endpointsliceac "github.com/ihcsim/nad-service-controller/internal/controllers/applyconfiguration/endpointslice"
	"github.com/ihcsim/nad-service-controller/internal/indexer"
	networkv1 "github.com/ihcsim/nad-service-controller/pkg/apis/k8s.cni.cncf.io/network/v1"
)

type NADServiceReconciler struct {
	client.Client
	ControllerName string
	Log            logr.Logger
	Debug          logr.Logger
}

func (r *NADServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctrllog.IntoContext(ctx, r.Log), "service", req.NamespacedName)
	debug := ctrllog.FromContext(ctrllog.IntoContext(ctx, r.Debug), "service", req.NamespacedName)

	svc := &corev1.Service{}
	if err := r.Get(ctx, req.NamespacedName, svc); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("skipping service", "reason", "service not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "fail to get service")
		return ctrl.Result{}, fmt.Errorf("failed to get Service: %w", err)
	}

	namespacedNetwork, exists := svc.GetAnnotations()[indexer.ServiceNetworkAnnotation]
	if !exists {
		log.Info("skipping service", "reason", "don't have required annotation", "annotation", indexer.ServiceNetworkAnnotation)
		return ctrl.Result{}, nil
	}

	// prefix network with namespace to match the convention used in the pods'
	// network-status annotation
	namespacedNetwork = fmt.Sprintf("%s/%s", svc.GetNamespace(), namespacedNetwork)
	log.Info("found service network", "network", namespacedNetwork)

	// find pods with the cni network-status annotation matching the network of the
	// service
	var (
		pods     = &corev1.PodList{}
		listOpts = &client.ListOptions{
			Namespace: svc.GetNamespace(),
		}
		matchingFields = client.MatchingFields{indexer.IndexKeyPodNetwork: namespacedNetwork}
	)
	matchingFields.ApplyToList(listOpts)
	if err := r.List(ctx, pods, listOpts); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list Pods: %w", err)
	}
	log.Info("found pods", "count", len(pods.Items))
	debug.Info("pods spec", "spec", pods)

	// use server-side apply to synchronize the endpoint slice with the service and pods
	config, err := endpointsliceac.ApplyConfig(svc, pods.Items, namespacedNetwork, r.Client)
	if err != nil {
		if retryableError(err) {
			log.Error(err, "failed to create endpoint slice apply configuration, will retry")
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: 5 * time.Second,
			}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to create endpoint slice apply configuration: %w", err)
	}
	log.Info("applying changes to endpoint slice", "name", config.Name)
	debug.Info("config spec", "spec", config)

	if err := r.Apply(ctx, config, &client.ApplyOptions{
		FieldManager: r.ControllerName,
	}); err != nil {
		log.Error(err, "failed to apply changes to endpoint slice")
		return ctrl.Result{}, fmt.Errorf("failed to apply changes to endpoint slice: %w", err)
	}

	return ctrl.Result{}, nil
}

func retryableError(err error) bool {
	return errors.Is(errors.Unwrap(err), endpointsliceac.ErrMsgInvalidPodReadinessCondition) ||
		errors.Is(errors.Unwrap(err), endpointsliceac.ErrMsgEmptyPodNetworkIPs)
}

// SyncServicesForPod finds all services that are annotated with the network name that the specified pod is a part of.
func (r *NADServiceReconciler) SyncServicesForPod(ctx context.Context, pod client.Object) []ctrl.Request {
	log := ctrllog.FromContext(ctrllog.IntoContext(ctx, r.Log), "pod", fmt.Sprintf("%s/%s", pod.GetNamespace(), pod.GetName()))
	debug := ctrllog.FromContext(ctrllog.IntoContext(ctx, r.Debug), "pod", fmt.Sprintf("%s/%s", pod.GetNamespace(), pod.GetName()))

	// requests holds the list of ctrl.Requests containing the name of the services to be reconciled,
	// in response to pod event.
	requests := []ctrl.Request{}
	svcChan := make(chan types.NamespacedName)
	go func() {
		for svc := range svcChan {
			log.Info("syncing service in response to pod event", "service", svc)
			requests = append(requests, ctrl.Request{
				NamespacedName: svc,
			})
		}
	}()

	annotation, exists := pod.GetAnnotations()[networkv1.NetworkStatusAnnot]
	if !exists {
		return nil
	}

	// unmarshal the JSON annotation into a slice of NetworkSelectionElement structs
	raw := []byte(annotation)
	if !json.Valid([]byte(raw)) {
		log.Error(fmt.Errorf("invalid JSON"), "failed to unmarshal network selection elements from pod annotation", "annotation", annotation)
		return nil
	}

	var networkSelectionElements []networkv1.NetworkSelectionElement
	if err := json.Unmarshal(raw, &networkSelectionElements); err != nil {
		log.Error(err, "failed to unmarshal network selection elements from pod annotations", "annotation", annotation)
		return nil
	}

	wg := sync.WaitGroup{}
LOOP:
	for _, element := range networkSelectionElements {
		select {
		case <-ctx.Done():
			break LOOP
		default:
			wg.Add(1)
			go func(networkName string) {
				defer wg.Done()

				var (
					svcs     = &corev1.ServiceList{}
					listOpts = &client.ListOptions{
						Namespace: pod.GetNamespace(),
					}
					matchingFields = client.MatchingFields{indexer.IndexKeyServiceNetwork: networkName}
				)
				matchingFields.ApplyToList(listOpts)
				if err := r.List(ctx, svcs, listOpts); err != nil {
					log.Error(err, "failed to list services")
					return
				}

				for _, svc := range svcs.Items {
					svcChan <- types.NamespacedName{
						Name:      svc.GetName(),
						Namespace: svc.GetNamespace(),
					}
				}
			}(element.Name)
		}
	}
	wg.Wait()
	close(svcChan)

	debug.Info("exiting service/pod sync")
	return requests
}
