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
	Upsert(ctx context.Context, user *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error)
	List(ctx context.Context, user *user.SignedInUser, limit int) ([]RecentItemDTO, error)
	Patch(ctx context.Context, user *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error)
	Delete(ctx context.Context, user *user.SignedInUser, uid string) error
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

func (s *RecentItemsService) Upsert(ctx context.Context, user *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	return s.upsert(ctx, user, cmd)
}

func (s *RecentItemsService) List(ctx context.Context, user *user.SignedInUser, limit int) ([]RecentItemDTO, error) {
	return s.list(ctx, user, limit)
}

func (s *RecentItemsService) Patch(ctx context.Context, user *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	return s.patch(ctx, user, uid, cmd)
}

func (s *RecentItemsService) Delete(ctx context.Context, user *user.SignedInUser, uid string) error {
	return s.delete(ctx, user, uid)
}
