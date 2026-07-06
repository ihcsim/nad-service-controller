package controllers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	endpointsliceac "github.com/ihcsim/nad-service-controller/internal/controllers/applyconfiguration/endpointslice"
)

const (
	serviceAnnotation          = "isim.dev/networks"
	IndexKeyServicePodSelector = "svc.pod.selector"
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

	network, exists := svc.GetAnnotations()[serviceAnnotation]
	if !exists {
		log.Info("skipping service", "reason", "don't have required annotation", "annotation", serviceAnnotation)
		return ctrl.Result{}, nil
	}
	log.Info("found network", "network", network)

	selector, err := labels.Set(svc.Spec.Selector).AsValidatedSelector()
	if err != nil {
		log.Error(err, "failed to create label selector from Service spec")
		return ctrl.Result{}, err
	}

	pods := &corev1.PodList{}
	if err := r.List(ctx, pods, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list Pods: %w", err)
	}
	log.Info("found pods", "count", len(pods.Items))
	debug.Info("pods spec", "list", pods)

	config, err := endpointsliceac.ApplyConfig(svc, pods.Items, r.Client)
	if err != nil {
		if errors.Is(errors.Unwrap(err), endpointsliceac.ErrMsgInvalidPodReadinessCondition) || errors.Is(errors.Unwrap(err), endpointsliceac.ErrMsgInvalidPodIP) {
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: 5 * time.Second,
			}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to create endpointSlice apply configuration: %w", err)
	}
	r.Log.Info("applying changes to endpointSlice", "name", config.Name, "config", config)

	if err := r.Apply(ctx, config, &client.ApplyOptions{
		FieldManager: r.ControllerName,
	}); err != nil {
		r.Log.Error(err, "failed to apply changes to endpointSlice")
		return ctrl.Result{}, fmt.Errorf("failed to apply changes to endpointSlice: %w", err)
	}

	return ctrl.Result{}, nil
}

// FindServicesForPod finds all services that have a selector matching the labels of the given pod.
func (r *NADServiceReconciler) FindServicesForPod(ctx context.Context, pod client.Object) []reconcile.Request {
	svcChan := make(chan types.NamespacedName)

	// requests holds the list of reconcile.Requests with the name of the services to be reconciled.
	requests := []reconcile.Request{}
	go func() {
		for svc := range svcChan {
			r.Log.Info("found service for pod", "service", svc.Name, "pod", pod.GetName())
			requests = append(requests, reconcile.Request{
				NamespacedName: svc,
			})
		}
	}()

	wg := sync.WaitGroup{}
LOOP:
	for k, v := range pod.GetLabels() {
		select {
		case <-ctx.Done():
			break LOOP
		default:
			wg.Add(1)
			go func(k, v string) {
				defer wg.Done()
				svcs := &corev1.ServiceList{}
				listOpts := &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector(IndexKeyServicePodSelector, fmt.Sprintf("%s=%s", k, v)),
					Namespace:     pod.GetNamespace(),
				}
				if err := r.List(ctx, svcs, listOpts); err != nil {
					r.Log.Error(err, "additionalInfo", "failed to list Services for Pod", "pod", pod.GetName())
					return
				}

				for _, svc := range svcs.Items {
					for k, v := range svc.GetAnnotations() {
						if k == serviceAnnotation && v != "" {
							svcChan <- types.NamespacedName{
								Name:      svc.GetName(),
								Namespace: svc.GetNamespace(),
							}
						}
					}
				}
			}(k, v)
		}
	}
	wg.Wait()
	close(svcChan)

	r.Debug.Info("exiting watcher")
	return requests
}
