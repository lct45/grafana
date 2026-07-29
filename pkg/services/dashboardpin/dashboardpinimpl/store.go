package dashboardpinimpl

import (
	"context"

	"github.com/grafana/grafana/pkg/services/dashboardpin"
)

type store interface {
	List(context.Context, *dashboardpin.ListPinsQuery) ([]dashboardpin.DashboardPin, error)
	Insert(context.Context, *dashboardpin.PinDashboardCommand, int) (*dashboardpin.DashboardPin, error)
	Get(context.Context, int64, int64, string) (*dashboardpin.DashboardPin, error)
	Delete(context.Context, *dashboardpin.UnpinDashboardCommand) error
	UpdateNote(context.Context, *dashboardpin.UpdatePinNoteCommand) (*dashboardpin.DashboardPin, error)
	Reorder(context.Context, *dashboardpin.ReorderPinsCommand) error
	DeleteByUser(context.Context, int64) error
}
