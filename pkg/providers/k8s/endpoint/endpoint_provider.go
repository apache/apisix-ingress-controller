package endpoint

import (
	"context"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
)

type Provider interface {
}

type provider struct {
	cfg *config.Config

	endpointsController     *endpointsController
	endpointSliceController *endpointSliceController
}

func NewProvider(cfg *config.Config) (Provider, error) {
	p := &provider{
		cfg: cfg,
	}

	if cfg.Kubernetes.WatchEndpointSlices {
		p.endpointSliceController = newEndpointSliceController()
	} else {
		p.endpointsController = NewEndpointsController()
	}

	return p, nil
}

func (p *provider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		if p.cfg.Kubernetes.WatchEndpointSlices {
			p.endpointSliceController.run(ctx)
		} else {
			p.endpointsController.run(ctx)
		}
	})

	e.Wait()
}
