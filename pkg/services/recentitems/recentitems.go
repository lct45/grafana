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
	CreateItem(ctx context.Context, user *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error)
	ListItems(ctx context.Context, user *user.SignedInUser, limit int) ([]RecentItemDTO, error)
	PatchItem(ctx context.Context, user *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error)
	DeleteItem(ctx context.Context, user *user.SignedInUser, uid string) error
}

type RecentItemsService struct {
	store         db.DB
	RouteRegister routing.RouteRegister
	log           log.Logger
	now           func() time.Time
}

func (s *RecentItemsService) CreateItem(ctx context.Context, user *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	return s.createItem(ctx, user, cmd)
}

func (s *RecentItemsService) ListItems(ctx context.Context, user *user.SignedInUser, limit int) ([]RecentItemDTO, error) {
	return s.listItems(ctx, user, limit)
}

func (s *RecentItemsService) PatchItem(ctx context.Context, user *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	return s.patchItem(ctx, user, uid, cmd)
}

func (s *RecentItemsService) DeleteItem(ctx context.Context, user *user.SignedInUser, uid string) error {
	return s.deleteItem(ctx, user, uid)
}
