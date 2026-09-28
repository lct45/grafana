package recentitems

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	authlib "github.com/grafana/authlib/types"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/authn"
	"github.com/grafana/grafana/pkg/services/authn/authntest"
	"github.com/grafana/grafana/pkg/services/contexthandler"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tests/testsuite"
	"github.com/grafana/grafana/pkg/web"
)

const (
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

		service := RecentItemsService{
			store: db.InitTestDB(t),
			now:   time.Now,
		}

		usr := user.SignedInUser{
			UserID:  testUserID,
			Login:   "signed_in_user",
			OrgID:   testOrgID,
			OrgRole: org.RoleEditor,
		}

		fn(t, scenarioContext{
			ctx:     &ctx,
			service: &service,
			reqContext: &contextmodel.ReqContext{
				Context:      &ctx,
				SignedInUser: &usr,
			},
		})
	})
}

func validCommand(resourceUID string) CreateRecentItemCommand {
	return CreateRecentItemCommand{
		ResourceType: ResourceTypeDashboard,
		ResourceUID:  resourceUID,
		Title:        "Dashboard " + resourceUID,
		URL:          "/d/" + resourceUID + "/dashboard",
	}
}

func (sc scenarioContext) create(t *testing.T, cmd CreateRecentItemCommand) RecentItemResponse {
	t.Helper()
	sc.reqContext.Req.Body = mockRequestBody(cmd)
	resp := sc.service.createHandler(sc.reqContext)
	require.Equal(t, http.StatusCreated, resp.Status(), string(resp.Body()))
	return decodeItem(t, resp)
}

func (sc scenarioContext) list(t *testing.T) []RecentItemDTO {
	t.Helper()
	resp := sc.service.listHandler(sc.reqContext)
	require.Equal(t, http.StatusOK, resp.Status(), string(resp.Body()))
	var result ListRecentItemsResponse
	require.NoError(t, json.Unmarshal(resp.Body(), &result))
	return result.Items
}

func (sc scenarioContext) withUID(uid string) {
	sc.ctx.Req = web.SetURLParams(sc.ctx.Req, map[string]string{":uid": uid})
}

func TestCreateRecentItem(t *testing.T) {
	testScenario(t, "creates an item and returns 201 with a Location header", func(t *testing.T, sc scenarioContext) {
		fixed := time.Unix(1_700_000_000, 0)
		sc.service.now = func() time.Time { return fixed }

		sc.reqContext.Req.Body = mockRequestBody(validCommand("abc"))
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, http.StatusCreated, resp.Status())

		result := decodeItem(t, resp)
		require.NotEmpty(t, result.Item.UID)
		require.Equal(t, ResourceTypeDashboard, result.Item.ResourceType)
		require.Equal(t, "abc", result.Item.ResourceUID)
		require.Equal(t, "Dashboard abc", result.Item.Title)
		require.Equal(t, "/d/abc/dashboard", result.Item.URL)
		require.Equal(t, fixed.Unix(), result.Item.LastViewedAt)

		normal, ok := resp.(*response.NormalResponse)
		require.True(t, ok)
		rec := httptest.NewRecorder()
		normal.WriteTo(&contextmodel.ReqContext{Context: &web.Context{Resp: web.NewResponseWriter(http.MethodPost, rec)}})
		require.Equal(t, basePath+"/"+result.Item.UID, rec.Header().Get("Location"))
	})

	testScenario(t, "trims whitespace from title, url and resourceUid", func(t *testing.T, sc scenarioContext) {
		cmd := validCommand("abc")
		cmd.ResourceUID = "  abc  "
		cmd.Title = "  Padded  "
		cmd.URL = "  /d/abc  "
		result := sc.create(t, cmd)
		require.Equal(t, "abc", result.Item.ResourceUID)
		require.Equal(t, "Padded", result.Item.Title)
		require.Equal(t, "/d/abc", result.Item.URL)
	})

	testScenario(t, "re-recording the same resource refreshes it and returns 200", func(t *testing.T, sc scenarioContext) {
		start := time.Unix(1_700_000_000, 0)
		sc.service.now = func() time.Time { return start }
		first := sc.create(t, validCommand("abc"))

		sc.service.now = func() time.Time { return start.Add(time.Minute) }
		cmd := validCommand("abc")
		cmd.Title = "Renamed"
		cmd.URL = "/d/abc/renamed"
		sc.reqContext.Req.Body = mockRequestBody(cmd)
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, http.StatusOK, resp.Status())

		second := decodeItem(t, resp)
		require.Equal(t, first.Item.UID, second.Item.UID)
		require.Equal(t, "Renamed", second.Item.Title)
		require.Equal(t, "/d/abc/renamed", second.Item.URL)
		require.Equal(t, start.Add(time.Minute).Unix(), second.Item.LastViewedAt)

		require.Len(t, sc.list(t), 1)
	})

	testScenario(t, "same resourceUid with a different resourceType is a distinct item", func(t *testing.T, sc scenarioContext) {
		sc.create(t, validCommand("abc"))
		cmd := validCommand("abc")
		cmd.ResourceType = ResourceTypeFolder
		sc.create(t, cmd)
		require.Len(t, sc.list(t), 2)
	})
}

