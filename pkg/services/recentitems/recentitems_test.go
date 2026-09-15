package recentitems

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/db"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/tests/testsuite"
	"github.com/grafana/grafana/pkg/web"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

func TestProvideServiceRegistersAuthenticatedRoutes(t *testing.T) {
	sqlStore, _ := db.InitTestDBWithCfg(t)
	routeRegister := routing.NewRouteRegister()
	service := ProvideService(sqlStore, routeRegister)
	require.NotNil(t, service)

	router := &recordingRouter{}
	routeRegister.Register(router)
	require.Len(t, router.routes, 4)

	expectedRoutes := map[string]string{
		http.MethodPost:   recentItemsPath + "/",
		http.MethodGet:    recentItemsPath + "/",
		http.MethodPatch:  recentItemsPath + "/:uid",
		http.MethodDelete: recentItemsPath + "/:uid",
	}
	for _, route := range router.routes {
		require.Equal(t, expectedRoutes[route.method], route.pattern)
		require.Len(t, route.handlers, 2)
		middlewareName := runtime.FuncForPC(reflect.ValueOf(route.handlers[0]).Pointer()).Name()
		require.Contains(t, middlewareName, "ReqSignedInNoAnonymous")
	}
}

func TestRecentItemsCRUDAndIsolation(t *testing.T) {
	service, signedInUser := newTestService(t)

	item, created, err := service.Upsert(t.Context(), signedInUser, createCommand("dashboard", "dashboard-1"))
	require.NoError(t, err)
	require.True(t, created)
	require.NotEmpty(t, item.UID)
	require.Equal(t, "Resource dashboard-1", item.Title)
	require.Equal(t, time.Unix(100, 0).UTC(), item.LastViewedAt)

	service.now = func() time.Time { return time.Unix(200, 0) }
	updated, created, err := service.Upsert(t.Context(), signedInUser, CreateRecentItemCommand{
		ResourceType: "dashboard",
		ResourceUID:  "dashboard-1",
		Title:        " Updated dashboard ",
		URL:          " /d/dashboard-1/updated ",
	})
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, item.UID, updated.UID)
	require.Equal(t, "Updated dashboard", updated.Title)
	require.Equal(t, "/d/dashboard-1/updated", updated.URL)
	require.Equal(t, time.Unix(200, 0).UTC(), updated.LastViewedAt)

	items, err := service.List(t.Context(), signedInUser, DefaultLimit, "")
	require.NoError(t, err)
	require.Len(t, items, 1)

	for _, principal := range []*user.SignedInUser{
		{OrgID: signedInUser.OrgID, UserID: 2},
		{OrgID: 2, UserID: signedInUser.UserID},
	} {
		isolatedItems, err := service.List(t.Context(), principal, DefaultLimit, "")
		require.NoError(t, err)
		require.Empty(t, isolatedItems)

		_, err = service.Patch(t.Context(), principal, item.UID, PatchRecentItemCommand{Title: pointer("No access")})
		require.ErrorIs(t, err, ErrItemNotFound)
		require.ErrorIs(t, service.Delete(t.Context(), principal, item.UID), ErrItemNotFound)
	}

	newTitle := "Patched dashboard"
	newURL := "/d/dashboard-1/patched"
	patched, err := service.Patch(t.Context(), signedInUser, item.UID, PatchRecentItemCommand{
		Title: &newTitle,
		URL:   &newURL,
	})
	require.NoError(t, err)
	require.Equal(t, newTitle, patched.Title)
	require.Equal(t, newURL, patched.URL)
	require.Equal(t, updated.LastViewedAt, patched.LastViewedAt)

	require.NoError(t, service.Delete(t.Context(), signedInUser, item.UID))
	require.ErrorIs(t, service.Delete(t.Context(), signedInUser, item.UID), ErrItemNotFound)
}

