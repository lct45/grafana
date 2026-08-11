package recentitems

import (
	"context"
	"strings"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
)

func (s *RecentItemsService) createOrUpdateRecentItem(ctx context.Context, user *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	if err := validateCreateCommand(cmd); err != nil {
		return RecentItemDTO{}, false, err
	}

	resourceType := strings.TrimSpace(cmd.ResourceType)
	resourceUID := strings.TrimSpace(cmd.ResourceUID)
	title := strings.TrimSpace(cmd.Title)
	itemURL := strings.TrimSpace(cmd.URL)
	now := s.now().Unix()

	var result RecentItem
	created := false

	err := s.store.WithTransactionalDbSession(ctx, func(session *db.Session) error {
		var existing RecentItem
		has, err := session.Where(
			"org_id = ? AND user_id = ? AND resource_type = ? AND resource_uid = ?",
			user.OrgID, user.UserID, resourceType, resourceUID,
		).Get(&existing)
		if err != nil {
			return err
		}

		if has {
			existing.Title = title
			existing.URL = itemURL
			existing.LastViewedAt = now
			_, err = session.ID(existing.ID).Cols("title", "url", "last_viewed_at").Update(&existing)
			if err != nil {
				return err
			}
			result = existing
			return nil
		}

		count, err := session.Where("org_id = ? AND user_id = ?", user.OrgID, user.UserID).Count(RecentItem{})
		if err != nil {
			return err
		}
		if count >= maxStoredItems {
			var oldest RecentItem
			hasOldest, err := session.Where("org_id = ? AND user_id = ?", user.OrgID, user.UserID).
				Asc("last_viewed_at").
				Limit(1).
				Get(&oldest)
			if err != nil {
				return err
			}
			if hasOldest {
				_, err = session.ID(oldest.ID).Delete(RecentItem{})
				if err != nil {
					return err
				}
			}
		}

		item := RecentItem{
			UID:          util.GenerateShortUID(),
			OrgID:        user.OrgID,
			UserID:       user.UserID,
			ResourceType: resourceType,
			ResourceUID:  resourceUID,
			Title:        title,
			URL:          itemURL,
			LastViewedAt: now,
			CreatedAt:    now,
		}
		_, err = session.Insert(&item)
		if err != nil {
			return err
		}
		result = item
		created = true
		return nil
	})
	if err != nil {
		return RecentItemDTO{}, false, err
	}

	return toDTO(result), created, nil
}

func (s *RecentItemsService) listRecentItems(ctx context.Context, user *user.SignedInUser, limit int) ([]RecentItemDTO, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		return nil, ErrInvalidLimit
	}

	var items []RecentItem
	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		return session.Where("org_id = ? AND user_id = ?", user.OrgID, user.UserID).
			Desc("last_viewed_at").
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

func (s *RecentItemsService) patchRecentItem(ctx context.Context, user *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	if err := validatePatchCommand(cmd); err != nil {
		return RecentItemDTO{}, err
	}

	var result RecentItem
	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		has, err := session.Where("org_id = ? AND user_id = ? AND uid = ?", user.OrgID, user.UserID, uid).Get(&result)
		if err != nil {
			return err
		}
		if !has {
			return ErrRecentItemNotFound
		}

		cols := make([]string, 0, 3)
		if cmd.Title != nil {
			result.Title = strings.TrimSpace(*cmd.Title)
			cols = append(cols, "title")
		}
		if cmd.URL != nil {
			result.URL = strings.TrimSpace(*cmd.URL)
			cols = append(cols, "url")
		}
		if cmd.LastViewedAt != nil {
			result.LastViewedAt = *cmd.LastViewedAt
			cols = append(cols, "last_viewed_at")
		}

		_, err = session.ID(result.ID).Cols(cols...).Update(&result)
		return err
	})
	if err != nil {
		return RecentItemDTO{}, err
	}

	return toDTO(result), nil
}

func (s *RecentItemsService) deleteRecentItem(ctx context.Context, user *user.SignedInUser, uid string) error {
	return s.store.WithDbSession(ctx, func(session *db.Session) error {
		count, err := session.Where("org_id = ? AND user_id = ? AND uid = ?", user.OrgID, user.UserID, uid).Delete(RecentItem{})
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrRecentItemNotFound
		}
		return nil
	})
}

func validateCreateCommand(cmd CreateRecentItemCommand) error {
	resourceType := strings.TrimSpace(cmd.ResourceType)
	if resourceType == "" {
		return ErrResourceTypeRequired
	}
	if _, ok := allowedResourceTypes[resourceType]; !ok {
		return ErrResourceTypeInvalid
	}

	resourceUID := strings.TrimSpace(cmd.ResourceUID)
	if resourceUID == "" {
		return ErrResourceUIDRequired
	}
	if len(resourceUID) > 40 {
		return ErrResourceUIDTooLong
	}

	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return ErrTitleRequired
	}
	if len(title) > 255 {
		return ErrTitleTooLong
	}

	itemURL := strings.TrimSpace(cmd.URL)
	if itemURL == "" {
		return ErrURLRequired
	}
	if len(itemURL) > 1024 {
		return ErrURLTooLong
	}

	return nil
}

func validatePatchCommand(cmd PatchRecentItemCommand) error {
	if cmd.Title == nil && cmd.URL == nil && cmd.LastViewedAt == nil {
		return ErrPatchEmpty
	}
	if cmd.Title != nil {
		title := strings.TrimSpace(*cmd.Title)
		if title == "" {
			return ErrTitleRequired
		}
		if len(title) > 255 {
			return ErrTitleTooLong
		}
	}
	if cmd.URL != nil {
		itemURL := strings.TrimSpace(*cmd.URL)
		if itemURL == "" {
			return ErrURLRequired
		}
		if len(itemURL) > 1024 {
			return ErrURLTooLong
		}
	}
	return nil
}

func toDTO(item RecentItem) RecentItemDTO {
	return RecentItemDTO{
		UID:          item.UID,
		ResourceType: item.ResourceType,
		ResourceUID:  item.ResourceUID,
		Title:        item.Title,
		URL:          item.URL,
		LastViewedAt: item.LastViewedAt,
		CreatedAt:    item.CreatedAt,
	}
}