func TestCreateRecentItemValidation(t *testing.T) {
	tooLong := func(n int) string { return strings.Repeat("x", n) }

	cases := []struct {
		desc   string
		mutate func(cmd *CreateRecentItemCommand)
		want   error
	}{
		{"rejects unknown resource type", func(c *CreateRecentItemCommand) { c.ResourceType = "widget" }, ErrResourceTypeInvalid},
		{"rejects empty resource type", func(c *CreateRecentItemCommand) { c.ResourceType = "" }, ErrResourceTypeInvalid},
		{"rejects blank resourceUid", func(c *CreateRecentItemCommand) { c.ResourceUID = "  " }, ErrResourceUIDRequired},
		{"rejects overlong resourceUid", func(c *CreateRecentItemCommand) { c.ResourceUID = tooLong(maxResourceUIDLength + 1) }, ErrResourceUIDTooLong},
		{"rejects blank title", func(c *CreateRecentItemCommand) { c.Title = " " }, ErrTitleRequired},
		{"rejects overlong title", func(c *CreateRecentItemCommand) { c.Title = tooLong(maxTitleLength + 1) }, ErrTitleTooLong},
		{"rejects blank url", func(c *CreateRecentItemCommand) { c.URL = "" }, ErrURLRequired},
		{"rejects overlong url", func(c *CreateRecentItemCommand) { c.URL = "/" + tooLong(maxURLLength) }, ErrURLTooLong},
		{"rejects absolute url", func(c *CreateRecentItemCommand) { c.URL = "https://evil.example/d/abc" }, ErrURLNotRelative},
		{"rejects protocol-relative url", func(c *CreateRecentItemCommand) { c.URL = "//evil.example/d/abc" }, ErrURLNotRelative},
		{"rejects backslash-relative url", func(c *CreateRecentItemCommand) { c.URL = `/\evil.example` }, ErrURLNotRelative},
	}

	for _, tc := range cases {
		testScenario(t, tc.desc, func(t *testing.T, sc scenarioContext) {
			cmd := validCommand("abc")
			tc.mutate(&cmd)
			sc.reqContext.Req.Body = mockRequestBody(cmd)
			resp := sc.service.createHandler(sc.reqContext)
			require.Equal(t, http.StatusBadRequest, resp.Status())
			require.Contains(t, string(resp.Body()), tc.want.Error())
			require.Empty(t, sc.list(t))
		})
	}
}

func TestCreateRecentItemBodyDecoding(t *testing.T) {
	cases := []struct {
		desc string
		body io.ReadCloser
	}{
		{"rejects malformed JSON", rawBody(`{"resourceType": "dashboard",`)},
		{"rejects unknown fields", rawBody(`{"resourceType":"dashboard","resourceUid":"abc","title":"t","url":"/d/abc","extra":true}`)},
		{"rejects multiple JSON objects", rawBody(`{"resourceType":"dashboard","resourceUid":"abc","title":"t","url":"/d/abc"}{}`)},
		{"rejects a JSON array", rawBody(`[]`)},
		{"rejects a missing body", nil},
		{"rejects an oversized body", rawBody(`{"title":"` + strings.Repeat("x", maxBodyBytes) + `"}`)},
	}

	for _, tc := range cases {
		testScenario(t, tc.desc, func(t *testing.T, sc scenarioContext) {
			sc.reqContext.Req.Body = tc.body
			resp := sc.service.createHandler(sc.reqContext)
			require.Equal(t, http.StatusBadRequest, resp.Status())
			require.Empty(t, sc.list(t))
		})
	}
}