func TestRecentItemsOrderingFilteringLimitAndTrim(t *testing.T) {
	service, signedInUser := newTestService(t)
	nextTimestamp := int64(0)
	service.now = func() time.Time {
		nextTimestamp++
		return time.Unix(nextTimestamp, 0)
	}

	for i := 0; i < MaxLimit+1; i++ {
		resourceType := "dashboard"
		if i == MaxLimit {
			resourceType = "folder"
		}
		_, _, err := service.Upsert(t.Context(), signedInUser, createCommand(resourceType, fmt.Sprintf("resource-%d", i)))
		require.NoError(t, err)
	}

	items, err := service.List(t.Context(), signedInUser, 2, "")
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "resource-50", items[0].ResourceUID)
	require.Equal(t, "resource-49", items[1].ResourceUID)

	folders, err := service.List(t.Context(), signedInUser, MaxLimit, "folder")
	require.NoError(t, err)
	require.Len(t, folders, 1)
	require.Equal(t, "resource-50", folders[0].ResourceUID)

	allItems, err := service.List(t.Context(), signedInUser, MaxLimit, "")
	require.NoError(t, err)
	require.Len(t, allItems, MaxLimit)
	require.Equal(t, "resource-1", allItems[MaxLimit-1].ResourceUID)
}

func TestRecentItemsConcurrentUpsert(t *testing.T) {
	service, signedInUser := newTestService(t)
	cmd := createCommand("dashboard", "dashboard-1")

	const workers = 8
	errs := make([]error, workers)
	uids := make([]string, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			item, _, err := service.Upsert(t.Context(), signedInUser, cmd)
			errs[i] = err
			uids[i] = item.UID
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
	items, err := service.List(t.Context(), signedInUser, DefaultLimit, "")
	require.NoError(t, err)
	require.Len(t, items, 1)
	for _, uid := range uids {
		require.Equal(t, items[0].UID, uid)
	}
}

func TestRecentItemsValidation(t *testing.T) {
	service, signedInUser := newTestService(t)
	tests := []struct {
		name string
		cmd  CreateRecentItemCommand
		err  error
	}{
		{name: "resource type", cmd: createCommand("unknown", "uid"), err: ErrInvalidResourceType},
		{name: "empty resource UID", cmd: createCommand("dashboard", " "), err: ErrInvalidResourceUID},
		{name: "long resource UID", cmd: createCommand("dashboard", strings.Repeat("a", maxResourceUIDLength+1)), err: ErrInvalidResourceUID},
		{name: "empty title", cmd: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "uid", URL: "/d/uid"}, err: ErrInvalidTitle},
		{name: "long title", cmd: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "uid", Title: strings.Repeat("a", maxTitleLength+1), URL: "/d/uid"}, err: ErrInvalidTitle},
		{name: "empty URL", cmd: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "uid", Title: "Title"}, err: ErrInvalidURL},
		{name: "long URL", cmd: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "uid", Title: "Title", URL: "/" + strings.Repeat("a", maxURLLength)}, err: ErrInvalidURL},
		{name: "absolute URL", cmd: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "uid", Title: "Title", URL: "https://example.com"}, err: ErrInvalidURL},
		{name: "protocol-relative URL", cmd: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "uid", Title: "Title", URL: "//example.com/path"}, err: ErrInvalidURL},
		{name: "triple-slash URL", cmd: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "uid", Title: "Title", URL: "///evil.com"}, err: ErrInvalidURL},
		{name: "backslash URL", cmd: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "uid", Title: "Title", URL: "/\\evil.com"}, err: ErrInvalidURL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := service.Upsert(t.Context(), signedInUser, tt.cmd)
			require.ErrorIs(t, err, tt.err)
		})
	}

	for _, resourceType := range []string{"dashboard", "alert-rule", "folder", "datasource", "explore"} {
		_, created, err := service.Upsert(t.Context(), signedInUser, createCommand(resourceType, resourceType+"-uid"))
		require.NoError(t, err)
		require.True(t, created)
	}

	_, err := service.List(t.Context(), signedInUser, 0, "")
	require.ErrorIs(t, err, ErrInvalidLimit)
	_, err = service.List(t.Context(), signedInUser, MaxLimit+1, "")
	require.ErrorIs(t, err, ErrInvalidLimit)
	_, err = service.List(t.Context(), signedInUser, DefaultLimit, "unknown")
	require.ErrorIs(t, err, ErrInvalidResourceType)

	_, err = service.Patch(t.Context(), signedInUser, "missing", PatchRecentItemCommand{})
	require.ErrorIs(t, err, ErrEmptyPatch)
	_, err = service.Patch(t.Context(), signedInUser, "not valid!", PatchRecentItemCommand{Title: pointer("Nope")})
	require.ErrorIs(t, err, ErrItemNotFound)
	require.ErrorIs(t, service.Delete(t.Context(), signedInUser, "not valid!"), ErrItemNotFound)

	item, _, err := service.Upsert(t.Context(), signedInUser, createCommand("dashboard", "patch-validation"))
	require.NoError(t, err)
	_, err = service.Patch(t.Context(), signedInUser, item.UID, PatchRecentItemCommand{Title: pointer(" ")})
	require.ErrorIs(t, err, ErrInvalidTitle)
	_, err = service.Patch(t.Context(), signedInUser, item.UID, PatchRecentItemCommand{URL: pointer("https://example.com")})
	require.ErrorIs(t, err, ErrInvalidURL)
}

