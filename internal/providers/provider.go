package providers

import (
	"context"
	"k8s.io/api/networking/v1beta1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Provider interface {
	Reconcile(context.Context, reconcile.Request, *v1beta1.Ingress, record.EventRecorder) error
	Delete(context.Context, reconcile.Request, *v1beta1.Ingress) error
}
