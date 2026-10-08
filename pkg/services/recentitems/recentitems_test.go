package recentitems

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/db/dbtest"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/tests/testsuite"
	"github.com/grafana/grafana/pkg/web"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

type scenarioContext struct {
	ctx        *web.Context
	service    *RecentItemsService
	reqContext *contextmodel.ReqContext
}

func testScenario(t *testing.T, desc string, fn func(t *testing.T, sc scenarioContext)) {
	t.Helper()

	t.Run(desc, func(t *testing.T) {
		ctx := web.Context{Req: &http.Request{
			Header: http.Header{},
			URL:    &url.URL{},
			Form:   url.Values{},
		}}
		ctx.Req.Header.Add("Content-Type", "application/json")
		sqlStore := db.InitTestDB(t)
		service := &RecentItemsService{
			store: sqlStore,
			now:   time.Now,
		}
		usr := user.SignedInUser{
			UserID:  1,
			OrgID:   1,
			OrgRole: org.RoleEditor,
		}
		sc := scenarioContext{
			ctx:     &ctx,
			service: service,
			reqContext: &contextmodel.ReqContext{
				Context:      &ctx,
				SignedInUser: &usr,
				IsSignedIn:   true,
			},
		}
		fn(t, sc)
	})
}

func validCommand(resourceUID, title string) CreateRecentItemCommand {
	return CreateRecentItemCommand{
		ResourceType: "dashboard",
		ResourceUID:  resourceUID,
		Title:        title,
		URL:          "/d/" + resourceUID,
	}
}

func TestCreateRecentItem(t *testing.T) {
	testScenario(t, "creates an item and refreshes it on the next view", func(t *testing.T, sc scenarioContext) {
		start := time.UnixMilli(1_700_000_000_000)
		sc.service.now = func() time.Time { return start }

		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "CPU"))
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))
		require.Equal(t, "dashboard", created.ResourceType)
		require.Equal(t, "dash1", created.ResourceUID)
		require.Equal(t, "CPU", created.Title)
		require.Equal(t, "/d/dash1", created.URL)
		require.Equal(t, start.UnixMilli(), created.LastViewedAt)

		sc.service.now = func() time.Time { return start.Add(time.Second) }
		refreshedCmd := validCommand("dash1", "CPU usage")
		refreshedCmd.URL = "/d/dash1?viewPanel=1"
		sc.reqContext.Req.Body = mockRequestBody(refreshedCmd)
		refreshedResp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, http.StatusOK, refreshedResp.Status())
		refreshed := decodeItem(t, refreshedResp)
		require.Equal(t, created.UID, refreshed.UID)
		require.Equal(t, "CPU usage", refreshed.Title)
		require.Equal(t, "/d/dash1?viewPanel=1", refreshed.URL)
		require.Equal(t, start.Add(time.Second).UnixMilli(), refreshed.LastViewedAt)

		list := listItems(t, sc, "")
		require.Len(t, list.RecentItems, 1)
	})
}

func TestCreateRecentItemValidation(t *testing.T) {
	testScenario(t, "rejects malformed JSON", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = io.NopCloser(strings.NewReader("{"))
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, http.StatusBadRequest, resp.Status())
	})

	testScenario(t, "rejects an invalid resource type", func(t *testing.T, sc scenarioContext) {
		cmd := validCommand("dash1", "CPU")
		cmd.ResourceType = "Dashboard"
		sc.reqContext.Req.Body = mockRequestBody(cmd)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, http.StatusBadRequest, resp.Status())
	})

	testScenario(t, "rejects an absolute URL", func(t *testing.T, sc scenarioContext) {
		cmd := validCommand("dash1", "CPU")
		cmd.URL = "https://example.com/d/dash1"
		sc.reqContext.Req.Body = mockRequestBody(cmd)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, http.StatusBadRequest, resp.Status())
	})
}

