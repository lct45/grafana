package explorebookmark

import (
	"errors"

	"github.com/grafana/grafana/pkg/components/simplejson"
)

var ErrBookmarkNotFound = errors.New("explore bookmark not found")

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
	UpdatedAt     int64  `xorm:"updated_at"`
}

type ExploreBookmarkDTO struct {
	UID           string           `json:"uid" xorm:"uid"`
	Name          string           `json:"name" xorm:"name"`
	DatasourceUID string           `json:"datasourceUid" xorm:"datasource_uid"`
	Queries       *simplejson.Json `json:"queries"`
	TimeFrom      string           `json:"timeFrom" xorm:"time_from"`
	TimeTo        string           `json:"timeTo" xorm:"time_to"`
	CreatedAt     int64            `json:"createdAt" xorm:"created_at"`
	UpdatedAt     int64            `json:"updatedAt" xorm:"updated_at"`
}

type ExploreBookmarkResponse struct {
	Result ExploreBookmarkDTO `json:"result"`
}

type ExploreBookmarkListResponse struct {
	Result []ExploreBookmarkDTO `json:"result"`
}

type ExploreBookmarkDeleteResponse struct {
	Message string `json:"message"`
	UID     string `json:"uid"`
}

// CreateExploreBookmarkCommand is the command for creating an explore bookmark.
// swagger:model
type CreateExploreBookmarkCommand struct {
	Name          string           `json:"name"`
	DatasourceUID string           `json:"datasourceUid"`
	Queries       *simplejson.Json `json:"queries"`
	TimeFrom      string           `json:"timeFrom"`
	TimeTo        string           `json:"timeTo"`
}
