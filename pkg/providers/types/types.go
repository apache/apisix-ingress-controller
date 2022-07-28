package types

import "context"

type Provider interface {
	Run(ctx context.Context)
}
