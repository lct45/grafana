package api

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
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/tracing"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/dashboardpins"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/org/orgimpl"
	"github.com/grafana/grafana/pkg/services/quota/quotatest"
	"github.com/grafana/grafana/pkg/services/supportbundles/supportbundlestest"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/services/user/userimpl"
	"github.com/grafana/grafana/pkg/tests/testsuite"
	"github.com/grafana/grafana/pkg/web"
)

const (
	testOrgID   = int64(1)
	testOrgID2  = int64(2)
	testUserID  = int64(1)
	testDashUID = "test-dashboard-uid"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

type scenarioContext struct {
	ctx        *web.Context
	api        *API
	reqContext *contextmodel.ReqContext
}

type mockDashboardService struct {
	existing map[int64]map[string]bool
}

func (m *mockDashboardService) GetDashboard(_ context.Context, query *dashboards.GetDashboardQuery) (*dashboards.Dashboard, error) {
	orgDashboards, ok := m.existing[query.OrgID]
	if !ok {
		return nil, dashboards.ErrDashboardNotFound
	}
	if orgDashboards[query.UID] {
		return &dashboards.Dashboard{UID: query.UID, OrgID: query.OrgID}, nil
	}
	return nil, dashboards.ErrDashboardNotFound
}

func testScenario(t *testing.T, desc string, orgDashboards map[int64][]string, fn func(t *testing.T, sc scenarioContext)) {
	t.Helper()

	t.Run(desc, func(t *testing.T) {
		ctx := web.Context{Req: &http.Request{
			Header: http.Header{},
			Form:   url.Values{},
		}}
		ctx.Req.Header.Add("Content-Type", "application/json")

		sqlStore, cfg := db.InitTestDBWithCfg(t)
		existing := make(map[int64]map[string]bool)
		for orgID, uids := range orgDashboards {
			existing[orgID] = make(map[string]bool, len(uids))
			for _, uid := range uids {
				existing[orgID][uid] = true
			}
		}

		service := dashboardpins.NewTestService(sqlStore, &mockDashboardService{existing: existing}, time.Now)
		api := ProvideApi(service)

		quotaService := quotatest.New(false, nil)
		orgSvc, err := orgimpl.ProvideService(sqlStore, cfg, quotaService)
		require.NoError(t, err)
		usrSvc, err := userimpl.ProvideService(
			sqlStore, orgSvc, cfg, nil, nil, tracing.InitializeTracerForTest(),
			quotaService, supportbundlestest.NewFakeBundleService(), nil,
		)
		require.NoError(t, err)

		_, err = usrSvc.Create(context.Background(), &user.CreateUserCommand{
			Email: "signed.in.user@test.com",
			Name:  "Signed In User",
			Login: "signed_in_user",
		})
		require.NoError(t, err)

		usr := user.SignedInUser{
			UserID:  testUserID,
			Name:    "Signed In User",
			Login:   "signed_in_user",
			Email:   "signed.in.user@test.com",
			OrgID:   testOrgID,
			OrgRole: org.RoleEditor,
		}

		sc := scenarioContext{
			ctx: &ctx,
			api: api,
			reqContext: &contextmodel.ReqContext{
				Context:      &ctx,
				SignedInUser: &usr,
			},
		}
		fn(t, sc)
	})
}

func TestCreateDashboardPinHandler(t *testing.T) {
	testScenario(t, "creates a dashboard pin with location header", map[int64][]string{testOrgID: {testDashUID}}, func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(dashboardpins.CreateDashboardPinCommand{
			DashboardUID: testDashUID,
			Note:         notePtr("Home dashboard"),
		})

		resp := sc.api.CreateDashboardPin(sc.reqContext)
		require.Equal(t, http.StatusCreated, resp.Status())
		normalResp, ok := resp.(*response.NormalResponse)
		require.True(t, ok)
		require.Equal(t, "/api/user/dashboard-pins/"+testDashUID, normalResp.Header().Get("Location"))

		var result dashboardpins.CreateDashboardPinResponse
		err := json.Unmarshal(resp.Body(), &result)
		require.NoError(t, err)
		require.Equal(t, testDashUID, result.Pin.DashboardUID)
		require.NotNil(t, result.Pin.Note)
		require.Equal(t, "Home dashboard", *result.Pin.Note)
	})
}

