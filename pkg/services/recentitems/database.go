package recentitems

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
)

var resourceTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)

func (s *RecentItemsService) record(ctx context.Context, usr *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	normalized, err := normalizeCreateCommand(cmd)
	if err != nil {
		return RecentItemDTO{}, false, err
	}

	dto, created, err := s.upsert(ctx, usr, normalized)
	if err != nil {
		if dialect := s.store.GetDialect(); dialect != nil && dialect.IsUniqueConstraintViolation(err) {
			// A concurrent insert of the same resource won the unique index. Refresh that row.
			return s.upsert(ctx, usr, normalized)
		}
	}
	return dto, created, err
}

func (s *RecentItemsService) upsert(ctx context.Context, usr *user.SignedInUser, cmd CreateRecentItemCommand) (RecentItemDTO, bool, error) {
	now := s.now().UnixMilli()
	var dto RecentItemDTO
	var created bool

	err := s.store.InTransaction(ctx, func(ctx context.Context) error {
		return s.store.WithDbSession(ctx, func(session *db.Session) error {
			existing, has, err := getByResource(session, usr, cmd.ResourceType, cmd.ResourceUID)
			if err != nil {
				return err
			}
			if has {
				existing.Title = cmd.Title
				existing.URL = cmd.URL
				existing.LastViewedAt = now
				_, err = session.ID(existing.ID).Cols("title", "url", "last_viewed_at").Update(&existing)
				if err != nil {
					return err
				}
				dto = toDTO(existing)
				created = false
				return nil
			}

			row := UserRecentItem{
				UID:          util.GenerateShortUID(),
				OrgID:        usr.OrgID,
				UserID:       usr.UserID,
				ResourceType: cmd.ResourceType,
				ResourceUID:  cmd.ResourceUID,
				Title:        cmd.Title,
				URL:          cmd.URL,
				LastViewedAt: now,
			}
			if _, err = session.Insert(&row); err != nil {
				return err
			}
			if err = trimOverflow(session, usr); err != nil {
				return err
			}
			dto = toDTO(row)
			created = true
			return nil
		})
	})

	return dto, created, err
}

func (s *RecentItemsService) list(ctx context.Context, usr *user.SignedInUser, query ListRecentItemsQuery) ([]RecentItemDTO, error) {
	if query.ResourceType != "" && !resourceTypePattern.MatchString(query.ResourceType) {
		return nil, ErrResourceTypeInvalid
	}
	if query.Limit < 1 || query.Limit > maxRecentItemsLimit {
		return nil, ErrInvalidLimit
	}

	var rows []UserRecentItem
	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		sess := session.Where("org_id = ? AND user_id = ?", usr.OrgID, usr.UserID)
		if query.ResourceType != "" {
			sess = sess.And("resource_type = ?", query.ResourceType)
		}
		return sess.OrderBy("last_viewed_at DESC, id DESC").Limit(query.Limit).Find(&rows)
	})
	if err != nil {
		return nil, err
	}

	items := make([]RecentItemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDTO(row))
	}
	return items, nil
}

func (s *RecentItemsService) update(ctx context.Context, usr *user.SignedInUser, uid string, cmd PatchRecentItemCommand) (RecentItemDTO, error) {
	if err := validatePatchCommand(cmd); err != nil {
		return RecentItemDTO{}, err
	}

	var dto RecentItemDTO
	err := s.store.WithDbSession(ctx, func(session *db.Session) error {
		existing, has, err := getByUID(session, usr, uid)
		if err != nil {
			return err
		}
		if !has {
			return ErrRecentItemNotFound
		}
		cols := make([]string, 0, 2)
		if cmd.Title != nil {
			existing.Title = *cmd.Title
			cols = append(cols, "title")
		}
		if cmd.URL != nil {
			existing.URL = *cmd.URL
			cols = append(cols, "url")
		}
		if _, err = session.ID(existing.ID).Cols(cols...).Update(&existing); err != nil {
			return err
		}
		dto = toDTO(existing)
		return nil
	})
	return dto, err
}

func (s *RecentItemsService) remove(ctx context.Context, usr *user.SignedInUser, uid string) error {
	return s.store.WithDbSession(ctx, func(session *db.Session) error {
		count, err := session.Where("org_id = ? AND user_id = ? AND uid = ?", usr.OrgID, usr.UserID, uid).Delete(&UserRecentItem{})
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrRecentItemNotFound
		}
		return nil
	})
}

func getByResource(session *db.Session, usr *user.SignedInUser, resourceType, resourceUID string) (UserRecentItem, bool, error) {
	var existing UserRecentItem
	has, err := session.Where(
		"org_id = ? AND user_id = ? AND resource_type = ? AND resource_uid = ?",
		usr.OrgID, usr.UserID, resourceType, resourceUID,
	).Get(&existing)
	return existing, has, err
}

