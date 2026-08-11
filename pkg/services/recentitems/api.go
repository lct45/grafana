package recentitems

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/middleware"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/web"
)

func (s *RecentItemsService) registerAPIEndpoints() {
	s.RouteRegister.Group("/api/user/RECENT_ITEMS", func(entities routing.RouteRegister) {
		entities.Post("/", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.createHandler))
		entities.Get("/", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.listHandler))
		entities.Patch("/:uid", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.patchHandler))
		entities.Delete("/:uid", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.deleteHandler))
	})
}

// swagger:route POST /user/RECENT_ITEMS recent_items createRecentItem
//
// Create or update a recently viewed item for the signed-in user.
//
// Responses:
// 200: createRecentItemResponse
// 201: createRecentItemResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *RecentItemsService) createHandler(c *contextmodel.ReqContext) response.Response {
	cmd := CreateRecentItemCommand{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	item, created, err := s.CreateOrUpdateRecentItem(c.Req.Context(), c.SignedInUser, cmd)
	if err != nil {
		if isValidationError(err) {
			return response.Error(http.StatusBadRequest, err.Error(), err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to create recent item", err)
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	return response.JSON(status, CreateRecentItemResponse{Item: item})
}

// swagger:route GET /user/RECENT_ITEMS recent_items listRecentItems
//
// List recently viewed items for the signed-in user.
//
// Responses:
// 200: listRecentItemsResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *RecentItemsService) listHandler(c *contextmodel.ReqContext) response.Response {
	limit := defaultListLimit
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxListLimit {
			return response.Error(http.StatusBadRequest, ErrInvalidLimit.Error(), ErrInvalidLimit)
		}
		limit = parsed
	}

	items, err := s.ListRecentItems(c.Req.Context(), c.SignedInUser, limit)
	if err != nil {
		if errors.Is(err, ErrInvalidLimit) {
			return response.Error(http.StatusBadRequest, err.Error(), err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to list recent items", err)
	}

	return response.JSON(http.StatusOK, ListRecentItemsResponse{Items: items})
}

// swagger:route PATCH /user/RECENT_ITEMS/{uid} recent_items patchRecentItem
//
// Partially update a recently viewed item for the signed-in user.
//
// Responses:
// 200: patchRecentItemResponse
// 400: badRequestError
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *RecentItemsService) patchHandler(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if len(uid) > 0 && !util.IsValidShortUID(uid) {
		return response.Error(http.StatusNotFound, "Recent item not found", nil)
	}

	var raw map[string]json.RawMessage
	dec := json.NewDecoder(c.Req.Body)
	if err := dec.Decode(&raw); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}
	if _, ok := raw["resourceType"]; ok {
		return response.Error(http.StatusBadRequest, ErrImmutableField.Error(), ErrImmutableField)
	}
	if _, ok := raw["resourceUid"]; ok {
		return response.Error(http.StatusBadRequest, ErrImmutableField.Error(), ErrImmutableField)
	}

	cmd := PatchRecentItemCommand{}
	body, err := json.Marshal(raw)
	if err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}
	if err := json.Unmarshal(body, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	item, err := s.PatchRecentItem(c.Req.Context(), c.SignedInUser, uid, cmd)
	if err != nil {
		if errors.Is(err, ErrRecentItemNotFound) {
			return response.Error(http.StatusNotFound, "Recent item not found", err)
		}
		if isValidationError(err) {
			return response.Error(http.StatusBadRequest, err.Error(), err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to update recent item", err)
	}

	return response.JSON(http.StatusOK, PatchRecentItemResponse{Item: item})
}

// swagger:route DELETE /user/RECENT_ITEMS/{uid} recent_items deleteRecentItem
//
// Delete a recently viewed item for the signed-in user.
//
// Responses:
// 200: deleteRecentItemResponse
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *RecentItemsService) deleteHandler(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if len(uid) > 0 && !util.IsValidShortUID(uid) {
		return response.Error(http.StatusNotFound, "Recent item not found", nil)
	}

	err := s.DeleteRecentItem(c.Req.Context(), c.SignedInUser, uid)
	if err != nil {
		if errors.Is(err, ErrRecentItemNotFound) {
			return response.Error(http.StatusNotFound, "Recent item not found", err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to delete recent item", err)
	}

	return response.JSON(http.StatusOK, DeleteRecentItemResponse{Message: "Recent item deleted"})
}

func isValidationError(err error) bool {
	return errors.Is(err, ErrResourceTypeRequired) ||
		errors.Is(err, ErrResourceTypeInvalid) ||
		errors.Is(err, ErrResourceUIDRequired) ||
		errors.Is(err, ErrTitleRequired) ||
		errors.Is(err, ErrURLRequired) ||
		errors.Is(err, ErrTitleTooLong) ||
		errors.Is(err, ErrURLTooLong) ||
		errors.Is(err, ErrResourceUIDTooLong) ||
		errors.Is(err, ErrPatchEmpty) ||
		errors.Is(err, ErrImmutableField) ||
		errors.Is(err, ErrInvalidLimit)
}

// swagger:parameters createRecentItem
type CreateRecentItemParams struct {
	// in:body
	// required:true
	Body CreateRecentItemCommand `json:"body"`
}

// swagger:parameters listRecentItems
type ListRecentItemsParams struct {
	// Maximum number of items to return (default 50, max 100).
	// in:query
	// required:false
	Limit *int `json:"limit"`
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
type RecentItemByUID struct {
	// in:path
	// required:true
	UID string `json:"uid"`
}

// swagger:response createRecentItemResponse
type CreateRecentItemResponseSwagger struct {
	// in: body
	Body CreateRecentItemResponse `json:"body"`
}

// swagger:response listRecentItemsResponse
type ListRecentItemsResponseSwagger struct {
	// in: body
	Body ListRecentItemsResponse `json:"body"`
}

// swagger:response patchRecentItemResponse
type PatchRecentItemResponseSwagger struct {
	// in: body
	Body PatchRecentItemResponse `json:"body"`
}

// swagger:response deleteRecentItemResponse
type DeleteRecentItemResponseSwagger struct {
	// in: body
	Body DeleteRecentItemResponse `json:"body"`
}
