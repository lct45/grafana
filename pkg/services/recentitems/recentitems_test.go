package recentitems

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/tracing"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/org/orgimpl"
	"github.com/grafana/grafana/pkg/services/quota/quotatest"
	"github.com/grafana/grafana/pkg/services/supportbundles/supportbundlestest"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/services/user/userimpl"
	"github.com/grafana/grafana/pkg/tests/testsuite"
	"github.com/grafana/grafana/pkg/web"
)

var (
	testOrgID  = int64(1)
	testUserID = int64(1)
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
			Form:   url.Values{},
		}}
		ctx.Req.Header.Add("Content-Type", "application/json")
		sqlStore, cfg := db.InitTestDBWithCfg(t)
		service := RecentItemsService{
			store: sqlStore,
			now:   time.Now,
		}
		quotaService := quotatest.New(false, nil)
		orgSvc, err := orgimpl.ProvideService(sqlStore, cfg, quotaService)
		require.NoError(t, err)
		usrSvc, err := userimpl.ProvideService(
			sqlStore, orgSvc, cfg, nil, nil, tracing.InitializeTracerForTest(),
			quotaService, supportbundlestest.NewFakeBundleService(), nil,
		)
		require.NoError(t, err)

		usr := user.SignedInUser{
			UserID:     testUserID,
			Name:       "Signed In User",
			Login:      "signed_in_user",
			Email:      "signed.in.user@test.com",
			OrgID:      testOrgID,
			OrgRole:    org.RoleEditor,
			LastSeenAt: service.now(),
		}

		_, err = usrSvc.Create(context.Background(), &user.CreateUserCommand{
			Email: "signed.in.user@test.com",
			Name:  "Signed In User",
			Login: "signed_in_user",
		})
		require.NoError(t, err)

		sc := scenarioContext{
			ctx:     &ctx,
			service: &service,
			reqContext: &contextmodel.ReqContext{
				Context:      &ctx,
				SignedInUser: &usr,
			},
		}
		fn(t, sc)
	})
}

func validCommand(title string) CreateRecentItemCommand {
	return CreateRecentItemCommand{
		ResourceType: ResourceTypeDashboard,
		ResourceUID:  "dash-uid-1",
		Title:        title,
		URL:          "/d/dash-uid-1",
	}
}

func TestCreateRecentItem(t *testing.T) {
	testScenario(t, "creates a recent item", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("Overview"))
		resp := sc.service.createHandler(sc.reqContext)
		result := validateCreateResponse(t, resp, 201)

		require.Equal(t, "Overview", result.Item.Title)
		require.Equal(t, ResourceTypeDashboard, result.Item.ResourceType)
		require.Equal(t, "dash-uid-1", result.Item.ResourceUID)
		require.Equal(t, "/d/dash-uid-1", result.Item.URL)
		require.NotEmpty(t, result.Item.UID)
		require.NotZero(t, result.Item.LastViewedAt)
		require.NotZero(t, result.Item.CreatedAt)
	})
}

func TestCreateRecentItemUpsert(t *testing.T) {
	testScenario(t, "upserts the same resource and returns 200", func(t *testing.T, sc scenarioContext) {
		start := time.UnixMilli(1_700_000_000_000)
		sc.service.now = func() time.Time { return start }
		sc.reqContext.Req.Body = mockRequestBody(validCommand("Overview"))
		first := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		sc.service.now = func() time.Time { return start.Add(time.Minute) }
		cmd := validCommand("Overview updated")
		cmd.URL = "/d/dash-uid-1?orgId=1"
		sc.reqContext.Req.Body = mockRequestBody(cmd)
		second := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 200)

		require.Equal(t, first.Item.UID, second.Item.UID)
		require.Equal(t, "Overview updated", second.Item.Title)
		require.Equal(t, "/d/dash-uid-1?orgId=1", second.Item.URL)
		require.Greater(t, second.Item.LastViewedAt, first.Item.LastViewedAt)
		require.Equal(t, first.Item.CreatedAt, second.Item.CreatedAt)

		listResp := sc.service.listHandler(sc.reqContext)
		require.Equal(t, 200, listResp.Status())
		var listResult ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(listResp.Body(), &listResult))
		require.Len(t, listResult.Items, 1)
	})
}

