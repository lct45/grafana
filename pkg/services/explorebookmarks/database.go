package explorebookmarks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
)

func (s *ExploreBookmarksService) createBookmark(ctx context.Context, user *user.SignedInUser, cmd CreateBookmarkCommand) (ExploreBookmarkDTO, error) {
	if err := validateCreateCommand(cmd); err != nil {
		return ExploreBookmarkDTO{}, err
	}

	queriesBytes, err := json.Marshal(cmd.Queries)
	if err != nil {
		return ExploreBookmarkDTO{}, err
	}

	bookmark := ExploreBookmark{
		UID:           util.GenerateShortUID(),
		OrgID:         user.OrgID,
		UserID:        user.UserID,
		Name:          strings.TrimSpace(cmd.Name),
		DatasourceUID: cmd.DatasourceUID,
		Queries:       string(queriesBytes),
		TimeFrom:      cmd.TimeRange.From,
		TimeTo:        cmd.TimeRange.To,
		CreatedAt:     s.now().Unix(),
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

func (s *ExploreBookmarksService) listBookmarks(ctx context.Context, user *user.SignedInUser) ([]ExploreBookmarkDTO, error) {
	var bookmarks []ExploreBookmark

	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		return session.Where("org_id = ? AND user_id = ?", user.OrgID, user.UserID).
			Desc("created_at").
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

func (s *ExploreBookmarksService) deleteBookmark(ctx context.Context, user *user.SignedInUser, uid string) error {
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

func validateCreateCommand(cmd CreateBookmarkCommand) error {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return ErrBookmarkNameRequired
	}
	if len(name) > 255 {
		return fmt.Errorf("bookmark name must be at most 255 characters")
	}
	if strings.TrimSpace(cmd.DatasourceUID) == "" {
		return fmt.Errorf("datasourceUid is required")
	}
	if cmd.Queries == nil {
		return fmt.Errorf("queries are required")
	}
	raw, err := cmd.Queries.MarshalJSON()
	if err != nil {
		return fmt.Errorf("queries must be valid JSON")
	}
	var queries []any
	if err := json.Unmarshal(raw, &queries); err != nil {
		return fmt.Errorf("queries must be a JSON array")
	}
	if len(queries) == 0 {
		return fmt.Errorf("queries must not be empty")
	}
	if strings.TrimSpace(cmd.TimeRange.From) == "" || strings.TrimSpace(cmd.TimeRange.To) == "" {
		return fmt.Errorf("timeRange.from and timeRange.to are required")
	}
	return nil
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
		TimeRange: TimeRangeDTO{
			From: bookmark.TimeFrom,
			To:   bookmark.TimeTo,
		},
		CreatedAt: bookmark.CreatedAt,
	}, nil
}
