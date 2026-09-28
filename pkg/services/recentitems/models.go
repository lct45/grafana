package recentitems

import "errors"

const (
	defaultRecentItemsLimit = 20
	maxRecentItemsLimit     = 100
	// maxRecentItemsPerUser is how many rows are kept for one user in one org.
	// Recording a new resource past this cap deletes the oldest rows in that scope.
	maxRecentItemsPerUser = 100

	maxTitleLength = 255
	maxURLLength   = 1024
)

var (
	ErrRecentItemNotFound = errors.New("recent item not found")
	errRecentItemInvalid  = errors.New("invalid recent item")
)

type invalidRecentItem struct {
	reason string
}

func (e invalidRecentItem) Error() string {
	return e.reason
}

func (e invalidRecentItem) Unwrap() error {
	return errRecentItemInvalid
}

func invalid(reason string) error {
	return invalidRecentItem{reason: reason}
}

// UserRecentItem is the persistence model for a recently viewed resource.
type UserRecentItem struct {
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

func (UserRecentItem) TableName() string {
	return "user_recent_item"
}

// RecentItemDTO is the API representation of a recently viewed resource.
// swagger:model
type RecentItemDTO struct {
	UID          string `json:"uid"`
	ResourceType string `json:"resourceType"`
	ResourceUID  string `json:"resourceUid"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	LastViewedAt int64  `json:"lastViewedAt"`
}

// CreateRecentItemCommand records a view of a resource.
// swagger:model
type CreateRecentItemCommand struct {
	// Type of resource that was viewed.
	// required: true
	// enum: dashboard,alert_rule,folder,datasource,explore
	ResourceType string `json:"resourceType"`
	// UID of the viewed resource.
	// required: true
	ResourceUID string `json:"resourceUid"`
	// Display title captured when the resource was viewed.
	// required: true
	Title string `json:"title"`
	// Relative Grafana path for the resource.
	// required: true
	URL string `json:"url"`
}

// PatchRecentItemCommand updates the stored title and/or URL.
// swagger:model
type PatchRecentItemCommand struct {
	Title *string `json:"title,omitempty"`
	URL   *string `json:"url,omitempty"`
}

// ListRecentItemsResponse wraps a list of recent items.
type ListRecentItemsResponse struct {
	RecentItems []RecentItemDTO `json:"recentItems"`
}

func toDTO(item UserRecentItem) RecentItemDTO {
	return RecentItemDTO{
		UID:          item.UID,
		ResourceType: item.ResourceType,
		ResourceUID:  item.ResourceUID,
		Title:        item.Title,
		URL:          item.URL,
		LastViewedAt: item.LastViewedAt,
	}
}
