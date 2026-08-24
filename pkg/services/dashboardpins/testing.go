package dashboardpins

import (
	"time"

	"github.com/grafana/grafana/pkg/infra/db"
)

// NewTestService constructs a DashboardPinsService for tests.
func NewTestService(sqlStore db.DB, dashboardService dashboardLookup, now func() time.Time) *DashboardPinsService {
	return &DashboardPinsService{
		store:            sqlStore,
		dashboardService: dashboardService,
		now:              now,
	}
}
