package recentitems

import (
	"bytes"
	"context"
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
		req := &http.Request{
			Header: http.Header{"Content-Type": []string{"application/json"}},
			URL:    &url.URL{Path: "/api/user/RECENT_ITEMS"},
		}
		ctx := web.Context{Req: req}
		sqlStore := db.InitTestDB(t)
		service := ProvideService(sqlStore)
		service.now = func() time.Time { return time.Unix(1_700_000_000, 0).UTC() }

		usr := user.SignedInUser{
			UserID:  1,
			OrgID:   1,
			OrgRole: org.RoleEditor,
			Login:   "signed_in_user",
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

func itemCommand(resourceType, uid, title string) CreateRecentItemCommand {
	return CreateRecentItemCommand{
		ResourceType: resourceType,
		ResourceUID:  uid,
		Title:        title,
		URL:          "/d/" + uid,
	}
}

func mockRequestBody(v any) io.ReadCloser {
	if raw, ok := v.(string); ok {
		return io.NopCloser(strings.NewReader(raw))
	}
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return io.NopCloser(bytes.NewReader(b))
}

func createItem(t *testing.T, sc scenarioContext, cmd CreateRecentItemCommand) RecentItemDTO {
	t.Helper()
	sc.reqContext.Req.Body = mockRequestBody(cmd)
	resp := sc.service.CreateHandler(sc.reqContext)
	require.Equal(t, http.StatusCreated, resp.Status())

	var dto RecentItemDTO
	require.NoError(t, json.Unmarshal(resp.Body(), &dto))
	require.Equal(t, "/api/user/RECENT_ITEMS/"+dto.UID, responseHeader(resp, "Location"))
	require.NotEmpty(t, dto.UID)
	return dto
}

func responseHeader(resp response.Response, key string) string {
	normal, ok := resp.(*response.NormalResponse)
	if !ok || normal.Header() == nil {
		return ""
	}
	return normal.Header().Get(key)
}

func setQuery(sc scenarioContext, rawQuery string) {
	sc.ctx.Req.URL.RawQuery = rawQuery
	sc.ctx.Req.Form = nil
}

func TestCreateRecentItem(t *testing.T) {
	testScenario(t, "creates an item and returns its location", func(t *testing.T, sc scenarioContext) {
		dto := createItem(t, sc, itemCommand("dashboard", "dashuid", "CPU"))
		require.Equal(t, "dashboard", dto.ResourceType)
		require.Equal(t, "dashuid", dto.ResourceUID)
		require.Equal(t, "CPU", dto.Title)
		require.Equal(t, "/d/dashuid", dto.URL)
		require.Equal(t, sc.service.now().UnixMilli(), dto.LastViewedAt)
	})

	testScenario(t, "refreshes an existing resource without creating another row", func(t *testing.T, sc scenarioContext) {
		start := sc.service.now()
		created := createItem(t, sc, itemCommand("dashboard", "dashuid", "CPU"))

		sc.service.now = func() time.Time { return start.Add(time.Millisecond) }
		cmd := itemCommand("dashboard", "dashuid", "CPU renamed")
		cmd.URL = "/d/dashuid/new"
		sc.reqContext.Req.Body = mockRequestBody(cmd)
		resp := sc.service.CreateHandler(sc.reqContext)
		require.Equal(t, http.StatusOK, resp.Status())
		require.Empty(t, responseHeader(resp, "Location"))

		var refreshed RecentItemDTO
		require.NoError(t, json.Unmarshal(resp.Body(), &refreshed))
		require.Equal(t, created.UID, refreshed.UID)
		require.Equal(t, "CPU renamed", refreshed.Title)
		require.Equal(t, "/d/dashuid/new", refreshed.URL)
		require.Greater(t, refreshed.LastViewedAt, created.LastViewedAt)

		setQuery(sc, "limit=100")
		listResp := sc.service.ListHandler(sc.reqContext)
		var list ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(listResp.Body(), &list))
		require.Len(t, list.RecentItems, 1)
	})

	testScenario(t, "keeps separate items when the same resource uid is used for another type", func(t *testing.T, sc scenarioContext) {
		createItem(t, sc, itemCommand("dashboard", "sameuid", "Dash"))
		createItem(t, sc, itemCommand("folder", "sameuid", "Folder"))

		setQuery(sc, "limit=100")
		listResp := sc.service.ListHandler(sc.reqContext)
		var list ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(listResp.Body(), &list))
		require.Len(t, list.RecentItems, 2)
	})

	testScenario(t, "concurrent first views of the same resource succeed", func(t *testing.T, sc scenarioContext) {
		cmd := itemCommand("dashboard", "dashuid", "CPU")
		errCh := make(chan error, 2)
		for i := 0; i < 2; i++ {
			go func() {
				_, _, err := sc.service.Record(context.Background(), sc.reqContext.SignedInUser, cmd)
				errCh <- err
			}()
		}
		require.NoError(t, <-errCh)
		require.NoError(t, <-errCh)

		setQuery(sc, "limit=100")
		listResp := sc.service.ListHandler(sc.reqContext)
		var list ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(listResp.Body(), &list))
		require.Len(t, list.RecentItems, 1)
	})
}