func TestCreateRecentItemValidation(t *testing.T) {
	testScenario(t, "rejects invalid resource type", func(t *testing.T, sc scenarioContext) {
		cmd := validCommand("Bad type")
		cmd.ResourceType = "playlist"
		sc.reqContext.Req.Body = mockRequestBody(cmd)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, 400, resp.Status())
	})

	testScenario(t, "rejects empty title", func(t *testing.T, sc scenarioContext) {
		cmd := validCommand("   ")
		sc.reqContext.Req.Body = mockRequestBody(cmd)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, 400, resp.Status())
	})
}

func TestListRecentItems(t *testing.T) {
	testScenario(t, "lists items newest first and respects limit", func(t *testing.T, sc scenarioContext) {
		start := time.UnixMilli(1_700_000_000_000)
		sc.service.now = func() time.Time { return start }
		sc.reqContext.Req.Body = mockRequestBody(CreateRecentItemCommand{
			ResourceType: ResourceTypeDashboard,
			ResourceUID:  "older",
			Title:        "Older",
			URL:          "/d/older",
		})
		older := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		sc.service.now = func() time.Time { return start.Add(time.Second) }
		sc.reqContext.Req.Body = mockRequestBody(CreateRecentItemCommand{
			ResourceType: ResourceTypeFolder,
			ResourceUID:  "newer",
			Title:        "Newer",
			URL:          "/dashboards/f/newer",
		})
		newer := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		listResp := sc.service.listHandler(sc.reqContext)
		require.Equal(t, 200, listResp.Status())
		var listResult ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(listResp.Body(), &listResult))
		require.Len(t, listResult.Items, 2)
		require.Equal(t, newer.Item.UID, listResult.Items[0].UID)
		require.Equal(t, older.Item.UID, listResult.Items[1].UID)

		sc.reqContext.Req.Form.Set("limit", "1")
		limited := sc.service.listHandler(sc.reqContext)
		require.Equal(t, 200, limited.Status())
		var limitedResult ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(limited.Body(), &limitedResult))
		require.Len(t, limitedResult.Items, 1)
		require.Equal(t, newer.Item.UID, limitedResult.Items[0].UID)

		sc.reqContext.Req.Form.Set("limit", "0")
		badLimit := sc.service.listHandler(sc.reqContext)
		require.Equal(t, 400, badLimit.Status())
	})
}

func TestPatchRecentItem(t *testing.T) {
	testScenario(t, "patches title and url", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("Overview"))
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.Item.UID})
		sc.reqContext.Req.Body = mockRequestBody(map[string]any{
			"title": "Renamed",
			"url":   "/d/dash-uid-1/renamed",
		})
		resp := sc.service.patchHandler(sc.reqContext)
		require.Equal(t, 200, resp.Status())

		var result CreateRecentItemResponse
		require.NoError(t, json.Unmarshal(resp.Body(), &result))
		require.Equal(t, "Renamed", result.Item.Title)
		require.Equal(t, "/d/dash-uid-1/renamed", result.Item.URL)
		require.Equal(t, created.Item.ResourceType, result.Item.ResourceType)
	})

	testScenario(t, "rejects immutable resourceType", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("Overview"))
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.Item.UID})
		sc.reqContext.Req.Body = mockRequestBody(map[string]any{
			"resourceType": ResourceTypeFolder,
			"title":        "Nope",
		})
		resp := sc.service.patchHandler(sc.reqContext)
		require.Equal(t, 400, resp.Status())
	})

	testScenario(t, "returns not found for missing uid", func(t *testing.T, sc scenarioContext) {
		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": "missinguid"})
		sc.reqContext.Req.Body = mockRequestBody(map[string]any{"title": "Missing"})
		resp := sc.service.patchHandler(sc.reqContext)
		require.Equal(t, 404, resp.Status())
	})
}

