package recentitems

import (
	"context"
	"strings"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
)

func (s *ServiceImpl) upsertRecentItem(ctx context.Context, signedInUser *user.SignedInUser, cmd CreateRecentItemCommand) (UpsertRecentItemResult, error) {
	cmd.ResourceType = strings.TrimSpace(cmd.ResourceType)
	cmd.ResourceUID = strings.TrimSpace(cmd.ResourceUID)
	cmd.Title = strings.TrimSpace(cmd.Title)
	cmd.URL = strings.TrimSpace(cmd.URL)
	if err := validateCreateCommand(cmd); err != nil {
		return UpsertRecentItemResult{}, err
	}

	var item RecentItem
	created := false
	err := s.store.WithTransactionalDbSession(ctx, func(session *db.Session) error {
		exists, err := session.Where(
			"user_id = ? AND org_id = ? AND resource_type = ? AND resource_uid = ?",
			signedInUser.UserID,
			signedInUser.OrgID,
			cmd.ResourceType,
			cmd.ResourceUID,
		).Get(&item)
		if err != nil {
			return err
		}

		if exists {
			item.Title = cmd.Title
			item.URL = cmd.URL
			item.LastViewedAt = s.now().Unix()
			_, err = session.ID(item.ID).Cols("title", "url", "last_viewed_at").Update(&item)
			if err != nil {
				return err
			}
		} else {
			item = RecentItem{
				UID:          util.GenerateShortUID(),
				OrgID:        signedInUser.OrgID,
				UserID:       signedInUser.UserID,
				ResourceType: cmd.ResourceType,
				ResourceUID:  cmd.ResourceUID,
				Title:        cmd.Title,
				URL:          cmd.URL,
				LastViewedAt: s.now().Unix(),
			}
			if _, err = session.Insert(&item); err != nil {
				return err
			}
			created = true
		}

		return enforceRecentItemLimit(session, signedInUser)
	})
	if err != nil {
		return UpsertRecentItemResult{}, err
	}

	return UpsertRecentItemResult{Item: toDTO(item), Created: created}, nil
}

func (s *ServiceImpl) listRecentItems(ctx context.Context, signedInUser *user.SignedInUser, query ListRecentItemsQuery) ([]RecentItemDTO, error) {
	query.ResourceType = strings.TrimSpace(query.ResourceType)
	if query.ResourceType != "" && !isValidResourceType(query.ResourceType) {
		return nil, ErrInvalidResourceType
	}
	if query.Limit <= 0 || query.Limit > DefaultLimit {
		query.Limit = DefaultLimit
	}

	var items []RecentItem
	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		dbQuery := session.Where("user_id = ? AND org_id = ?", signedInUser.UserID, signedInUser.OrgID)
		if query.ResourceType != "" {
			dbQuery = dbQuery.And("resource_type = ?", query.ResourceType)
		}
		return dbQuery.Desc("last_viewed_at", "id").Limit(query.Limit).Find(&items)
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

func (s *ServiceImpl) patchRecentItem(ctx context.Context, signedInUser *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	if err := validatePatchCommand(cmd); err != nil {
		return RecentItemDTO{}, err
	}

	var item RecentItem
	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		exists, err := session.Where(
			"user_id = ? AND org_id = ? AND uid = ?",
			signedInUser.UserID,
			signedInUser.OrgID,
			uid,
		).Get(&item)
		if err != nil {
			return err
		}
		if !exists {
			return ErrRecentItemNotFound
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

func (s *ServiceImpl) deleteRecentItem(ctx context.Context, signedInUser *user.SignedInUser, uid string) error {
	return s.store.WithDbSession(ctx, func(session *db.Session) error {
		count, err := session.Where(
			"user_id = ? AND org_id = ? AND uid = ?",
			signedInUser.UserID,
			signedInUser.OrgID,
			uid,
		).Delete(RecentItem{})
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrRecentItemNotFound
		}
		return nil
	})
}

func enforceRecentItemLimit(session *db.Session, signedInUser *user.SignedInUser) error {
	var count int64
	count, err := session.Where("user_id = ? AND org_id = ?", signedInUser.UserID, signedInUser.OrgID).Count(RecentItem{})
	if err != nil || count <= DefaultLimit {
		return err
	}

	var staleItems []RecentItem
	err = session.Where("user_id = ? AND org_id = ?", signedInUser.UserID, signedInUser.OrgID).
		Asc("last_viewed_at", "id").
		Limit(int(count) - DefaultLimit).
		Find(&staleItems)
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(staleItems))
	for _, item := range staleItems {
		ids = append(ids, item.ID)
	}
	if len(ids) == 0 {
		return nil
	}
	_, err = session.In("id", ids).Delete(RecentItem{})
	return err
}

func validateCreateCommand(cmd CreateRecentItemCommand) error {
	if !isValidResourceType(cmd.ResourceType) {
		return ErrInvalidResourceType
	}
	if cmd.ResourceUID == "" {
		return ErrResourceUIDRequired
	}
	if len(cmd.ResourceUID) > MaxResourceUIDLen {
		return ErrResourceUIDTooLong
	}
	if len(cmd.Title) > MaxTitleLen {
		return ErrTitleTooLong
	}
	if len(cmd.URL) > MaxURLLen {
		return ErrURLTooLong
	}
	return nil
}

func validatePatchCommand(cmd PatchRecentItemCommand) error {
	if cmd.Title == nil && cmd.URL == nil {
		return ErrNoPatchFields
	}
	if cmd.Title != nil && len(strings.TrimSpace(*cmd.Title)) > MaxTitleLen {
		return ErrTitleTooLong
	}
	if cmd.URL != nil && len(strings.TrimSpace(*cmd.URL)) > MaxURLLen {
		return ErrURLTooLong
	}
	return nil
}

func isValidResourceType(resourceType string) bool {
	switch resourceType {
	case ResourceTypeAlert, ResourceTypeDash, ResourceTypeSource, ResourceTypeExplore, ResourceTypeFolder:
		return true
	default:
		return false
	}
}

func toDTO(item RecentItem) RecentItemDTO {
	return RecentItemDTO{
		UID:          item.UID,
		ResourceType: item.ResourceType,
		ResourceUID:  item.ResourceUID,
		Title:        item.Title,
		URL:          item.URL,
		LastViewedAt: item.LastViewedAt,
	}
}
