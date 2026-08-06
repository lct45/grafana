package recentitems

import (
	"context"
	"time"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/user"
)

func ProvideService(sqlStore db.DB, routeRegister routing.RouteRegister) *ServiceImpl {
	s := &ServiceImpl{
		store:         sqlStore,
		routeRegister: routeRegister,
		now:           time.Now,
	}
	s.registerAPIEndpoints()
	return s
}

type Service interface {
	UpsertRecentItem(ctx context.Context, user *user.SignedInUser, cmd CreateRecentItemCommand) (UpsertRecentItemResult, error)
	ListRecentItems(ctx context.Context, user *user.SignedInUser, query ListRecentItemsQuery) ([]RecentItemDTO, error)
	PatchRecentItem(ctx context.Context, user *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error)
	DeleteRecentItem(ctx context.Context, user *user.SignedInUser, uid string) error
}

type ServiceImpl struct {
	store         db.DB
	routeRegister routing.RouteRegister
	now           func() time.Time
}

func (s *ServiceImpl) UpsertRecentItem(ctx context.Context, user *user.SignedInUser, cmd CreateRecentItemCommand) (UpsertRecentItemResult, error) {
	return s.upsertRecentItem(ctx, user, cmd)
}

func (s *ServiceImpl) ListRecentItems(ctx context.Context, user *user.SignedInUser, query ListRecentItemsQuery) ([]RecentItemDTO, error) {
	return s.listRecentItems(ctx, user, query)
}

func (s *ServiceImpl) PatchRecentItem(ctx context.Context, user *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	return s.patchRecentItem(ctx, user, uid, cmd)
}

func (s *ServiceImpl) DeleteRecentItem(ctx context.Context, user *user.SignedInUser, uid string) error {
	return s.deleteRecentItem(ctx, user, uid)
}
