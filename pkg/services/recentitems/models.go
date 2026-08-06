package recentitems

import "errors"

const (
	DefaultLimit        = 50
	MaxResourceUIDLen   = 255
	MaxTitleLen         = 255
	MaxURLLen           = 2048
	ResourceTypeAlert   = "alertRule"
	ResourceTypeDash    = "dashboard"
	ResourceTypeSource  = "datasource"
	ResourceTypeExplore = "explore"
	ResourceTypeFolder  = "folder"
)

var (
	ErrInvalidResourceType = errors.New("invalid resource type")
	ErrNoPatchFields       = errors.New("at least one mutable field is required")
	ErrRecentItemNotFound  = errors.New("recent item not found")
	ErrResourceUIDRequired = errors.New("resourceUid is required")
	ErrResourceUIDTooLong  = errors.New("resourceUid must be at most 255 characters")
	ErrTitleTooLong        = errors.New("title must be at most 255 characters")
	ErrURLTooLong          = errors.New("url must be at most 2048 characters")
)

type RecentItem struct {
	ID           int64  `xorm:"pk autoincr 'id'"`
	UID          string `xorm:"uid"`
	OrgID        int64  `xorm:"org_id"`
	UserID       int64  `xorm:"user_id"`
	ResourceType string `xorm:"resource_type"`
	ResourceUID  string `xorm:"resource_uid"`
	Title        string `xorm:"title"`
	URL          string `xorm:"url"`
	LastViewedAt int64  `xorm:"last_viewed_at"`
}

type RecentItemDTO struct {
	UID          string `json:"uid"`
	ResourceType string `json:"resourceType"`
	ResourceUID  string `json:"resourceUid"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	LastViewedAt int64  `json:"lastViewedAt"`
}

type UpsertRecentItemResult struct {
	Item    RecentItemDTO
	Created bool
}

// swagger:model
type CreateRecentItemCommand struct {
	ResourceType string `json:"resourceType"`
	ResourceUID  string `json:"resourceUid"`
	Title        string `json:"title"`
	URL          string `json:"url"`
}

// swagger:model
type PatchRecentItemCommand struct {
	Title *string `json:"title,omitempty"`
	URL   *string `json:"url,omitempty"`
}

type ListRecentItemsQuery struct {
	ResourceType string
	Limit        int
}

type RecentItemResponse struct {
	Item RecentItemDTO `json:"item"`
}

type RecentItemsResponse struct {
	Items []RecentItemDTO `json:"items"`
}
