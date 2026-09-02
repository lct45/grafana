package recentitems

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/middleware"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/web"
)

const recentItemsPath = "/api/user/RECENT_ITEMS"

func (s *RecentItemsService) registerAPIEndpoints() {
	s.routeRegister.Group(recentItemsPath, func(items routing.RouteRegister) {
		items.Post("/", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.createHandler))
		items.Get("/", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.listHandler))
		items.Patch("/:uid", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.patchHandler))
		items.Delete("/:uid", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.deleteHandler))
	})
}

// swagger:route POST /user/RECENT_ITEMS signed_in_user createRecentItem
//
// Record a recently viewed resource.
//
// Responses:
// 200: recentItemResponse
// 201: recentItemResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *RecentItemsService) createHandler(c *contextmodel.ReqContext) response.Response {
	var cmd CreateRecentItemCommand
	if err := decodeStrictJSON(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "Invalid request body", err)
	}

	item, created, err := s.Upsert(c.Req.Context(), c.SignedInUser, cmd)
	if err != nil {
		return recentItemError("Failed to record recent item", err)
	}

	if !created {
		return response.JSON(http.StatusOK, item)
	}
	return response.JSON(http.StatusCreated, item).
		SetHeader("Location", fmt.Sprintf("%s/%s", recentItemsPath, item.UID))
}

// swagger:route GET /user/RECENT_ITEMS signed_in_user listRecentItems
//
// List recently viewed resources for the signed-in user and organization.
//
// Responses:
// 200: listRecentItemsResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *RecentItemsService) listHandler(c *contextmodel.ReqContext) response.Response {
	limit := DefaultLimit
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			return response.Error(http.StatusBadRequest, "Invalid limit", ErrInvalidLimit)
		}
		limit = parsed
	}

	items, err := s.List(c.Req.Context(), c.SignedInUser, limit)
	if err != nil {
		return recentItemError("Failed to list recent items", err)
	}
	return response.JSON(http.StatusOK, ListRecentItemsResponse{Items: items})
}

// swagger:route PATCH /user/RECENT_ITEMS/{uid} signed_in_user patchRecentItem
//
// Update a recently viewed resource.
//
// Responses:
// 200: recentItemResponse
// 400: badRequestError
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *RecentItemsService) patchHandler(c *contextmodel.ReqContext) response.Response {
	var cmd PatchRecentItemCommand
	if err := decodeStrictJSON(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "Invalid request body", err)
	}

	item, err := s.Patch(c.Req.Context(), c.SignedInUser, web.Params(c.Req)[":uid"], cmd)
	if err != nil {
		return recentItemError("Failed to update recent item", err)
	}
	return response.JSON(http.StatusOK, item)
}

// swagger:route DELETE /user/RECENT_ITEMS/{uid} signed_in_user deleteRecentItem
//
// Delete a recently viewed resource.
//
// Responses:
// 200: deleteRecentItemResponse
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *RecentItemsService) deleteHandler(c *contextmodel.ReqContext) response.Response {
	err := s.Delete(c.Req.Context(), c.SignedInUser, web.Params(c.Req)[":uid"])
	if err != nil {
		return recentItemError("Failed to delete recent item", err)
	}
	return response.JSON(http.StatusOK, DeleteRecentItemResponse{Message: "Recent item deleted"})
}

func recentItemError(message string, err error) response.Response {
	switch {
	case errors.Is(err, ErrItemNotFound):
		return response.Error(http.StatusNotFound, "Recent item not found", err)
	case errors.Is(err, ErrInvalidResourceType),
		errors.Is(err, ErrInvalidResourceUID),
		errors.Is(err, ErrInvalidTitle),
		errors.Is(err, ErrInvalidURL),
		errors.Is(err, ErrInvalidTimestamp),
		errors.Is(err, ErrInvalidLimit),
		errors.Is(err, ErrEmptyPatch):
		return response.Error(http.StatusBadRequest, err.Error(), err)
	default:
		return response.Error(http.StatusInternalServerError, message, err)
	}
}

func decodeStrictJSON(req *http.Request, value any) error {
	decoder := json.NewDecoder(req.Body)
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

// swagger:parameters createRecentItem
type CreateRecentItemParams struct {
	// in:body
	// required:true
	Body CreateRecentItemCommand `json:"body"`
}

// swagger:parameters listRecentItems
type ListRecentItemsParams struct {
	// Maximum number of items to return.
	// in:query
	// minimum:1
	// maximum:50
	Limit int `json:"limit"`
}

// swagger:parameters patchRecentItem
type PatchRecentItemParams struct {
	// in:path
	// required:true
	UID string `json:"uid"`
	// in:body
	// required:true
	Body PatchRecentItemCommand `json:"body"`
}

// swagger:parameters deleteRecentItem
type DeleteRecentItemParams struct {
	// in:path
	// required:true
	UID string `json:"uid"`
}

// swagger:response recentItemResponse
type RecentItemResponseSwagger struct {
	// in:body
	Body RecentItemDTO `json:"body"`
}

// swagger:response listRecentItemsResponse
type ListRecentItemsResponseSwagger struct {
	// in:body
	Body ListRecentItemsResponse `json:"body"`
}

// swagger:response deleteRecentItemResponse
type DeleteRecentItemResponseSwagger struct {
	// in:body
	Body DeleteRecentItemResponse `json:"body"`
}