func TestRecentItemsHandlers(t *testing.T) {
	service, signedInUser := newTestService(t)

	createResponse := service.createHandler(requestContext(http.MethodPost, recentItemsPath, `{
		"resourceType": "dashboard",
		"resourceUid": "dashboard-1",
		"title": "Dashboard",
		"url": "/d/dashboard-1/dashboard"
	}`, signedInUser))
	require.Equal(t, http.StatusCreated, createResponse.Status())
	var created RecentItemDTO
	require.NoError(t, json.Unmarshal(createResponse.Body(), &created))
	require.NotEmpty(t, created.UID)
	normalResponse, ok := createResponse.(interface{ Header() http.Header })
	require.True(t, ok)
	require.Equal(t, recentItemsPath+"/"+created.UID, normalResponse.Header().Get("Location"))

	require.Equal(t, http.StatusOK, service.createHandler(requestContext(http.MethodPost, recentItemsPath, `{
		"resourceType": "dashboard",
		"resourceUid": "dashboard-1",
		"title": "Updated",
		"url": "/d/dashboard-1/updated"
	}`, signedInUser)).Status())

	for _, body := range []string{
		`{"unknown": true}`,
		`{"resourceType":`,
		`{} {}`,
		``,
	} {
		require.Equal(t, http.StatusBadRequest, service.createHandler(requestContext(http.MethodPost, recentItemsPath, body, signedInUser)).Status())
	}

	_, _, err := service.Upsert(t.Context(), signedInUser, createCommand("folder", "folder-1"))
	require.NoError(t, err)
	listResponse := service.listHandler(requestContext(http.MethodGet, recentItemsPath+"?limit=1&resourceType=folder", "", signedInUser))
	require.Equal(t, http.StatusOK, listResponse.Status())
	var listed ListRecentItemsResponse
	require.NoError(t, json.Unmarshal(listResponse.Body(), &listed))
	require.Len(t, listed.Items, 1)
	require.Equal(t, "folder-1", listed.Items[0].ResourceUID)

	for _, query := range []string{"?limit=invalid", "?limit=0", "?resourceType=unknown"} {
		require.Equal(t, http.StatusBadRequest, service.listHandler(requestContext(http.MethodGet, recentItemsPath+query, "", signedInUser)).Status())
	}

	immutablePatchCtx := requestContext(http.MethodPatch, recentItemsPath+"/"+created.UID, `{"resourceUid":"other"}`, signedInUser)
	setUIDParam(immutablePatchCtx, created.UID)
	require.Equal(t, http.StatusBadRequest, service.patchHandler(immutablePatchCtx).Status())

	emptyPatchCtx := requestContext(http.MethodPatch, recentItemsPath+"/"+created.UID, `{}`, signedInUser)
	setUIDParam(emptyPatchCtx, created.UID)
	require.Equal(t, http.StatusBadRequest, service.patchHandler(emptyPatchCtx).Status())

	patchCtx := requestContext(http.MethodPatch, recentItemsPath+"/"+created.UID, `{"title":"Patched dashboard","url":"/d/dashboard-1/patched"}`, signedInUser)
	setUIDParam(patchCtx, created.UID)
	patchResponse := service.patchHandler(patchCtx)
	require.Equal(t, http.StatusOK, patchResponse.Status())
	var patched RecentItemDTO
	require.NoError(t, json.Unmarshal(patchResponse.Body(), &patched))
	require.Equal(t, "Patched dashboard", patched.Title)

	missingPatchCtx := requestContext(http.MethodPatch, recentItemsPath+"/missing", `{"title":"Missing"}`, signedInUser)
	setUIDParam(missingPatchCtx, "missing")
	require.Equal(t, http.StatusNotFound, service.patchHandler(missingPatchCtx).Status())

	deleteCtx := requestContext(http.MethodDelete, recentItemsPath+"/"+created.UID, "", signedInUser)
	setUIDParam(deleteCtx, created.UID)
	require.Equal(t, http.StatusNoContent, service.deleteHandler(deleteCtx).Status())
	require.Empty(t, service.deleteHandler(deleteCtx).Body())
	require.Equal(t, http.StatusNotFound, service.deleteHandler(deleteCtx).Status())
}

