package main

import (
	"github.com/DingGGu/cloudflare-access-controller/v2/internal/controllers"
	"github.com/DingGGu/cloudflare-access-controller/v2/internal/providers"
	"net/http"
	"os"
	runtime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func main() {
	options := getOptions()
	log.SetLogger(zap.New(zap.UseDevMode(options.Debug)))

	logger := log.Log.WithName("main")

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Namespace:              options.WatchNamespace,
		SyncPeriod:             &options.SyncPeriodSecond,
		MetricsBindAddress:     "0",
		HealthProbeBindAddress: ":8888",
	})
	if nil != err {
		logger.Error(err, "could not create manager")
		os.Exit(1)
	}

	provider := providers.NewCloudflare(
		options.CloudflareApiToken,
		log.Log.WithName("providers").WithName("cloudflare"),
		options.ZoneName,
		options.ClusterName,
	)

	if err := (&controllers.Controller{
		Client:   mgr.GetClient(),
		Log:      log.Log.WithName("controllers").WithName("ingress"),
		Recorder: mgr.GetEventRecorderFor("cloudflare-access-controller"),
		Provider: provider,
	}).New(mgr); err != nil {
		logger.Error(err, "could not create controller", "controllers", "ingress")
		os.Exit(1)
	}

	_ = mgr.AddHealthzCheck("check", func(_ *http.Request) error { return nil })
	_ = mgr.AddReadyzCheck("check", func(_ *http.Request) error { return nil })

	if err := mgr.Start(runtime.SetupSignalHandler()); err != nil {
		log.Log.Error(err, "could not start manager")
		os.Exit(1)
	}
}
