package dashboardpinimpl

import (
	"context"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/dashboardpin"
)

type Service struct {
	store  store
	db     db.DB
	logger log.Logger
}

func ProvideService(db db.DB) dashboardpin.Service {
	return &Service{
		store: &sqlStore{
			db: db,
		},
		db:     db,
		logger: log.New("dashboardpin"),
	}
}

func (s *Service) List(ctx context.Context, query *dashboardpin.ListPinsQuery) ([]dashboardpin.DashboardPinDTO, error) {
	pins, err := s.store.List(ctx, query)
	if err != nil {
		return nil, err
	}

	dtos := make([]dashboardpin.DashboardPinDTO, 0, len(pins))
	for _, pin := range pins {
		dtos = append(dtos, dashboardpin.ToDTO(pin))
	}
	return dtos, nil
}

func (s *Service) Pin(ctx context.Context, cmd *dashboardpin.PinDashboardCommand) (*dashboardpin.DashboardPinDTO, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.store.Get(ctx, cmd.UserID, cmd.OrgID, cmd.DashboardUID)
	if err == nil {
		dto := dashboardpin.ToDTO(*existing)
		return &dto, nil
	}
	if err != dashboardpin.ErrPinNotFound {
		return nil, err
	}

	pins, err := s.store.List(ctx, &dashboardpin.ListPinsQuery{
		UserID: cmd.UserID,
		OrgID:  cmd.OrgID,
	})
	if err != nil {
		return nil, err
	}

	pin, err := s.store.Insert(ctx, cmd, len(pins))
	if err != nil {
		if s.db.GetDialect().IsUniqueConstraintViolation(err) {
			existing, getErr := s.store.Get(ctx, cmd.UserID, cmd.OrgID, cmd.DashboardUID)
			if getErr != nil {
				return nil, getErr
			}
			dto := dashboardpin.ToDTO(*existing)
			return &dto, nil
		}
		return nil, err
	}

	dto := dashboardpin.ToDTO(*pin)
	return &dto, nil
}

func (s *Service) Unpin(ctx context.Context, cmd *dashboardpin.UnpinDashboardCommand) error {
	if err := cmd.Validate(); err != nil {
		return err
	}
	return s.store.Delete(ctx, cmd)
}

func (s *Service) UpdateNote(ctx context.Context, cmd *dashboardpin.UpdatePinNoteCommand) (*dashboardpin.DashboardPinDTO, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	pin, err := s.store.UpdateNote(ctx, cmd)
	if err != nil {
		return nil, err
	}

	dto := dashboardpin.ToDTO(*pin)
	return &dto, nil
}

func (s *Service) Reorder(ctx context.Context, cmd *dashboardpin.ReorderPinsCommand) error {
	if err := cmd.Validate(); err != nil {
		return err
	}
	return s.store.Reorder(ctx, cmd)
}

func (s *Service) DeleteByUser(ctx context.Context, userID int64) error {
	return s.store.DeleteByUser(ctx, userID)
}
