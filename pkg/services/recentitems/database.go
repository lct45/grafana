package recentitems

import (
	"context"
	"errors"
	"strings"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
)

// upsertAttempts bounds retries when concurrent writers race on the
// (org, user, resource) unique key. Two attempts is the common case: the
// first insert loses the race, the second finds the row and refreshes it.
const upsertAttempts = 3

var errUpsertConflict = errors.New("recent item insert conflicted with a concurrent write")

// recordView creates a recent item or refreshes the existing one for the same
// resource. It reports whether a new row was created.
func (s *RecentItemsService) recordView(ctx context.Context, usr *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	if err := validateCreateCommand(&cmd); err != nil {
		return RecentItemDTO{}, false, err
	}

	var (
		item    RecentItem
		created bool
	)
	for attempt := 0; attempt < upsertAttempts; attempt++ {
		err := s.store.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
			var err error
			item, created, err = s.upsertInSession(sess, usr, cmd)
			return err
		})
		if errors.Is(err, errUpsertConflict) {
			continue
		}
		if err != nil {
			return RecentItemDTO{}, false, err
		}
		return toDTO(item), created, nil
	}
	return RecentItemDTO{}, false, errPersistenceRaceExceeded
}

func (s *RecentItemsService) upsertInSession(sess *db.Session, usr *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItem, bool, error) {
	now := s.now().Unix()

	existing := RecentItem{}
	has, err := sess.Where(
		"org_id = ? AND user_id = ? AND resource_type = ? AND resource_uid = ?",
		usr.OrgID, usr.UserID, string(cmd.ResourceType), cmd.ResourceUID,
	).Get(&existing)
	if err != nil {
		return RecentItem{}, false, err
	}

	if has {
		existing.Title = cmd.Title
		existing.URL = cmd.URL
		existing.LastViewedAt = now
		if _, err := sess.ID(existing.ID).Cols("title", "url", "last_viewed_at").Update(&existing); err != nil {
			return RecentItem{}, false, err
		}
		return existing, false, nil
	}

	if s.beforeInsert != nil {
		if err := s.beforeInsert(sess); err != nil {
			return RecentItem{}, false, err
		}
	}

	item := RecentItem{
		UID:          util.GenerateShortUID(),
		OrgID:        usr.OrgID,
		UserID:       usr.UserID,
		ResourceType: string(cmd.ResourceType),
		ResourceUID:  cmd.ResourceUID,
		Title:        cmd.Title,
		URL:          cmd.URL,
		LastViewedAt: now,
	}
	if _, err := sess.Insert(&item); err != nil {
		if s.store.GetDialect().IsUniqueConstraintViolation(err) {
			return RecentItem{}, false, errUpsertConflict
		}
		return RecentItem{}, false, err
	}

	if err := trimToCap(sess, usr); err != nil {
		return RecentItem{}, false, err
	}
	return item, true, nil
}

// trimToCap deletes the oldest rows beyond MaxStoredItems. It runs after every
// insert, so at most a handful of excess rows exist at any time; selecting IDs
// first (instead of DELETE ... IN (SELECT ... LIMIT)) keeps it portable across
// MySQL, which rejects LIMIT inside IN subqueries.
func trimToCap(sess *db.Session, usr *user.SignedInUser) error {
	var excess []int64
	err := sess.Table(RecentItem{}).Cols("id").
		Where("org_id = ? AND user_id = ?", usr.OrgID, usr.UserID).
		Desc("last_viewed_at", "id").
		Limit(MaxStoredItems, MaxStoredItems).
		Find(&excess)
	if err != nil {
		return err
	}
	if len(excess) == 0 {
		return nil
	}
	_, err = sess.In("id", excess).Delete(RecentItem{})
	return err
}

