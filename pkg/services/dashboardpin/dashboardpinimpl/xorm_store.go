package dashboardpinimpl

import (
	"context"
	"time"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/dashboardpin"
)

type sqlStore struct {
	db db.DB
}

func (s *sqlStore) List(ctx context.Context, query *dashboardpin.ListPinsQuery) ([]dashboardpin.DashboardPin, error) {
	var pins []dashboardpin.DashboardPin
	err := s.db.WithDbSession(ctx, func(sess *db.Session) error {
		return sess.Where("user_id = ? AND org_id = ?", query.UserID, query.OrgID).
			OrderBy("sort_order ASC, id ASC").
			Find(&pins)
	})
	return pins, err
}

func (s *sqlStore) Get(ctx context.Context, userID, orgID int64, dashboardUID string) (*dashboardpin.DashboardPin, error) {
	var pin dashboardpin.DashboardPin
	err := s.db.WithDbSession(ctx, func(sess *db.Session) error {
		has, err := sess.Where("user_id = ? AND org_id = ? AND dashboard_uid = ?", userID, orgID, dashboardUID).Get(&pin)
		if err != nil {
			return err
		}
		if !has {
			return dashboardpin.ErrPinNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pin, nil
}

func (s *sqlStore) Insert(ctx context.Context, cmd *dashboardpin.PinDashboardCommand, sortOrder int) (*dashboardpin.DashboardPin, error) {
	now := time.Now()
	pin := dashboardpin.DashboardPin{
		OrgID:        cmd.OrgID,
		UserID:       cmd.UserID,
		DashboardUID: cmd.DashboardUID,
		SortOrder:    sortOrder,
		Note:         cmd.Note,
		Created:      now,
		Updated:      now,
	}

	err := s.db.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		_, err := sess.Insert(&pin)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &pin, nil
}

func (s *sqlStore) Delete(ctx context.Context, cmd *dashboardpin.UnpinDashboardCommand) error {
	return s.db.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		result, err := sess.Exec(
			"DELETE FROM dashboard_pin WHERE user_id = ? AND org_id = ? AND dashboard_uid = ?",
			cmd.UserID, cmd.OrgID, cmd.DashboardUID,
		)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return dashboardpin.ErrPinNotFound
		}
		return nil
	})
}

func (s *sqlStore) UpdateNote(ctx context.Context, cmd *dashboardpin.UpdatePinNoteCommand) (*dashboardpin.DashboardPin, error) {
	var pin dashboardpin.DashboardPin
	err := s.db.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		has, err := sess.Where("user_id = ? AND org_id = ? AND dashboard_uid = ?", cmd.UserID, cmd.OrgID, cmd.DashboardUID).Get(&pin)
		if err != nil {
			return err
		}
		if !has {
			return dashboardpin.ErrPinNotFound
		}

		pin.Note = cmd.Note
		pin.Updated = time.Now()
		_, err = sess.ID(pin.ID).Cols("note", "updated").Update(&pin)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &pin, nil
}

func (s *sqlStore) Reorder(ctx context.Context, cmd *dashboardpin.ReorderPinsCommand) error {
	return s.db.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		var existing []dashboardpin.DashboardPin
		if err := sess.Where("user_id = ? AND org_id = ?", cmd.UserID, cmd.OrgID).Find(&existing); err != nil {
			return err
		}

		if len(existing) != len(cmd.DashboardUIDs) {
			return dashboardpin.ErrInvalidReorder
		}

		existingUIDs := make(map[string]bool, len(existing))
		for _, pin := range existing {
			existingUIDs[pin.DashboardUID] = true
		}
		for _, uid := range cmd.DashboardUIDs {
			if !existingUIDs[uid] {
				return dashboardpin.ErrInvalidReorder
			}
		}

		now := time.Now()
		for i, uid := range cmd.DashboardUIDs {
			_, err := sess.Exec(
				"UPDATE dashboard_pin SET sort_order = ?, updated = ? WHERE user_id = ? AND org_id = ? AND dashboard_uid = ?",
				i, now, cmd.UserID, cmd.OrgID, uid,
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *sqlStore) DeleteByUser(ctx context.Context, userID int64) error {
	return s.db.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		_, err := sess.Exec("DELETE FROM dashboard_pin WHERE user_id = ?", userID)
		return err
	})
}
