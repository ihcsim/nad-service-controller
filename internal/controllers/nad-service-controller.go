package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const serviceAnnotation = "isim.dev/networks"

type NADServiceReconciler struct {
	client.Client
}

func (r *NADServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx, "service", req.NamespacedName)

	log.Info("reconciling service")
	svc := &corev1.Service{}
	if err := r.Get(ctx, req.NamespacedName, svc); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("skipping service", "reason", "service not found")
			return reconcile.Result{}, nil
		}
		log.Error(err, "additionalInfo", "fail to get service")
		return reconcile.Result{}, fmt.Errorf("failed to get Service: %w", err)
	}

	network, exists := svc.GetAnnotations()[serviceAnnotation]
	if !exists {
		log.Info("skipping service", "reason", "missing required annotation", "annotation", serviceAnnotation)
		return reconcile.Result{}, nil
	}
	log.Info("found network", "network", network)

	selector, err := labels.Set(svc.Spec.Selector).AsValidatedSelector()
	if err != nil {
		log.Error(err, "additionalInfo", "failed to create label selector from Service spec")
		return reconcile.Result{}, err
	}

	pods := &corev1.PodList{}
	if err := r.List(ctx, pods, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to list Pods: %w", err)
	}
	log.Info("discovered pods", "list", pods)

	return ctrl.Result{}, nil
}
