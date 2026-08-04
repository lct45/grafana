package explorebookmarks

import (
	"errors"
	"net/http"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/middleware"
	ac "github.com/grafana/grafana/pkg/services/accesscontrol"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/web"
)

func (s *ExploreBookmarksService) registerAPIEndpoints() {
	s.RouteRegister.Group("/api/explore/bookmarks", func(entities routing.RouteRegister) {
		entities.Post("/", middleware.ReqSignedIn, routing.Wrap(s.permissionsMiddleware(s.createHandler, "Failed to create explore bookmark")))
		entities.Get("/", middleware.ReqSignedIn, routing.Wrap(s.permissionsMiddleware(s.listHandler, "Failed to list explore bookmarks")))
		entities.Delete("/:uid", middleware.ReqSignedIn, routing.Wrap(s.permissionsMiddleware(s.deleteHandler, "Failed to delete explore bookmark")))
	})
}

type callbackHandler func(c *contextmodel.ReqContext) response.Response

func (s *ExploreBookmarksService) permissionsMiddleware(handler callbackHandler, errorMessage string) callbackHandler {
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
// 201: createExploreBookmarkResponse
// 400: badRequestError
// 401: unauthorisedError
// 500: internalServerError
func (s *ExploreBookmarksService) createHandler(c *contextmodel.ReqContext) response.Response {
	cmd := CreateBookmarkCommand{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	bookmark, err := s.CreateBookmark(c.Req.Context(), c.SignedInUser, cmd)
	if err != nil {
		if errors.Is(err, ErrBookmarkNameRequired) || isValidationError(err) {
			return response.Error(http.StatusBadRequest, err.Error(), err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to create explore bookmark", err)
	}

	return response.JSON(http.StatusCreated, CreateBookmarkResponse{Bookmark: bookmark})
}

// swagger:route GET /explore/bookmarks explore_bookmarks listExploreBookmarks
//
// List explore bookmarks for the signed-in user.
//
// Responses:
// 200: listExploreBookmarksResponse
// 401: unauthorisedError
// 500: internalServerError
func (s *ExploreBookmarksService) listHandler(c *contextmodel.ReqContext) response.Response {
	bookmarks, err := s.ListBookmarks(c.Req.Context(), c.SignedInUser)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to list explore bookmarks", err)
	}

	return response.JSON(http.StatusOK, ListBookmarksResponse{Bookmarks: bookmarks})
}

// swagger:route DELETE /explore/bookmarks/{uid} explore_bookmarks deleteExploreBookmark
//
// Delete an explore bookmark.
//
// Responses:
// 200: deleteExploreBookmarkResponse
// 401: unauthorisedError
// 404: notFoundError
// 500: internalServerError
func (s *ExploreBookmarksService) deleteHandler(c *contextmodel.ReqContext) response.Response {
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

	return response.JSON(http.StatusOK, DeleteBookmarkResponse{Message: "Bookmark deleted"})
}

func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "datasourceUid is required" ||
		msg == "queries are required" ||
		msg == "queries must be valid JSON" ||
		msg == "queries must be a JSON array" ||
		msg == "queries must not be empty" ||
		msg == "timeRange.from and timeRange.to are required" ||
		msg == "bookmark name must be at most 255 characters"
}

// swagger:parameters createExploreBookmark
type CreateExploreBookmarkParams struct {
	// in:body
	// required:true
	Body CreateBookmarkCommand `json:"body"`
}

// swagger:parameters deleteExploreBookmark
type ExploreBookmarkByUID struct {
	// in:path
	// required:true
	UID string `json:"uid"`
}

// swagger:response createExploreBookmarkResponse
type CreateExploreBookmarkResponseSwagger struct {
	// in: body
	Body CreateBookmarkResponse `json:"body"`
}

// swagger:response listExploreBookmarksResponse
type ListExploreBookmarksResponseSwagger struct {
	// in: body
	Body ListBookmarksResponse `json:"body"`
}

// swagger:response deleteExploreBookmarkResponse
type DeleteExploreBookmarkResponseSwagger struct {
	// in: body
	Body DeleteBookmarkResponse `json:"body"`
}
