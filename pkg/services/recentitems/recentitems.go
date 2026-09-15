package recentitems

import (
	"context"
	"time"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/user"
)

type Service interface {
	Upsert(context.Context, *user.SignedInUser, CreateRecentItemCommand) (RecentItemDTO, bool, error)
	List(context.Context, *user.SignedInUser, int, string) ([]RecentItemDTO, error)
	Patch(context.Context, *user.SignedInUser, string, PatchRecentItemCommand) (RecentItemDTO, error)
	Delete(context.Context, *user.SignedInUser, string) error
}

type RecentItemsService struct {
	store         db.DB
	routeRegister routing.RouteRegister
	log           log.Logger
	now           func() time.Time
}

func ProvideService(sqlStore db.DB, routeRegister routing.RouteRegister) *RecentItemsService {
	service := &RecentItemsService{
		store:         sqlStore,
		routeRegister: routeRegister,
		log:           log.New("recent-items"),
		now:           time.Now,
	}
	service.registerAPIEndpoints()
	return service
}

func (s *RecentItemsService) Upsert(ctx context.Context, signedInUser *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	return s.upsert(ctx, signedInUser, cmd)
}

func (s *RecentItemsService) List(ctx context.Context, signedInUser *user.SignedInUser, limit int, resourceType string) ([]RecentItemDTO, error) {
	return s.list(ctx, signedInUser, limit, resourceType)
}

func (s *RecentItemsService) Patch(ctx context.Context, signedInUser *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	return s.patch(ctx, signedInUser, uid, cmd)
}

func (s *RecentItemsService) Delete(ctx context.Context, signedInUser *user.SignedInUser, uid string) error {
	return s.delete(ctx, signedInUser, uid)
}
