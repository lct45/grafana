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
			URL:    &url.URL{},
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

func validCommand(resourceUID, title string) CreateRecentItemCommand {
	return CreateRecentItemCommand{
		ResourceType: "dashboard",
		ResourceUID:  resourceUID,
		Title:        title,
		URL:          fmt.Sprintf("/d/%s", resourceUID),
	}
}

func TestCreateRecentItem(t *testing.T) {
	testScenario(t, "creates a recent item with all fields", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "My Dashboard"))
		resp := sc.service.createHandler(sc.reqContext)
		result := validateCreateResponse(t, resp, 201)

		require.Equal(t, "dashboard", result.Item.ResourceType)
		require.Equal(t, "dash1", result.Item.ResourceUID)
		require.Equal(t, "My Dashboard", result.Item.Title)
		require.Equal(t, "/d/dash1", result.Item.URL)
		require.NotEmpty(t, result.Item.UID)
		require.NotZero(t, result.Item.LastViewedAt)
		require.NotZero(t, result.Item.CreatedAt)
	})
}

func TestCreateRecentItemUpsert(t *testing.T) {
	testScenario(t, "upserts an existing resource and returns 200", func(t *testing.T, sc scenarioContext) {
		start := time.Now()
		sc.service.now = func() time.Time { return start }
		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "Original"))
		first := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		sc.service.now = func() time.Time { return start.Add(time.Minute) }
		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "Updated Title"))
		second := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 200)

		require.Equal(t, first.Item.UID, second.Item.UID)
		require.Equal(t, "Updated Title", second.Item.Title)
		require.Greater(t, second.Item.LastViewedAt, first.Item.LastViewedAt)
		require.Equal(t, first.Item.CreatedAt, second.Item.CreatedAt)

		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(listResp.Body(), &listResult))
		require.Len(t, listResult.Items, 1)
	})
}

func TestCreateRecentItemValidation(t *testing.T) {
	testScenario(t, "rejects invalid resource type", func(t *testing.T, sc scenarioContext) {
		cmd := validCommand("dash1", "Title")
		cmd.ResourceType = "unknown"
		sc.reqContext.Req.Body = mockRequestBody(cmd)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, 400, resp.Status())
	})

	testScenario(t, "rejects empty title", func(t *testing.T, sc scenarioContext) {
		cmd := validCommand("dash1", "   ")
		sc.reqContext.Req.Body = mockRequestBody(cmd)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, 400, resp.Status())
	})
}

func TestListRecentItems(t *testing.T) {
	testScenario(t, "lists items newest first and respects limit", func(t *testing.T, sc scenarioContext) {
		start := time.Now()
		sc.service.now = func() time.Time { return start }
		sc.reqContext.Req.Body = mockRequestBody(validCommand("older", "Older"))
		older := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		sc.service.now = func() time.Time { return start.Add(time.Second) }
		sc.reqContext.Req.Body = mockRequestBody(validCommand("newer", "Newer"))
		newer := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		listResp := sc.service.listHandler(sc.reqContext)
		require.Equal(t, 200, listResp.Status())

		var listResult ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(listResp.Body(), &listResult))
		require.Len(t, listResult.Items, 2)
		require.Equal(t, newer.Item.UID, listResult.Items[0].UID)
		require.Equal(t, older.Item.UID, listResult.Items[1].UID)

		sc.ctx.Req.Form = url.Values{"limit": []string{"1"}}
		limited := sc.service.listHandler(sc.reqContext)
		var limitedResult ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(limited.Body(), &limitedResult))
		require.Len(t, limitedResult.Items, 1)
		require.Equal(t, newer.Item.UID, limitedResult.Items[0].UID)

		sc.ctx.Req.Form = url.Values{"limit": []string{"101"}}
		badLimit := sc.service.listHandler(sc.reqContext)
		require.Equal(t, 400, badLimit.Status())
	})
}

func TestPatchRecentItem(t *testing.T) {
	testScenario(t, "patches title and url", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "Original"))
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		title := "Patched"
		urlVal := "/d/dash1?view=new"
		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.Item.UID})
		sc.reqContext.Req.Body = mockRequestBody(PatchRecentItemCommand{Title: &title, URL: &urlVal})
		resp := sc.service.patchHandler(sc.reqContext)
		require.Equal(t, 200, resp.Status())

		var result PatchRecentItemResponse
		require.NoError(t, json.Unmarshal(resp.Body(), &result))
		require.Equal(t, "Patched", result.Item.Title)
		require.Equal(t, "/d/dash1?view=new", result.Item.URL)
		require.Equal(t, created.Item.ResourceUID, result.Item.ResourceUID)
	})

	testScenario(t, "rejects immutable resourceType", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "Original"))
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.Item.UID})
		sc.reqContext.Req.Body = mockRequestBody(map[string]any{
			"title":        "Nope",
			"resourceType": "folder",
		})
		resp := sc.service.patchHandler(sc.reqContext)
		require.Equal(t, 400, resp.Status())
	})

	testScenario(t, "returns not found for missing uid", func(t *testing.T, sc scenarioContext) {
		title := "Missing"
		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": "missinguid"})
		sc.reqContext.Req.Body = mockRequestBody(PatchRecentItemCommand{Title: &title})
		resp := sc.service.patchHandler(sc.reqContext)
		require.Equal(t, 404, resp.Status())
	})
}

func TestDeleteRecentItem(t *testing.T) {
	testScenario(t, "deletes an existing recent item", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("dash1", "To Delete"))
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
	testScenario(t, "does not list or delete items from another user", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("private", "Private"))
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
		sc.reqContext.Req.Body = mockRequestBody(validCommand("org1", "Org One"))
		validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		sc.reqContext.SignedInUser.OrgID = 2
		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(listResp.Body(), &listResult))
		require.Empty(t, listResult.Items)
	})
}

func TestCreateRecentItemTrimsToMax(t *testing.T) {
	testScenario(t, "trims oldest item when exceeding max stored items", func(t *testing.T, sc scenarioContext) {
		start := time.Now()
		for i := 0; i < maxStoredItems; i++ {
			sc.service.now = func() time.Time { return start.Add(time.Duration(i) * time.Second) }
			uid := fmt.Sprintf("d%02d", i)
			sc.reqContext.Req.Body = mockRequestBody(validCommand(uid, uid))
			validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)
		}

		sc.service.now = func() time.Time { return start.Add(time.Duration(maxStoredItems) * time.Second) }
		sc.reqContext.Req.Body = mockRequestBody(validCommand("newest", "Newest"))
		validateCreateResponse(t, sc.service.createHandler(sc.reqContext), 201)

		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(listResp.Body(), &listResult))
		require.Len(t, listResult.Items, maxStoredItems)
		require.Equal(t, "newest", listResult.Items[0].ResourceUID)

		for _, item := range listResult.Items {
			require.NotEqual(t, "d00", item.ResourceUID)
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
