package endpoint

import (
	"context"

	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
)

type Provider interface {
}

type provider struct {
	cfg *providertypes.CommonConfig

	endpointsController     *endpointsController
	endpointSliceController *endpointSliceController
}

func NewProvider(cfg *providertypes.CommonConfig) (Provider, error) {
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
