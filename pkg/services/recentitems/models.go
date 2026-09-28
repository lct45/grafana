package recentitems

import (
	"errors"
)

const (
	// MaxStoredItems caps how many recent items are kept per user and org.
	// Older entries are trimmed on write once the cap is exceeded.
	MaxStoredItems = 50
	// DefaultListLimit is used when the list request omits a limit.
	DefaultListLimit = 20

	maxTitleLength       = 255
	maxURLLength         = 2048
	maxResourceUIDLength = 128
)

var (
	ErrRecentItemNotFound      = errors.New("recent item not found")
	ErrResourceTypeInvalid     = errors.New("resourceType must be one of: dashboard, folder, alert-rule, datasource, explore")
	ErrResourceUIDRequired     = errors.New("resourceUid is required")
	ErrResourceUIDTooLong      = errors.New("resourceUid must be at most 128 characters")
	ErrTitleRequired           = errors.New("title is required")
	ErrTitleTooLong            = errors.New("title must be at most 255 characters")
	ErrURLRequired             = errors.New("url is required")
	ErrURLTooLong              = errors.New("url must be at most 2048 characters")
	ErrURLNotRelative          = errors.New("url must be a relative path starting with a single '/'")
	ErrPatchNoFields           = errors.New("at least one of title or url must be provided")
	ErrListLimitInvalid        = errors.New("limit must be a positive integer")
	ErrRequestBodyInvalid      = errors.New("request body must be a single JSON object with known fields")
	errPersistenceRaceExceeded = errors.New("recent item upsert conflicted repeatedly")
)

// ResourceType identifies the kind of Grafana resource a recent item points to.
type ResourceType string

const (
	ResourceTypeDashboard  ResourceType = "dashboard"
	ResourceTypeFolder     ResourceType = "folder"
	ResourceTypeAlertRule  ResourceType = "alert-rule"
	ResourceTypeDatasource ResourceType = "datasource"
	ResourceTypeExplore    ResourceType = "explore"
)

var allowedResourceTypes = map[ResourceType]struct{}{
	ResourceTypeDashboard:  {},
	ResourceTypeFolder:     {},
	ResourceTypeAlertRule:  {},
	ResourceTypeDatasource: {},
	ResourceTypeExplore:    {},
}

func (r ResourceType) isValid() bool {
	_, ok := allowedResourceTypes[r]
	return ok
}

// RecentItem is the persistence model for a recently viewed resource.
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

// RecentItemDTO is the API-facing representation of a recent item.
type RecentItemDTO struct {
	UID          string       `json:"uid"`
	ResourceType ResourceType `json:"resourceType"`
	ResourceUID  string       `json:"resourceUid"`
	Title        string       `json:"title"`
	URL          string       `json:"url"`
	// Unix timestamp (seconds) of the most recent view.
	LastViewedAt int64 `json:"lastViewedAt"`
}

// CreateRecentItemCommand records a view of a resource. Re-recording the same
// resource refreshes the existing item instead of creating a duplicate.
// swagger:model
type CreateRecentItemCommand struct {
	// Kind of resource that was viewed.
	// required: true
	// enum: dashboard,folder,alert-rule,datasource,explore
	ResourceType ResourceType `json:"resourceType"`
	// UID of the viewed resource.
	// required: true
	// example: PE1C5CBDA0504A6A3
	ResourceUID string `json:"resourceUid"`
	// Display title at the time of viewing.
	// required: true
	Title string `json:"title"`
	// Relative Grafana URL that reopens the resource.
	// required: true
	// example: /d/PE1C5CBDA0504A6A3/my-dashboard
	URL string `json:"url"`
}

// PatchRecentItemCommand updates the mutable display fields of a recent item.
// resourceType and resourceUid are immutable and rejected if supplied.
// swagger:model
type PatchRecentItemCommand struct {
	// New display title.
	Title *string `json:"title,omitempty"`
	// New relative Grafana URL.
	URL *string `json:"url,omitempty"`
}

// ListRecentItemsQuery holds the supported list filters.
type ListRecentItemsQuery struct {
	// Restrict results to a single resource type.
	ResourceType ResourceType
	// Maximum number of items to return; bounded by MaxStoredItems.
	Limit int
}

// RecentItemResponse wraps a single recent item.
type RecentItemResponse struct {
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
		ResourceType: ResourceType(item.ResourceType),
		ResourceUID:  item.ResourceUID,
		Title:        item.Title,
		URL:          item.URL,
		LastViewedAt: item.LastViewedAt,
	}
}
