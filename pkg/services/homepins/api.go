package homepins

import (
	"errors"
	"net/http"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/middleware"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/web"
)

func (s *HomePinsService) registerAPIEndpoints() {
	s.RouteRegister.Group("/api/home/dashboard-pins", func(entities routing.RouteRegister) {
		entities.Post("/", middleware.ReqSignedIn, routing.Wrap(s.createHandler))
		entities.Get("/", middleware.ReqSignedIn, routing.Wrap(s.listHandler))
		entities.Put("/reorder", middleware.ReqSignedIn, routing.Wrap(s.reorderHandler))
		entities.Patch("/:uid", middleware.ReqSignedIn, routing.Wrap(s.updateHandler))
		entities.Delete("/:uid", middleware.ReqSignedIn, routing.Wrap(s.deleteHandler))
	})
}

// swagger:route POST /home/dashboard-pins home_pins createDashboardPin
//
// Pin a dashboard to the Home shelf.
//
// Responses:
// 201: createDashboardPinResponse
// 400: badRequestError
// 401: unauthorisedError
// 409: conflictError
// 500: internalServerError
func (s *HomePinsService) createHandler(c *contextmodel.ReqContext) response.Response {
	cmd := CreatePinCommand{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	pin, err := s.CreatePin(c.Req.Context(), c.SignedInUser, cmd)
	if err != nil {
		if errors.Is(err, ErrPinAlreadyExists) {
			return response.Error(http.StatusConflict, err.Error(), err)
		}
		if isValidationError(err) {
			return response.Error(http.StatusBadRequest, err.Error(), err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to create dashboard pin", err)
	}

	return response.JSON(http.StatusCreated, CreatePinResponse{Pin: pin})
}

// swagger:route GET /home/dashboard-pins home_pins listDashboardPins
//
// List pinned dashboards for the signed-in user.
//
// Responses:
// 200: listDashboardPinsResponse
// 401: unauthorisedError
// 500: internalServerError
func (s *HomePinsService) listHandler(c *contextmodel.ReqContext) response.Response {
	pins, err := s.ListPins(c.Req.Context(), c.SignedInUser)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to list dashboard pins", err)
	}

	return response.JSON(http.StatusOK, ListPinsResponse{Pins: pins})
}

// swagger:route PATCH /home/dashboard-pins/{uid} home_pins updateDashboardPin
//
// Update a dashboard pin note.
//
// Responses:
// 200: updateDashboardPinResponse
// 400: badRequestError
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *HomePinsService) updateHandler(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if len(uid) > 0 && !util.IsValidShortUID(uid) {
		return response.Error(http.StatusNotFound, "Dashboard pin not found", nil)
	}

	cmd := UpdatePinCommand{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	pin, err := s.UpdatePin(c.Req.Context(), c.SignedInUser, uid, cmd)
	if err != nil {
		if errors.Is(err, ErrPinNotFound) {
			return response.Error(http.StatusNotFound, "Dashboard pin not found", err)
		}
		if isValidationError(err) {
			return response.Error(http.StatusBadRequest, err.Error(), err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to update dashboard pin", err)
	}

	return response.JSON(http.StatusOK, UpdatePinResponse{Pin: pin})
}

// swagger:route PUT /home/dashboard-pins/reorder home_pins reorderDashboardPins
//
// Reorder pinned dashboards for the signed-in user.
//
// Responses:
// 200: reorderDashboardPinsResponse
// 400: badRequestError
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *HomePinsService) reorderHandler(c *contextmodel.ReqContext) response.Response {
	cmd := ReorderPinsCommand{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	err := s.ReorderPins(c.Req.Context(), c.SignedInUser, cmd)
	if err != nil {
		if errors.Is(err, ErrPinNotFound) {
			return response.Error(http.StatusNotFound, "Dashboard pin not found", err)
		}
		if isValidationError(err) {
			return response.Error(http.StatusBadRequest, err.Error(), err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to reorder dashboard pins", err)
	}

	return response.JSON(http.StatusOK, ReorderPinsResponse{Message: "Pins reordered"})
}

// swagger:route DELETE /home/dashboard-pins/{uid} home_pins deleteDashboardPin
//
// Unpin a dashboard from the Home shelf.
//
// Responses:
// 200: deleteDashboardPinResponse
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *HomePinsService) deleteHandler(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if len(uid) > 0 && !util.IsValidShortUID(uid) {
		return response.Error(http.StatusNotFound, "Dashboard pin not found", nil)
	}

	err := s.DeletePin(c.Req.Context(), c.SignedInUser, uid)
	if err != nil {
		if errors.Is(err, ErrPinNotFound) {
			return response.Error(http.StatusNotFound, "Dashboard pin not found", err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to delete dashboard pin", err)
	}

	return response.JSON(http.StatusOK, DeletePinResponse{Message: "Pin deleted"})
}

func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "dashboardUid is required" ||
		msg == "uids are required" ||
		msg == "uids must include all pins" ||
		msg == "duplicate uid in reorder request" ||
		msg == "note must be at most 255 characters"
}
