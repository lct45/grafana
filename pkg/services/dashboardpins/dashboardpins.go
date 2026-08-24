package dashboardpins

import (
	"context"
	"time"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/user"
)

type dashboardLookup interface {
	GetDashboard(ctx context.Context, query *dashboards.GetDashboardQuery) (*dashboards.Dashboard, error)
}

func ProvideService(
	sqlStore db.DB,
	dashboardService dashboards.DashboardService,
) *DashboardPinsService {
	return &DashboardPinsService{
		store:            sqlStore,
		dashboardService: dashboardService,
		log:              log.New("dashboard-pins"),
		now:              time.Now,
	}
}

type Service interface {
	ListPins(ctx context.Context, user *user.SignedInUser) ([]DashboardPinDTO, error)
	CreatePin(ctx context.Context, user *user.SignedInUser, cmd CreateDashboardPinCommand) (DashboardPinDTO, error)
	ReorderPins(ctx context.Context, user *user.SignedInUser, cmd ReorderDashboardPinsCommand) ([]DashboardPinDTO, error)
	PatchPin(ctx context.Context, user *user.SignedInUser, dashboardUID string, cmd PatchDashboardPinCommand) (DashboardPinDTO, error)
	DeletePin(ctx context.Context, user *user.SignedInUser, dashboardUID string) error
}

type DashboardPinsService struct {
	store            db.DB
	dashboardService dashboardLookup
	log              log.Logger
	now              func() time.Time
}

func (s *DashboardPinsService) ListPins(ctx context.Context, user *user.SignedInUser) ([]DashboardPinDTO, error) {
	return s.listPins(ctx, user)
}

func (s *DashboardPinsService) CreatePin(ctx context.Context, user *user.SignedInUser, cmd CreateDashboardPinCommand) (DashboardPinDTO, error) {
	return s.createPin(ctx, user, cmd)
}

func (s *DashboardPinsService) ReorderPins(ctx context.Context, user *user.SignedInUser, cmd ReorderDashboardPinsCommand) ([]DashboardPinDTO, error) {
	return s.reorderPins(ctx, user, cmd)
}

func (s *DashboardPinsService) PatchPin(ctx context.Context, user *user.SignedInUser, dashboardUID string, cmd PatchDashboardPinCommand) (DashboardPinDTO, error) {
	return s.patchPin(ctx, user, dashboardUID, cmd)
}

func (s *DashboardPinsService) DeletePin(ctx context.Context, user *user.SignedInUser, dashboardUID string) error {
	return s.deletePin(ctx, user, dashboardUID)
}
