package explorebookmarks

import (
	"errors"

	"github.com/grafana/grafana/pkg/components/simplejson"
)

var (
	ErrBookmarkNotFound     = errors.New("explore bookmark not found")
	ErrBookmarkNameRequired = errors.New("bookmark name is required")
)

// ExploreBookmark is the persistence model for explore bookmarks.
type ExploreBookmark struct {
	ID            int64  `xorm:"pk autoincr 'id'"`
	UID           string `xorm:"uid"`
	OrgID         int64  `xorm:"org_id"`
	UserID        int64  `xorm:"user_id"`
	Name          string `xorm:"name"`
	DatasourceUID string `xorm:"datasource_uid"`
	Queries       string `xorm:"queries"`
	TimeFrom      string `xorm:"time_from"`
	TimeTo        string `xorm:"time_to"`
	CreatedAt     int64  `xorm:"created_at"`
}

// TimeRangeDTO is the JSON time range stored with a bookmark.
type TimeRangeDTO struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ExploreBookmarkDTO is the API-facing bookmark representation.
type ExploreBookmarkDTO struct {
	UID           string           `json:"uid"`
	Name          string           `json:"name"`
	DatasourceUID string           `json:"datasourceUid"`
	Queries       *simplejson.Json `json:"queries"`
	TimeRange     TimeRangeDTO     `json:"timeRange"`
	CreatedAt     int64            `json:"createdAt"`
}

// CreateBookmarkResponse wraps a created bookmark.
type CreateBookmarkResponse struct {
	Bookmark ExploreBookmarkDTO `json:"bookmark"`
}

// ListBookmarksResponse wraps a list of bookmarks.
type ListBookmarksResponse struct {
	Bookmarks []ExploreBookmarkDTO `json:"bookmarks"`
}

// DeleteBookmarkResponse is returned after deleting a bookmark.
type DeleteBookmarkResponse struct {
	Message string `json:"message"`
}

// CreateBookmarkCommand is the command for creating an explore bookmark.
// swagger:model
type CreateBookmarkCommand struct {
	// Display name for the bookmark.
	// required: true
	Name string `json:"name"`
	// UID of the data source for which queries are stored.
	// required: true
	// example: PE1C5CBDA0504A6A3
	DatasourceUID string `json:"datasourceUid"`
	// The JSON model of queries.
	// required: true
	Queries *simplejson.Json `json:"queries"`
	// Time range associated with the bookmark.
	// required: true
	TimeRange TimeRangeDTO `json:"timeRange"`
}