func TestRecentItemErrorMapping(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
	}{
		{ErrItemNotFound, http.StatusNotFound},
		{ErrInvalidResourceType, http.StatusBadRequest},
		{ErrInvalidResourceUID, http.StatusBadRequest},
		{ErrInvalidTitle, http.StatusBadRequest},
		{ErrInvalidURL, http.StatusBadRequest},
		{ErrInvalidLimit, http.StatusBadRequest},
		{ErrEmptyPatch, http.StatusBadRequest},
		{errors.New("database unavailable"), http.StatusInternalServerError},
	} {
		require.Equal(t, tt.status, recentItemError("operation failed", tt.err).Status())
	}
}

func newTestService(t *testing.T) (*RecentItemsService, *user.SignedInUser) {
	t.Helper()
	sqlStore, _ := db.InitTestDBWithCfg(t)
	return &RecentItemsService{
		store: sqlStore,
		now:   func() time.Time { return time.Unix(100, 0) },
	}, &user.SignedInUser{OrgID: 1, UserID: 1}
}

func createCommand(resourceType, resourceUID string) CreateRecentItemCommand {
	return CreateRecentItemCommand{
		ResourceType: resourceType,
		ResourceUID:  resourceUID,
		Title:        "Resource " + resourceUID,
		URL:          "/" + resourceType + "/" + resourceUID,
	}
}

func requestContext(method, target, body string, signedInUser *user.SignedInUser) *contextmodel.ReqContext {
	req, _ := http.NewRequest(method, target, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return &contextmodel.ReqContext{
		Context:      &web.Context{Req: req},
		SignedInUser: signedInUser,
		IsSignedIn:   true,
	}
}

func setUIDParam(ctx *contextmodel.ReqContext, uid string) {
	ctx.Req = web.SetURLParams(ctx.Req, map[string]string{":uid": uid})
}

func pointer(value string) *string {
	return &value
}

type recordedRoute struct {
	method   string
	pattern  string
	handlers []web.Handler
}

type recordingRouter struct {
	routes []recordedRoute
}

func (r *recordingRouter) Handle(method, pattern string, handlers []web.Handler) {
	r.routes = append(r.routes, recordedRoute{method: method, pattern: pattern, handlers: handlers})
}

func (r *recordingRouter) Get(pattern string, handlers ...web.Handler) {
	r.Handle(http.MethodGet, pattern, handlers)
}
