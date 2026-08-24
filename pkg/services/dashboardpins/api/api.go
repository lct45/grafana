package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/grafana/grafana/pkg/api/response"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/dashboardpins"
	"github.com/grafana/grafana/pkg/web"
)

type API struct {
	service dashboardpins.Service
}

func ProvideApi(service dashboardpins.Service) *API {
	return &API{service: service}
}

// swagger:route GET /user/dashboard-pins signed_in_user listDashboardPins
//
// List dashboard pins for the signed-in user.
//
// Responses:
// 200: listDashboardPinsResponse
// 401: unauthorisedError
// 500: internalServerError
func (a *API) ListDashboardPins(c *contextmodel.ReqContext) response.Response {
	pins, err := a.service.ListPins(c.Req.Context(), c.SignedInUser)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to list dashboard pins", err)
	}

	return response.JSON(http.StatusOK, dashboardpins.ListDashboardPinsResponse{Pins: pins})
}

// swagger:route POST /user/dashboard-pins signed_in_user createDashboardPin
//
// Pin a dashboard for the signed-in user.
//
// Responses:
// 201: createDashboardPinResponse
// 400: badRequestError
// 401: unauthorisedError
// 404: notFoundError
// 409: conflictError
// 500: internalServerError
func (a *API) CreateDashboardPin(c *contextmodel.ReqContext) response.Response {
	cmd := dashboardpins.CreateDashboardPinCommand{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	pin, err := a.service.CreatePin(c.Req.Context(), c.SignedInUser, cmd)
	if err != nil {
		return mapDashboardPinError(err, "Failed to create dashboard pin")
	}

	location := "/api/user/dashboard-pins/" + pin.DashboardUID
	return response.JSON(http.StatusCreated, dashboardpins.CreateDashboardPinResponse{Pin: pin}).SetHeader("Location", location)
}

// swagger:route PUT /user/dashboard-pins signed_in_user reorderDashboardPins
//
// Reorder dashboard pins for the signed-in user.
//
// The request body must contain the full list of dashboard UIDs in the desired order. Sort order is derived from array index. The set of dashboard UIDs must match the current pins exactly.
//
// Responses:
// 200: listDashboardPinsResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (a *API) ReorderDashboardPins(c *contextmodel.ReqContext) response.Response {
	cmd := dashboardpins.ReorderDashboardPinsCommand{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	pins, err := a.service.ReorderPins(c.Req.Context(), c.SignedInUser, cmd)
	if err != nil {
		return mapDashboardPinError(err, "Failed to reorder dashboard pins")
	}

	return response.JSON(http.StatusOK, dashboardpins.ListDashboardPinsResponse{Pins: pins})
}

// swagger:route PATCH /user/dashboard-pins/{dashboard_uid} signed_in_user patchDashboardPin
//
// Update the note on a dashboard pin.
//
// Responses:
// 200: patchDashboardPinResponse
// 400: badRequestError
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (a *API) PatchDashboardPin(c *contextmodel.ReqContext) response.Response {
	dashboardUID := web.Params(c.Req)[":dashboardUid"]

	cmd := dashboardpins.PatchDashboardPinCommand{}
	if err := bindPatchDashboardPinCommand(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	pin, err := a.service.PatchPin(c.Req.Context(), c.SignedInUser, dashboardUID, cmd)
	if err != nil {
		return mapDashboardPinError(err, "Failed to update dashboard pin")
	}

	return response.JSON(http.StatusOK, dashboardpins.PatchDashboardPinResponse{Pin: pin})
}

// swagger:route DELETE /user/dashboard-pins/{dashboard_uid} signed_in_user deleteDashboardPin
//
// Unpin a dashboard for the signed-in user.
//
// Responses:
// 200: okResponse
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (a *API) DeleteDashboardPin(c *contextmodel.ReqContext) response.Response {
	dashboardUID := web.Params(c.Req)[":dashboardUid"]

	err := a.service.DeletePin(c.Req.Context(), c.SignedInUser, dashboardUID)
	if err != nil {
		return mapDashboardPinError(err, "Failed to delete dashboard pin")
	}

	return response.Success("Dashboard pin deleted")
}

func bindPatchDashboardPinCommand(req *http.Request, cmd *dashboardpins.PatchDashboardPinCommand) error {
	body := http.MaxBytesReader(nil, req.Body, web.MaxBindBodyBytes)
	defer func() { _ = body.Close() }()

	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(cmd); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	if decoder.More() {
		return errors.New("multiple JSON objects not allowed")
	}

	return nil
}

func mapDashboardPinError(err error, fallbackMessage string) *response.NormalResponse {
	switch {
	case errors.Is(err, dashboardpins.ErrPinNotFound), errors.Is(err, dashboardpins.ErrDashboardNotFound):
		return response.Error(http.StatusNotFound, err.Error(), err)
	case errors.Is(err, dashboardpins.ErrPinAlreadyExists):
		return response.Error(http.StatusConflict, err.Error(), err)
	case errors.Is(err, dashboardpins.ErrPinLimitReached),
		errors.Is(err, dashboardpins.ErrDashboardUIDRequired),
		errors.Is(err, dashboardpins.ErrNoteTooLong),
		errors.Is(err, dashboardpins.ErrInvalidReorder),
		errors.Is(err, dashboardpins.ErrInvalidDashboardUID):
		return response.Error(http.StatusBadRequest, err.Error(), err)
	default:
		return response.Error(http.StatusInternalServerError, fallbackMessage, err)
	}
}

// swagger:parameters createDashboardPin
type CreateDashboardPinParams struct {
	// in:body
	// required:true
	Body dashboardpins.CreateDashboardPinCommand `json:"body"`
}

// swagger:parameters reorderDashboardPins
type ReorderDashboardPinsParams struct {
	// in:body
	// required:true
	Body dashboardpins.ReorderDashboardPinsCommand `json:"body"`
}

// swagger:parameters patchDashboardPin
type PatchDashboardPinParams struct {
	// in:body
	// required:true
	Body dashboardpins.PatchDashboardPinCommand `json:"body"`
}

// swagger:parameters patchDashboardPin deleteDashboardPin
type DashboardPinByDashboardUID struct {
	// in:path
	// required:true
	DashboardUID string `json:"dashboard_uid"`
}

// swagger:response listDashboardPinsResponse
type ListDashboardPinsResponseSwagger struct {
	// in: body
	Body dashboardpins.ListDashboardPinsResponse `json:"body"`
}

// swagger:response createDashboardPinResponse
type CreateDashboardPinResponseSwagger struct {
	// in: body
	Body dashboardpins.CreateDashboardPinResponse `json:"body"`
}

// swagger:response patchDashboardPinResponse
type PatchDashboardPinResponseSwagger struct {
	// in: body
	Body dashboardpins.PatchDashboardPinResponse `json:"body"`
}
