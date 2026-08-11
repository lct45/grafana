package recentitems

import "errors"

var (
	ErrRecentItemNotFound   = errors.New("recent item not found")
	ErrResourceTypeRequired = errors.New("resourceType is required")
	ErrResourceTypeInvalid  = errors.New("resourceType is invalid")
	ErrResourceUIDRequired  = errors.New("resourceUid is required")
	ErrTitleRequired        = errors.New("title is required")
	ErrURLRequired          = errors.New("url is required")
	ErrTitleTooLong         = errors.New("title must be at most 255 characters")
	ErrURLTooLong           = errors.New("url must be at most 1024 characters")
	ErrResourceUIDTooLong   = errors.New("resourceUid must be at most 40 characters")
	ErrPatchEmpty           = errors.New("at least one of title, url, or lastViewedAt is required")
	ErrImmutableField       = errors.New("resourceType and resourceUid cannot be updated")
	ErrInvalidLimit         = errors.New("limit must be between 1 and 100")
)

const (
	defaultListLimit = 50
	maxListLimit     = 100
	maxStoredItems   = 50
)

// Allowed resource types for recent items.
var allowedResourceTypes = map[string]struct{}{
	"dashboard":       {},
	"alert_rule":      {},
	"folder":          {},
	"datasource":      {},
	"explore_session": {},
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

// CreateRecentItemCommand is the command for recording a recently viewed resource.
// swagger:model
type CreateRecentItemCommand struct {
	// Type of the viewed resource.
	// required: true
	// example: dashboard
	ResourceType string `json:"resourceType"`
	// UID of the viewed resource.
	// required: true
	// example: abc123
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

// PatchRecentItemResponse wraps an updated recent item.
type PatchRecentItemResponse struct {
	Item RecentItemDTO `json:"item"`
}

// DeleteRecentItemResponse is returned after deleting a recent item.
type DeleteRecentItemResponse struct {
	Message string `json:"message"`
}
