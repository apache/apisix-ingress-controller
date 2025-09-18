package manager

import (
	"context"

	webhookv1 "github.com/apache/apisix-ingress-controller/internal/webhook/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func setupWebhooks(ctx context.Context, mgr manager.Manager) error {
	if err := webhookv1.SetupIngressWebhookWithManager(mgr); err != nil {
		return err
	}
	return nil
}
