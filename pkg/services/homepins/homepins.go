package homepins

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
) *HomePinsService {
	s := &HomePinsService{
		store:         sqlStore,
		RouteRegister: routeRegister,
		log:           log.New("home-pins"),
		now:           time.Now,
	}

	s.registerAPIEndpoints()

	return s
}

type Service interface {
	CreatePin(ctx context.Context, user *user.SignedInUser, cmd CreatePinCommand) (DashboardPinDTO, error)
	ListPins(ctx context.Context, user *user.SignedInUser) ([]DashboardPinDTO, error)
	UpdatePin(ctx context.Context, user *user.SignedInUser, uid string, cmd UpdatePinCommand) (DashboardPinDTO, error)
	ReorderPins(ctx context.Context, user *user.SignedInUser, cmd ReorderPinsCommand) error
	DeletePin(ctx context.Context, user *user.SignedInUser, uid string) error
}

type HomePinsService struct {
	store         db.DB
	RouteRegister routing.RouteRegister
	log           log.Logger
	now           func() time.Time
}

func (s *HomePinsService) CreatePin(ctx context.Context, user *user.SignedInUser, cmd CreatePinCommand) (DashboardPinDTO, error) {
	return s.createPin(ctx, user, cmd)
}

func (s *HomePinsService) ListPins(ctx context.Context, user *user.SignedInUser) ([]DashboardPinDTO, error) {
	return s.listPins(ctx, user)
}

func (s *HomePinsService) UpdatePin(ctx context.Context, user *user.SignedInUser, uid string, cmd UpdatePinCommand) (DashboardPinDTO, error) {
	return s.updatePin(ctx, user, uid, cmd)
}

func (s *HomePinsService) ReorderPins(ctx context.Context, user *user.SignedInUser, cmd ReorderPinsCommand) error {
	return s.reorderPins(ctx, user, cmd)
}

func (s *HomePinsService) DeletePin(ctx context.Context, user *user.SignedInUser, uid string) error {
	return s.deletePin(ctx, user, uid)
}
