package dashboardpin

import (
	"errors"
	"time"
)

const MaxNoteLength = 500

var (
	ErrCommandValidationFailed = errors.New("command missing required fields")
	ErrPinNotFound             = errors.New("dashboard pin not found")
	ErrInvalidReorder          = errors.New("reorder list does not match current pins")
	ErrNoteTooLong             = errors.New("note exceeds maximum length")
)

type DashboardPin struct {
	ID           int64     `xorm:"pk autoincr 'id'" db:"id"`
	OrgID        int64     `xorm:"org_id" db:"org_id"`
	UserID       int64     `xorm:"user_id" db:"user_id"`
	DashboardUID string    `xorm:"dashboard_uid" db:"dashboard_uid"`
	SortOrder    int       `xorm:"sort_order" db:"sort_order"`
	Note         string    `xorm:"note" db:"note"`
	Created      time.Time `xorm:"created" db:"created"`
	Updated      time.Time `xorm:"updated" db:"updated"`
}

type DashboardPinDTO struct {
	DashboardUID string    `json:"dashboardUid"`
	SortOrder    int       `json:"sortOrder"`
	Note         string    `json:"note,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ListPinsQuery struct {
	UserID int64
	OrgID  int64
}

type PinDashboardCommand struct {
	UserID       int64
	OrgID        int64
	DashboardUID string
	Note         string
}

func (cmd *PinDashboardCommand) Validate() error {
	if cmd.UserID == 0 || cmd.OrgID == 0 || cmd.DashboardUID == "" {
		return ErrCommandValidationFailed
	}
	if len(cmd.Note) > MaxNoteLength {
		return ErrNoteTooLong
	}
	return nil
}

type UnpinDashboardCommand struct {
	UserID       int64
	OrgID        int64
	DashboardUID string
}

func (cmd *UnpinDashboardCommand) Validate() error {
	if cmd.UserID == 0 || cmd.OrgID == 0 || cmd.DashboardUID == "" {
		return ErrCommandValidationFailed
	}
	return nil
}

type UpdatePinNoteCommand struct {
	UserID       int64
	OrgID        int64
	DashboardUID string
	Note         string
}

func (cmd *UpdatePinNoteCommand) Validate() error {
	if cmd.UserID == 0 || cmd.OrgID == 0 || cmd.DashboardUID == "" {
		return ErrCommandValidationFailed
	}
	if len(cmd.Note) > MaxNoteLength {
		return ErrNoteTooLong
	}
	return nil
}

type ReorderPinsCommand struct {
	UserID        int64    `json:"-"`
	OrgID         int64    `json:"-"`
	DashboardUIDs []string `json:"dashboardUids"`
}

type PinDashboardRequest struct {
	Note string `json:"note"`
}

type UpdatePinNoteRequest struct {
	Note string `json:"note"`
}

func (cmd *ReorderPinsCommand) Validate() error {
	if cmd.UserID == 0 || cmd.OrgID == 0 || len(cmd.DashboardUIDs) == 0 {
		return ErrCommandValidationFailed
	}
	return nil
}

func ToDTO(pin DashboardPin) DashboardPinDTO {
	return DashboardPinDTO{
		DashboardUID: pin.DashboardUID,
		SortOrder:    pin.SortOrder,
		Note:         pin.Note,
		CreatedAt:    pin.Created,
		UpdatedAt:    pin.Updated,
	}
}
