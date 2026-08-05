package explorebookmarks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/tracing"
	accesscontrolmock "github.com/grafana/grafana/pkg/services/accesscontrol/mock"
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
	testDsUID  = "NCzh67i"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

type scenarioContext struct {
	ctx        *web.Context
	service    *ExploreBookmarksService
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
		service := ExploreBookmarksService{
			store:         sqlStore,
			now:           time.Now,
			accessControl: accesscontrolmock.New(),
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

func validCommand(name string) CreateBookmarkCommand {
	return CreateBookmarkCommand{
		Name:          name,
		DatasourceUID: testDsUID,
		Queries: simplejson.NewFromAny([]interface{}{
			map[string]any{
				"refId": "A",
				"expr":  "rate(cpu_usage[5m])",
			},
		}),
		TimeRange: TimeRangeDTO{From: "now-6h", To: "now"},
	}
}

func TestCreateBookmark(t *testing.T) {
	testScenario(t, "creates a bookmark with all fields", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("CPU usage"))
		resp := sc.service.createHandler(sc.reqContext)
		result := validateCreateResponse(t, resp)

		require.Equal(t, "CPU usage", result.Bookmark.Name)
		require.Equal(t, testDsUID, result.Bookmark.DatasourceUID)
		require.Equal(t, "now-6h", result.Bookmark.TimeRange.From)
		require.Equal(t, "now", result.Bookmark.TimeRange.To)
		require.NotEmpty(t, result.Bookmark.UID)
	})
}

func TestCreateBookmarkValidation(t *testing.T) {
	testScenario(t, "rejects empty name", func(t *testing.T, sc scenarioContext) {
		cmd := validCommand("   ")
		sc.reqContext.Req.Body = mockRequestBody(cmd)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, 400, resp.Status())
	})

	testScenario(t, "rejects empty queries", func(t *testing.T, sc scenarioContext) {
		cmd := validCommand("Empty queries")
		cmd.Queries = simplejson.NewFromAny([]interface{}{})
		sc.reqContext.Req.Body = mockRequestBody(cmd)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, 400, resp.Status())
	})
}

func TestListBookmarks(t *testing.T) {
	testScenario(t, "lists bookmarks for the signed-in user newest first", func(t *testing.T, sc scenarioContext) {
		start := time.Now()
		sc.service.now = func() time.Time { return start }
		sc.reqContext.Req.Body = mockRequestBody(validCommand("Older"))
		older := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))

		sc.service.now = func() time.Time { return start.Add(time.Second) }
		sc.reqContext.Req.Body = mockRequestBody(validCommand("Newer"))
		newer := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))

		listResp := sc.service.listHandler(sc.reqContext)
		require.Equal(t, 200, listResp.Status())

		var listResult ListBookmarksResponse
		err := json.Unmarshal(listResp.Body(), &listResult)
		require.NoError(t, err)
		require.Len(t, listResult.Bookmarks, 2)
		require.Equal(t, newer.Bookmark.UID, listResult.Bookmarks[0].UID)
		require.Equal(t, older.Bookmark.UID, listResult.Bookmarks[1].UID)
	})
}

func TestDeleteBookmark(t *testing.T) {
	testScenario(t, "deletes an existing bookmark", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("Disk IO"))
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.Bookmark.UID})
		deleteResp := sc.service.deleteHandler(sc.reqContext)
		require.Equal(t, 200, deleteResp.Status())

		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ListBookmarksResponse
		err := json.Unmarshal(listResp.Body(), &listResult)
		require.NoError(t, err)
		require.Empty(t, listResult.Bookmarks)
	})
}

func TestDeleteBookmarkNotFound(t *testing.T) {
	testScenario(t, "returns not found for missing bookmark", func(t *testing.T, sc scenarioContext) {
		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": "missinguid"})
		deleteResp := sc.service.deleteHandler(sc.reqContext)
		require.Equal(t, 404, deleteResp.Status())
	})
}

func TestBookmarksAreUserScoped(t *testing.T) {
	testScenario(t, "does not list bookmarks from another user", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCommand("Private query"))
		sc.service.createHandler(sc.reqContext)

		otherUser := user.SignedInUser{
			UserID:     999,
			OrgID:      testOrgID,
			OrgRole:    org.RoleEditor,
			LastSeenAt: sc.service.now(),
		}
		sc.reqContext.SignedInUser = &otherUser

		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ListBookmarksResponse
		err := json.Unmarshal(listResp.Body(), &listResult)
		require.NoError(t, err)
		require.Empty(t, listResult.Bookmarks)
	})
}

func mockRequestBody(v any) io.ReadCloser {
	b, _ := json.Marshal(v)
	return io.NopCloser(bytes.NewReader(b))
}

func validateCreateResponse(t *testing.T, resp response.Response) CreateBookmarkResponse {
	t.Helper()

	require.Equal(t, 201, resp.Status())

	var result CreateBookmarkResponse
	err := json.Unmarshal(resp.Body(), &result)
	require.NoError(t, err)

	return result
}