func TestListRecentItems(t *testing.T) {
	testScenario(t, "orders by most recently viewed", func(t *testing.T, sc scenarioContext) {
		start := time.Unix(1_700_000_000, 0)
		for i, uid := range []string{"first", "second", "third"} {
			sc.service.now = func() time.Time { return start.Add(time.Duration(i) * time.Second) }
			sc.create(t, validCommand(uid))
		}

		// Viewing "first" again should move it to the top.
		sc.service.now = func() time.Time { return start.Add(time.Hour) }
		sc.reqContext.Req.Body = mockRequestBody(validCommand("first"))
		require.Equal(t, http.StatusOK, sc.service.createHandler(sc.reqContext).Status())

		items := sc.list(t)
		require.Len(t, items, 3)
		require.Equal(t, "first", items[0].ResourceUID)
		require.Equal(t, "third", items[1].ResourceUID)
		require.Equal(t, "second", items[2].ResourceUID)
	})

	testScenario(t, "applies the default limit", func(t *testing.T, sc scenarioContext) {
		for i := 0; i < DefaultListLimit+5; i++ {
			sc.create(t, validCommand(fmt.Sprintf("uid-%02d", i)))
		}
		require.Len(t, sc.list(t), DefaultListLimit)
	})

	testScenario(t, "honours an explicit limit and caps it at the stored maximum", func(t *testing.T, sc scenarioContext) {
		for i := 0; i < 5; i++ {
			sc.create(t, validCommand(fmt.Sprintf("uid-%d", i)))
		}

		sc.reqContext.Req.Form.Set("limit", "2")
		require.Len(t, sc.list(t), 2)

		sc.reqContext.Req.Form.Set("limit", fmt.Sprint(MaxStoredItems*10))
		require.Len(t, sc.list(t), 5)
	})

	testScenario(t, "filters by resource type", func(t *testing.T, sc scenarioContext) {
		sc.create(t, validCommand("dash"))
		folderCmd := validCommand("fold")
		folderCmd.ResourceType = ResourceTypeFolder
		sc.create(t, folderCmd)

		sc.reqContext.Req.Form.Set("resourceType", string(ResourceTypeFolder))
		items := sc.list(t)
		require.Len(t, items, 1)
		require.Equal(t, "fold", items[0].ResourceUID)
	})

	testScenario(t, "rejects an unknown resource type filter", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Form.Set("resourceType", "widget")
		resp := sc.service.listHandler(sc.reqContext)
		require.Equal(t, http.StatusBadRequest, resp.Status())
		require.Contains(t, string(resp.Body()), ErrResourceTypeInvalid.Error())
	})

	testScenario(t, "rejects a non-numeric limit", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Form.Set("limit", "ten")
		resp := sc.service.listHandler(sc.reqContext)
		require.Equal(t, http.StatusBadRequest, resp.Status())
	})

	testScenario(t, "rejects a negative limit", func(t *testing.T, sc scenarioContext) {
		sc.reqContext.Req.Form.Set("limit", "-1")
		resp := sc.service.listHandler(sc.reqContext)
		require.Equal(t, http.StatusBadRequest, resp.Status())
		require.Contains(t, string(resp.Body()), ErrListLimitInvalid.Error())
	})

	testScenario(t, "returns an empty array rather than null", func(t *testing.T, sc scenarioContext) {
		resp := sc.service.listHandler(sc.reqContext)
		require.Equal(t, http.StatusOK, resp.Status())
		require.JSONEq(t, `{"items":[]}`, string(resp.Body()))
	})
}

func TestRecentItemsAreCapped(t *testing.T) {
	testScenario(t, "keeps only the most recent MaxStoredItems entries", func(t *testing.T, sc scenarioContext) {
		start := time.Unix(1_700_000_000, 0)
		for i := 0; i < MaxStoredItems+3; i++ {
			sc.service.now = func() time.Time { return start.Add(time.Duration(i) * time.Second) }
			sc.create(t, validCommand(fmt.Sprintf("uid-%03d", i)))
		}

		sc.reqContext.Req.Form.Set("limit", fmt.Sprint(MaxStoredItems))
		items := sc.list(t)
		require.Len(t, items, MaxStoredItems)
		require.Equal(t, fmt.Sprintf("uid-%03d", MaxStoredItems+2), items[0].ResourceUID)
		require.Equal(t, "uid-003", items[MaxStoredItems-1].ResourceUID)

		var stored int64
		err := sc.service.store.WithDbSession(context.Background(), func(sess *db.Session) error {
			var err error
			stored, err = sess.Where("org_id = ? AND user_id = ?", testOrgID, testUserID).Count(RecentItem{})
			return err
		})
		require.NoError(t, err)
		require.EqualValues(t, MaxStoredItems, stored)
	})
}