func TestCreateDashboardPinNotFound(t *testing.T) {
	testScenario(t, "returns not found when dashboard does not exist", nil, func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(dashboardpins.CreateDashboardPinCommand{
			DashboardUID: testDashUID,
		})

		resp := sc.api.CreateDashboardPin(sc.reqContext)
		require.Equal(t, http.StatusNotFound, resp.Status())
	})
}

func TestListDashboardPinsHandler(t *testing.T) {
	testScenario(t, "lists dashboard pins for the signed-in user", map[int64][]string{testOrgID: {testDashUID, "second-dashboard"}}, func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(dashboardpins.CreateDashboardPinCommand{DashboardUID: testDashUID})
		require.Equal(t, http.StatusCreated, sc.api.CreateDashboardPin(sc.reqContext).Status())

		sc.reqContext.Req.Body = mockRequestBody(dashboardpins.CreateDashboardPinCommand{DashboardUID: "second-dashboard"})
		require.Equal(t, http.StatusCreated, sc.api.CreateDashboardPin(sc.reqContext).Status())

		resp := sc.api.ListDashboardPins(sc.reqContext)
		require.Equal(t, http.StatusOK, resp.Status())

		var result dashboardpins.ListDashboardPinsResponse
		err := json.Unmarshal(resp.Body(), &result)
		require.NoError(t, err)
		require.Len(t, result.Pins, 2)
	})
}

func TestReorderDashboardPinsHandler(t *testing.T) {
	testScenario(t, "reorders dashboard pins by array index", map[int64][]string{testOrgID: {testDashUID, "second-dashboard"}}, func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(dashboardpins.CreateDashboardPinCommand{DashboardUID: testDashUID})
		require.Equal(t, http.StatusCreated, sc.api.CreateDashboardPin(sc.reqContext).Status())
		sc.reqContext.Req.Body = mockRequestBody(dashboardpins.CreateDashboardPinCommand{DashboardUID: "second-dashboard"})
		require.Equal(t, http.StatusCreated, sc.api.CreateDashboardPin(sc.reqContext).Status())

		sc.reqContext.Req.Body = mockRequestBody(dashboardpins.ReorderDashboardPinsCommand{
			DashboardUIDs: []string{"second-dashboard", testDashUID},
		})
		resp := sc.api.ReorderDashboardPins(sc.reqContext)
		require.Equal(t, http.StatusOK, resp.Status())

		var result dashboardpins.ListDashboardPinsResponse
		err := json.Unmarshal(resp.Body(), &result)
		require.NoError(t, err)
		require.Equal(t, "second-dashboard", result.Pins[0].DashboardUID)
		require.Equal(t, 0, result.Pins[0].SortOrder)
		require.Equal(t, testDashUID, result.Pins[1].DashboardUID)
		require.Equal(t, 1, result.Pins[1].SortOrder)
	})
}

func TestPatchDashboardPinHandler(t *testing.T) {
	testScenario(t, "updates dashboard pin note", map[int64][]string{testOrgID: {testDashUID}}, func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(dashboardpins.CreateDashboardPinCommand{DashboardUID: testDashUID})
		require.Equal(t, http.StatusCreated, sc.api.CreateDashboardPin(sc.reqContext).Status())

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":dashboardUid": testDashUID})
		sc.reqContext.Req.Body = mockRequestBody(dashboardpins.PatchDashboardPinCommand{
			Note: notePtr("Updated"),
		})

		resp := sc.api.PatchDashboardPin(sc.reqContext)
		require.Equal(t, http.StatusOK, resp.Status())

		var result dashboardpins.PatchDashboardPinResponse
		err := json.Unmarshal(resp.Body(), &result)
		require.NoError(t, err)
		require.NotNil(t, result.Pin.Note)
		require.Equal(t, "Updated", *result.Pin.Note)
	})
}

