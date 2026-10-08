package recentitems

import (
	"context"
	"time"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/user"
)

func ProvideService(sqlStore db.DB, routeRegister routing.RouteRegister) *RecentItemsService {
	s := &RecentItemsService{
		store:         sqlStore,
		RouteRegister: routeRegister,
		now:           time.Now,
	}

	s.registerAPIEndpoints()

	return s
}

type Service interface {
	Record(ctx context.Context, user *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error)
	List(ctx context.Context, user *user.SignedInUser, query ListRecentItemsQuery) ([]RecentItemDTO, error)
	Update(ctx context.Context, user *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error)
	Remove(ctx context.Context, user *user.SignedInUser, uid string) error
}

type RecentItemsService struct {
	store         db.DB
	RouteRegister routing.RouteRegister
	now           func() time.Time
}

func (s *RecentItemsService) Record(ctx context.Context, user *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	return s.record(ctx, user, cmd)
}

func (s *RecentItemsService) List(ctx context.Context, user *user.SignedInUser, query ListRecentItemsQuery) ([]RecentItemDTO, error) {
	return s.list(ctx, user, query)
}

func (s *RecentItemsService) Update(ctx context.Context, user *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	return s.update(ctx, user, uid, cmd)
}

func (s *RecentItemsService) Remove(ctx context.Context, user *user.SignedInUser, uid string) error {
	return s.remove(ctx, user, uid)
}