func TestCreateRecentItemValidation(t *testing.T) {
	cases := []struct {
		name string
		body any
	}{
		{name: "malformed json", body: "{"},
		{name: "unknown field", body: map[string]any{
			"resourceType": "dashboard", "resourceUid": "dashuid", "title": "CPU", "url": "/d/dashuid", "extra": true,
		}},
		{name: "invalid type", body: itemCommand("widget", "dashuid", "CPU")},
		{name: "empty title", body: itemCommand("dashboard", "dashuid", "   ")},
		{name: "external url", body: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "dashuid", Title: "CPU", URL: "https://example.com/d"}},
		{name: "protocol relative url", body: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "dashuid", Title: "CPU", URL: "//example.com/d"}},
		{name: "backslash protocol relative url", body: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "dashuid", Title: "CPU", URL: `/\evil.com`}},
		{name: "double backslash protocol relative url", body: CreateRecentItemCommand{ResourceType: "dashboard", ResourceUID: "dashuid", Title: "CPU", URL: `/\\evil.com`}},
		{name: "title too long", body: itemCommand("dashboard", "dashuid", strings.Repeat("a", 256))},
	}

	for _, tc := range cases {
		testScenario(t, tc.name, func(t *testing.T, sc scenarioContext) {
			sc.reqContext.Req.Body = mockRequestBody(tc.body)
			resp := sc.service.CreateHandler(sc.reqContext)
			require.Equal(t, http.StatusBadRequest, resp.Status())
		})
	}
}

func TestListRecentItems(t *testing.T) {
	testScenario(t, "returns newest first and defaults to 20", func(t *testing.T, sc scenarioContext) {
		start := sc.service.now()
		for i := 0; i < 21; i++ {
			captured := i
			sc.service.now = func() time.Time { return start.Add(time.Duration(captured) * time.Millisecond) }
			createItem(t, sc, itemCommand("dashboard", "res"+strconv.Itoa(captured), "Item "+strconv.Itoa(captured)))
		}

		resp := sc.service.ListHandler(sc.reqContext)
		require.Equal(t, http.StatusOK, resp.Status())
		var list ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(resp.Body(), &list))
		require.Len(t, list.RecentItems, 20)
		require.Equal(t, "res20", list.RecentItems[0].ResourceUID)
		require.Equal(t, "res1", list.RecentItems[19].ResourceUID)
	})

	testScenario(t, "does not change last viewed time", func(t *testing.T, sc scenarioContext) {
		created := createItem(t, sc, itemCommand("explore", "expluid", "Explore"))
		resp := sc.service.ListHandler(sc.reqContext)
		var list ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(resp.Body(), &list))
		require.Equal(t, created.LastViewedAt, list.RecentItems[0].LastViewedAt)
	})

	for _, rawQuery := range []string{"limit=0", "limit=101", "limit=-1", "limit=abc"} {
		testScenario(t, "rejects "+rawQuery, func(t *testing.T, sc scenarioContext) {
			setQuery(sc, rawQuery)
			resp := sc.service.ListHandler(sc.reqContext)
			require.Equal(t, http.StatusBadRequest, resp.Status())
			require.Contains(t, string(resp.Body()), "limit is invalid")
		})
	}
}

