package recentitems

import (
	"context"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
)

type sqlItemStore struct {
	db db.DB
}

func (s *sqlItemStore) record(ctx context.Context, usr *user.SignedInUser, cmd CreateRecentItemCommand, viewedAt int64) (RecentItemDTO, bool, error) {
	var (
		dto     RecentItemDTO
		created bool
	)

	err := s.db.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		existing, found, err := getByResource(sess, usr, cmd.ResourceType, cmd.ResourceUID)
		if err != nil {
			return err
		}
		if found {
			existing.Title = cmd.Title
			existing.URL = cmd.URL
			existing.LastViewedAt = viewedAt
			if _, err = sess.ID(existing.ID).Cols("title", "url", "last_viewed_at").Update(&existing); err != nil {
				return err
			}
			dto = toDTO(existing)
			created = false
			return nil
		}

		item := UserRecentItem{
			UID:          util.GenerateShortUID(),
			OrgID:        usr.OrgID,
			UserID:       usr.UserID,
			ResourceType: cmd.ResourceType,
			ResourceUID:  cmd.ResourceUID,
			Title:        cmd.Title,
			URL:          cmd.URL,
			LastViewedAt: viewedAt,
		}
		if _, err = sess.Insert(&item); err != nil {
			if !s.db.GetDialect().IsUniqueConstraintViolation(err) {
				return err
			}
			insertErr := err
			existing, found, err = getByResource(sess, usr, cmd.ResourceType, cmd.ResourceUID)
			if err != nil {
				return err
			}
			if !found {
				return insertErr
			}
			existing.Title = cmd.Title
			existing.URL = cmd.URL
			existing.LastViewedAt = viewedAt
			if _, err = sess.ID(existing.ID).Cols("title", "url", "last_viewed_at").Update(&existing); err != nil {
				return err
			}
			dto = toDTO(existing)
			created = false
			return nil
		}

		if err = pruneOldest(sess, usr); err != nil {
			return err
		}
		dto = toDTO(item)
		created = true
		return nil
	})
	return dto, created, err
}

func (s *sqlItemStore) list(ctx context.Context, usr *user.SignedInUser, limit int) ([]RecentItemDTO, error) {
	var items []UserRecentItem
	err := s.db.WithDbSession(ctx, func(sess *db.Session) error {
		return sess.Where("org_id = ? AND user_id = ?", usr.OrgID, usr.UserID).
			Desc("last_viewed_at", "id").
			Limit(limit).
			Find(&items)
	})
	if err != nil {
		return nil, err
	}

	dtos := make([]RecentItemDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, toDTO(item))
	}
	return dtos, nil
}

func (s *sqlItemStore) patch(ctx context.Context, usr *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	var dto RecentItemDTO
	err := s.db.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		item, found, err := getByUID(sess, usr, uid)
		if err != nil {
			return err
		}
		if !found {
			return ErrRecentItemNotFound
		}

		cols := make([]string, 0, 2)
		if cmd.Title != nil {
			item.Title = *cmd.Title
			cols = append(cols, "title")
		}
		if cmd.URL != nil {
			item.URL = *cmd.URL
			cols = append(cols, "url")
		}
		if _, err = sess.ID(item.ID).Cols(cols...).Update(&item); err != nil {
			return err
		}
		dto = toDTO(item)
		return nil
	})
	return dto, err
}

func (s *sqlItemStore) delete(ctx context.Context, usr *user.SignedInUser, uid string) error {
	return s.db.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		count, err := sess.Where("org_id = ? AND user_id = ? AND uid = ?", usr.OrgID, usr.UserID, uid).Delete(&UserRecentItem{})
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrRecentItemNotFound
		}
		return nil
	})
}

func getByResource(sess *db.Session, usr *user.SignedInUser, resourceType, resourceUID string) (UserRecentItem, bool, error) {
	var item UserRecentItem
	found, err := sess.Where(
		"org_id = ? AND user_id = ? AND resource_type = ? AND resource_uid = ?",
		usr.OrgID, usr.UserID, resourceType, resourceUID,
	).Get(&item)
	return item, found, err
}

func getByUID(sess *db.Session, usr *user.SignedInUser, uid string) (UserRecentItem, bool, error) {
	var item UserRecentItem
	found, err := sess.Where("org_id = ? AND user_id = ? AND uid = ?", usr.OrgID, usr.UserID, uid).Get(&item)
	return item, found, err
}

func pruneOldest(sess *db.Session, usr *user.SignedInUser) error {
	var items []UserRecentItem
	err := sess.Where("org_id = ? AND user_id = ?", usr.OrgID, usr.UserID).
		Desc("last_viewed_at", "id").
		Find(&items)
	if err != nil {
		return err
	}
	if len(items) <= maxRecentItemsPerUser {
		return nil
	}

	stale := make([]int64, 0, len(items)-maxRecentItemsPerUser)
	for _, item := range items[maxRecentItemsPerUser:] {
		stale = append(stale, item.ID)
	}
	_, err = sess.Where("org_id = ? AND user_id = ?", usr.OrgID, usr.UserID).In("id", stale).Delete(&UserRecentItem{})
	return err
}
