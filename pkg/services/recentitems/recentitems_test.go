package recentitems

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/tests/testsuite"
	"github.com/grafana/grafana/pkg/web/webtest"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

func TestRecentItemsAPI(t *testing.T) {
	sqlStore := db.InitTestDB(t)
	routeRegister := routing.NewRouteRegister()
	service := ProvideService(sqlStore, routeRegister)
	server := webtest.NewServer(t, routeRegister)
	userOne := testUser(1, 1)
	userTwo := testUser(2, 1)

	t.Run("rejects unauthenticated requests", func(t *testing.T) {
		resp := sendRequest(t, server, nil, http.MethodGet, "/api/user/recent-items/", nil)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	var item RecentItemDTO
	t.Run("creates a recent item", func(t *testing.T) {
		resp := sendRequest(t, server, userOne, http.MethodPost, "/api/user/recent-items/", CreateRecentItemCommand{
			ResourceType: ResourceTypeDash,
			ResourceUID:  "dashboard-1",
			Title:        "Production overview",
			URL:          "/d/dashboard-1",
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotEmpty(t, resp.Header.Get("Location"))
		item = decodeItemResponse(t, resp).Item
		require.Equal(t, "dashboard-1", item.ResourceUID)
	})

	t.Run("refreshes an existing recent item", func(t *testing.T) {
		service.now = func() time.Time { return time.Unix(item.LastViewedAt+1, 0) }
		resp := sendRequest(t, server, userOne, http.MethodPost, "/api/user/recent-items/", CreateRecentItemCommand{
			ResourceType: ResourceTypeDash,
			ResourceUID:  "dashboard-1",
			Title:        "Renamed overview",
			URL:          "/d/dashboard-1-renamed",
		})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		refreshed := decodeItemResponse(t, resp).Item
		require.Equal(t, item.UID, refreshed.UID)
		require.Equal(t, "Renamed overview", refreshed.Title)
		require.Greater(t, refreshed.LastViewedAt, item.LastViewedAt)
	})

	t.Run("lists only the signed-in user's items", func(t *testing.T) {
		resp := sendRequest(t, server, userOne, http.MethodGet, "/api/user/recent-items/", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Len(t, decodeItemsResponse(t, resp).Items, 1)

		resp = sendRequest(t, server, userTwo, http.MethodGet, "/api/user/recent-items/", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Empty(t, decodeItemsResponse(t, resp).Items)
	})

	t.Run("prevents another user from patching or deleting an item", func(t *testing.T) {
		title := "Private title"
		resp := sendRequest(t, server, userTwo, http.MethodPatch, "/api/user/recent-items/"+item.UID, PatchRecentItemCommand{Title: &title})
		require.Equal(t, http.StatusNotFound, resp.StatusCode)

		resp = sendRequest(t, server, userTwo, http.MethodDelete, "/api/user/recent-items/"+item.UID, nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("patches mutable metadata", func(t *testing.T) {
		title := "Updated title"
		url := "/d/updated"
		resp := sendRequest(t, server, userOne, http.MethodPatch, "/api/user/recent-items/"+item.UID, PatchRecentItemCommand{
			Title: &title,
			URL:   &url,
		})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		updated := decodeItemResponse(t, resp).Item
		require.Equal(t, title, updated.Title)
		require.Equal(t, url, updated.URL)
	})

	t.Run("deletes an item", func(t *testing.T) {
		resp := sendRequest(t, server, userOne, http.MethodDelete, "/api/user/recent-items/"+item.UID, nil)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)

		resp = sendRequest(t, server, userOne, http.MethodGet, "/api/user/recent-items/", nil)
		require.Empty(t, decodeItemsResponse(t, resp).Items)
	})
}

func TestRecentItemsValidation(t *testing.T) {
	sqlStore := db.InitTestDB(t)
	routeRegister := routing.NewRouteRegister()
	ProvideService(sqlStore, routeRegister)
	server := webtest.NewServer(t, routeRegister)
	signedInUser := testUser(1, 1)

	tests := []struct {
		name string
		body string
	}{
		{name: "invalid resource type", body: `{"resourceType":"unknown","resourceUid":"uid"}`},
		{name: "missing resource UID", body: `{"resourceType":"dashboard"}`},
		{name: "resource UID too long", body: fmt.Sprintf(`{"resourceType":"dashboard","resourceUid":%q}`, strings.Repeat("a", MaxResourceUIDLen+1))},
		{name: "title too long", body: fmt.Sprintf(`{"resourceType":"dashboard","resourceUid":"uid","title":%q}`, strings.Repeat("a", MaxTitleLen+1))},
		{name: "URL too long", body: fmt.Sprintf(`{"resourceType":"dashboard","resourceUid":"uid","url":%q}`, strings.Repeat("a", MaxURLLen+1))},
		{name: "unknown field", body: `{"resourceType":"dashboard","resourceUid":"uid","immutable":true}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := sendRawRequest(t, server, signedInUser, http.MethodPost, "/api/user/recent-items/", tt.body)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}

	createResp := sendRequest(t, server, signedInUser, http.MethodPost, "/api/user/recent-items/", CreateRecentItemCommand{
		ResourceType: ResourceTypeDash,
		ResourceUID:  "dashboard-1",
	})
	item := decodeItemResponse(t, createResp).Item

	t.Run("rejects an empty patch", func(t *testing.T) {
		resp := sendRawRequest(t, server, signedInUser, http.MethodPatch, "/api/user/recent-items/"+item.UID, `{}`)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("rejects immutable patch fields", func(t *testing.T) {
		resp := sendRawRequest(t, server, signedInUser, http.MethodPatch, "/api/user/recent-items/"+item.UID, `{"resourceUid":"other"}`)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestRecentItemsOrderingFilteringAndLimit(t *testing.T) {
	sqlStore := db.InitTestDB(t)
	service := &ServiceImpl{store: sqlStore, now: time.Now}
	signedInUser := testUser(1, 1)
	start := time.Unix(1_700_000_000, 0)

	var firstDashboard RecentItemDTO
	for i := 0; i < DefaultLimit+5; i++ {
		service.now = func() time.Time { return start.Add(time.Duration(i) * time.Second) }
		resourceType := ResourceTypeFolder
		if i%2 == 0 {
			resourceType = ResourceTypeDash
		}
		result, err := service.UpsertRecentItem(context.Background(), signedInUser, CreateRecentItemCommand{
			ResourceType: resourceType,
			ResourceUID:  fmt.Sprintf("resource-%d", i),
		})
		require.NoError(t, err)
		if i == 6 {
			firstDashboard = result.Item
		}
	}

	items, err := service.ListRecentItems(context.Background(), signedInUser, ListRecentItemsQuery{Limit: DefaultLimit + 10})
	require.NoError(t, err)
	require.Len(t, items, DefaultLimit)
	require.Equal(t, "resource-54", items[0].ResourceUID)
	require.Equal(t, "resource-5", items[len(items)-1].ResourceUID)

	dashboards, err := service.ListRecentItems(context.Background(), signedInUser, ListRecentItemsQuery{
		ResourceType: ResourceTypeDash,
		Limit:        2,
	})
	require.NoError(t, err)
	require.Len(t, dashboards, 2)
	require.Equal(t, "resource-54", dashboards[0].ResourceUID)
	require.Equal(t, "resource-52", dashboards[1].ResourceUID)

	service.now = func() time.Time { return start.Add(2 * time.Hour) }
	refreshed, err := service.UpsertRecentItem(context.Background(), signedInUser, CreateRecentItemCommand{
		ResourceType: ResourceTypeDash,
		ResourceUID:  firstDashboard.ResourceUID,
		Title:        "Refreshed",
	})
	require.NoError(t, err)
	require.False(t, refreshed.Created)

	dashboards, err = service.ListRecentItems(context.Background(), signedInUser, ListRecentItemsQuery{
		ResourceType: ResourceTypeDash,
		Limit:        2,
	})
	require.NoError(t, err)
	require.Equal(t, firstDashboard.UID, dashboards[0].UID)

	_, err = service.ListRecentItems(context.Background(), signedInUser, ListRecentItemsQuery{ResourceType: "invalid"})
	require.ErrorIs(t, err, ErrInvalidResourceType)
}

func TestRecentItemsAreOrgScoped(t *testing.T) {
	sqlStore := db.InitTestDB(t)
	service := &ServiceImpl{store: sqlStore, now: time.Now}
	orgOneUser := testUser(1, 1)
	orgTwoUser := testUser(1, 2)

	result, err := service.UpsertRecentItem(context.Background(), orgOneUser, CreateRecentItemCommand{
		ResourceType: ResourceTypeDash,
		ResourceUID:  "dashboard-1",
	})
	require.NoError(t, err)

	items, err := service.ListRecentItems(context.Background(), orgTwoUser, ListRecentItemsQuery{})
	require.NoError(t, err)
	require.Empty(t, items)

	title := "Cross-tenant update"
	_, err = service.PatchRecentItem(context.Background(), orgTwoUser, result.Item.UID, PatchRecentItemCommand{Title: &title})
	require.ErrorIs(t, err, ErrRecentItemNotFound)
	require.ErrorIs(t, service.DeleteRecentItem(context.Background(), orgTwoUser, result.Item.UID), ErrRecentItemNotFound)
}

func testUser(userID, orgID int64) *user.SignedInUser {
	return &user.SignedInUser{
		UserID:  userID,
		OrgID:   orgID,
		OrgRole: org.RoleEditor,
	}
}

func sendRequest(t *testing.T, server *webtest.Server, signedInUser *user.SignedInUser, method, path string, body any) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(data)
	}
	return send(t, server, signedInUser, method, path, reader)
}

func sendRawRequest(t *testing.T, server *webtest.Server, signedInUser *user.SignedInUser, method, path, body string) *http.Response {
	t.Helper()
	return send(t, server, signedInUser, method, path, strings.NewReader(body))
}

func send(t *testing.T, server *webtest.Server, signedInUser *user.SignedInUser, method, path string, body io.Reader) *http.Response {
	t.Helper()
	req := server.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if signedInUser != nil {
		req = webtest.RequestWithSignedInUser(req, signedInUser)
	}
	resp, err := server.Send(req)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, resp.Body.Close())
	})
	return resp
}

func decodeItemResponse(t *testing.T, resp *http.Response) RecentItemResponse {
	t.Helper()
	var result RecentItemResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}

func decodeItemsResponse(t *testing.T, resp *http.Response) RecentItemsResponse {
	t.Helper()
	var result RecentItemsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}
