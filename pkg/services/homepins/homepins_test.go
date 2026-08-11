package homepins

import (
	"bytes"
	"encoding/json"
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
	testDashUID = "abc123"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

type scenarioContext struct {
	ctx        *web.Context
	service    *HomePinsService
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
		service := HomePinsService{
			store: sqlStore,
			now:   time.Now,
		}
		quotaService := quotatest.New(false, nil)
		orgSvc, err := orgimpl.ProvideService(sqlStore, cfg, quotaService)
		require.NoError(t, err)
		_, err = userimpl.ProvideService(
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

func validCreateCommand() CreatePinCommand {
	return CreatePinCommand{
		DashboardUID: testDashUID,
		Note:         "On-call overview",
	}
}

func TestCreatePin(t *testing.T) {
	testScenario(t, "creates a pin with note", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCreateCommand())
		resp := sc.service.createHandler(sc.reqContext)
		result := validateCreateResponse(t, resp)

		require.Equal(t, testDashUID, result.Pin.DashboardUID)
		require.Equal(t, "On-call overview", result.Pin.Note)
		require.Equal(t, 0, result.Pin.SortOrder)
		require.NotEmpty(t, result.Pin.UID)
	})
}

func TestCreatePinValidation(t *testing.T) {
	testScenario(t, "rejects empty dashboard uid", func(t *testing.T, sc scenarioContext) {
		cmd := validCreateCommand()
		cmd.DashboardUID = "   "
		sc.reqContext.Req.Body = mockRequestBody(cmd)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, 400, resp.Status())
	})

	testScenario(t, "rejects duplicate pin", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCreateCommand())
		sc.service.createHandler(sc.reqContext)

		sc.reqContext.Req.Body = mockRequestBody(validCreateCommand())
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, 409, resp.Status())
	})
}

func TestListPins(t *testing.T) {
	testScenario(t, "lists pins in sort order", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(CreatePinCommand{DashboardUID: "first"})
		first := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))

		sc.reqContext.Req.Body = mockRequestBody(CreatePinCommand{DashboardUID: "second"})
		second := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))

		listResp := sc.service.listHandler(sc.reqContext)
		require.Equal(t, 200, listResp.Status())

		var listResult ListPinsResponse
		err := json.Unmarshal(listResp.Body(), &listResult)
		require.NoError(t, err)
		require.Len(t, listResult.Pins, 2)
		require.Equal(t, first.Pin.UID, listResult.Pins[0].UID)
		require.Equal(t, second.Pin.UID, listResult.Pins[1].UID)
	})
}

func TestUpdatePin(t *testing.T) {
	testScenario(t, "updates pin note", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCreateCommand())
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.Pin.UID})
		sc.reqContext.Req.Body = mockRequestBody(UpdatePinCommand{Note: "Updated note"})
		updateResp := sc.service.updateHandler(sc.reqContext)
		require.Equal(t, 200, updateResp.Status())

		var updateResult UpdatePinResponse
		err := json.Unmarshal(updateResp.Body(), &updateResult)
		require.NoError(t, err)
		require.Equal(t, "Updated note", updateResult.Pin.Note)
	})
}

func TestReorderPins(t *testing.T) {
	testScenario(t, "reorders pins", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(CreatePinCommand{DashboardUID: "first"})
		first := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))

		sc.reqContext.Req.Body = mockRequestBody(CreatePinCommand{DashboardUID: "second"})
		second := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))

		sc.reqContext.Req.Body = mockRequestBody(ReorderPinsCommand{UIDs: []string{second.Pin.UID, first.Pin.UID}})
		reorderResp := sc.service.reorderHandler(sc.reqContext)
		require.Equal(t, 200, reorderResp.Status())

		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ListPinsResponse
		err := json.Unmarshal(listResp.Body(), &listResult)
		require.NoError(t, err)
		require.Equal(t, second.Pin.UID, listResult.Pins[0].UID)
		require.Equal(t, first.Pin.UID, listResult.Pins[1].UID)
	})
}

func TestDeletePin(t *testing.T) {
	testScenario(t, "deletes an existing pin", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCreateCommand())
		created := validateCreateResponse(t, sc.service.createHandler(sc.reqContext))

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.Pin.UID})
		deleteResp := sc.service.deleteHandler(sc.reqContext)
		require.Equal(t, 200, deleteResp.Status())

		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ListPinsResponse
		err := json.Unmarshal(listResp.Body(), &listResult)
		require.NoError(t, err)
		require.Empty(t, listResult.Pins)
	})
}

func TestDeletePinNotFound(t *testing.T) {
	testScenario(t, "returns not found for missing pin", func(t *testing.T, sc scenarioContext) {
		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": "missinguid"})
		deleteResp := sc.service.deleteHandler(sc.reqContext)
		require.Equal(t, 404, deleteResp.Status())
	})
}

func TestPinsAreUserScoped(t *testing.T) {
	testScenario(t, "does not list pins from another user", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(validCreateCommand())
		sc.service.createHandler(sc.reqContext)

		otherUser := user.SignedInUser{
			UserID:     999,
			OrgID:      testOrgID,
			OrgRole:    org.RoleEditor,
			LastSeenAt: sc.service.now(),
		}
		sc.reqContext.SignedInUser = &otherUser

		listResp := sc.service.listHandler(sc.reqContext)
		var listResult ListPinsResponse
		err := json.Unmarshal(listResp.Body(), &listResult)
		require.NoError(t, err)
		require.Empty(t, listResult.Pins)
	})
}

func mockRequestBody(v any) io.ReadCloser {
	b, _ := json.Marshal(v)
	return io.NopCloser(bytes.NewReader(b))
}

func validateCreateResponse(t *testing.T, resp response.Response) CreatePinResponse {
	t.Helper()

	require.Equal(t, 201, resp.Status())

	var result CreatePinResponse
	err := json.Unmarshal(resp.Body(), &result)
	require.NoError(t, err)

	return result
}
