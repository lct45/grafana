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
	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/web"
)

const (
	basePath = "/api/user/recent-items"
	// maxBodyBytes is far above any valid payload (title 255 + url 2048) and
	// stops oversized bodies before they reach the JSON decoder.
	maxBodyBytes = 16 << 10
)

// Recent items are personal data, so anonymous sessions are rejected even
// though they may otherwise be "signed in" via anonymous access.
func (s *RecentItemsService) registerAPIEndpoints() {
	s.RouteRegister.Group(basePath, func(entities routing.RouteRegister) {
		entities.Post("/", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.createHandler))
		entities.Get("/", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.listHandler))
		entities.Patch("/:uid", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.patchHandler))
		entities.Delete("/:uid", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.deleteHandler))
	})
}

// swagger:route POST /user/recent-items recent_items createRecentItem
//
// Record that the signed-in user viewed a resource.
//
// Creates a recent item, or refreshes the title, URL and last-viewed time of
// the existing item for the same resource type and UID.
//
// Responses:
// 200: recentItemResponse
// 201: recentItemResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *RecentItemsService) createHandler(c *contextmodel.ReqContext) response.Response {
	cmd := CreateRecentItemCommand{}
	if err := decodeStrictJSON(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, ErrRequestBodyInvalid.Error(), err)
	}

	item, created, err := s.RecordView(c.Req.Context(), c.SignedInUser, cmd)
	if err != nil {
		if isValidationError(err) {
			return response.Error(http.StatusBadRequest, err.Error(), err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to record recent item", err)
	}

	if !created {
		return response.JSON(http.StatusOK, RecentItemResponse{Item: item})
	}
	return response.JSON(http.StatusCreated, RecentItemResponse{Item: item}).
		SetHeader("Location", fmt.Sprintf("%s/%s", basePath, item.UID))
}

// swagger:route GET /user/recent-items recent_items listRecentItems
//
// List the signed-in user's recently viewed resources, most recent first.
//
// Responses:
// 200: listRecentItemsResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *RecentItemsService) listHandler(c *contextmodel.ReqContext) response.Response {
	query := ListRecentItemsQuery{
		ResourceType: ResourceType(c.Query("resourceType")),
	}
	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			return response.Error(http.StatusBadRequest, ErrListLimitInvalid.Error(), err)
		}
		query.Limit = limit
	}

	items, err := s.ListItems(c.Req.Context(), c.SignedInUser, query)
	if err != nil {
		if isValidationError(err) {
			return response.Error(http.StatusBadRequest, err.Error(), err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to list recent items", err)
	}

	return response.JSON(http.StatusOK, ListRecentItemsResponse{Items: items})
}

// swagger:route PATCH /user/recent-items/{uid} recent_items patchRecentItem
//
// Update the title or URL of a recent item.
//
// Responses:
// 200: recentItemResponse
// 400: badRequestError
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *RecentItemsService) patchHandler(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if !isPlausibleUID(uid) {
		return response.Error(http.StatusNotFound, "Recent item not found", nil)
	}

	cmd := PatchRecentItemCommand{}
	if err := decodeStrictJSON(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, ErrRequestBodyInvalid.Error(), err)
	}

	item, err := s.PatchItem(c.Req.Context(), c.SignedInUser, uid, cmd)
	if err != nil {
		if isValidationError(err) {
			return response.Error(http.StatusBadRequest, err.Error(), err)
		}
		if errors.Is(err, ErrRecentItemNotFound) {
			return response.Error(http.StatusNotFound, "Recent item not found", err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to update recent item", err)
	}

	return response.JSON(http.StatusOK, RecentItemResponse{Item: item})
}

// swagger:route DELETE /user/recent-items/{uid} recent_items deleteRecentItem
//
// Delete a recent item.
//
// Responses:
// 200: deleteRecentItemResponse
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *RecentItemsService) deleteHandler(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if !isPlausibleUID(uid) {
		return response.Error(http.StatusNotFound, "Recent item not found", nil)
	}

	if err := s.DeleteItem(c.Req.Context(), c.SignedInUser, uid); err != nil {
		if errors.Is(err, ErrRecentItemNotFound) {
			return response.Error(http.StatusNotFound, "Recent item not found", err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to delete recent item", err)
	}

	return response.JSON(http.StatusOK, DeleteRecentItemResponse{Message: "Recent item deleted"})
}

func isPlausibleUID(uid string) bool {
	return uid != "" && !util.IsShortUIDTooLong(uid) && util.IsValidShortUID(uid)
}

// decodeStrictJSON rejects unknown fields (which also enforces immutability of
// resourceType/resourceUid on PATCH) and trailing content after the object.
func decodeStrictJSON(req *http.Request, v any) error {
	if req.Body == nil {
		return io.ErrUnexpectedEOF
	}
	defer func() { _ = req.Body.Close() }()

	dec := json.NewDecoder(http.MaxBytesReader(nil, req.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return ErrRequestBodyInvalid
	}
	return nil
}

func isValidationError(err error) bool {
	for _, target := range []error{
		ErrResourceTypeInvalid,
		ErrResourceUIDRequired,
		ErrResourceUIDTooLong,
		ErrTitleRequired,
		ErrTitleTooLong,
		ErrURLRequired,
		ErrURLTooLong,
		ErrURLNotRelative,
		ErrPatchNoFields,
		ErrListLimitInvalid,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// swagger:parameters createRecentItem
type CreateRecentItemParams struct {
	// in:body
	// required:true
	Body CreateRecentItemCommand `json:"body"`
}

// swagger:parameters listRecentItems
type ListRecentItemsParams struct {
	// Only return items of this resource type.
	// in:query
	// required:false
	// enum: dashboard,folder,alert-rule,datasource,explore
	ResourceType string `json:"resourceType"`
	// Maximum number of items to return (default 20, max 50).
	// in:query
	// required:false
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
	// in: body
	Body RecentItemResponse `json:"body"`
}

// swagger:response listRecentItemsResponse
type ListRecentItemsResponseSwagger struct {
	// in: body
	Body ListRecentItemsResponse `json:"body"`
}

// swagger:response deleteRecentItemResponse
type DeleteRecentItemResponseSwagger struct {
	// in: body
	Body DeleteRecentItemResponse `json:"body"`
}
