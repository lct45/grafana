package recentitems

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
)

func (s *RecentItemsService) createItem(ctx context.Context, user *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	if err := validateCreateCommand(cmd); err != nil {
		return RecentItemDTO{}, false, err
	}

	now := s.now().UnixMilli()
	resourceType := strings.TrimSpace(cmd.ResourceType)
	resourceUID := strings.TrimSpace(cmd.ResourceUID)
	title := strings.TrimSpace(cmd.Title)
	itemURL := strings.TrimSpace(cmd.URL)

	var (
		result  RecentItem
		created bool
	)

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
			created = false
			return nil
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

		if _, err := session.Insert(&item); err != nil {
			return err
		}

		count, err := session.Where("org_id = ? AND user_id = ?", user.OrgID, user.UserID).Count(RecentItem{})
		if err != nil {
			return err
		}
		if count > MaxStoredItems {
			overflow := int(count - MaxStoredItems)
			var oldest []RecentItem
			err = session.Where("org_id = ? AND user_id = ?", user.OrgID, user.UserID).
				Asc("last_viewed_at").
				Limit(overflow).
				Find(&oldest)
			if err != nil {
				return err
			}
			for _, old := range oldest {
				if _, err := session.ID(old.ID).Delete(RecentItem{}); err != nil {
					return err
				}
			}
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

func (s *RecentItemsService) listItems(ctx context.Context, user *user.SignedInUser, limit int) ([]RecentItemDTO, error) {
	if limit <= 0 {
		limit = DefaultListLimit
	}
	if limit > MaxListLimit {
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

func (s *RecentItemsService) patchItem(ctx context.Context, user *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	if err := validatePatchCommand(cmd); err != nil {
		return RecentItemDTO{}, err
	}

	var updated RecentItem
	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		has, err := session.Where("org_id = ? AND user_id = ? AND uid = ?", user.OrgID, user.UserID, uid).Get(&updated)
		if err != nil {
			return err
		}
		if !has {
			return ErrRecentItemNotFound
		}

		cols := make([]string, 0, 3)
		if cmd.Title != nil {
			updated.Title = strings.TrimSpace(*cmd.Title)
			cols = append(cols, "title")
		}
		if cmd.URL != nil {
			updated.URL = strings.TrimSpace(*cmd.URL)
			cols = append(cols, "url")
		}
		if cmd.LastViewedAt != nil {
			updated.LastViewedAt = *cmd.LastViewedAt
			cols = append(cols, "last_viewed_at")
		}

		_, err = session.ID(updated.ID).Cols(cols...).Update(&updated)
		return err
	})
	if err != nil {
		return RecentItemDTO{}, err
	}

	return toDTO(updated), nil
}

func (s *RecentItemsService) deleteItem(ctx context.Context, user *user.SignedInUser, uid string) error {
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
		return ErrInvalidResourceType
	}

	if strings.TrimSpace(cmd.ResourceUID) == "" {
		return ErrResourceUIDRequired
	}
	if len(strings.TrimSpace(cmd.ResourceUID)) > 40 {
		return fmt.Errorf("resourceUid must be at most 40 characters")
	}

	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return ErrTitleRequired
	}
	if len(title) > MaxTitleLength {
		return fmt.Errorf("title must be at most %d characters", MaxTitleLength)
	}

	itemURL := strings.TrimSpace(cmd.URL)
	if itemURL == "" {
		return ErrURLRequired
	}
	if len(itemURL) > MaxURLLength {
		return fmt.Errorf("url must be at most %d characters", MaxURLLength)
	}

	return nil
}

func validatePatchCommand(cmd PatchRecentItemCommand) error {
	if cmd.Title == nil && cmd.URL == nil && cmd.LastViewedAt == nil {
		return ErrEmptyPatch
	}
	if cmd.Title != nil {
		title := strings.TrimSpace(*cmd.Title)
		if title == "" {
			return ErrTitleRequired
		}
		if len(title) > MaxTitleLength {
			return fmt.Errorf("title must be at most %d characters", MaxTitleLength)
		}
	}
	if cmd.URL != nil {
		itemURL := strings.TrimSpace(*cmd.URL)
		if itemURL == "" {
			return ErrURLRequired
		}
		if len(itemURL) > MaxURLLength {
			return fmt.Errorf("url must be at most %d characters", MaxURLLength)
		}
	}
	return nil
}
