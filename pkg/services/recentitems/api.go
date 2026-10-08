package recentitems

import (
	"encoding/json"
	"errors"
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

func (s *RecentItemsService) registerAPIEndpoints() {
	s.RouteRegister.Group("/api/USER/RECENT_ITEMS", func(items routing.RouteRegister) {
		items.Post("/", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.createHandler))
		items.Get("/", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.listHandler))
		items.Patch("/:uid", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.patchHandler))
		items.Delete("/:uid", middleware.ReqSignedInNoAnonymous, routing.Wrap(s.deleteHandler))
	})
}

func unauthorized(c *contextmodel.ReqContext) bool {
	return c.SignedInUser == nil || c.IsAnonymous || c.UserID <= 0
}

// swagger:route POST /USER/RECENT_ITEMS signed_in_user createRecentItem
//
// Record a recently viewed resource.
//
// Creates a recent item for the signed-in user, or refreshes the existing item when the same resource was already viewed.
//
// Responses:
// 200: recentItemResponse
// 201: recentItemResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *RecentItemsService) createHandler(c *contextmodel.ReqContext) response.Response {
	if unauthorized(c) {
		return response.Error(http.StatusUnauthorized, "Unauthorized", nil)
	}

	cmd := CreateRecentItemCommand{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, ErrMalformedBody.Error(), ErrMalformedBody)
	}

	item, created, err := s.Record(c.Req.Context(), c.SignedInUser, cmd)
	if err != nil {
		return writeServiceError(err, "Failed to record recent item")
	}

	resp := response.JSON(http.StatusOK, item)
	if created {
		resp = response.JSON(http.StatusCreated, item)
		resp.SetHeader("Location", "/api/USER/RECENT_ITEMS/"+item.UID)
	}
	return resp
}

// swagger:route GET /USER/RECENT_ITEMS signed_in_user listRecentItems
//
// List recently viewed resources.
//
// Returns the signed-in user's recent items, newest first.
//
// Responses:
// 200: listRecentItemsResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *RecentItemsService) listHandler(c *contextmodel.ReqContext) response.Response {
	if unauthorized(c) {
		return response.Error(http.StatusUnauthorized, "Unauthorized", nil)
	}

	limit, err := parseLimit(c.Req.URL.Query().Get("limit"))
	if err != nil {
		return response.Error(http.StatusBadRequest, err.Error(), err)
	}

	items, err := s.List(c.Req.Context(), c.SignedInUser, ListRecentItemsQuery{
		ResourceType: c.Req.URL.Query().Get("resourceType"),
		Limit:        limit,
	})
	if err != nil {
		return writeServiceError(err, "Failed to list recent items")
	}

	return response.JSON(http.StatusOK, ListRecentItemsResponse{RecentItems: items})
}

// swagger:route PATCH /USER/RECENT_ITEMS/{uid} signed_in_user patchRecentItem
//
// Update a recently viewed resource.
//
// Updates the title or URL of one recent item owned by the signed-in user.
//
// Responses:
// 200: recentItemResponse
// 400: badRequestError
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *RecentItemsService) patchHandler(c *contextmodel.ReqContext) response.Response {
	if unauthorized(c) {
		return response.Error(http.StatusUnauthorized, "Unauthorized", nil)
	}

	uid := web.Params(c.Req)[":uid"]
	if !validItemUID(uid) {
		return response.Error(http.StatusNotFound, "Recent item not found", nil)
	}

	cmd, err := decodePatch(c.Req.Body)
	if err != nil {
		return writeServiceError(err, "Failed to update recent item")
	}

	item, err := s.Update(c.Req.Context(), c.SignedInUser, uid, cmd)
	if err != nil {
		return writeServiceError(err, "Failed to update recent item")
	}

	return response.JSON(http.StatusOK, item)
}

// swagger:route DELETE /USER/RECENT_ITEMS/{uid} signed_in_user deleteRecentItem
//
// Delete a recently viewed resource.
//
// Removes one recent item owned by the signed-in user.
//
// Responses:
// 204: deleteRecentItemResponse
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *RecentItemsService) deleteHandler(c *contextmodel.ReqContext) response.Response {
	if unauthorized(c) {
		return response.Error(http.StatusUnauthorized, "Unauthorized", nil)
	}

	uid := web.Params(c.Req)[":uid"]
	if !validItemUID(uid) {
		return response.Error(http.StatusNotFound, "Recent item not found", nil)
	}

	err := s.Remove(c.Req.Context(), c.SignedInUser, uid)
	if err != nil {
		return writeServiceError(err, "Failed to delete recent item")
	}

	return response.Empty(http.StatusNoContent)
}

func validItemUID(uid string) bool {
	return uid != "" && util.IsValidShortUID(uid) && !util.IsShortUIDTooLong(uid)
}

func parseLimit(raw string) (int, error) {
	if raw == "" {
		return defaultRecentItemsLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > maxRecentItemsLimit {
		return 0, ErrInvalidLimit
	}
	return limit, nil
}

func decodePatch(body io.Reader) (PatchRecentItemCommand, error) {
	if body == nil {
		return PatchRecentItemCommand{}, ErrPatchEmpty
	}
	raw, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		return PatchRecentItemCommand{}, ErrMalformedBody
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return PatchRecentItemCommand{}, ErrMalformedBody
	}

	cmd := PatchRecentItemCommand{}
	for key, value := range fields {
		switch key {
		case "title":
			title, err := decodeOptionalString(value)
			if err != nil {
				return PatchRecentItemCommand{}, err
			}
			cmd.Title = title
		case "url":
			rawURL, err := decodeOptionalString(value)
			if err != nil {
				return PatchRecentItemCommand{}, err
			}
			cmd.URL = rawURL
		case "uid", "resourceType", "resourceUid", "lastViewedAt":
			return PatchRecentItemCommand{}, ErrImmutableField
		default:
			return PatchRecentItemCommand{}, ErrUnknownField
		}
	}

	if cmd.Title == nil && cmd.URL == nil {
		return PatchRecentItemCommand{}, ErrPatchEmpty
	}
	return cmd, nil
}

func decodeOptionalString(value json.RawMessage) (*string, error) {
	if string(value) == "null" {
		return nil, nil
	}
	var decoded string
	if err := json.Unmarshal(value, &decoded); err != nil {
		return nil, ErrMalformedBody
	}
	return &decoded, nil
}

func writeServiceError(err error, action string) response.Response {
	if isValidationError(err) {
		return response.Error(http.StatusBadRequest, err.Error(), err)
	}
	if errors.Is(err, ErrRecentItemNotFound) {
		return response.Error(http.StatusNotFound, "Recent item not found", err)
	}
	return response.Error(http.StatusInternalServerError, action, err)
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
	// required:false
	ResourceType string `json:"resourceType"`
	// in:query
	// required:false
	// maximum: 50
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
type RecentItemResponse struct {
	// in:body
	Body RecentItemDTO `json:"body"`
}

// swagger:response listRecentItemsResponse
type ListRecentItemsResponseSwagger struct {
	// in:body
	Body ListRecentItemsResponse `json:"body"`
}

// swagger:response deleteRecentItemResponse
type DeleteRecentItemResponse struct{}
