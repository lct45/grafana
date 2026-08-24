package dashboardpins

import (
	"context"
	"errors"
	"strings"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
)

func (s *DashboardPinsService) listPins(ctx context.Context, user *user.SignedInUser) ([]DashboardPinDTO, error) {
	var pins []DashboardPin

	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		return session.Where("org_id = ? AND user_id = ?", user.OrgID, user.UserID).
			Asc("sort_order").
			Find(&pins)
	})
	if err != nil {
		return nil, err
	}

	dtos := make([]DashboardPinDTO, 0, len(pins))
	for _, pin := range pins {
		dtos = append(dtos, toDTO(pin))
	}

	return dtos, nil
}

func (s *DashboardPinsService) createPin(ctx context.Context, user *user.SignedInUser, cmd CreateDashboardPinCommand) (DashboardPinDTO, error) {
	if err := validateCreateCommand(cmd); err != nil {
		return DashboardPinDTO{}, err
	}

	dashboardUID := strings.TrimSpace(cmd.DashboardUID)
	if err := s.validateDashboardExists(ctx, user.OrgID, dashboardUID); err != nil {
		return DashboardPinDTO{}, err
	}

	var created DashboardPin
	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		count, err := session.Where("org_id = ? AND user_id = ?", user.OrgID, user.UserID).Count(&DashboardPin{})
		if err != nil {
			return err
		}
		if count >= MaxPins {
			return ErrPinLimitReached
		}

		var maxPin DashboardPin
		hasMax, err := session.Where("org_id = ? AND user_id = ?", user.OrgID, user.UserID).
			Desc("sort_order").
			Limit(1).
			Get(&maxPin)
		if err != nil {
			return err
		}

		nextSortOrder := 0
		if hasMax {
			nextSortOrder = maxPin.SortOrder + 1
		}

		pin := DashboardPin{
			OrgID:        user.OrgID,
			UserID:       user.UserID,
			DashboardUID: dashboardUID,
			Note:         normalizeNote(cmd.Note),
			SortOrder:    nextSortOrder,
			CreatedAt:    s.now().Unix(),
		}

		_, err = session.Insert(&pin)
		if err != nil {
			if s.store.GetDialect().IsUniqueConstraintViolation(err) {
				return ErrPinAlreadyExists
			}
			return err
		}

		created = pin
		return nil
	})
	if err != nil {
		return DashboardPinDTO{}, err
	}

	return toDTO(created), nil
}

func (s *DashboardPinsService) reorderPins(ctx context.Context, user *user.SignedInUser, cmd ReorderDashboardPinsCommand) ([]DashboardPinDTO, error) {
	dashboardUIDs, err := normalizeReorderCommand(cmd)
	if err != nil {
		return nil, err
	}

	err = s.store.WithTransactionalDbSession(ctx, func(session *db.Session) error {
		var existing []DashboardPin
		if err := session.Where("org_id = ? AND user_id = ?", user.OrgID, user.UserID).Find(&existing); err != nil {
			return err
		}

		if err := validateReorderMatchesExisting(existing, dashboardUIDs); err != nil {
			return err
		}

		for index, dashboardUID := range dashboardUIDs {
			_, err := session.Where("org_id = ? AND user_id = ? AND dashboard_uid = ?", user.OrgID, user.UserID, dashboardUID).
				Cols("sort_order").
				Update(&DashboardPin{SortOrder: index})
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.listPins(ctx, user)
}

func (s *DashboardPinsService) patchPin(ctx context.Context, user *user.SignedInUser, dashboardUID string, cmd PatchDashboardPinCommand) (DashboardPinDTO, error) {
	dashboardUID = strings.TrimSpace(dashboardUID)
	if err := validateDashboardUID(dashboardUID); err != nil {
		return DashboardPinDTO{}, err
	}
	if err := validateNote(cmd.Note); err != nil {
		return DashboardPinDTO{}, err
	}

	var updated DashboardPin
	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		has, err := session.Where("org_id = ? AND user_id = ? AND dashboard_uid = ?", user.OrgID, user.UserID, dashboardUID).Get(&updated)
		if err != nil {
			return err
		}
		if !has {
			return ErrPinNotFound
		}

		updated.Note = normalizeNote(cmd.Note)
		_, err = session.Where("org_id = ? AND user_id = ? AND dashboard_uid = ?", user.OrgID, user.UserID, dashboardUID).
			Cols("note").
			Update(&updated)
		return err
	})
	if err != nil {
		return DashboardPinDTO{}, err
	}

	return toDTO(updated), nil
}

func (s *DashboardPinsService) deletePin(ctx context.Context, user *user.SignedInUser, dashboardUID string) error {
	dashboardUID = strings.TrimSpace(dashboardUID)
	if err := validateDashboardUID(dashboardUID); err != nil {
		return err
	}

	return s.store.WithDbSession(ctx, func(session *db.Session) error {
		count, err := session.Where("org_id = ? AND user_id = ? AND dashboard_uid = ?", user.OrgID, user.UserID, dashboardUID).Delete(&DashboardPin{})
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrPinNotFound
		}
		return nil
	})
}

