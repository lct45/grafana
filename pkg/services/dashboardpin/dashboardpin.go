package dashboardpin

import (
	"context"
)

type Service interface {
	List(ctx context.Context, query *ListPinsQuery) ([]DashboardPinDTO, error)
	Pin(ctx context.Context, cmd *PinDashboardCommand) (*DashboardPinDTO, error)
	Unpin(ctx context.Context, cmd *UnpinDashboardCommand) error
	UpdateNote(ctx context.Context, cmd *UpdatePinNoteCommand) (*DashboardPinDTO, error)
	Reorder(ctx context.Context, cmd *ReorderPinsCommand) error
	DeleteByUser(ctx context.Context, userID int64) error
}
