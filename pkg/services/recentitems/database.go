package recentitems

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
)

var validResourceTypes = map[string]struct{}{
	"dashboard":  {},
	"alert-rule": {},
	"folder":     {},
	"datasource": {},
	"explore":    {},
}

func (s *RecentItemsService) upsert(ctx context.Context, signedInUser *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	if err := validateCreateCommand(cmd); err != nil {
		return RecentItemDTO{}, false, err
	}

	item, created, err := s.upsertInTransaction(ctx, signedInUser, cmd)
	if err == nil {
		return toDTO(item), created, nil
	}
	if !s.store.GetDialect().IsUniqueConstraintViolation(err) {
		return RecentItemDTO{}, false, err
	}

	item, err = s.updateExistingInTransaction(ctx, signedInUser, cmd)
	if err != nil {
		return RecentItemDTO{}, false, err
	}
	return toDTO(item), false, nil
}

func (s *RecentItemsService) upsertInTransaction(ctx context.Context, signedInUser *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItem, bool, error) {
	var item RecentItem
	created := false
	err := s.store.WithTransactionalDbSession(ctx, func(session *db.Session) error {
		found, err := getByResource(session, signedInUser, cmd.ResourceType, cmd.ResourceUID, &item)
		if err != nil {
			return err
		}

		if found {
			updateItem(&item, cmd, s.now().Unix())
			if _, err := session.ID(item.ID).Cols("title", "url", "last_viewed_at").Update(&item); err != nil {
				return err
			}
		} else {
			item = RecentItem{
				UID:          util.GenerateShortUID(),
				OrgID:        signedInUser.OrgID,
				UserID:       signedInUser.UserID,
				ResourceType: cmd.ResourceType,
				ResourceUID:  strings.TrimSpace(cmd.ResourceUID),
			}
			updateItem(&item, cmd, s.now().Unix())
			if _, err := session.Insert(&item); err != nil {
				return err
			}
			created = true
		}

		return trimRecentItems(session, signedInUser)
	})
	return item, created, err
}

func (s *RecentItemsService) updateExistingInTransaction(ctx context.Context, signedInUser *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItem, error) {
	var item RecentItem
	err := s.store.WithTransactionalDbSession(ctx, func(session *db.Session) error {
		found, err := getByResource(session, signedInUser, cmd.ResourceType, cmd.ResourceUID, &item)
		if err != nil {
			return err
		}
		if !found {
			return ErrItemNotFound
		}

		updateItem(&item, cmd, s.now().Unix())
		if _, err := session.ID(item.ID).Cols("title", "url", "last_viewed_at").Update(&item); err != nil {
			return err
		}
		return trimRecentItems(session, signedInUser)
	})
	return item, err
}

func getByResource(session *db.Session, signedInUser *user.SignedInUser, resourceType, resourceUID string, item *RecentItem) (bool, error) {
	return session.Where(
		"org_id = ? AND user_id = ? AND resource_type = ? AND resource_uid = ?",
		signedInUser.OrgID, signedInUser.UserID, resourceType, strings.TrimSpace(resourceUID),
	).Get(item)
}

func updateItem(item *RecentItem, cmd CreateRecentItemCommand, timestamp int64) {
	item.Title = strings.TrimSpace(cmd.Title)
	item.URL = strings.TrimSpace(cmd.URL)
	item.LastViewedAt = timestamp
}

