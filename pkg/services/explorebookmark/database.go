package explorebookmark

import (
	"context"
	"encoding/json"

	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
)

func (s *ExploreBookmarkService) createBookmark(ctx context.Context, user *user.SignedInUser, cmd CreateExploreBookmarkCommand) (ExploreBookmarkDTO, error) {
	now := s.now().Unix()
	queriesBytes, err := json.Marshal(cmd.Queries)
	if err != nil {
		return ExploreBookmarkDTO{}, err
	}

	bookmark := ExploreBookmark{
		UID:           util.GenerateShortUID(),
		OrgID:         user.OrgID,
		UserID:        user.UserID,
		Name:          cmd.Name,
		DatasourceUID: cmd.DatasourceUID,
		Queries:       string(queriesBytes),
		TimeFrom:      cmd.TimeFrom,
		TimeTo:        cmd.TimeTo,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err = s.store.WithDbSession(ctx, func(session *db.Session) error {
		_, err := session.Insert(&bookmark)
		return err
	})
	if err != nil {
		return ExploreBookmarkDTO{}, err
	}

	return toDTO(bookmark)
}

func (s *ExploreBookmarkService) listBookmarks(ctx context.Context, user *user.SignedInUser) ([]ExploreBookmarkDTO, error) {
	var bookmarks []ExploreBookmark

	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		return session.Where("org_id = ? AND user_id = ?", user.OrgID, user.UserID).
			OrderBy("updated_at DESC").
			Find(&bookmarks)
	})
	if err != nil {
		return nil, err
	}

	dtos := make([]ExploreBookmarkDTO, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		dto, err := toDTO(bookmark)
		if err != nil {
			return nil, err
		}
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

func (s *ExploreBookmarkService) deleteBookmark(ctx context.Context, user *user.SignedInUser, uid string) error {
	return s.store.WithDbSession(ctx, func(session *db.Session) error {
		count, err := session.Where("org_id = ? AND user_id = ? AND uid = ?", user.OrgID, user.UserID, uid).Delete(ExploreBookmark{})
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrBookmarkNotFound
		}
		return nil
	})
}

func toDTO(bookmark ExploreBookmark) (ExploreBookmarkDTO, error) {
	queries, err := simplejson.NewJson([]byte(bookmark.Queries))
	if err != nil {
		return ExploreBookmarkDTO{}, err
	}

	return ExploreBookmarkDTO{
		UID:           bookmark.UID,
		Name:          bookmark.Name,
		DatasourceUID: bookmark.DatasourceUID,
		Queries:       queries,
		TimeFrom:      bookmark.TimeFrom,
		TimeTo:        bookmark.TimeTo,
		CreatedAt:     bookmark.CreatedAt,
		UpdatedAt:     bookmark.UpdatedAt,
	}, nil
}
