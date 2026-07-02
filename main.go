package main

import (
	"context"
	"fmt"
	"os"
	"time"

	inctrl "github.com/ihcsim/nad-service-controller/internal/controllers"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	ControllerName      = "nad-service-controller"
	ControllerNamespace = "kube-system"
)

func main() {
	ctrl.SetLogger(zap.New())
	log := ctrl.Log.WithName(ControllerName)

	var (
		leaseDuration = 100 * time.Second
		renewDeadline = 80 * time.Second
		retryPeriod   = 20 * time.Second
	)
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		LeaderElection:          true,
		LeaderElectionID:        ControllerName,
		LeaderElectionNamespace: ControllerNamespace,
		LeaseDuration:           &leaseDuration,
		RenewDeadline:           &renewDeadline,
		RetryPeriod:             &retryPeriod,
		Metrics: ctrlmetrics.Options{
			SecureServing: true,
			BindAddress:   ":9443",
		},
	})
	if err != nil {
		log.Error(err, "could not create manager")
		os.Exit(1)
	}

	r := &inctrl.NADServiceReconciler{
		Client: mgr.GetClient(),
		Log:    log,
		Debug:  log.V(3),
	}
	if err := ctrl.NewControllerManagedBy(mgr).
		Named(ControllerName).
		For(&corev1.Service{}).
		Watches(&corev1.Pod{}, handler.TypedEnqueueRequestsFromMapFunc(r.FindServicesForPod), builder.WithPredicates()).
		Complete(r); err != nil {
		log.Error(err, "could not create controller")
		os.Exit(1)
	}

	// set up indices based on service's spec.selector for faster pods selection
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, inctrl.IndexKeyServicePodSelector, func(obj client.Object) []string {
		svc := obj.(*corev1.Service)
		var indexValues []string
		for k, v := range svc.Spec.Selector {
			indexValues = append(indexValues, fmt.Sprintf("%s=%s", k, v))
		}
		return indexValues
	}); err != nil {
		log.Error(err, "could not create field indexer")
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "problem starting manager")
		os.Exit(1)
	}
}