func (s *RecentItemsService) list(ctx context.Context, signedInUser *user.SignedInUser, limit int, resourceType string) ([]RecentItemDTO, error) {
	if limit < 1 || limit > MaxLimit {
		return nil, ErrInvalidLimit
	}
	if resourceType != "" && !isValidResourceType(resourceType) {
		return nil, ErrInvalidResourceType
	}

	items := make([]RecentItem, 0, limit)
	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		query := session.Where("org_id = ? AND user_id = ?", signedInUser.OrgID, signedInUser.UserID)
		if resourceType != "" {
			query = query.And("resource_type = ?", resourceType)
		}
		return query.Desc("last_viewed_at", "id").Limit(limit).Find(&items)
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

func (s *RecentItemsService) patch(ctx context.Context, signedInUser *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	if !isValidItemUID(uid) {
		return RecentItemDTO{}, ErrItemNotFound
	}
	if err := validatePatchCommand(cmd); err != nil {
		return RecentItemDTO{}, err
	}

	var item RecentItem
	err := s.store.WithTransactionalDbSession(ctx, func(session *db.Session) error {
		found, err := session.Where(
			"uid = ? AND org_id = ? AND user_id = ?",
			uid, signedInUser.OrgID, signedInUser.UserID,
		).Get(&item)
		if err != nil {
			return err
		}
		if !found {
			return ErrItemNotFound
		}

		columns := make([]string, 0, 2)
		if cmd.Title != nil {
			item.Title = strings.TrimSpace(*cmd.Title)
			columns = append(columns, "title")
		}
		if cmd.URL != nil {
			item.URL = strings.TrimSpace(*cmd.URL)
			columns = append(columns, "url")
		}
		_, err = session.ID(item.ID).Cols(columns...).Update(&item)
		return err
	})
	if err != nil {
		return RecentItemDTO{}, err
	}
	return toDTO(item), nil
}

func (s *RecentItemsService) delete(ctx context.Context, signedInUser *user.SignedInUser, uid string) error {
	if !isValidItemUID(uid) {
		return ErrItemNotFound
	}

	return s.store.WithDbSession(ctx, func(session *db.Session) error {
		count, err := session.Where(
			"uid = ? AND org_id = ? AND user_id = ?",
			uid, signedInUser.OrgID, signedInUser.UserID,
		).Delete(&RecentItem{})
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrItemNotFound
		}
		return nil
	})
}

func trimRecentItems(session *db.Session, signedInUser *user.SignedInUser) error {
	for {
		items := make([]RecentItem, 0, MaxLimit)
		if err := session.Where("org_id = ? AND user_id = ?", signedInUser.OrgID, signedInUser.UserID).
			Desc("last_viewed_at", "id").
			Limit(MaxLimit, MaxLimit).
			Find(&items); err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}

		ids := make([]int64, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		if _, err := session.In("id", ids).Delete(&RecentItem{}); err != nil {
			return err
		}
	}
}

func validateCreateCommand(cmd CreateRecentItemCommand) error {
	if !isValidResourceType(cmd.ResourceType) {
		return ErrInvalidResourceType
	}
	resourceUID := strings.TrimSpace(cmd.ResourceUID)
	if resourceUID == "" || len(resourceUID) > maxResourceUIDLength {
		return ErrInvalidResourceUID
	}
	if err := validateTitle(cmd.Title); err != nil {
		return err
	}
	return validateURL(cmd.URL)
}

func validatePatchCommand(cmd PatchRecentItemCommand) error {
	if cmd.Title == nil && cmd.URL == nil {
		return ErrEmptyPatch
	}
	if cmd.Title != nil {
		if err := validateTitle(*cmd.Title); err != nil {
			return err
		}
	}
	if cmd.URL != nil {
		if err := validateURL(*cmd.URL); err != nil {
			return err
		}
	}
	return nil
}

func validateTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" || len(title) > maxTitleLength {
		return ErrInvalidTitle
	}
	return nil
}

func validateURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || len(rawURL) > maxURLLength || !strings.HasPrefix(rawURL, "/") ||
		strings.HasPrefix(rawURL, "//") || strings.Contains(rawURL, "\\") {
		return ErrInvalidURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return ErrInvalidURL
	}
	return nil
}

func isValidResourceType(resourceType string) bool {
	_, ok := validResourceTypes[resourceType]
	return ok
}

func isValidItemUID(uid string) bool {
	return uid != "" && util.IsValidShortUID(uid) && !util.IsShortUIDTooLong(uid)
}

func toDTO(item RecentItem) RecentItemDTO {
	return RecentItemDTO{
		UID:          item.UID,
		ResourceType: item.ResourceType,
		ResourceUID:  item.ResourceUID,
		Title:        item.Title,
		URL:          item.URL,
		LastViewedAt: time.Unix(item.LastViewedAt, 0).UTC(),
	}
}
