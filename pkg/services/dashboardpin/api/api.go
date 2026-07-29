package api

import (
	"errors"
	"net/http"

	"github.com/grafana/grafana/pkg/api/response"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/dashboardpin"
	"github.com/grafana/grafana/pkg/web"
)

type API struct {
	service          dashboardpin.Service
	dashboardService dashboards.DashboardService
}

func ProvideApi(service dashboardpin.Service, dashboardService dashboards.DashboardService) *API {
	return &API{
		service:          service,
		dashboardService: dashboardService,
	}
}

func (api *API) ListPinnedDashboards(c *contextmodel.ReqContext) response.Response {
	pins, err := api.service.List(c.Req.Context(), &dashboardpin.ListPinsQuery{
		UserID: c.UserID,
		OrgID:  c.GetOrgID(),
	})
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to list pinned dashboards", err)
	}
	return response.JSON(http.StatusOK, pins)
}

func (api *API) PinDashboardByUID(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if uid == "" {
		return response.Error(http.StatusBadRequest, "Invalid dashboard UID", nil)
	}

	if _, err := api.dashboardService.GetDashboard(c.Req.Context(), &dashboards.GetDashboardQuery{
		UID:   uid,
		OrgID: c.GetOrgID(),
	}); err != nil {
		return response.Error(http.StatusNotFound, "Dashboard not found", err)
	}

	cmd := dashboardpin.PinDashboardCommand{
		UserID:       c.UserID,
		OrgID:        c.GetOrgID(),
		DashboardUID: uid,
	}
	req := dashboardpin.PinDashboardRequest{}
	if c.Req.ContentLength > 0 {
		if err := web.Bind(c.Req, &req); err != nil {
			return response.Error(http.StatusBadRequest, "bad request data", err)
		}
	}
	cmd.Note = req.Note

	pin, err := api.service.Pin(c.Req.Context(), &cmd)
	if err != nil {
		return pinErrorResponse(err)
	}
	return response.JSON(http.StatusOK, pin)
}

func (api *API) UnpinDashboardByUID(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if uid == "" {
		return response.Error(http.StatusBadRequest, "Invalid dashboard UID", nil)
	}

	err := api.service.Unpin(c.Req.Context(), &dashboardpin.UnpinDashboardCommand{
		UserID:       c.UserID,
		OrgID:        c.GetOrgID(),
		DashboardUID: uid,
	})
	if err != nil {
		return pinErrorResponse(err)
	}
	return response.Success("Dashboard unpinned")
}

func (api *API) UpdatePinNoteByUID(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if uid == "" {
		return response.Error(http.StatusBadRequest, "Invalid dashboard UID", nil)
	}

	req := dashboardpin.UpdatePinNoteRequest{}
	if err := web.Bind(c.Req, &req); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}
	cmd := dashboardpin.UpdatePinNoteCommand{
		UserID:       c.UserID,
		OrgID:        c.GetOrgID(),
		DashboardUID: uid,
		Note:         req.Note,
	}

	pin, err := api.service.UpdateNote(c.Req.Context(), &cmd)
	if err != nil {
		return pinErrorResponse(err)
	}
	return response.JSON(http.StatusOK, pin)
}

func (api *API) ReorderPinnedDashboards(c *contextmodel.ReqContext) response.Response {
	cmd := dashboardpin.ReorderPinsCommand{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}
	cmd.UserID = c.UserID
	cmd.OrgID = c.GetOrgID()

	if err := api.service.Reorder(c.Req.Context(), &cmd); err != nil {
		return pinErrorResponse(err)
	}
	return response.Success("Pinned dashboards reordered")
}

func pinErrorResponse(err error) response.Response {
	switch {
	case errors.Is(err, dashboardpin.ErrPinNotFound):
		return response.Error(http.StatusNotFound, "Dashboard pin not found", err)
	case errors.Is(err, dashboardpin.ErrInvalidReorder):
		return response.Error(http.StatusBadRequest, "Invalid reorder request", err)
	case errors.Is(err, dashboardpin.ErrNoteTooLong):
		return response.Error(http.StatusBadRequest, "Note exceeds maximum length", err)
	case errors.Is(err, dashboardpin.ErrCommandValidationFailed):
		return response.Error(http.StatusBadRequest, "Invalid request", err)
	default:
		return response.Error(http.StatusInternalServerError, "Failed to update pinned dashboards", err)
	}
}
