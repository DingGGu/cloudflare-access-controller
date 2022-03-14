package providers

import (
	"context"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Provider interface {
	Reconcile(context.Context, reconcile.Request, *networkingv1.Ingress, record.EventRecorder) error
	Delete(context.Context, reconcile.Request, *networkingv1.Ingress) error
}
