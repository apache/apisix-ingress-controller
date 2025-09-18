package manager

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	webhookv1 "github.com/apache/apisix-ingress-controller/internal/webhook/v1"
)

func setupWebhooks(_ context.Context, mgr manager.Manager) error {
	if err := webhookv1.SetupIngressWebhookWithManager(mgr); err != nil {
		return err
	}
	return nil
}
