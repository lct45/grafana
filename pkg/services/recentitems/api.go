package recentitems

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/grafana/grafana/pkg/api/response"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/web"
)

// swagger:route POST /user/RECENT_ITEMS signed_in_user createRecentItem
//
// Record a recently viewed resource.
//
// Creates a recent item for the signed-in user in the current organization.
// Repeating the same resource refreshes its title, URL, and last-viewed time.
//
// Responses:
// 201: recentItemResponse
// 200: recentItemResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *RecentItemsService) CreateHandler(c *contextmodel.ReqContext) response.Response {
	usr, resp := requireUser(c)
	if resp != nil {
		return resp
	}

	cmd := CreateRecentItemCommand{}
	if err := decodeJSON(c.Req.Body, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", nil)
	}

	item, created, err := s.Record(c.Req.Context(), usr, cmd)
	if err != nil {
		return mapRecentItemError(err, "Failed to record recent item")
	}
	if created {
		return response.JSON(http.StatusCreated, item).
			SetHeader("Location", "/api/user/RECENT_ITEMS/"+item.UID)
	}
	return response.JSON(http.StatusOK, item)
}

// swagger:route GET /user/RECENT_ITEMS signed_in_user listRecentItems
//
// List recently viewed resources.
//
// Returns the signed-in user's recent items in the current organization, newest first.
//
// Responses:
// 200: listRecentItemsResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *RecentItemsService) ListHandler(c *contextmodel.ReqContext) response.Response {
	usr, resp := requireUser(c)
	if resp != nil {
		return resp
	}

	limit, err := parseLimit(c.Query("limit"))
	if err != nil {
		return response.Error(http.StatusBadRequest, "limit is invalid", nil)
	}

	items, err := s.List(c.Req.Context(), usr, limit)
	if err != nil {
		return mapRecentItemError(err, "Failed to list recent items")
	}
	return response.JSON(http.StatusOK, ListRecentItemsResponse{RecentItems: items})
}

// swagger:route PATCH /user/RECENT_ITEMS/{uid} signed_in_user patchRecentItem
//
// Update a recently viewed resource.
//
// Updates the title and/or URL of a recent item. Does not change when it was last viewed.
//
// Responses:
// 200: recentItemResponse
// 400: badRequestError
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *RecentItemsService) PatchHandler(c *contextmodel.ReqContext) response.Response {
	usr, resp := requireUser(c)
	if resp != nil {
		return resp
	}

	uid := web.Params(c.Req)[":uid"]
	if !validItemUID(uid) {
		return response.Error(http.StatusNotFound, "Recent item not found", nil)
	}

	cmd := PatchRecentItemCommand{}
	if err := decodeJSON(c.Req.Body, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", nil)
	}

	item, err := s.Update(c.Req.Context(), usr, uid, cmd)
	if err != nil {
		return mapRecentItemError(err, "Failed to update recent item")
	}
	return response.JSON(http.StatusOK, item)
}

// swagger:route DELETE /user/RECENT_ITEMS/{uid} signed_in_user deleteRecentItem
//
// Delete a recently viewed resource.
//
// Responses:
// 204: recentItemDeletedResponse
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *RecentItemsService) DeleteHandler(c *contextmodel.ReqContext) response.Response {
	usr, resp := requireUser(c)
	if resp != nil {
		return resp
	}

	uid := web.Params(c.Req)[":uid"]
	if !validItemUID(uid) {
		return response.Error(http.StatusNotFound, "Recent item not found", nil)
	}

	if err := s.Remove(c.Req.Context(), usr, uid); err != nil {
		return mapRecentItemError(err, "Failed to delete recent item")
	}
	return response.Empty(http.StatusNoContent)
}

func requireUser(c *contextmodel.ReqContext) (*user.SignedInUser, response.Response) {
	if c == nil || c.SignedInUser == nil || c.IsAnonymous || c.UserID <= 0 || c.OrgID <= 0 {
		return nil, response.Error(http.StatusUnauthorized, "Unauthorized", nil)
	}
	return c.SignedInUser, nil
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
		return 0, errRecentItemInvalid
	}
	return limit, nil
}

func decodeJSON(body io.Reader, dest any) error {
	if body == nil {
		return io.EOF
	}
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("unexpected trailing data")
	}
	return nil
}

func mapRecentItemError(err error, fallback string) response.Response {
	if errors.Is(err, errRecentItemInvalid) {
		return response.Error(http.StatusBadRequest, err.Error(), nil)
	}
	if errors.Is(err, ErrRecentItemNotFound) {
		return response.Error(http.StatusNotFound, "Recent item not found", nil)
	}
	return response.Error(http.StatusInternalServerError, fallback, err)
}

// swagger:parameters createRecentItem
type CreateRecentItemParams struct {
	// in:body
	// required:true
	Body CreateRecentItemCommand `json:"body"`
}

// swagger:parameters listRecentItems
type ListRecentItemsParams struct {
	// Maximum number of items to return. Defaults to 20. Maximum is 100.
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
type RecentItemResponse struct {
	// in:body
	Body RecentItemDTO `json:"body"`
}

// swagger:response listRecentItemsResponse
type ListRecentItemsResponseSwagger struct {
	// in:body
	Body ListRecentItemsResponse `json:"body"`
}

// swagger:response recentItemDeletedResponse
type RecentItemDeletedResponse struct{}
