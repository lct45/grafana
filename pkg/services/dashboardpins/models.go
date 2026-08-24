package dashboardpins

import (
	"errors"
)

const (
	MaxPins               = 50
	MaxNoteLength         = 256
	MaxDashboardUIDLength = 40
)

var (
	ErrPinNotFound          = errors.New("dashboard pin not found")
	ErrPinAlreadyExists     = errors.New("dashboard pin already exists")
	ErrPinLimitReached      = errors.New("dashboard pin limit reached")
	ErrDashboardUIDRequired = errors.New("dashboardUid is required")
	ErrDashboardNotFound    = errors.New("dashboard not found")
	ErrNoteTooLong          = errors.New("note must be at most 256 characters")
	ErrInvalidReorder       = errors.New("dashboardUids must contain exactly the same dashboard UIDs as the current list")
	ErrInvalidDashboardUID  = errors.New("invalid dashboard UID")
)

// DashboardPin is the persistence model for dashboard pins.
type DashboardPin struct {
	ID           int64   `xorm:"pk autoincr 'id'"`
	OrgID        int64   `xorm:"org_id"`
	UserID       int64   `xorm:"user_id"`
	DashboardUID string  `xorm:"dashboard_uid"`
	Note         *string `xorm:"note"`
	SortOrder    int     `xorm:"sort_order"`
	CreatedAt    int64   `xorm:"created_at"`
}

// DashboardPinDTO is the API-facing pin representation.
type DashboardPinDTO struct {
	DashboardUID string  `json:"dashboardUid"`
	Note         *string `json:"note,omitempty"`
	SortOrder    int     `json:"sortOrder"`
	CreatedAt    int64   `json:"createdAt"`
}

// ListDashboardPinsResponse wraps a list of dashboard pins.
type ListDashboardPinsResponse struct {
	Pins []DashboardPinDTO `json:"pins"`
}

// CreateDashboardPinResponse wraps a created dashboard pin.
type CreateDashboardPinResponse struct {
	Pin DashboardPinDTO `json:"pin"`
}

// PatchDashboardPinResponse wraps an updated dashboard pin.
type PatchDashboardPinResponse struct {
	Pin DashboardPinDTO `json:"pin"`
}

// CreateDashboardPinCommand is the command for creating a dashboard pin.
// swagger:model
type CreateDashboardPinCommand struct {
	// UID of the dashboard to pin.
	// required: true
	// example: abc123
	DashboardUID string `json:"dashboardUid"`
	// Optional note for the pin.
	Note *string `json:"note,omitempty"`
}

// PatchDashboardPinCommand is the command for updating a dashboard pin note.
// swagger:model
type PatchDashboardPinCommand struct {
	// Updated note for the pin. Pass null to clear the note.
	Note *string `json:"note"`
}

// ReorderDashboardPinsCommand is the command for reordering dashboard pins.
// swagger:model
type ReorderDashboardPinsCommand struct {
	// Full list of dashboard UIDs in the desired order. Sort order is derived from array index.
	// required: true
	DashboardUIDs []string `json:"dashboardUids"`
}
