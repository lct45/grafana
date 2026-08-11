package homepins

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
)

const maxNoteLength = 255

func (s *HomePinsService) createPin(ctx context.Context, user *user.SignedInUser, cmd CreatePinCommand) (DashboardPinDTO, error) {
	dashboardUID := strings.TrimSpace(cmd.DashboardUID)
	if dashboardUID == "" {
		return DashboardPinDTO{}, fmt.Errorf("dashboardUid is required")
	}
	if err := validateNote(cmd.Note); err != nil {
		return DashboardPinDTO{}, err
	}

	now := s.now().Unix()
	pin := DashboardHomePin{
		UID:          util.GenerateShortUID(),
		OrgID:        user.OrgID,
		UserID:       user.UserID,
		DashboardUID: dashboardUID,
		Note:         strings.TrimSpace(cmd.Note),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err := s.store.WithTransactionalDbSession(ctx, func(session *db.Session) error {
		existing := DashboardHomePin{}
		has, err := session.Where("org_id = ? AND user_id = ? AND dashboard_uid = ?", user.OrgID, user.UserID, dashboardUID).Get(&existing)
		if err != nil {
			return err
		}
		if has {
			return ErrPinAlreadyExists
		}

		var maxSort int
		_, err = session.SQL("SELECT COALESCE(MAX(sort_order), -1) FROM dashboard_home_pin WHERE org_id = ? AND user_id = ?", user.OrgID, user.UserID).Get(&maxSort)
		if err != nil {
			return err
		}
		pin.SortOrder = maxSort + 1

		_, err = session.Insert(&pin)
		return err
	})
	if err != nil {
		return DashboardPinDTO{}, err
	}

	return toDTO(pin), nil
}

func (s *HomePinsService) listPins(ctx context.Context, user *user.SignedInUser) ([]DashboardPinDTO, error) {
	var pins []DashboardHomePin

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

func (s *HomePinsService) updatePin(ctx context.Context, user *user.SignedInUser, uid string, cmd UpdatePinCommand) (DashboardPinDTO, error) {
	if err := validateNote(cmd.Note); err != nil {
		return DashboardPinDTO{}, err
	}

	var updated DashboardHomePin
	err := s.store.WithTransactionalDbSession(ctx, func(session *db.Session) error {
		pin := DashboardHomePin{}
		has, err := session.Where("org_id = ? AND user_id = ? AND uid = ?", user.OrgID, user.UserID, uid).Get(&pin)
		if err != nil {
			return err
		}
		if !has {
			return ErrPinNotFound
		}

		pin.Note = strings.TrimSpace(cmd.Note)
		pin.UpdatedAt = s.now().Unix()

		_, err = session.ID(pin.ID).Cols("note", "updated_at").Update(&pin)
		if err != nil {
			return err
		}
		updated = pin
		return nil
	})
	if err != nil {
		return DashboardPinDTO{}, err
	}

	return toDTO(updated), nil
}

func (s *HomePinsService) reorderPins(ctx context.Context, user *user.SignedInUser, cmd ReorderPinsCommand) error {
	if len(cmd.UIDs) == 0 {
		return fmt.Errorf("uids are required")
	}

	return s.store.WithTransactionalDbSession(ctx, func(session *db.Session) error {
		var existing []DashboardHomePin
		if err := session.Where("org_id = ? AND user_id = ?", user.OrgID, user.UserID).Find(&existing); err != nil {
			return err
		}

		existingByUID := make(map[string]DashboardHomePin, len(existing))
		for _, pin := range existing {
			existingByUID[pin.UID] = pin
		}

		if len(cmd.UIDs) != len(existing) {
			return fmt.Errorf("uids must include all pins")
		}

		seen := make(map[string]struct{}, len(cmd.UIDs))
		now := s.now().Unix()
		for i, uid := range cmd.UIDs {
			if _, ok := seen[uid]; ok {
				return fmt.Errorf("duplicate uid in reorder request")
			}
			seen[uid] = struct{}{}

			pin, ok := existingByUID[uid]
			if !ok {
				return ErrPinNotFound
			}
			pin.SortOrder = i
			pin.UpdatedAt = now
			if _, err := session.ID(pin.ID).Cols("sort_order", "updated_at").Update(&pin); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *HomePinsService) deletePin(ctx context.Context, user *user.SignedInUser, uid string) error {
	return s.store.WithDbSession(ctx, func(session *db.Session) error {
		count, err := session.Where("org_id = ? AND user_id = ? AND uid = ?", user.OrgID, user.UserID, uid).Delete(DashboardHomePin{})
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrPinNotFound
		}
		return nil
	})
}

func validateNote(note string) error {
	if len(note) > maxNoteLength {
		return fmt.Errorf("note must be at most %d characters", maxNoteLength)
	}
	return nil
}

func toDTO(pin DashboardHomePin) DashboardPinDTO {
	return DashboardPinDTO{
		UID:          pin.UID,
		DashboardUID: pin.DashboardUID,
		SortOrder:    pin.SortOrder,
		Note:         pin.Note,
		CreatedAt:    pin.CreatedAt,
		UpdatedAt:    pin.UpdatedAt,
	}
}
