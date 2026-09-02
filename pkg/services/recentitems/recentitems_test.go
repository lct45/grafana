package recentitems

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/infra/db"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/tests/testsuite"
	"github.com/grafana/grafana/pkg/web"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
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
		Title:        "Updated dashboard",
		URL:          "/d/dashboard-1/updated",
	})
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, item.UID, updated.UID)
	require.Equal(t, "Updated dashboard", updated.Title)
	require.Equal(t, time.Unix(200, 0).UTC(), updated.LastViewedAt)

	items, err := service.List(t.Context(), signedInUser, DefaultLimit)
	require.NoError(t, err)
	require.Len(t, items, 1)

	otherUser := &user.SignedInUser{OrgID: signedInUser.OrgID, UserID: 2}
	otherOrg := &user.SignedInUser{OrgID: 2, UserID: signedInUser.UserID}
	for _, principal := range []*user.SignedInUser{otherUser, otherOrg} {
		isolatedItems, err := service.List(t.Context(), principal, DefaultLimit)
		require.NoError(t, err)
		require.Empty(t, isolatedItems)

		_, err = service.Patch(t.Context(), principal, item.UID, PatchRecentItemCommand{Title: pointer("No access")})
		require.ErrorIs(t, err, ErrItemNotFound)
		require.ErrorIs(t, service.Delete(t.Context(), principal, item.UID), ErrItemNotFound)
	}

	newTitle := "Patched dashboard"
	newURL := "/d/dashboard-1/patched"
	newTime := time.Unix(300, 0).UTC()
	patched, err := service.Patch(t.Context(), signedInUser, item.UID, PatchRecentItemCommand{
		Title:        &newTitle,
		URL:          &newURL,
		LastViewedAt: &newTime,
	})
	require.NoError(t, err)
	require.Equal(t, newTitle, patched.Title)
	require.Equal(t, newURL, patched.URL)
	require.Equal(t, newTime, patched.LastViewedAt)

	require.NoError(t, service.Delete(t.Context(), signedInUser, item.UID))
	require.ErrorIs(t, service.Delete(t.Context(), signedInUser, item.UID), ErrItemNotFound)
}

func TestRecentItemsOrderingLimitAndTrim(t *testing.T) {
	service, signedInUser := newTestService(t)
	nextTimestamp := int64(0)
	service.now = func() time.Time {
		nextTimestamp++
		return time.Unix(nextTimestamp, 0)
	}

	for i := 0; i < MaxLimit+1; i++ {
		_, _, err := service.Upsert(t.Context(), signedInUser, createCommand("dashboard", fmt.Sprintf("dashboard-%d", i)))
		require.NoError(t, err)
	}

	items, err := service.List(t.Context(), signedInUser, 2)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "dashboard-50", items[0].ResourceUID)
	require.Equal(t, "dashboard-49", items[1].ResourceUID)

	allItems, err := service.List(t.Context(), signedInUser, MaxLimit)
	require.NoError(t, err)
	require.Len(t, allItems, MaxLimit)
	require.Equal(t, "dashboard-1", allItems[MaxLimit-1].ResourceUID)
}

func TestRecentItemsValidation(t *testing.T) {
	service, signedInUser := newTestService(t)

	tests := []struct {
		name string
		cmd  CreateRecentItemCommand
		err  error
	}{
		{name: "resource type", cmd: createCommand("unknown", "uid"), err: ErrInvalidResourceType},
		{name: "resource UID", cmd: createCommand("dashboard", "bad uid"), err: ErrInvalidResourceUID},
		{name: "empty resource UID", cmd: createCommand("dashboard", ""), err: ErrInvalidResourceUID},
		{name: "title", cmd: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "uid", URL: "/d/uid"}, err: ErrInvalidTitle},
		{name: "long title", cmd: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "uid", Title: strings.Repeat("a", 256), URL: "/d/uid"}, err: ErrInvalidTitle},
		{name: "absolute URL", cmd: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "uid", Title: "Title", URL: "https://example.com"}, err: ErrInvalidURL},
		{name: "protocol-relative URL", cmd: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "uid", Title: "Title", URL: "//example.com/path"}, err: ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := service.Upsert(t.Context(), signedInUser, tt.cmd)
			require.ErrorIs(t, err, tt.err)
		})
	}

	_, created, err := service.Upsert(t.Context(), signedInUser, CreateRecentItemCommand{
		ResourceType: "explore",
		ResourceUID:  strings.Repeat("e", 200),
		Title:        "Explore session",
		URL:          "/explore",
	})
	require.NoError(t, err)
	require.True(t, created)

	_, err = service.List(t.Context(), signedInUser, 0)
	require.ErrorIs(t, err, ErrInvalidLimit)
	_, err = service.List(t.Context(), signedInUser, MaxLimit+1)
	require.ErrorIs(t, err, ErrInvalidLimit)
	_, err = service.Patch(t.Context(), signedInUser, "missing", PatchRecentItemCommand{})
	require.ErrorIs(t, err, ErrEmptyPatch)
	_, err = service.Patch(t.Context(), signedInUser, "not valid!", PatchRecentItemCommand{Title: pointer("Nope")})
	require.ErrorIs(t, err, ErrItemNotFound)
	require.ErrorIs(t, service.Delete(t.Context(), signedInUser, "not valid!"), ErrItemNotFound)
}