func (s *DashboardPinsService) validateDashboardExists(ctx context.Context, orgID int64, dashboardUID string) error {
	_, err := s.dashboardService.GetDashboard(ctx, &dashboards.GetDashboardQuery{
		UID:   dashboardUID,
		OrgID: orgID,
	})
	if err != nil {
		if errors.Is(err, dashboards.ErrDashboardNotFound) {
			return ErrDashboardNotFound
		}
		return err
	}
	return nil
}

func validateCreateCommand(cmd CreateDashboardPinCommand) error {
	dashboardUID := strings.TrimSpace(cmd.DashboardUID)
	if dashboardUID == "" {
		return ErrDashboardUIDRequired
	}
	if err := validateDashboardUID(dashboardUID); err != nil {
		return err
	}
	return validateNote(cmd.Note)
}

func normalizeReorderCommand(cmd ReorderDashboardPinsCommand) ([]string, error) {
	if cmd.DashboardUIDs == nil {
		return nil, ErrInvalidReorder
	}

	dashboardUIDs := make([]string, 0, len(cmd.DashboardUIDs))
	seen := make(map[string]struct{}, len(cmd.DashboardUIDs))
	for _, dashboardUID := range cmd.DashboardUIDs {
		trimmed := strings.TrimSpace(dashboardUID)
		if trimmed == "" {
			return nil, ErrDashboardUIDRequired
		}
		if err := validateDashboardUID(trimmed); err != nil {
			return nil, err
		}
		if _, ok := seen[trimmed]; ok {
			return nil, ErrInvalidReorder
		}
		seen[trimmed] = struct{}{}
		dashboardUIDs = append(dashboardUIDs, trimmed)
	}

	return dashboardUIDs, nil
}

func validateReorderMatchesExisting(existing []DashboardPin, dashboardUIDs []string) error {
	if len(existing) != len(dashboardUIDs) {
		return ErrInvalidReorder
	}

	existingUIDs := make(map[string]struct{}, len(existing))
	for _, pin := range existing {
		existingUIDs[pin.DashboardUID] = struct{}{}
	}

	for _, dashboardUID := range dashboardUIDs {
		if _, ok := existingUIDs[dashboardUID]; !ok {
			return ErrInvalidReorder
		}
	}

	return nil
}

func validateDashboardUID(dashboardUID string) error {
	if strings.TrimSpace(dashboardUID) == "" {
		return ErrDashboardUIDRequired
	}
	if !util.IsValidShortUID(dashboardUID) {
		return ErrInvalidDashboardUID
	}
	if util.IsShortUIDTooLong(dashboardUID) {
		return ErrInvalidDashboardUID
	}
	return nil
}

func validateNote(note *string) error {
	if note == nil {
		return nil
	}
	if len(*note) > MaxNoteLength {
		return ErrNoteTooLong
	}
	return nil
}

func normalizeNote(note *string) *string {
	if note == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*note)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func toDTO(pin DashboardPin) DashboardPinDTO {
	return DashboardPinDTO{
		DashboardUID: pin.DashboardUID,
		Note:         pin.Note,
		SortOrder:    pin.SortOrder,
		CreatedAt:    pin.CreatedAt,
	}
}
