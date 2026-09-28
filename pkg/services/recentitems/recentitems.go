package recentitems

import (
	"context"
	"strings"
	"time"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
)

type itemStore interface {
	record(ctx context.Context, usr *user.SignedInUser, cmd CreateRecentItemCommand, viewedAt int64) (RecentItemDTO, bool, error)
	list(ctx context.Context, usr *user.SignedInUser, limit int) ([]RecentItemDTO, error)
	patch(ctx context.Context, usr *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error)
	delete(ctx context.Context, usr *user.SignedInUser, uid string) error
}

// Service records and reads recently viewed resources for a signed-in user.
type Service interface {
	Record(ctx context.Context, usr *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error)
	List(ctx context.Context, usr *user.SignedInUser, limit int) ([]RecentItemDTO, error)
	Update(ctx context.Context, usr *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error)
	Remove(ctx context.Context, usr *user.SignedInUser, uid string) error
}

// RecentItemsService is the SQL-backed recent items service.
type RecentItemsService struct {
	db    db.DB
	now   func() time.Time
	store itemStore
}

func ProvideService(sqlStore db.DB) *RecentItemsService {
	s := &RecentItemsService{
		db:  sqlStore,
		now: time.Now,
	}
	s.store = &sqlItemStore{db: sqlStore}
	return s
}

func (s *RecentItemsService) Record(ctx context.Context, usr *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	normalized, err := normalizeCreate(cmd)
	if err != nil {
		return RecentItemDTO{}, false, err
	}
	return s.store.record(ctx, usr, normalized, s.now().UnixMilli())
}

func (s *RecentItemsService) List(ctx context.Context, usr *user.SignedInUser, limit int) ([]RecentItemDTO, error) {
	if limit < 1 || limit > maxRecentItemsLimit {
		return nil, invalid("limit is invalid")
	}
	return s.store.list(ctx, usr, limit)
}

func (s *RecentItemsService) Update(ctx context.Context, usr *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	normalized, err := normalizePatch(cmd)
	if err != nil {
		return RecentItemDTO{}, err
	}
	return s.store.patch(ctx, usr, uid, normalized)
}

func (s *RecentItemsService) Remove(ctx context.Context, usr *user.SignedInUser, uid string) error {
	return s.store.delete(ctx, usr, uid)
}

func normalizeCreate(cmd CreateRecentItemCommand) (CreateRecentItemCommand, error) {
	cmd.ResourceType = strings.TrimSpace(cmd.ResourceType)
	if !validResourceType(cmd.ResourceType) {
		return CreateRecentItemCommand{}, invalid("resourceType is invalid")
	}

	cmd.ResourceUID = strings.TrimSpace(cmd.ResourceUID)
	if cmd.ResourceUID == "" || !util.IsValidShortUID(cmd.ResourceUID) || util.IsShortUIDTooLong(cmd.ResourceUID) {
		return CreateRecentItemCommand{}, invalid("resourceUid is invalid")
	}

	title, err := normalizeTitle(cmd.Title)
	if err != nil {
		return CreateRecentItemCommand{}, err
	}
	cmd.Title = title

	rawURL, err := normalizeURL(cmd.URL)
	if err != nil {
		return CreateRecentItemCommand{}, err
	}
	cmd.URL = rawURL
	return cmd, nil
}

func normalizePatch(cmd PatchRecentItemCommand) (PatchRecentItemCommand, error) {
	if cmd.Title == nil && cmd.URL == nil {
		return PatchRecentItemCommand{}, invalid("title or url is required")
	}
	if cmd.Title != nil {
		title, err := normalizeTitle(*cmd.Title)
		if err != nil {
			return PatchRecentItemCommand{}, err
		}
		cmd.Title = &title
	}
	if cmd.URL != nil {
		rawURL, err := normalizeURL(*cmd.URL)
		if err != nil {
			return PatchRecentItemCommand{}, err
		}
		cmd.URL = &rawURL
	}
	return cmd, nil
}

func normalizeTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", invalid("title is required")
	}
	if len(title) > maxTitleLength {
		return "", invalid("title must be at most 255 characters")
	}
	return title, nil
}

func normalizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", invalid("url is required")
	}
	if len(rawURL) > maxURLLength {
		return "", invalid("url must be at most 1024 characters")
	}
	// Browsers treat \ as / in hrefs, so /\evil.com becomes //evil.com.
	if strings.Contains(rawURL, `\`) || !strings.HasPrefix(rawURL, "/") || strings.HasPrefix(rawURL, "//") || strings.Contains(rawURL, "://") {
		return "", invalid("url must be a relative path")
	}
	return rawURL, nil
}

func validResourceType(resourceType string) bool {
	switch resourceType {
	case "dashboard", "alert_rule", "folder", "datasource", "explore":
		return true
	default:
		return false
	}
}
