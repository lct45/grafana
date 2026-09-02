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
	"alertRule":  {},
	"folder":     {},
	"datasource": {},
	"explore":    {},
}

func (s *RecentItemsService) upsert(ctx context.Context, signedInUser *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	if err := validateCreateCommand(cmd); err != nil {
		return RecentItemDTO{}, false, err
	}

	var item RecentItem
	created := false
	err := s.store.WithTransactionalDbSession(ctx, func(session *db.Session) error {
		found, err := session.Where(
			"org_id = ? AND user_id = ? AND resource_type = ? AND resource_uid = ?",
			signedInUser.OrgID, signedInUser.UserID, cmd.ResourceType, cmd.ResourceUID,
		).Get(&item)
		if err != nil {
			return err
		}

		now := s.now().Unix()
		if found {
			item.Title = strings.TrimSpace(cmd.Title)
			item.URL = cmd.URL
			item.LastViewedAt = now
			if _, err := session.ID(item.ID).Cols("title", "url", "last_viewed_at").Update(&item); err != nil {
				return err
			}
		} else {
			item = RecentItem{
				UID:          util.GenerateShortUID(),
				OrgID:        signedInUser.OrgID,
				UserID:       signedInUser.UserID,
				ResourceType: cmd.ResourceType,
				ResourceUID:  cmd.ResourceUID,
				Title:        strings.TrimSpace(cmd.Title),
				URL:          cmd.URL,
				LastViewedAt: now,
			}
			if _, err := session.Insert(&item); err != nil {
				return err
			}
			created = true
		}

		return trimRecentItems(session, signedInUser)
	})
	if err != nil {
		return RecentItemDTO{}, false, err
	}

	return toDTO(item), created, nil
}

func (s *RecentItemsService) list(ctx context.Context, signedInUser *user.SignedInUser, limit int) ([]RecentItemDTO, error) {
	if limit < 1 || limit > MaxLimit {
		return nil, ErrInvalidLimit
	}

	items := make([]RecentItem, 0, limit)
	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		return session.Where("org_id = ? AND user_id = ?", signedInUser.OrgID, signedInUser.UserID).
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

		columns := make([]string, 0, 3)
		if cmd.Title != nil {
			item.Title = strings.TrimSpace(*cmd.Title)
			columns = append(columns, "title")
		}
		if cmd.URL != nil {
			item.URL = *cmd.URL
			columns = append(columns, "url")
		}
		if cmd.LastViewedAt != nil {
			item.LastViewedAt = cmd.LastViewedAt.Unix()
			columns = append(columns, "last_viewed_at")
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
	var items []RecentItem
	if err := session.Where("org_id = ? AND user_id = ?", signedInUser.OrgID, signedInUser.UserID).
		Desc("last_viewed_at", "id").
		Find(&items); err != nil {
		return err
	}
	if len(items) <= MaxLimit {
		return nil
	}

	ids := make([]int64, 0, len(items)-MaxLimit)
	for _, item := range items[MaxLimit:] {
		ids = append(ids, item.ID)
	}
	_, err := session.In("id", ids).Delete(&RecentItem{})
	return err
}

func validateCreateCommand(cmd CreateRecentItemCommand) error {
	if _, ok := validResourceTypes[cmd.ResourceType]; !ok {
		return ErrInvalidResourceType
	}
	if err := validateResourceUID(cmd.ResourceType, cmd.ResourceUID); err != nil {
		return err
	}
	if err := validateTitle(cmd.Title); err != nil {
		return err
	}
	return validateURL(cmd.URL)
}

func validatePatchCommand(cmd PatchRecentItemCommand) error {
	if cmd.Title == nil && cmd.URL == nil && cmd.LastViewedAt == nil {
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
	if cmd.LastViewedAt != nil && cmd.LastViewedAt.IsZero() {
		return ErrInvalidTimestamp
	}
	return nil
}

func validateResourceUID(resourceType, uid string) error {
	if uid == "" {
		return ErrInvalidResourceUID
	}
	if resourceType == "explore" {
		if len(uid) > 255 {
			return ErrInvalidResourceUID
		}
		return nil
	}
	if !util.IsValidShortUID(uid) || util.IsShortUIDTooLong(uid) {
		return ErrInvalidResourceUID
	}
	return nil
}

func validateTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" || len(title) > 255 {
		return ErrInvalidTitle
	}
	return nil
}

func validateURL(rawURL string) error {
	if rawURL == "" || len(rawURL) > 1024 || !strings.HasPrefix(rawURL, "/") {
		return ErrInvalidURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return ErrInvalidURL
	}
	return nil
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
