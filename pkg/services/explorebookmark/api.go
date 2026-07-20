package explorebookmark

import (
	"errors"
	"net/http"
	"strings"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/middleware"
	ac "github.com/grafana/grafana/pkg/services/accesscontrol"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/web"
)

func (s *ExploreBookmarkService) registerAPIEndpoints() {
	s.RouteRegister.Group("/api/explore/bookmarks", func(entities routing.RouteRegister) {
		entities.Post("/", middleware.ReqSignedIn, routing.Wrap(s.permissionsMiddleware(s.createHandler, "Failed to create explore bookmark")))
		entities.Get("/", middleware.ReqSignedIn, routing.Wrap(s.permissionsMiddleware(s.listHandler, "Failed to list explore bookmarks")))
		entities.Delete("/:uid", middleware.ReqSignedIn, routing.Wrap(s.permissionsMiddleware(s.deleteHandler, "Failed to delete explore bookmark")))
	})
}

type callbackHandler func(c *contextmodel.ReqContext) response.Response

func (s *ExploreBookmarkService) permissionsMiddleware(handler callbackHandler, errorMessage string) callbackHandler {
	return func(c *contextmodel.ReqContext) response.Response {
		hasAccess := ac.HasAccess(s.accessControl, c)
		if c.GetOrgRole() == org.RoleViewer && !hasAccess(ac.EvalPermission(ac.ActionDatasourcesExplore)) {
			return response.Error(http.StatusUnauthorized, errorMessage, nil)
		}
		return handler(c)
	}
}

// swagger:route POST /explore/bookmarks explore_bookmarks createExploreBookmark
//
// Create an explore bookmark.
//
// Responses:
// 200: getExploreBookmarkResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *ExploreBookmarkService) createHandler(c *contextmodel.ReqContext) response.Response {
	cmd := CreateExploreBookmarkCommand{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	cmd.Name = strings.TrimSpace(cmd.Name)
	if cmd.Name == "" {
		return response.Error(http.StatusBadRequest, "name is required", nil)
	}
	if cmd.DatasourceUID == "" {
		return response.Error(http.StatusBadRequest, "datasourceUid is required", nil)
	}
	if cmd.Queries == nil {
		return response.Error(http.StatusBadRequest, "queries are required", nil)
	}
	if cmd.TimeFrom == "" || cmd.TimeTo == "" {
		return response.Error(http.StatusBadRequest, "time range is required", nil)
	}

	bookmark, err := s.CreateBookmark(c.Req.Context(), c.SignedInUser, cmd)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to create explore bookmark", err)
	}

	return response.JSON(http.StatusOK, ExploreBookmarkResponse{Result: bookmark})
}

// swagger:route GET /explore/bookmarks explore_bookmarks listExploreBookmarks
//
// List explore bookmarks for the signed-in user.
//
// Responses:
// 200: getExploreBookmarkListResponse
// 401: unauthorisedError
// 500: internalServerError
func (s *ExploreBookmarkService) listHandler(c *contextmodel.ReqContext) response.Response {
	bookmarks, err := s.ListBookmarks(c.Req.Context(), c.SignedInUser)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to list explore bookmarks", err)
	}

	return response.JSON(http.StatusOK, ExploreBookmarkListResponse{Result: bookmarks})
}

// swagger:route DELETE /explore/bookmarks/{uid} explore_bookmarks deleteExploreBookmark
//
// Delete an explore bookmark.
//
// Responses:
// 200: getExploreBookmarkDeleteResponse
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *ExploreBookmarkService) deleteHandler(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if len(uid) > 0 && !util.IsValidShortUID(uid) {
		return response.Error(http.StatusNotFound, "Explore bookmark not found", nil)
	}

	err := s.DeleteBookmark(c.Req.Context(), c.SignedInUser, uid)
	if err != nil {
		if errors.Is(err, ErrBookmarkNotFound) {
			return response.Error(http.StatusNotFound, "Explore bookmark not found", err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to delete explore bookmark", err)
	}

	return response.JSON(http.StatusOK, ExploreBookmarkDeleteResponse{
		Message: "Bookmark deleted",
		UID:     uid,
	})
}

// swagger:parameters createExploreBookmark
type CreateExploreBookmarkParams struct {
	// in:body
	// required:true
	Body CreateExploreBookmarkCommand `json:"body"`
}

// swagger:parameters deleteExploreBookmark
type ExploreBookmarkByUID struct {
	// in:path
	// required:true
	UID string `json:"uid"`
}

// swagger:response getExploreBookmarkResponse
type GetExploreBookmarkResponse struct {
	// in: body
	Body ExploreBookmarkResponse `json:"body"`
}

// swagger:response getExploreBookmarkListResponse
type GetExploreBookmarkListResponse struct {
	// in: body
	Body ExploreBookmarkListResponse `json:"body"`
}

// swagger:response getExploreBookmarkDeleteResponse
type GetExploreBookmarkDeleteResponse struct {
	// in: body
	Body ExploreBookmarkDeleteResponse `json:"body"`
}