func (s *RecentItemsService) listItems(ctx context.Context, usr *user.SignedInUser, query ListRecentItemsQuery) ([]RecentItemDTO, error) {
	if err := validateListQuery(&query); err != nil {
		return nil, err
	}

	var items []RecentItem
	err := s.store.WithDbSession(ctx, func(sess *db.Session) error {
		q := sess.Where("org_id = ? AND user_id = ?", usr.OrgID, usr.UserID)
		if query.ResourceType != "" {
			q = q.And("resource_type = ?", string(query.ResourceType))
		}
		return q.Desc("last_viewed_at", "id").Limit(query.Limit).Find(&items)
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

func (s *RecentItemsService) patchItem(ctx context.Context, usr *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	if err := validatePatchCommand(&cmd); err != nil {
		return RecentItemDTO{}, err
	}

	var item RecentItem
	err := s.store.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		has, err := sess.Where("org_id = ? AND user_id = ? AND uid = ?", usr.OrgID, usr.UserID, uid).Get(&item)
		if err != nil {
			return err
		}
		if !has {
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
		_, err = sess.ID(item.ID).Cols(cols...).Update(&item)
		return err
	})
	if err != nil {
		return RecentItemDTO{}, err
	}
	return toDTO(item), nil
}

func (s *RecentItemsService) deleteItem(ctx context.Context, usr *user.SignedInUser, uid string) error {
	return s.store.WithDbSession(ctx, func(sess *db.Session) error {
		count, err := sess.Where("org_id = ? AND user_id = ? AND uid = ?", usr.OrgID, usr.UserID, uid).Delete(RecentItem{})
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrRecentItemNotFound
		}
		return nil
	})
}

// validateCreateCommand normalizes whitespace in place and checks limits.
func validateCreateCommand(cmd *CreateRecentItemCommand) error {
	if !cmd.ResourceType.isValid() {
		return ErrResourceTypeInvalid
	}
	cmd.ResourceUID = strings.TrimSpace(cmd.ResourceUID)
	if cmd.ResourceUID == "" {
		return ErrResourceUIDRequired
	}
	if len(cmd.ResourceUID) > maxResourceUIDLength {
		return ErrResourceUIDTooLong
	}
	title, err := validateTitle(cmd.Title)
	if err != nil {
		return err
	}
	cmd.Title = title
	url, err := validateURL(cmd.URL)
	if err != nil {
		return err
	}
	cmd.URL = url
	return nil
}

func validatePatchCommand(cmd *PatchRecentItemCommand) error {
	if cmd.Title == nil && cmd.URL == nil {
		return ErrPatchNoFields
	}
	if cmd.Title != nil {
		title, err := validateTitle(*cmd.Title)
		if err != nil {
			return err
		}
		cmd.Title = &title
	}
	if cmd.URL != nil {
		url, err := validateURL(*cmd.URL)
		if err != nil {
			return err
		}
		cmd.URL = &url
	}
	return nil
}

func validateListQuery(query *ListRecentItemsQuery) error {
	if query.ResourceType != "" && !query.ResourceType.isValid() {
		return ErrResourceTypeInvalid
	}
	if query.Limit < 0 {
		return ErrListLimitInvalid
	}
	if query.Limit == 0 {
		query.Limit = DefaultListLimit
	}
	if query.Limit > MaxStoredItems {
		query.Limit = MaxStoredItems
	}
	return nil
}

func validateTitle(raw string) (string, error) {
	title := strings.TrimSpace(raw)
	if title == "" {
		return "", ErrTitleRequired
	}
	if len(title) > maxTitleLength {
		return "", ErrTitleTooLong
	}
	return title, nil
}

// validateURL only accepts same-origin relative paths so a stored item can
// never redirect the user off the Grafana instance when reopened.
func validateURL(raw string) (string, error) {
	url := strings.TrimSpace(raw)
	if url == "" {
		return "", ErrURLRequired
	}
	if len(url) > maxURLLength {
		return "", ErrURLTooLong
	}
	if !strings.HasPrefix(url, "/") || strings.HasPrefix(url, "//") || strings.HasPrefix(url, "/\\") {
		return "", ErrURLNotRelative
	}
	return url, nil
}
