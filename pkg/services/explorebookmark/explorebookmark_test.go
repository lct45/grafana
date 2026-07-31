package explorebookmark

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
	service    *ExploreBookmarkService
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
		service := ExploreBookmarkService{
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

func TestCreateBookmark(t *testing.T) {
	testScenario(t, "creates a bookmark with all fields", func(t *testing.T, sc scenarioContext) {
		command := CreateExploreBookmarkCommand{
			Name:          "CPU usage",
			DatasourceUID: testDsUID,
			Queries: simplejson.NewFromAny([]interface{}{
				map[string]any{
					"expr": "rate(cpu_usage[5m])",
				},
			}),
			TimeFrom: "now-6h",
			TimeTo:   "now",
		}
		sc.reqContext.Req.Body = mockRequestBody(command)
		resp := sc.service.createHandler(sc.reqContext)
		result := validateAndUnMarshalResponse(t, resp)

		require.Equal(t, "CPU usage", result.Result.Name)
		require.Equal(t, testDsUID, result.Result.DatasourceUID)
		require.Equal(t, "now-6h", result.Result.TimeFrom)
		require.Equal(t, "now", result.Result.TimeTo)
		require.NotEmpty(t, result.Result.UID)
	})
}

func TestCreateBookmarkValidation(t *testing.T) {
	testScenario(t, "rejects empty name", func(t *testing.T, sc scenarioContext) {
		command := CreateExploreBookmarkCommand{
			Name:          "   ",
			DatasourceUID: testDsUID,
			Queries:       simplejson.NewFromAny([]interface{}{}),
			TimeFrom:      "now-6h",
			TimeTo:        "now",
		}
		sc.reqContext.Req.Body = mockRequestBody(command)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, 400, resp.Status())
	})

	testScenario(t, "rejects missing datasource uid", func(t *testing.T, sc scenarioContext) {
		command := CreateExploreBookmarkCommand{
			Name:     "CPU usage",
			Queries:  simplejson.NewFromAny([]interface{}{map[string]any{"expr": "up"}}),
			TimeFrom: "now-6h",
			TimeTo:   "now",
		}
		sc.reqContext.Req.Body = mockRequestBody(command)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, 400, resp.Status())
	})

	testScenario(t, "rejects missing queries", func(t *testing.T, sc scenarioContext) {
		command := CreateExploreBookmarkCommand{
			Name:          "CPU usage",
			DatasourceUID: testDsUID,
			TimeFrom:      "now-6h",
			TimeTo:        "now",
		}
		sc.reqContext.Req.Body = mockRequestBody(command)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, 400, resp.Status())
	})

	testScenario(t, "rejects missing time range", func(t *testing.T, sc scenarioContext) {
		command := CreateExploreBookmarkCommand{
			Name:          "CPU usage",
			DatasourceUID: testDsUID,
			Queries:       simplejson.NewFromAny([]interface{}{map[string]any{"expr": "up"}}),
			TimeFrom:      "now-6h",
		}
		sc.reqContext.Req.Body = mockRequestBody(command)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, 400, resp.Status())
	})
}

func TestListBookmarks(t *testing.T) {
	testScenario(t, "lists bookmarks for the signed-in user", func(t *testing.T, sc scenarioContext) {
		command := CreateExploreBookmarkCommand{
			Name:          "Memory usage",
			DatasourceUID: testDsUID,
			Queries: simplejson.NewFromAny([]interface{}{
				map[string]any{
					"expr": "node_memory_MemAvailable_bytes",
				},
			}),
			TimeFrom: "now-1h",
			TimeTo:   "now",
		}
		sc.reqContext.Req.Body = mockRequestBody(command)
		createResp := sc.service.createHandler(sc.reqContext)
		created := validateAndUnMarshalResponse(t, createResp)

		listResp := sc.service.listHandler(sc.reqContext)
		require.Equal(t, 200, listResp.Status())

		var listResult ExploreBookmarkListResponse
		err := json.Unmarshal(listResp.Body(), &listResult)
		require.NoError(t, err)
		require.Len(t, listResult.Result, 1)
		require.Equal(t, created.Result.UID, listResult.Result[0].UID)
	})
}

func TestDeleteBookmark(t *testing.T) {
	testScenario(t, "deletes an existing bookmark", func(t *testing.T, sc scenarioContext) {
		command := CreateExploreBookmarkCommand{
			Name:          "Disk IO",
			DatasourceUID: testDsUID,
			Queries: simplejson.NewFromAny([]interface{}{
				map[string]any{
					"expr": "rate(node_disk_io_time_seconds_total[5m])",
				},
			}),
			TimeFrom: "now-24h",
			TimeTo:   "now",
		}
		sc.reqContext.Req.Body = mockRequestBody(command)
		createResp := sc.service.createHandler(sc.reqContext)
		created := validateAndUnMarshalResponse(t, createResp)

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.Result.UID})
		deleteResp := sc.service.deleteHandler(sc.reqContext)
		require.Equal(t, 200, deleteResp.Status())

		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ExploreBookmarkListResponse
		err := json.Unmarshal(listResp.Body(), &listResult)
		require.NoError(t, err)
		require.Empty(t, listResult.Result)
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
		command := CreateExploreBookmarkCommand{
			Name:          "Private query",
			DatasourceUID: testDsUID,
			Queries: simplejson.NewFromAny([]interface{}{
				map[string]any{
					"expr": "up",
				},
			}),
			TimeFrom: "now-6h",
			TimeTo:   "now",
		}
		sc.reqContext.Req.Body = mockRequestBody(command)
		sc.service.createHandler(sc.reqContext)

		otherUser := user.SignedInUser{
			UserID:     999,
			OrgID:      testOrgID,
			OrgRole:    org.RoleEditor,
			LastSeenAt: sc.service.now(),
		}
		sc.reqContext.SignedInUser = &otherUser

		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ExploreBookmarkListResponse
		err := json.Unmarshal(listResp.Body(), &listResult)
		require.NoError(t, err)
		require.Empty(t, listResult.Result)
	})
}

func mockRequestBody(v any) io.ReadCloser {
	b, _ := json.Marshal(v)
	return io.NopCloser(bytes.NewReader(b))
}

func validateAndUnMarshalResponse(t *testing.T, resp response.Response) ExploreBookmarkResponse {
	t.Helper()

	require.Equal(t, 200, resp.Status())

	var result ExploreBookmarkResponse
	err := json.Unmarshal(resp.Body(), &result)
	require.NoError(t, err)

	return result
}
