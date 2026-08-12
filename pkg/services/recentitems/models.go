package recentitems

import "errors"

var (
	ErrRecentItemNotFound   = errors.New("recent item not found")
	ErrInvalidResourceType  = errors.New("invalid resourceType")
	ErrResourceTypeRequired = errors.New("resourceType is required")
	ErrResourceUIDRequired  = errors.New("resourceUid is required")
	ErrTitleRequired        = errors.New("title is required")
	ErrURLRequired          = errors.New("url is required")
	ErrImmutableField       = errors.New("resourceType and resourceUid cannot be updated")
	ErrUnknownField         = errors.New("unknown or unsupported field")
	ErrEmptyPatch           = errors.New("at least one field is required")
	ErrInvalidLimit         = errors.New("limit must be between 1 and 100")
)

const (
	ResourceTypeDashboard      = "dashboard"
	ResourceTypeAlertRule      = "alert_rule"
	ResourceTypeFolder         = "folder"
	ResourceTypeDatasource     = "datasource"
	ResourceTypeExploreSession = "explore_session"

	DefaultListLimit = 50
	MaxListLimit     = 100
	MaxStoredItems   = 50
	MaxTitleLength   = 255
	MaxURLLength     = 1024
)

var allowedResourceTypes = map[string]struct{}{
	ResourceTypeDashboard:      {},
	ResourceTypeAlertRule:      {},
	ResourceTypeFolder:         {},
	ResourceTypeDatasource:     {},
	ResourceTypeExploreSession: {},
}

// RecentItem is the persistence model for recently viewed resources.
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
	CreatedAt    int64  `xorm:"created_at"`
}

// RecentItemDTO is the API-facing recent item representation.
type RecentItemDTO struct {
	UID          string `json:"uid"`
	ResourceType string `json:"resourceType"`
	ResourceUID  string `json:"resourceUid"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	LastViewedAt int64  `json:"lastViewedAt"`
	CreatedAt    int64  `json:"createdAt"`
}

// CreateRecentItemCommand is the command for recording a recent item.
// swagger:model
type CreateRecentItemCommand struct {
	// Type of the viewed resource.
	// required: true
	// example: dashboard
	ResourceType string `json:"resourceType"`
	// UID of the viewed resource.
	// required: true
	ResourceUID string `json:"resourceUid"`
	// Display title for the resource.
	// required: true
	Title string `json:"title"`
	// Deep link URL for the resource.
	// required: true
	URL string `json:"url"`
}

// PatchRecentItemCommand is the command for partially updating a recent item.
// swagger:model
type PatchRecentItemCommand struct {
	Title        *string `json:"title,omitempty"`
	URL          *string `json:"url,omitempty"`
	LastViewedAt *int64  `json:"lastViewedAt,omitempty"`
}

// CreateRecentItemResponse wraps a created or upserted recent item.
type CreateRecentItemResponse struct {
	Item RecentItemDTO `json:"item"`
}

// ListRecentItemsResponse wraps a list of recent items.
type ListRecentItemsResponse struct {
	Items []RecentItemDTO `json:"items"`
}

// DeleteRecentItemResponse is returned after deleting a recent item.
type DeleteRecentItemResponse struct {
	Message string `json:"message"`
}

func toDTO(item RecentItem) RecentItemDTO {
	return RecentItemDTO{
		UID:          item.UID,
		ResourceType: item.ResourceType,
		ResourceUID:  item.ResourceUID,
		Title:        item.Title,
		URL:          item.URL,
		LastViewedAt: item.LastViewedAt,
		CreatedAt:    item.CreatedAt,
	}
}