func TestCreateRecentItemConcurrentWriters(t *testing.T) {
	conflictingRow := func(sess *db.Session) error {
		_, err := sess.Insert(&RecentItem{
			UID: "racer", OrgID: testOrgID, UserID: testUserID,
			ResourceType: string(ResourceTypeDashboard), ResourceUID: "abc",
			Title: "Inserted by a concurrent request", URL: "/d/abc", LastViewedAt: 1,
		})
		return err
	}

	testScenario(t, "refreshes the row inserted by a concurrent writer instead of failing", func(t *testing.T, sc scenarioContext) {
		calls := 0
		sc.service.beforeInsert = func(sess *db.Session) error {
			calls++
			if calls == 1 {
				return conflictingRow(sess)
			}
			return nil
		}

		sc.reqContext.Req.Body = mockRequestBody(validCommand("abc"))
		resp := sc.service.createHandler(sc.reqContext)
		// The first transaction rolls back (taking the simulated concurrent row
		// with it), so the retry inserts cleanly and reports a creation.
		require.Equal(t, http.StatusCreated, resp.Status(), string(resp.Body()))
		require.Equal(t, 2, calls)
		require.Len(t, sc.list(t), 1)
	})

	testScenario(t, "gives up after repeated conflicts", func(t *testing.T, sc scenarioContext) {
		sc.service.beforeInsert = conflictingRow

		sc.reqContext.Req.Body = mockRequestBody(validCommand("abc"))
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, http.StatusInternalServerError, resp.Status())
	})

	testScenario(t, "propagates unexpected insert errors", func(t *testing.T, sc scenarioContext) {
		sc.service.beforeInsert = func(*db.Session) error { return fmt.Errorf("connection lost") }

		sc.reqContext.Req.Body = mockRequestBody(validCommand("abc"))
		resp := sc.service.createHandler(sc.reqContext)
		require.Equal(t, http.StatusInternalServerError, resp.Status())
	})
}

