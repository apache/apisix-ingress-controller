package apisix

import (
	"context"
	"strings"

	"github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	"go.uber.org/zap"
)

// There is no need to ensure the consistency between the upstream to services, only need to ensure that the upstream-node can be delete after deleting the service
type upstreamService struct {
	cluster *cluster
}

func newUpstreamServiceRelation(c *cluster) *upstreamService {
	return &upstreamService{
		cluster: c,
	}
}

func (u *upstreamService) Get(ctx context.Context, svcId string) (*v1.UpstreamServiceRelation, error) {
	log.Debugw("try to get upstreamService in cache",
		zap.String("svcId", svcId),
		zap.String("cluster", "default"),
	)
	us, err := u.cluster.cache.GetUpstreamServiceRelation(svcId)
	if err == nil {
		return us, nil
	}
	if err != cache.ErrNotFound {
		log.Error("failed to find upstreamService in cache",
			zap.String("svcId", svcId), zap.Error(err))
	} else {
		log.Debugw("failed to find upstreamService in cache",
			zap.String("svcId", svcId), zap.Error(err))
	}
	return nil, err
}

func (u *upstreamService) Delete(ctx context.Context, svcId string) error {
	log.Debugw("try to delete upstreamService in cache",
		zap.String("svcId", svcId),
		zap.String("cluster", "default"),
	)
	us, err := u.cluster.cache.GetUpstreamServiceRelation(svcId)
	if err != nil {
		return err
	}
	ups, err := u.cluster.upstream.Get(ctx, us.UpstreamId)
	if err != nil {
		log.Errorf("failed to get upstream in cache: %s", err)
		return err
	}
	ups.Nodes = make(v1.UpstreamNodes, 0)
	_, err = u.cluster.upstream.Update(ctx, ups)
	if err != nil {
		return err
	}
	u.cluster.cache.DeleteUpstreamServiceRelation(us)
	_, err = u.List(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (u *upstreamService) Create(ctx context.Context, upstreamName string) error {
	log.Debugw("try to create upstreamService in cache",
		zap.String("upstreamName", upstreamName),
		zap.String("cluster", "default"),
	)
	upsId, svcId := u.parseUpstreamName(upstreamName)
	if upsId == "" || svcId == "" {
		log.Error("failed to parse upstreamName",
			zap.String("upstreamName", upstreamName),
		)
		return nil
	}
	us, err := u.cluster.cache.GetUpstreamServiceRelation(svcId)
	if err != nil {
		if err != cache.ErrNotFound {
			return err
		}
		us = &v1.UpstreamServiceRelation{
			Metadata: v1.Metadata{
				ID: svcId,
			},
			UpstreamId: upsId,
		}
	}
	if us != nil {
		us.UpstreamId = upsId
	}
	if err := u.cluster.cache.InsertUpstreamServiceRelation(us); err != nil {
		log.Errorf("failed to reflect upstreamService create to cache: %s", err)
		return err
	}
	return nil
}

func (u *upstreamService) List(ctx context.Context) ([]*v1.UpstreamServiceRelation, error) {
	log.Debugw("try to create upstreamService in cache",
		zap.String("cluster", "default"),
	)
	usrs, err := u.cluster.cache.ListUpstreamServiceRelation()
	if err != nil {
		log.Errorw("failed to list upstream in cache",
			zap.Error(err),
		)
		return nil, err
	}
	return usrs, nil
}

func (u *upstreamService) parseUpstreamName(upsName string) (upsId string, svcId string) {
	log.Debugw("try to parse upstreamName",
		zap.String("upstreamName", upsName),
		zap.String("cluster", "default"),
	)
	args := strings.Split(upsName, "_")
	// namespace_service_subcret_port
	if len(args) < 2 {
		return
	}
	return upsName, args[0] + "_" + args[1]
}