func TestRecentItemCap(t *testing.T) {
	testScenario(t, "drops the oldest item past 100 for one user and org", func(t *testing.T, sc scenarioContext) {
		start := sc.service.now()
		for i := 0; i < maxRecentItemsPerUser+1; i++ {
			captured := i
			sc.service.now = func() time.Time { return start.Add(time.Duration(captured) * time.Millisecond) }
			createItem(t, sc, itemCommand("dashboard", "res"+strconv.Itoa(captured), "Item "+strconv.Itoa(captured)))
		}

		setQuery(sc, "limit=100")
		resp := sc.service.ListHandler(sc.reqContext)
		require.Equal(t, http.StatusOK, resp.Status())
		var list ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(resp.Body(), &list))
		require.Len(t, list.RecentItems, maxRecentItemsPerUser)
		require.Equal(t, "res100", list.RecentItems[0].ResourceUID)
		for _, item := range list.RecentItems {
			require.NotEqual(t, "res0", item.ResourceUID)
		}
	})
}

func TestPatchRecentItem(t *testing.T) {
	testScenario(t, "updates title and url without changing last viewed time", func(t *testing.T, sc scenarioContext) {
		created := createItem(t, sc, itemCommand("datasource", "dsuid", "Prometheus"))
		start := sc.service.now()
		sc.service.now = func() time.Time { return start.Add(time.Hour) }

		title := "Prom"
		rawURL := "/connections/datasources/edit/dsuid"
		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.UID})
		sc.reqContext.Req.Body = mockRequestBody(PatchRecentItemCommand{Title: &title, URL: &rawURL})
		resp := sc.service.PatchHandler(sc.reqContext)
		require.Equal(t, http.StatusOK, resp.Status())

		var updated RecentItemDTO
		require.NoError(t, json.Unmarshal(resp.Body(), &updated))
		require.Equal(t, "Prom", updated.Title)
		require.Equal(t, rawURL, updated.URL)
		require.Equal(t, created.LastViewedAt, updated.LastViewedAt)
		require.Equal(t, created.ResourceType, updated.ResourceType)
		require.Equal(t, created.ResourceUID, updated.ResourceUID)
	})

	testScenario(t, "rejects an empty patch", func(t *testing.T, sc scenarioContext) {
		created := createItem(t, sc, itemCommand("folder", "folderuid", "Folder"))
		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.UID})
		sc.reqContext.Req.Body = mockRequestBody(map[string]any{})
		resp := sc.service.PatchHandler(sc.reqContext)
		require.Equal(t, http.StatusBadRequest, resp.Status())
	})

	testScenario(t, "rejects immutable and unknown fields", func(t *testing.T, sc scenarioContext) {
		created := createItem(t, sc, itemCommand("alert_rule", "ruleuid", "Rule"))
		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.UID})
		sc.reqContext.Req.Body = mockRequestBody(map[string]any{"title": "Next", "resourceType": "dashboard"})
		resp := sc.service.PatchHandler(sc.reqContext)
		require.Equal(t, http.StatusBadRequest, resp.Status())
	})

	testScenario(t, "rejects a bad url", func(t *testing.T, sc scenarioContext) {
		created := createItem(t, sc, itemCommand("dashboard", "dashuid", "CPU"))
		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.UID})
		rawURL := "https://example.com"
		sc.reqContext.Req.Body = mockRequestBody(PatchRecentItemCommand{URL: &rawURL})
		resp := sc.service.PatchHandler(sc.reqContext)
		require.Equal(t, http.StatusBadRequest, resp.Status())
	})

	testScenario(t, "returns not found for a missing item", func(t *testing.T, sc scenarioContext) {
		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": "missinguid"})
		title := "Nope"
		sc.reqContext.Req.Body = mockRequestBody(PatchRecentItemCommand{Title: &title})
		resp := sc.service.PatchHandler(sc.reqContext)
		require.Equal(t, http.StatusNotFound, resp.Status())
	})
}

func TestDeleteRecentItem(t *testing.T) {
	testScenario(t, "deletes an item and then returns not found", func(t *testing.T, sc scenarioContext) {
		created := createItem(t, sc, itemCommand("dashboard", "dashuid", "CPU"))
		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": created.UID})

		resp := sc.service.DeleteHandler(sc.reqContext)
		require.Equal(t, http.StatusNoContent, resp.Status())
		require.Empty(t, resp.Body())

		again := sc.service.DeleteHandler(sc.reqContext)
		require.Equal(t, http.StatusNotFound, again.Status())
	})
}