func TestPatchRecentItem(t *testing.T) {
	testScenario(t, "updates title and url", func(t *testing.T, sc scenarioContext) {
		created := sc.create(t, validCommand("abc"))

		sc.withUID(created.Item.UID)
		sc.reqContext.Req.Body = rawBody(`{"title": "  New title ", "url": "/d/abc/new"}`)
		resp := sc.service.patchHandler(sc.reqContext)
		require.Equal(t, http.StatusOK, resp.Status(), string(resp.Body()))

		result := decodeItem(t, resp)
		require.Equal(t, created.Item.UID, result.Item.UID)
		require.Equal(t, "New title", result.Item.Title)
		require.Equal(t, "/d/abc/new", result.Item.URL)
		require.Equal(t, created.Item.LastViewedAt, result.Item.LastViewedAt)

		items := sc.list(t)
		require.Len(t, items, 1)
		require.Equal(t, "New title", items[0].Title)
	})

	testScenario(t, "updates only the title when url is omitted", func(t *testing.T, sc scenarioContext) {
		created := sc.create(t, validCommand("abc"))
		sc.withUID(created.Item.UID)
		sc.reqContext.Req.Body = rawBody(`{"title": "Only title"}`)
		result := decodeItem(t, sc.service.patchHandler(sc.reqContext))
		require.Equal(t, "Only title", result.Item.Title)
		require.Equal(t, created.Item.URL, result.Item.URL)
	})

	testScenario(t, "updates only the url when title is omitted", func(t *testing.T, sc scenarioContext) {
		created := sc.create(t, validCommand("abc"))
		sc.withUID(created.Item.UID)
		sc.reqContext.Req.Body = rawBody(`{"url": "/d/abc/only-url"}`)
		result := decodeItem(t, sc.service.patchHandler(sc.reqContext))
		require.Equal(t, created.Item.Title, result.Item.Title)
		require.Equal(t, "/d/abc/only-url", result.Item.URL)
	})

	testScenario(t, "rejects immutable fields", func(t *testing.T, sc scenarioContext) {
		created := sc.create(t, validCommand("abc"))
		sc.withUID(created.Item.UID)
		for _, body := range []string{
			`{"resourceUid": "other"}`,
			`{"resourceType": "folder"}`,
			`{"uid": "other"}`,
			`{"lastViewedAt": 1}`,
		} {
			sc.reqContext.Req.Body = rawBody(body)
			resp := sc.service.patchHandler(sc.reqContext)
			require.Equal(t, http.StatusBadRequest, resp.Status(), body)
		}
	})

	testScenario(t, "rejects an empty patch", func(t *testing.T, sc scenarioContext) {
		created := sc.create(t, validCommand("abc"))
		sc.withUID(created.Item.UID)
		sc.reqContext.Req.Body = rawBody(`{}`)
		resp := sc.service.patchHandler(sc.reqContext)
		require.Equal(t, http.StatusBadRequest, resp.Status())
		require.Contains(t, string(resp.Body()), ErrPatchNoFields.Error())
	})

	testScenario(t, "validates patched values", func(t *testing.T, sc scenarioContext) {
		created := sc.create(t, validCommand("abc"))
		sc.withUID(created.Item.UID)
		for body, want := range map[string]error{
			`{"title": "  "}`: ErrTitleRequired,
			`{"title": "` + strings.Repeat("x", maxTitleLength+1) + `"}`: ErrTitleTooLong,
			`{"url": ""}`: ErrURLRequired,
			`{"url": "/` + strings.Repeat("x", maxURLLength) + `"}`: ErrURLTooLong,
			`{"url": "https://evil.example"}`:                       ErrURLNotRelative,
		} {
			sc.reqContext.Req.Body = rawBody(body)
			resp := sc.service.patchHandler(sc.reqContext)
			require.Equal(t, http.StatusBadRequest, resp.Status(), body)
			require.Contains(t, string(resp.Body()), want.Error())
		}
	})

	testScenario(t, "returns 404 for a missing item", func(t *testing.T, sc scenarioContext) {
		sc.withUID("missinguid")
		sc.reqContext.Req.Body = rawBody(`{"title": "x"}`)
		resp := sc.service.patchHandler(sc.reqContext)
		require.Equal(t, http.StatusNotFound, resp.Status())
	})

	testScenario(t, "returns 404 for an implausible uid without reading the body", func(t *testing.T, sc scenarioContext) {
		for _, uid := range []string{"", "has space", "bad/slash", strings.Repeat("a", 41)} {
			sc.withUID(uid)
			sc.reqContext.Req.Body = rawBody(`not json`)
			resp := sc.service.patchHandler(sc.reqContext)
			require.Equal(t, http.StatusNotFound, resp.Status(), uid)
		}
	})
}

func TestDeleteRecentItem(t *testing.T) {
	testScenario(t, "deletes an existing item", func(t *testing.T, sc scenarioContext) {
		created := sc.create(t, validCommand("abc"))
		sc.create(t, validCommand("keep"))

		sc.withUID(created.Item.UID)
		resp := sc.service.deleteHandler(sc.reqContext)
		require.Equal(t, http.StatusOK, resp.Status())
		require.JSONEq(t, `{"message":"Recent item deleted"}`, string(resp.Body()))

		items := sc.list(t)
		require.Len(t, items, 1)
		require.Equal(t, "keep", items[0].ResourceUID)

		// Deleting again is a 404, not a silent success.
		require.Equal(t, http.StatusNotFound, sc.service.deleteHandler(sc.reqContext).Status())
	})

	testScenario(t, "returns 404 for an implausible uid", func(t *testing.T, sc scenarioContext) {
		sc.withUID("not valid")
		require.Equal(t, http.StatusNotFound, sc.service.deleteHandler(sc.reqContext).Status())
	})
}

