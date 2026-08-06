package recentitems

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/middleware"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/web"
)

func (s *ServiceImpl) registerAPIEndpoints() {
	s.routeRegister.Group("/api/user/recent-items", func(items routing.RouteRegister) {
		items.Post("/", middleware.ReqSignedIn, routing.Wrap(s.createHandler))
		items.Get("/", middleware.ReqSignedIn, routing.Wrap(s.listHandler))
		items.Patch("/:uid", middleware.ReqSignedIn, routing.Wrap(s.patchHandler))
		items.Delete("/:uid", middleware.ReqSignedIn, routing.Wrap(s.deleteHandler))
	})
}

// swagger:route POST /user/recent-items recent_items createRecentItem
//
// Create or refresh a recent item.
//
// Responses:
// 200: recentItemResponse
// 201: recentItemResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *ServiceImpl) createHandler(c *contextmodel.ReqContext) response.Response {
	cmd := CreateRecentItemCommand{}
	if err := decodeStrictJSON(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	result, err := s.UpsertRecentItem(c.Req.Context(), c.SignedInUser, cmd)
	if err != nil {
		if isValidationError(err) {
			return response.Error(http.StatusBadRequest, err.Error(), err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to save recent item", err)
	}

	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	resp := response.JSON(status, RecentItemResponse{Item: result.Item})
	if result.Created {
		resp.SetHeader("Location", fmt.Sprintf("/api/user/recent-items/%s", result.Item.UID))
	}
	return resp
}

// swagger:route GET /user/recent-items recent_items listRecentItems
//
// List recent items for the signed-in user.
//
// Responses:
// 200: recentItemsResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *ServiceImpl) listHandler(c *contextmodel.ReqContext) response.Response {
	items, err := s.ListRecentItems(c.Req.Context(), c.SignedInUser, ListRecentItemsQuery{
		ResourceType: c.Query("resourceType"),
		Limit:        c.QueryInt("limit"),
	})
	if err != nil {
		if isValidationError(err) {
			return response.Error(http.StatusBadRequest, err.Error(), err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to list recent items", err)
	}
	return response.JSON(http.StatusOK, RecentItemsResponse{Items: items})
}

// swagger:route PATCH /user/recent-items/{uid} recent_items patchRecentItem
//
// Update mutable recent-item metadata.
//
// Responses:
// 200: recentItemResponse
// 400: badRequestError
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *ServiceImpl) patchHandler(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if uid == "" || !util.IsValidShortUID(uid) {
		return response.Error(http.StatusNotFound, "Recent item not found", nil)
	}

	cmd := PatchRecentItemCommand{}
	if err := decodeStrictJSON(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	item, err := s.PatchRecentItem(c.Req.Context(), c.SignedInUser, uid, cmd)
	if err != nil {
		switch {
		case errors.Is(err, ErrRecentItemNotFound):
			return response.Error(http.StatusNotFound, "Recent item not found", err)
		case isValidationError(err):
			return response.Error(http.StatusBadRequest, err.Error(), err)
		default:
			return response.Error(http.StatusInternalServerError, "Failed to update recent item", err)
		}
	}
	return response.JSON(http.StatusOK, RecentItemResponse{Item: item})
}

// swagger:route DELETE /user/recent-items/{uid} recent_items deleteRecentItem
//
// Delete a recent item.
//
// Responses:
// 204: deleteRecentItemResponse
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *ServiceImpl) deleteHandler(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if uid == "" || !util.IsValidShortUID(uid) {
		return response.Error(http.StatusNotFound, "Recent item not found", nil)
	}

	err := s.DeleteRecentItem(c.Req.Context(), c.SignedInUser, uid)
	if err != nil {
		if errors.Is(err, ErrRecentItemNotFound) {
			return response.Error(http.StatusNotFound, "Recent item not found", err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to delete recent item", err)
	}
	return response.Empty(http.StatusNoContent)
}

func decodeStrictJSON(r *http.Request, value any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain a single JSON object")
		}
		return err
	}
	return nil
}

func isValidationError(err error) bool {
	return errors.Is(err, ErrInvalidResourceType) ||
		errors.Is(err, ErrNoPatchFields) ||
		errors.Is(err, ErrResourceUIDRequired) ||
		errors.Is(err, ErrResourceUIDTooLong) ||
		errors.Is(err, ErrTitleTooLong) ||
		errors.Is(err, ErrURLTooLong)
}

// swagger:parameters createRecentItem
type CreateRecentItemParams struct {
	// in:body
	// required:true
	Body CreateRecentItemCommand `json:"body"`
}

// swagger:parameters listRecentItems
type ListRecentItemsParams struct {
	// in:query
	ResourceType string `json:"resourceType"`
	// in:query
	Limit int `json:"limit"`
}

// swagger:parameters patchRecentItem deleteRecentItem
type RecentItemByUIDParams struct {
	// in:path
	// required:true
	UID string `json:"uid"`
}

// swagger:parameters patchRecentItem
type PatchRecentItemParams struct {
	// in:body
	// required:true
	Body PatchRecentItemCommand `json:"body"`
}

// swagger:response recentItemResponse
type RecentItemResponseSwagger struct {
	// in:body
	Body RecentItemResponse `json:"body"`
}

// swagger:response recentItemsResponse
type RecentItemsResponseSwagger struct {
	// in:body
	Body RecentItemsResponse `json:"body"`
}

// swagger:response deleteRecentItemResponse
type DeleteRecentItemResponseSwagger struct{}