func getByUID(session *db.Session, usr *user.SignedInUser, uid string) (UserRecentItem, bool, error) {
	var existing UserRecentItem
	has, err := session.Where("org_id = ? AND user_id = ? AND uid = ?", usr.OrgID, usr.UserID, uid).Get(&existing)
	return existing, has, err
}

func trimOverflow(session *db.Session, usr *user.SignedInUser) error {
	count, err := session.Where("org_id = ? AND user_id = ?", usr.OrgID, usr.UserID).Count(&UserRecentItem{})
	if err != nil {
		return err
	}
	extra := int(count) - maxRecentItemsPerUser
	if extra <= 0 {
		return nil
	}

	var stale []UserRecentItem
	err = session.Where("org_id = ? AND user_id = ?", usr.OrgID, usr.UserID).
		OrderBy("last_viewed_at ASC, id ASC").
		Limit(extra).
		Find(&stale)
	if err != nil {
		return err
	}
	for _, row := range stale {
		if _, err = session.ID(row.ID).Delete(&UserRecentItem{}); err != nil {
			return err
		}
	}
	return nil
}

func normalizeCreateCommand(cmd CreateRecentItemCommand) (CreateRecentItemCommand, error) {
	cmd.ResourceType = strings.TrimSpace(cmd.ResourceType)
	cmd.ResourceUID = strings.TrimSpace(cmd.ResourceUID)
	cmd.Title = strings.TrimSpace(cmd.Title)
	cmd.URL = strings.TrimSpace(cmd.URL)

	if err := validateResourceType(cmd.ResourceType); err != nil {
		return CreateRecentItemCommand{}, err
	}
	if err := validateResourceUID(cmd.ResourceUID); err != nil {
		return CreateRecentItemCommand{}, err
	}
	if err := validateTitle(cmd.Title); err != nil {
		return CreateRecentItemCommand{}, err
	}
	if err := validateURL(cmd.URL); err != nil {
		return CreateRecentItemCommand{}, err
	}
	return cmd, nil
}

func validatePatchCommand(cmd PatchRecentItemCommand) error {
	if cmd.Title == nil && cmd.URL == nil {
		return ErrPatchEmpty
	}
	if cmd.Title != nil {
		title := strings.TrimSpace(*cmd.Title)
		if err := validateTitle(title); err != nil {
			return err
		}
		*cmd.Title = title
	}
	if cmd.URL != nil {
		rawURL := strings.TrimSpace(*cmd.URL)
		if err := validateURL(rawURL); err != nil {
			return err
		}
		*cmd.URL = rawURL
	}
	return nil
}

func validateResourceType(resourceType string) error {
	if resourceType == "" {
		return ErrResourceTypeRequired
	}
	if !resourceTypePattern.MatchString(resourceType) {
		return ErrResourceTypeInvalid
	}
	return nil
}

func validateResourceUID(resourceUID string) error {
	if resourceUID == "" {
		return ErrResourceUIDRequired
	}
	if len(resourceUID) > maxResourceUIDLength {
		return ErrResourceUIDTooLong
	}
	return nil
}

func validateTitle(title string) error {
	if title == "" {
		return ErrTitleRequired
	}
	if len(title) > maxTitleLength {
		return ErrTitleTooLong
	}
	return nil
}

func validateURL(raw string) error {
	if raw == "" {
		return ErrURLRequired
	}
	if len(raw) > maxURLLength {
		return ErrURLTooLong
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "\\\n\r") {
		return ErrURLInvalid
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Scheme != "" {
		return ErrURLInvalid
	}
	return nil
}

func toDTO(row UserRecentItem) RecentItemDTO {
	return RecentItemDTO{
		UID:          row.UID,
		ResourceType: row.ResourceType,
		ResourceUID:  row.ResourceUID,
		Title:        row.Title,
		URL:          row.URL,
		LastViewedAt: row.LastViewedAt,
	}
}

func isValidationError(err error) bool {
	return errors.Is(err, ErrResourceTypeRequired) ||
		errors.Is(err, ErrResourceTypeInvalid) ||
		errors.Is(err, ErrResourceUIDRequired) ||
		errors.Is(err, ErrResourceUIDTooLong) ||
		errors.Is(err, ErrTitleRequired) ||
		errors.Is(err, ErrTitleTooLong) ||
		errors.Is(err, ErrURLRequired) ||
		errors.Is(err, ErrURLInvalid) ||
		errors.Is(err, ErrURLTooLong) ||
		errors.Is(err, ErrInvalidLimit) ||
		errors.Is(err, ErrPatchEmpty) ||
		errors.Is(err, ErrImmutableField) ||
		errors.Is(err, ErrUnknownField) ||
		errors.Is(err, ErrMalformedBody)
}
