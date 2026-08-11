package recentitems

import (
	"context"
	"time"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/user"
)

func ProvideService(
	sqlStore db.DB,
	routeRegister routing.RouteRegister,
) *RecentItemsService {
	s := &RecentItemsService{
		store:         sqlStore,
		RouteRegister: routeRegister,
		log:           log.New("recent-items"),
		now:           time.Now,
	}

	s.registerAPIEndpoints()

	return s
}

type Service interface {
	CreateOrUpdateRecentItem(ctx context.Context, user *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error)
	ListRecentItems(ctx context.Context, user *user.SignedInUser, limit int) ([]RecentItemDTO, error)
	PatchRecentItem(ctx context.Context, user *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error)
	DeleteRecentItem(ctx context.Context, user *user.SignedInUser, uid string) error
}

type RecentItemsService struct {
	store         db.DB
	RouteRegister routing.RouteRegister
	log           log.Logger
	now           func() time.Time
}

func (s *RecentItemsService) CreateOrUpdateRecentItem(ctx context.Context, user *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	return s.createOrUpdateRecentItem(ctx, user, cmd)
}

func (s *RecentItemsService) ListRecentItems(ctx context.Context, user *user.SignedInUser, limit int) ([]RecentItemDTO, error) {
	return s.listRecentItems(ctx, user, limit)
}

func (s *RecentItemsService) PatchRecentItem(ctx context.Context, user *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	return s.patchRecentItem(ctx, user, uid, cmd)
}

func (s *RecentItemsService) DeleteRecentItem(ctx context.Context, user *user.SignedInUser, uid string) error {
	return s.deleteRecentItem(ctx, user, uid)
}
