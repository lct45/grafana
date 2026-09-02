package recentitems

import (
	"errors"
	"time"
)

const (
	DefaultLimit = 50
	MaxLimit     = 50
)

var (
	ErrInvalidResourceType = errors.New("invalid resource type")
	ErrInvalidResourceUID  = errors.New("invalid resource UID")
	ErrInvalidTitle        = errors.New("invalid title")
	ErrInvalidURL          = errors.New("invalid URL")
	ErrInvalidTimestamp    = errors.New("invalid last viewed timestamp")
	ErrInvalidLimit        = errors.New("invalid limit")
	ErrItemNotFound        = errors.New("recent item not found")
	ErrEmptyPatch          = errors.New("patch must include at least one mutable field")
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

func (RecentItem) TableName() string {
	return "user_recent_item"
}

type RecentItemDTO struct {
	UID          string    `json:"uid"`
	ResourceType string    `json:"resourceType"`
	ResourceUID  string    `json:"resourceUid"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	LastViewedAt time.Time `json:"lastViewedAt"`
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
	Title        *string    `json:"title,omitempty"`
	URL          *string    `json:"url,omitempty"`
	LastViewedAt *time.Time `json:"lastViewedAt,omitempty"`
}

type ListRecentItemsResponse struct {
	Items []RecentItemDTO `json:"items"`
}

type DeleteRecentItemResponse struct {
	Message string `json:"message"`
}
