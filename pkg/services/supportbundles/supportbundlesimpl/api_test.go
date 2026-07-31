package supportbundlesimpl

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/grafana/pkg/infra/kvstore"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/contexthandler/ctxkey"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/web"
)

func newTestReqContext(method, target, uid string) *contextmodel.ReqContext {
	req := httptest.NewRequest(method, target, nil)
	req = web.SetURLParams(req, map[string]string{":uid": uid})

	ctx := &contextmodel.ReqContext{
		Context: &web.Context{
			Req: req,
		},
		Logger: log.New("test"),
	}
	req = req.WithContext(ctxkey.Set(req.Context(), ctx))
	ctx.Req = req
	return ctx
}

func TestHandleRemove_NotFound(t *testing.T) {
	s := &Service{
		store: newStore(kvstore.NewFakeKVStore()),
	}

	resp := s.handleRemove(newTestReqContext(http.MethodDelete, "/api/support-bundles/missing-uid", "missing-uid"))
	assert.Equal(t, http.StatusNotFound, resp.Status())
}

func TestHandleDownload_NotFound(t *testing.T) {
	s := &Service{
		store: newStore(kvstore.NewFakeKVStore()),
	}

	resp := s.handleDownload(newTestReqContext(http.MethodGet, "/api/support-bundles/missing-uid", "missing-uid"))
	assert.Equal(t, http.StatusNotFound, resp.Status())
}