func TestListRecentItems(t *testing.T) {
	testScenario(t, "returns newest items first and applies the default limit", func(t *testing.T, sc scenarioContext) {
		start := time.UnixMilli(1_700_000_000_000)
		for i := 0; i < defaultRecentItemsLimit+1; i++ {
			recordedAt := start.Add(time.Duration(i) * time.Second)
			sc.service.now = func() time.Time { return recordedAt }
			cmd := validCommand("res"+strconv.Itoa(i), "Item")
			sc.reqContext.Req.Body = mockRequestBody(cmd)
			require.Equal(t, http.StatusCreated, sc.service.createHandler(sc.reqContext).Status())
		}

		list := listItems(t, sc, "")
		require.Len(t, list.RecentItems, defaultRecentItemsLimit)
		require.Equal(t, "res"+strconv.Itoa(defaultRecentItemsLimit), list.RecentItems[0].ResourceUID)
		require.Equal(t, "res1", list.RecentItems[defaultRecentItemsLimit-1].ResourceUID)
	})

	testScenario(t, "filters by resource type and rejects a limit above the maximum", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "Dash"))
		require.Equal(t, http.StatusCreated, sc.service.createHandler(sc.reqContext).Status())

		folder := validCommand("fold1", "Folder")
		folder.ResourceType = "folder"
		folder.URL = "/dashboards/f/fold1"
		sc.reqContext.Req.Body = mockRequestBody(folder)
		require.Equal(t, http.StatusCreated, sc.service.createHandler(sc.reqContext).Status())

		list := listItems(t, sc, "resourceType=folder&limit=10")
		require.Len(t, list.RecentItems, 1)
		require.Equal(t, "fold1", list.RecentItems[0].ResourceUID)

		resp := listRaw(sc, "limit=51")
		require.Equal(t, http.StatusBadRequest, resp.Status())
	})
}

func TestRecentItemsTrimToCap(t *testing.T) {
	testScenario(t, "drops the oldest item when the cap is exceeded", func(t *testing.T, sc scenarioContext) {
		start := time.UnixMilli(1_700_000_000_000)
		for i := 0; i < maxRecentItemsPerUser+1; i++ {
			recordedAt := start.Add(time.Duration(i) * time.Second)
			sc.service.now = func() time.Time { return recordedAt }
			sc.reqContext.Req.Body = mockRequestBody(validCommand("res"+strconv.Itoa(i), "Item"))
			require.Equal(t, http.StatusCreated, sc.service.createHandler(sc.reqContext).Status())
		}

		list := listItems(t, sc, "limit=50")
		require.Len(t, list.RecentItems, maxRecentItemsPerUser)
		require.Equal(t, "res"+strconv.Itoa(maxRecentItemsPerUser), list.RecentItems[0].ResourceUID)
		for _, item := range list.RecentItems {
			require.NotEqual(t, "res0", item.ResourceUID)
		}
	})
}

func TestPatchRecentItem(t *testing.T) {
	testScenario(t, "updates title and url without changing lastViewedAt", func(t *testing.T, sc scenarioContext) {
		start := time.UnixMilli(1_700_000_000_000)
		sc.service.now = func() time.Time { return start }
		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "CPU"))
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))

		sc.service.now = func() time.Time { return start.Add(time.Hour) }
		setUID(sc, created.UID)
		sc.reqContext.Req.Body = mockRequestBody(map[string]string{
			"title": "CPU renamed",
			"url":   "/d/dash1?viewPanel=2",
		})
		resp := sc.service.patchHandler(sc.reqContext)
		require.Equal(t, http.StatusOK, resp.Status())
		patched := decodeItem(t, resp)
		require.Equal(t, "CPU renamed", patched.Title)
		require.Equal(t, "/d/dash1?viewPanel=2", patched.URL)
		require.Equal(t, created.LastViewedAt, patched.LastViewedAt)
	})

	testScenario(t, "rejects an empty patch and immutable fields", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "CPU"))
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))
		setUID(sc, created.UID)

		sc.reqContext.Req.Body = mockRequestBody(map[string]string{})
		require.Equal(t, http.StatusBadRequest, sc.service.patchHandler(sc.reqContext).Status())

		sc.reqContext.Req.Body = mockRequestBody(map[string]string{"resourceUid": "other"})
		require.Equal(t, http.StatusBadRequest, sc.service.patchHandler(sc.reqContext).Status())

		sc.reqContext.Req.Body = mockRequestBody(map[string]string{"title": "OK", "unknown": "nope"})
		require.Equal(t, http.StatusBadRequest, sc.service.patchHandler(sc.reqContext).Status())
	})

	testScenario(t, "returns not found for a missing uid", func(t *testing.T, sc scenarioContext) {
		setUID(sc, "missinguid")
		sc.reqContext.Req.Body = mockRequestBody(map[string]string{"title": "CPU"})
		require.Equal(t, http.StatusNotFound, sc.service.patchHandler(sc.reqContext).Status())
	})
}

