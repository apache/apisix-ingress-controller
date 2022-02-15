package apisix

import (
	"context"

	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

var (
	_ StreamRoute = (*noopClient)(nil)
)

type noopClient struct {
}

func (r *noopClient) Get(ctx context.Context, name string) (*v1.StreamRoute, error) {
	return nil, nil
}

func (r *noopClient) List(ctx context.Context) ([]*v1.StreamRoute, error) {
	return nil, nil
}

func (r *noopClient) Create(ctx context.Context, obj *v1.StreamRoute) (*v1.StreamRoute, error) {
	return nil, nil
}

func (r *noopClient) Delete(ctx context.Context, obj *v1.StreamRoute) error {
	return nil
}

func (r *noopClient) Update(ctx context.Context, obj *v1.StreamRoute) (*v1.StreamRoute, error) {
	return nil, nil
}
