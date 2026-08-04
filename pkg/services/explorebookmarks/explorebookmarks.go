package explorebookmarks

import (
	"context"
	"time"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	ac "github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/user"
)

func ProvideService(
	sqlStore db.DB,
	routeRegister routing.RouteRegister,
	accessControl ac.AccessControl,
) *ExploreBookmarksService {
	s := &ExploreBookmarksService{
		store:         sqlStore,
		RouteRegister: routeRegister,
		log:           log.New("explore-bookmarks"),
		now:           time.Now,
		accessControl: accessControl,
	}

	s.registerAPIEndpoints()

	return s
}

type Service interface {
	CreateBookmark(ctx context.Context, user *user.SignedInUser, cmd CreateBookmarkCommand) (ExploreBookmarkDTO, error)
	ListBookmarks(ctx context.Context, user *user.SignedInUser) ([]ExploreBookmarkDTO, error)
	DeleteBookmark(ctx context.Context, user *user.SignedInUser, uid string) error
}

type ExploreBookmarksService struct {
	store         db.DB
	RouteRegister routing.RouteRegister
	log           log.Logger
	now           func() time.Time
	accessControl ac.AccessControl
}

func (s *ExploreBookmarksService) CreateBookmark(ctx context.Context, user *user.SignedInUser, cmd CreateBookmarkCommand) (ExploreBookmarkDTO, error) {
	return s.createBookmark(ctx, user, cmd)
}

func (s *ExploreBookmarksService) ListBookmarks(ctx context.Context, user *user.SignedInUser) ([]ExploreBookmarkDTO, error) {
	return s.listBookmarks(ctx, user)
}

func (s *ExploreBookmarksService) DeleteBookmark(ctx context.Context, user *user.SignedInUser, uid string) error {
	return s.deleteBookmark(ctx, user, uid)
}