func TestDeleteRecentItem(t *testing.T) {
	testScenario(t, "deletes an item and returns not found the second time", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "CPU"))
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))

		setUID(sc, created.UID)
		require.Equal(t, http.StatusNoContent, sc.service.deleteHandler(sc.reqContext).Status())
		require.Empty(t, listItems(t, sc, "").RecentItems)

		setUID(sc, created.UID)
		require.Equal(t, http.StatusNotFound, sc.service.deleteHandler(sc.reqContext).Status())
	})

	testScenario(t, "returns not found for a missing uid", func(t *testing.T, sc scenarioContext) {
		setUID(sc, "missinguid")
		require.Equal(t, http.StatusNotFound, sc.service.deleteHandler(sc.reqContext).Status())
	})
}

func TestRecentItemsAreScopedToUserAndOrg(t *testing.T) {
	testScenario(t, "hides items from another user and another org", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "CPU"))
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))

		otherUser := user.SignedInUser{UserID: 2, OrgID: 1, OrgRole: org.RoleEditor}
		sc.reqContext.SignedInUser = &otherUser
		require.Empty(t, listItems(t, sc, "").RecentItems)

		setUID(sc, created.UID)
		sc.reqContext.Req.Body = mockRequestBody(map[string]string{"title": "stolen"})
		require.Equal(t, http.StatusNotFound, sc.service.patchHandler(sc.reqContext).Status())
		require.Equal(t, http.StatusNotFound, sc.service.deleteHandler(sc.reqContext).Status())

		otherOrg := user.SignedInUser{UserID: 1, OrgID: 2, OrgRole: org.RoleEditor}
		sc.reqContext.SignedInUser = &otherOrg
		require.Empty(t, listItems(t, sc, "").RecentItems)
		setUID(sc, created.UID)
		require.Equal(t, http.StatusNotFound, sc.service.deleteHandler(sc.reqContext).Status())
	})
}

func TestRecentItemsPersistenceFailure(t *testing.T) {
	testScenario(t, "returns a safe server error when the store fails", func(t *testing.T, sc scenarioContext) {
		sc.service.store = &dbtest.FakeDB{ExpectedError: errors.New("database unavailable")}
		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "CPU"))
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, http.StatusInternalServerError, resp.Status())
		require.NotContains(t, string(resp.Body()), "database unavailable")

		setUID(sc, "someuid")
		require.Equal(t, http.StatusInternalServerError, sc.service.deleteHandler(sc.reqContext).Status())
	})
}

func TestRecentItemsRequireASignedInUser(t *testing.T) {
	testScenario(t, "rejects anonymous and missing users", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.SignedInUser.IsAnonymous = true
		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "CPU"))
		require.Equal(t, http.StatusUnauthorized, sc.service.createHandler(sc.reqContext).Status())
		require.Equal(t, http.StatusUnauthorized, sc.service.listHandler(sc.reqContext).Status())

		sc.reqContext.SignedInUser = nil
		setUID(sc, "someuid")
		sc.reqContext.Req.Body = mockRequestBody(map[string]string{"title": "CPU"})
		require.Equal(t, http.StatusUnauthorized, sc.service.patchHandler(sc.reqContext).Status())
		require.Equal(t, http.StatusUnauthorized, sc.service.deleteHandler(sc.reqContext).Status())
	})
}

func listItems(t *testing.T, sc scenarioContext, rawQuery string) ListRecentItemsResponse {
	t.Helper()
	resp := listRaw(sc, rawQuery)
	require.Equal(t, http.StatusOK, resp.Status())
	var list ListRecentItemsResponse
	require.NoError(t, json.Unmarshal(resp.Body(), &list))
	return list
}

func listRaw(sc scenarioContext, rawQuery string) response.Response {
	sc.ctx.Req.URL.RawQuery = rawQuery
	return sc.service.listHandler(sc.reqContext)
}

func setUID(sc scenarioContext, uid string) {
	sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": uid})
	sc.reqContext.Req = sc.ctx.Req
}

func mockRequestBody(v any) io.ReadCloser {
	b, _ := json.Marshal(v)
	return io.NopCloser(bytes.NewReader(b))
}

func validateCreateResponse(t *testing.T, resp response.Response) RecentItemDTO {
	t.Helper()
	require.Equal(t, http.StatusCreated, resp.Status())
	item := decodeItem(t, resp)
	require.NotEmpty(t, item.UID)
	normal, ok := resp.(*response.NormalResponse)
	require.True(t, ok)
	require.Equal(t, "/api/USER/RECENT_ITEMS/"+item.UID, normal.Header().Get("Location"))
	return item
}

func decodeItem(t *testing.T, resp response.Response) RecentItemDTO {
	t.Helper()
	var item RecentItemDTO
	require.NoError(t, json.Unmarshal(resp.Body(), &item))
	return item
}