func TestPatchDashboardPinRejectsUnknownFields(t *testing.T) {
	testScenario(t, "rejects unknown patch fields", map[int64][]string{testOrgID: {testDashUID}}, func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(dashboardpins.CreateDashboardPinCommand{DashboardUID: testDashUID})
		require.Equal(t, http.StatusCreated, sc.api.CreateDashboardPin(sc.reqContext).Status())

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":dashboardUid": testDashUID})
		sc.reqContext.Req.Body = io.NopCloser(bytes.NewReader([]byte(`{"note":"Updated","sortOrder":1}`)))

		resp := sc.api.PatchDashboardPin(sc.reqContext)
		require.Equal(t, http.StatusBadRequest, resp.Status())
	})
}

func TestDeleteDashboardPinHandler(t *testing.T) {
	testScenario(t, "deletes an existing dashboard pin", map[int64][]string{testOrgID: {testDashUID}}, func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(dashboardpins.CreateDashboardPinCommand{DashboardUID: testDashUID})
		require.Equal(t, http.StatusCreated, sc.api.CreateDashboardPin(sc.reqContext).Status())

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":dashboardUid": testDashUID})
		resp := sc.api.DeleteDashboardPin(sc.reqContext)
		require.Equal(t, http.StatusOK, resp.Status())

		listResp := sc.api.ListDashboardPins(sc.reqContext)
		var result dashboardpins.ListDashboardPinsResponse
		err := json.Unmarshal(listResp.Body(), &result)
		require.NoError(t, err)
		require.Empty(t, result.Pins)
	})
}

func TestDeleteDashboardPinNotFound(t *testing.T) {
	testScenario(t, "returns not found for missing dashboard pin", map[int64][]string{testOrgID: {testDashUID}}, func(t *testing.T, sc scenarioContext) {
		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":dashboardUid": testDashUID})
		resp := sc.api.DeleteDashboardPin(sc.reqContext)
		require.Equal(t, http.StatusNotFound, resp.Status())
	})
}

func TestDashboardPinsAreUserScoped(t *testing.T) {
	testScenario(t, "does not expose pins from another user", map[int64][]string{testOrgID: {testDashUID}}, func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(dashboardpins.CreateDashboardPinCommand{DashboardUID: testDashUID})
		require.Equal(t, http.StatusCreated, sc.api.CreateDashboardPin(sc.reqContext).Status())

		otherUser := user.SignedInUser{
			UserID:  999,
			OrgID:   testOrgID,
			OrgRole: org.RoleEditor,
		}
		sc.reqContext.SignedInUser = &otherUser

		resp := sc.api.ListDashboardPins(sc.reqContext)
		var result dashboardpins.ListDashboardPinsResponse
		err := json.Unmarshal(resp.Body(), &result)
		require.NoError(t, err)
		require.Empty(t, result.Pins)
	})
}

func TestDashboardPinsAreOrgScoped(t *testing.T) {
	testScenario(t, "does not expose pins from another org", map[int64][]string{
		testOrgID:  {testDashUID},
		testOrgID2: {testDashUID},
	}, func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(dashboardpins.CreateDashboardPinCommand{DashboardUID: testDashUID})
		require.Equal(t, http.StatusCreated, sc.api.CreateDashboardPin(sc.reqContext).Status())

		otherOrgUser := user.SignedInUser{
			UserID:  testUserID,
			OrgID:   testOrgID2,
			OrgRole: org.RoleEditor,
		}
		sc.reqContext.SignedInUser = &otherOrgUser

		resp := sc.api.ListDashboardPins(sc.reqContext)
		var result dashboardpins.ListDashboardPinsResponse
		err := json.Unmarshal(resp.Body(), &result)
		require.NoError(t, err)
		require.Empty(t, result.Pins)

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":dashboardUid": testDashUID})
		deleteResp := sc.api.DeleteDashboardPin(sc.reqContext)
		require.Equal(t, http.StatusNotFound, deleteResp.Status())
	})
}

func mockRequestBody(v any) io.ReadCloser {
	b, _ := json.Marshal(v)
	return io.NopCloser(bytes.NewReader(b))
}

func notePtr(value string) *string {
	return &value
}
