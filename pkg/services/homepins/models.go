package homepins

import "errors"

var (
	ErrPinNotFound      = errors.New("dashboard pin not found")
	ErrPinAlreadyExists = errors.New("dashboard is already pinned")
)

// DashboardHomePin is the persistence model for home dashboard pins.
type DashboardHomePin struct {
	ID            int64  `xorm:"pk autoincr 'id'"`
	UID           string `xorm:"uid"`
	OrgID         int64  `xorm:"org_id"`
	UserID        int64  `xorm:"user_id"`
	DashboardUID  string `xorm:"dashboard_uid"`
	SortOrder     int    `xorm:"sort_order"`
	Note          string `xorm:"note"`
	CreatedAt     int64  `xorm:"created_at"`
	UpdatedAt     int64  `xorm:"updated_at"`
}

// DashboardPinDTO is the API-facing pin representation.
type DashboardPinDTO struct {
	UID          string `json:"uid"`
	DashboardUID string `json:"dashboardUid"`
	SortOrder    int    `json:"sortOrder"`
	Note         string `json:"note,omitempty"`
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`
}

// CreatePinResponse wraps a created pin.
type CreatePinResponse struct {
	Pin DashboardPinDTO `json:"pin"`
}

// ListPinsResponse wraps a list of pins.
type ListPinsResponse struct {
	Pins []DashboardPinDTO `json:"pins"`
}

// UpdatePinResponse wraps an updated pin.
type UpdatePinResponse struct {
	Pin DashboardPinDTO `json:"pin"`
}

// ReorderPinsResponse is returned after reordering pins.
type ReorderPinsResponse struct {
	Message string `json:"message"`
}

// DeletePinResponse is returned after deleting a pin.
type DeletePinResponse struct {
	Message string `json:"message"`
}

// CreatePinCommand is the command for pinning a dashboard.
// swagger:model
type CreatePinCommand struct {
	// UID of the dashboard to pin.
	// required: true
	DashboardUID string `json:"dashboardUid"`
	// Optional short note displayed on the Home shelf.
	Note string `json:"note"`
}

// UpdatePinCommand is the command for updating a pin.
// swagger:model
type UpdatePinCommand struct {
	Note string `json:"note"`
}

// ReorderPinsCommand is the command for reordering pins.
// swagger:model
type ReorderPinsCommand struct {
	// Ordered list of pin UIDs.
	// required: true
	UIDs []string `json:"uids"`
}