func TestRecentItemsHandlers(t *testing.T) {
	service, signedInUser := newTestService(t)

	createCtx := requestContext(http.MethodPost, recentItemsPath, `{
		"resourceType": "dashboard",
		"resourceUid": "dashboard-1",
		"title": "Dashboard",
		"url": "/d/dashboard-1/dashboard"
	}`, signedInUser)
	createResponse := service.createHandler(createCtx)
	require.Equal(t, http.StatusCreated, createResponse.Status())

	var created RecentItemDTO
	require.NoError(t, json.Unmarshal(createResponse.Body(), &created))
	require.NotEmpty(t, created.UID)

	normalResponse, ok := createResponse.(interface{ Header() http.Header })
	require.True(t, ok)
	require.Equal(t, recentItemsPath+"/"+created.UID, normalResponse.Header().Get("Location"))

	service.now = func() time.Time { return time.Unix(200, 0) }
	upsertCtx := requestContext(http.MethodPost, recentItemsPath, `{
		"resourceType": "dashboard",
		"resourceUid": "dashboard-1",
		"title": "Updated dashboard",
		"url": "/d/dashboard-1/updated"
	}`, signedInUser)
	upsertResponse := service.createHandler(upsertCtx)
	require.Equal(t, http.StatusOK, upsertResponse.Status())
	var upserted RecentItemDTO
	require.NoError(t, json.Unmarshal(upsertResponse.Body(), &upserted))
	require.Equal(t, created.UID, upserted.UID)

	service.now = func() time.Time { return time.Unix(300, 0) }
	secondCreate := requestContext(http.MethodPost, recentItemsPath, `{
		"resourceType": "folder",
		"resourceUid": "folder-1",
		"title": "Folder",
		"url": "/dashboards/f/folder-1"
	}`, signedInUser)
	require.Equal(t, http.StatusCreated, service.createHandler(secondCreate).Status())

	unknownFieldCtx := requestContext(http.MethodPost, recentItemsPath, `{"unknown": true}`, signedInUser)
	require.Equal(t, http.StatusBadRequest, service.createHandler(unknownFieldCtx).Status())

	listCtx := requestContext(http.MethodGet, recentItemsPath+"?limit=1", "", signedInUser)
	listResponse := service.listHandler(listCtx)
	require.Equal(t, http.StatusOK, listResponse.Status())
	var listed ListRecentItemsResponse
	require.NoError(t, json.Unmarshal(listResponse.Body(), &listed))
	require.Len(t, listed.Items, 1)
	require.Equal(t, "folder-1", listed.Items[0].ResourceUID)

	emptyListCtx := requestContext(http.MethodGet, recentItemsPath, "", &user.SignedInUser{OrgID: 9, UserID: 9})
	emptyListResponse := service.listHandler(emptyListCtx)
	require.Equal(t, http.StatusOK, emptyListResponse.Status())
	var empty ListRecentItemsResponse
	require.NoError(t, json.Unmarshal(emptyListResponse.Body(), &empty))
	require.Empty(t, empty.Items)
	require.NotNil(t, empty.Items)

	badLimitCtx := requestContext(http.MethodGet, recentItemsPath+"?limit=invalid", "", signedInUser)
	require.Equal(t, http.StatusBadRequest, service.listHandler(badLimitCtx).Status())
	zeroLimitCtx := requestContext(http.MethodGet, recentItemsPath+"?limit=0", "", signedInUser)
	require.Equal(t, http.StatusBadRequest, service.listHandler(zeroLimitCtx).Status())

	immutablePatchCtx := requestContext(http.MethodPatch, recentItemsPath+"/"+created.UID, `{"resourceUid":"other"}`, signedInUser)
	immutablePatchCtx.Req = web.SetURLParams(immutablePatchCtx.Req, map[string]string{":uid": created.UID})
	require.Equal(t, http.StatusBadRequest, service.patchHandler(immutablePatchCtx).Status())

	emptyPatchCtx := requestContext(http.MethodPatch, recentItemsPath+"/"+created.UID, `{}`, signedInUser)
	emptyPatchCtx.Req = web.SetURLParams(emptyPatchCtx.Req, map[string]string{":uid": created.UID})
	require.Equal(t, http.StatusBadRequest, service.patchHandler(emptyPatchCtx).Status())

	patchCtx := requestContext(http.MethodPatch, recentItemsPath+"/"+created.UID, `{
		"title": "Patched dashboard",
		"url": "/d/dashboard-1/patched",
		"lastViewedAt": "1970-01-01T00:06:40Z"
	}`, signedInUser)
	patchCtx.Req = web.SetURLParams(patchCtx.Req, map[string]string{":uid": created.UID})
	patchResponse := service.patchHandler(patchCtx)
	require.Equal(t, http.StatusOK, patchResponse.Status())
	var patched RecentItemDTO
	require.NoError(t, json.Unmarshal(patchResponse.Body(), &patched))
	require.Equal(t, "Patched dashboard", patched.Title)
	require.Equal(t, "/d/dashboard-1/patched", patched.URL)
	require.Equal(t, time.Unix(400, 0).UTC(), patched.LastViewedAt)

	invalidUIDCtx := requestContext(http.MethodDelete, recentItemsPath+"/not valid!", "", signedInUser)
	invalidUIDCtx.Req = web.SetURLParams(invalidUIDCtx.Req, map[string]string{":uid": "not valid!"})
	require.Equal(t, http.StatusNotFound, service.deleteHandler(invalidUIDCtx).Status())

	deleteCtx := requestContext(http.MethodDelete, recentItemsPath+"/"+created.UID, "", signedInUser)
	deleteCtx.Req = web.SetURLParams(deleteCtx.Req, map[string]string{":uid": created.UID})
	require.Equal(t, http.StatusOK, service.deleteHandler(deleteCtx).Status())
	require.Equal(t, http.StatusNotFound, service.deleteHandler(deleteCtx).Status())
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

func pointer(value string) *string {
	return &value
}
