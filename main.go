package main

import (
	"context"
	"os"
	"time"

	ctrlnad "github.com/ihcsim/nad-service-controller/internal/controllers"
	indexer "github.com/ihcsim/nad-service-controller/internal/indexer"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
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
		LeaderElection:                true,
		LeaderElectionID:              ControllerName,
		LeaderElectionNamespace:       ControllerNamespace,
		LeaderElectionReleaseOnCancel: true,
		LeaseDuration:                 &leaseDuration,
		RenewDeadline:                 &renewDeadline,
		RetryPeriod:                   &retryPeriod,
		Metrics: ctrlmetrics.Options{
			SecureServing: true,
			BindAddress:   ":9443",
		},
	})
	if err != nil {
		log.Error(err, "could not create manager")
		os.Exit(1)
	}

	r := &ctrlnad.NADServiceReconciler{
		Client:         mgr.GetClient(),
		ControllerName: ControllerName,
		Log:            log,
		Debug:          log.V(3),
	}
	if err := ctrl.NewControllerManagedBy(mgr).
		Named(ControllerName).
		For(&corev1.Service{}).
		Watches(&corev1.Pod{}, handler.TypedEnqueueRequestsFromMapFunc(r.SyncServicesForPod), builder.WithPredicates()).
		Complete(r); err != nil {
		log.Error(err, "could not create controller")
		os.Exit(1)
	}

	// set up index to search services by their network. the index values are used
	// as field selector in the controller's list calls
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, indexer.IndexKeyServiceNetwork, indexer.ServiceByNetworkFunc); err != nil {
		log.Error(err, "could not create field indexer for corev1/service")
		os.Exit(1)
	}

	// set up index to search pods by their network. the index values are used as
	// field selector in the contoller's list calls
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, indexer.IndexKeyPodNetwork, indexer.PodByNetworkFunc); err != nil {
		log.Error(err, "could not create field indexer for corev1/service")
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "problem starting manager")
		os.Exit(1)
	}
}
