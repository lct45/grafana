package recentitems

import "errors"

const (
	defaultRecentItemsLimit = 20
	maxRecentItemsLimit     = 50
	// maxRecentItemsPerUser is how many rows are kept for one user in one org.
	// Older rows are removed when a new distinct resource is recorded.
	maxRecentItemsPerUser = 50

	maxResourceTypeLength = 40
	maxResourceUIDLength  = 40
	maxTitleLength        = 255
	maxURLLength          = 1024
)

var (
	ErrRecentItemNotFound = errors.New("recent item not found")

	ErrResourceTypeRequired = errors.New("resourceType is required")
	ErrResourceTypeInvalid  = errors.New("resourceType is invalid")
	ErrResourceUIDRequired  = errors.New("resourceUid is required")
	ErrResourceUIDTooLong   = errors.New("resourceUid must be at most 40 characters")
	ErrTitleRequired        = errors.New("title is required")
	ErrTitleTooLong         = errors.New("title must be at most 255 characters")
	ErrURLRequired          = errors.New("url is required")
	ErrURLInvalid           = errors.New("url must be a relative path")
	ErrURLTooLong           = errors.New("url must be at most 1024 characters")
	ErrInvalidLimit         = errors.New("limit must be between 1 and 50")
	ErrPatchEmpty           = errors.New("title or url is required")
	ErrImmutableField       = errors.New("uid, resourceType, resourceUid, and lastViewedAt cannot be changed")
	ErrUnknownField         = errors.New("unknown field")
	ErrMalformedBody        = errors.New("bad request data")
)

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
type RecentItemDTO struct {
	UID          string `json:"uid"`
	ResourceType string `json:"resourceType"`
	ResourceUID  string `json:"resourceUid"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	LastViewedAt int64  `json:"lastViewedAt"`
}

// ListRecentItemsResponse wraps a list of recent items.
type ListRecentItemsResponse struct {
	RecentItems []RecentItemDTO `json:"recentItems"`
}

// ListRecentItemsQuery selects a page of recent items for the signed-in user.
type ListRecentItemsQuery struct {
	ResourceType string
	Limit        int
}

// CreateRecentItemCommand records a view of a resource.
// swagger:model
type CreateRecentItemCommand struct {
	// Kind of resource, such as dashboard or folder.
	// required: true
	ResourceType string `json:"resourceType"`
	// UID of the viewed resource.
	// required: true
	ResourceUID string `json:"resourceUid"`
	// Display title captured at view time.
	// required: true
	Title string `json:"title"`
	// Relative URL used to open the resource again.
	// required: true
	URL string `json:"url"`
}

// PatchRecentItemCommand updates display fields on an existing recent item.
// swagger:model
type PatchRecentItemCommand struct {
	Title *string `json:"title,omitempty"`
	URL   *string `json:"url,omitempty"`
}
