package recentitems

import (
	"context"
	"time"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/user"
)

func ProvideService(sqlStore db.DB, routeRegister routing.RouteRegister) *RecentItemsService {
	s := &RecentItemsService{
		store:         sqlStore,
		RouteRegister: routeRegister,
		log:           log.New("recent-items"),
		now:           time.Now,
	}

	s.registerAPIEndpoints()

	return s
}

// Service manages the signed-in user's recently viewed resources.
type Service interface {
	// RecordView creates a recent item or refreshes the existing entry for the
	// same resource. The bool result is true when a new item was created.
	RecordView(ctx context.Context, usr *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error)
	ListItems(ctx context.Context, usr *user.SignedInUser, query ListRecentItemsQuery) ([]RecentItemDTO, error)
	PatchItem(ctx context.Context, usr *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error)
	DeleteItem(ctx context.Context, usr *user.SignedInUser, uid string) error
}

type RecentItemsService struct {
	store         db.DB
	RouteRegister routing.RouteRegister
	log           log.Logger
	now           func() time.Time
	// beforeInsert is a test seam that runs inside the upsert transaction just
	// before a new row is inserted, so tests can simulate a concurrent writer.
	beforeInsert func(sess *db.Session) error
}

func (s *RecentItemsService) RecordView(ctx context.Context, usr *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	return s.recordView(ctx, usr, cmd)
}

func (s *RecentItemsService) ListItems(ctx context.Context, usr *user.SignedInUser, query ListRecentItemsQuery) ([]RecentItemDTO, error) {
	return s.listItems(ctx, usr, query)
}

func (s *RecentItemsService) PatchItem(ctx context.Context, usr *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	return s.patchItem(ctx, usr, uid, cmd)
}

func (s *RecentItemsService) DeleteItem(ctx context.Context, usr *user.SignedInUser, uid string) error {
	return s.deleteItem(ctx, usr, uid)
}
