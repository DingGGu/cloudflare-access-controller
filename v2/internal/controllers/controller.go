package controllers

import (
	"context"
	"github.com/DingGGu/cloudflare-access-controller/v2/internal/providers"
	"k8s.io/client-go/tools/record"

	"github.com/go-logr/logr"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	runtime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Controller struct {
	Client   client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
	Provider providers.Provider
}

func (c *Controller) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	log := c.Log.WithValues("ingress", req.NamespacedName)

	ingress := &networkingv1beta1.Ingress{}
	if err := c.Client.Get(ctx, req.NamespacedName, ingress); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Could not fetch ingress")
			return reconcile.Result{}, err
		}

		if err := c.Provider.Delete(ctx, req, ingress); err != nil {
			log.Error(err, "Cannot delete resource")
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	log.Info("Reconciling ingress")
	if err := c.Provider.Reconcile(ctx, req, ingress, c.Recorder); err != nil {
		log.Error(err, "Reconcile fail during upload to provider")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (c *Controller) New(mgr runtime.Manager) error {
	return runtime.NewControllerManagedBy(mgr).
		For(&networkingv1beta1.Ingress{}).
		For(&extensionsv1beta1.Ingress{}).
		Complete(c)
}