func TestDeleteRecentItem(t *testing.T) {
	testScenario(t, "deletes an existing item", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("Overview"))
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.Item.UID})
		deleteResp := sc.service.deleteHandler(sc.reqContext)
		require.Equal(t, 200, deleteResp.Status())

		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(listResp.Body(), &listResult))
		require.Empty(t, listResult.Items)
	})
}

func TestDeleteRecentItemNotFound(t *testing.T) {
	testScenario(t, "returns not found for missing item", func(t *testing.T, sc scenarioContext) {
		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": "missinguid"})
		deleteResp := sc.service.deleteHandler(sc.reqContext)
		require.Equal(t, 404, deleteResp.Status())
	})
}

func TestRecentItemsAreUserScoped(t *testing.T) {
	testScenario(t, "does not list or delete another user's items", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("Private"))
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		otherUser := user.SignedInUser{
			UserID:     999,
			OrgID:      testOrgID,
			OrgRole:    org.RoleEditor,
			LastSeenAt: sc.service.now(),
		}
		sc.reqContext.SignedInUser = &otherUser

		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(listResp.Body(), &listResult))
		require.Empty(t, listResult.Items)

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.Item.UID})
		deleteResp := sc.service.deleteHandler(sc.reqContext)
		require.Equal(t, 404, deleteResp.Status())
	})
}

func TestRecentItemsAreOrgScoped(t *testing.T) {
	testScenario(t, "does not list items from another org", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("Org one"))
		validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		sc.reqContext.SignedInUser.OrgID = 2
		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(listResp.Body(), &listResult))
		require.Empty(t, listResult.Items)
	})
}

func TestCreateRecentItemTrimsToMax(t *testing.T) {
	testScenario(t, "trims oldest items when exceeding max stored", func(t *testing.T, sc scenarioContext) {
		start := time.UnixMilli(1_700_000_000_000)
		for i := 0; i < MaxStoredItems; i++ {
			sc.service.now = func() time.Time { return start.Add(time.Duration(i) * time.Second) }
			sc.reqContext.Req.Body = mockRequestBody(CreateRecentItemCommand{
				ResourceType: ResourceTypeDashboard,
				ResourceUID:  fmt.Sprintf("dash-%d", i),
				Title:        fmt.Sprintf("Dash %d", i),
				URL:          fmt.Sprintf("/d/dash-%d", i),
			})
			validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)
		}

		sc.service.now = func() time.Time { return start.Add(time.Duration(MaxStoredItems) * time.Second) }
		sc.reqContext.Req.Body = mockRequestBody(CreateRecentItemCommand{
			ResourceType: ResourceTypeDashboard,
			ResourceUID:  "dash-newest",
			Title:        "Newest",
			URL:          "/d/dash-newest",
		})
		newest := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(listResp.Body(), &listResult))
		require.Len(t, listResult.Items, MaxStoredItems)
		require.Equal(t, newest.Item.UID, listResult.Items[0].UID)

		for _, item := range listResult.Items {
			require.NotEqual(t, "dash-0", item.ResourceUID)
		}
	})
}

func mockRequestBody(v any) io.ReadCloser {
	b, _ := json.Marshal(v)
	return io.NopCloser(bytes.NewReader(b))
}

func validateCreateResponse(t *testing.T, resp response.Response, expectedStatus int) CreateRecentItemResponse {
	t.Helper()

	require.Equal(t, expectedStatus, resp.Status())

	var result CreateRecentItemResponse
	err := json.Unmarshal(resp.Body(), &result)
	require.NoError(t, err)

	return result
}
