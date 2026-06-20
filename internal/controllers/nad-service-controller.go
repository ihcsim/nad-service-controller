package controllers

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
)

type NADServiceReconciler struct{}

func (c *NADServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