func TestRecentItemsAreScopedToUserAndOrg(t *testing.T) {
	testScenario(t, "another user in the same org cannot see, patch or delete items", func(t *testing.T, sc scenarioContext) {
		created := sc.create(t, validCommand("abc"))

		sc.reqContext.SignedInUser = &user.SignedInUser{UserID: 999, OrgID: testOrgID, OrgRole: org.RoleAdmin}
		require.Empty(t, sc.list(t))

		sc.withUID(created.Item.UID)
		sc.reqContext.Req.Body = rawBody(`{"title": "hijacked"}`)
		require.Equal(t, http.StatusNotFound, sc.service.patchHandler(sc.reqContext).Status())
		require.Equal(t, http.StatusNotFound, sc.service.deleteHandler(sc.reqContext).Status())

		// Recording the same resource as the other user creates a separate item.
		sc.create(t, validCommand("abc"))
		require.Len(t, sc.list(t), 1)

		sc.reqContext.SignedInUser = &user.SignedInUser{UserID: testUserID, OrgID: testOrgID, OrgRole: org.RoleEditor}
		items := sc.list(t)
		require.Len(t, items, 1)
		require.Equal(t, "Dashboard abc", items[0].Title)
	})

	testScenario(t, "the same user in another org has a separate list", func(t *testing.T, sc scenarioContext) {
		created := sc.create(t, validCommand("abc"))

		sc.reqContext.SignedInUser = &user.SignedInUser{UserID: testUserID, OrgID: 2, OrgRole: org.RoleEditor}
		require.Empty(t, sc.list(t))

		sc.withUID(created.Item.UID)
		require.Equal(t, http.StatusNotFound, sc.service.deleteHandler(sc.reqContext).Status())
	})
}

// TestRecentItemsHTTP drives the registered routes through the real router
// and auth middleware, covering route registration and anonymous rejection.
func TestRecentItemsHTTP(t *testing.T) {
	signedIn := &authn.Identity{ID: "1", Type: authlib.TypeUser, OrgID: testOrgID, OrgRoles: map[int64]org.RoleType{testOrgID: org.RoleViewer}}
	anonymous := &authn.Identity{Type: authlib.TypeAnonymous, OrgID: testOrgID}

	newServer := func(t *testing.T, identity *authn.Identity, authErr error) *web.Mux {
		routeRegister := routing.NewRouteRegister()
		ProvideService(db.InitTestDB(t), routeRegister)

		ctxHandler := contexthandler.ProvideService(setting.NewCfg(), &authntest.FakeService{
			ExpectedIdentity: identity,
			ExpectedErr:      authErr,
		}, featuremgmt.WithFeatures())

		mux := web.New()
		mux.Use(ctxHandler.Middleware)
		routeRegister.Register(mux)
		return mux
	}

	do := func(t *testing.T, mux *web.Mux, method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		req := httptest.NewRequest(method, path, reader)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	t.Run("full lifecycle for a signed-in user", func(t *testing.T) {
		mux := newServer(t, signedIn, nil)

		rec := do(t, mux, http.MethodPost, basePath, `{"resourceType":"explore","resourceUid":"sess-1","title":"Explore","url":"/explore?x=1"}`)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var created RecentItemResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
		require.Equal(t, basePath+"/"+created.Item.UID, rec.Header().Get("Location"))

		rec = do(t, mux, http.MethodPost, basePath, `{"resourceType":"explore","resourceUid":"sess-1","title":"Explore again","url":"/explore?x=2"}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		rec = do(t, mux, http.MethodGet, basePath+"?resourceType=explore&limit=10", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var listed ListRecentItemsResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
		require.Len(t, listed.Items, 1)
		require.Equal(t, "Explore again", listed.Items[0].Title)

		rec = do(t, mux, http.MethodPatch, basePath+"/"+created.Item.UID, `{"title":"Patched"}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		rec = do(t, mux, http.MethodDelete, basePath+"/"+created.Item.UID, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		rec = do(t, mux, http.MethodGet, basePath, "")
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"items":[]}`, rec.Body.String())
	})

	t.Run("anonymous users are rejected on every route", func(t *testing.T) {
		mux := newServer(t, anonymous, nil)
		for _, tc := range []struct{ method, path string }{
			{http.MethodPost, basePath},
			{http.MethodGet, basePath},
			{http.MethodPatch, basePath + "/abc"},
			{http.MethodDelete, basePath + "/abc"},
		} {
			rec := do(t, mux, tc.method, tc.path, `{}`)
			require.Equal(t, http.StatusUnauthorized, rec.Code, tc.method+" "+tc.path)
		}
	})

	t.Run("unauthenticated requests are rejected", func(t *testing.T) {
		mux := newServer(t, nil, fmt.Errorf("no session"))
		rec := do(t, mux, http.MethodGet, basePath, "")
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func mockRequestBody(v any) io.ReadCloser {
	b, _ := json.Marshal(v)
	return io.NopCloser(bytes.NewReader(b))
}

func rawBody(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

func decodeItem(t *testing.T, resp response.Response) RecentItemResponse {
	t.Helper()
	var result RecentItemResponse
	require.NoError(t, json.Unmarshal(resp.Body(), &result))
	return result
}
