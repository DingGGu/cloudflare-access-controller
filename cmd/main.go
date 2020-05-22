package main

import (
	"github.com/DingGGu/cloudflare-access-controller/v2/internal/controllers"
	"github.com/DingGGu/cloudflare-access-controller/v2/internal/providers"
	"os"
	runtime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"
)

var log = logf.Log.WithName("cloudflare-access-controller")

func main() {
	options := getOptions()
	logf.SetLogger(zap.New(zap.UseDevMode(options.Debug)))

	logger := log.WithName("main")

	syncPeriod := time.Second * 60

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Namespace:  options.WatchNamespace,
		SyncPeriod: &syncPeriod,
	})
	if nil != err {
		logger.Error(err, "could not create manager")
		os.Exit(1)
	}

	provider := providers.NewCloudflare(
		options.CloudflareApiToken,
		logf.Log.WithName("providers").WithName("cloudflare"),
		options.ZoneName,
		options.ClusterName,
	)

	if err := (&controllers.Controller{
		Client:   mgr.GetClient(),
		Log:      logf.Log.WithName("controllers").WithName("ingress"),
		Recorder: mgr.GetEventRecorderFor("cloudflare-access-controller"),
		Provider: provider,
	}).New(mgr); err != nil {
		logger.Error(err, "could not create controller", "controllers", "ingress")
		os.Exit(1)
	}

	if err := mgr.Start(runtime.SetupSignalHandler()); err != nil {
		log.Error(err, "could not start manager")
		os.Exit(1)
	}
}