func TestRecentItemsAreScopedToUserAndOrg(t *testing.T) {
	testScenario(t, "hides items from another user and another org", func(t *testing.T, sc scenarioContext) {
		created := createItem(t, sc, itemCommand("dashboard", "dashuid", "CPU"))
		owner := *sc.reqContext.SignedInUser

		otherUser := owner
		otherUser.UserID = 999
		sc.reqContext.SignedInUser = &otherUser
		assertHidden(t, sc, created.UID)

		otherOrg := owner
		otherOrg.OrgID = 2
		sc.reqContext.SignedInUser = &otherOrg
		assertHidden(t, sc, created.UID)

		sc.reqContext.SignedInUser = &owner
		setQuery(sc, "limit=20")
		resp := sc.service.ListHandler(sc.reqContext)
		var list ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(resp.Body(), &list))
		require.Len(t, list.RecentItems, 1)
		require.Equal(t, created.UID, list.RecentItems[0].UID)
	})
}

func assertHidden(t *testing.T, sc scenarioContext, uid string) {
	t.Helper()
	setQuery(sc, "")
	listResp := sc.service.ListHandler(sc.reqContext)
	require.Equal(t, http.StatusOK, listResp.Status())
	var list ListRecentItemsResponse
	require.NoError(t, json.Unmarshal(listResp.Body(), &list))
	require.Empty(t, list.RecentItems)

	title := "Stolen"
	sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": uid})
	sc.reqContext.Req.Body = mockRequestBody(PatchRecentItemCommand{Title: &title})
	require.Equal(t, http.StatusNotFound, sc.service.PatchHandler(sc.reqContext).Status())
	require.Equal(t, http.StatusNotFound, sc.service.DeleteHandler(sc.reqContext).Status())
}

func TestRecentItemsRequireASignedInUser(t *testing.T) {
	testScenario(t, "rejects signed-out and anonymous callers", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Body = mockRequestBody(itemCommand("dashboard", "dashuid", "CPU"))
		sc.reqContext.SignedInUser = nil
		sc.reqContext.IsSignedIn = false
		assertUnauthorized(t, sc)

		anon := user.SignedInUser{UserID: 1, OrgID: 1, IsAnonymous: true}
		sc.reqContext.SignedInUser = &anon
		sc.reqContext.Req.Body = mockRequestBody(itemCommand("dashboard", "dashuid", "CPU"))
		assertUnauthorized(t, sc)
	})
}

func assertUnauthorized(t *testing.T, sc scenarioContext) {
	t.Helper()
	require.Equal(t, http.StatusUnauthorized, sc.service.CreateHandler(sc.reqContext).Status())
	require.Equal(t, http.StatusUnauthorized, sc.service.ListHandler(sc.reqContext).Status())
	sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": "someuid"})
	require.Equal(t, http.StatusUnauthorized, sc.service.PatchHandler(sc.reqContext).Status())
	require.Equal(t, http.StatusUnauthorized, sc.service.DeleteHandler(sc.reqContext).Status())
}

func TestRecentItemStoreFailure(t *testing.T) {
	secret := errors.New("sql exploded: secret")
	testScenario(t, "returns a safe 500 when the store fails", func(t *testing.T, sc scenarioContext) {
		sc.service.store = failingStore{err: secret}

		sc.reqContext.Req.Body = mockRequestBody(itemCommand("dashboard", "dashuid", "CPU"))
		assertSafeFailure(t, sc.service.CreateHandler(sc.reqContext), secret)

		assertSafeFailure(t, sc.service.ListHandler(sc.reqContext), secret)

		sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": "someuid"})
		title := "Next"
		sc.reqContext.Req.Body = mockRequestBody(PatchRecentItemCommand{Title: &title})
		assertSafeFailure(t, sc.service.PatchHandler(sc.reqContext), secret)
		assertSafeFailure(t, sc.service.DeleteHandler(sc.reqContext), secret)
	})
}

func assertSafeFailure(t *testing.T, resp response.Response, secret error) {
	t.Helper()
	require.Equal(t, http.StatusInternalServerError, resp.Status())
	body := string(resp.Body())
	require.NotContains(t, body, secret.Error())
	require.NotContains(t, body, "sql exploded")
}

type failingStore struct {
	err error
}

func (f failingStore) record(context.Context, *user.SignedInUser, CreateRecentItemCommand, int64) (RecentItemDTO, bool, error) {
	return RecentItemDTO{}, false, f.err
}

func (f failingStore) list(context.Context, *user.SignedInUser, int) ([]RecentItemDTO, error) {
	return nil, f.err
}

func (f failingStore) patch(context.Context, *user.SignedInUser, string, PatchRecentItemCommand) (RecentItemDTO, error) {
	return RecentItemDTO{}, f.err
}

func (f failingStore) delete(context.Context, *user.SignedInUser, string) error {
	return f.err
}
