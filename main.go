package main

import (
	"os"
	"time"

	inctrl "github.com/ihcsim/nad-service-controller/internal/controllers"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
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

	if err := ctrl.NewControllerManagedBy(mgr).
		Named(ControllerName).
		For(&corev1.Service{}).
		Complete(&inctrl.NADServiceReconciler{
			Client: mgr.GetClient(),
		}); err != nil {
		log.Error(err, "could not create controller")
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "problem starting manager")
		os.Exit(1)
	}
}
