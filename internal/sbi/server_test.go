package sbi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/sbi/consumer"
	"github.com/free5gc/udm/internal/sbi/processor"
	"github.com/free5gc/udm/pkg/factory"
)

type testServerUdm struct {
	ctx *udm_context.UDMContext
}

func (t *testServerUdm) SetLogEnable(enable bool) {}

func (t *testServerUdm) SetLogLevel(level string) {}

func (t *testServerUdm) SetReportCaller(reportCaller bool) {}

func (t *testServerUdm) Start() {}

func (t *testServerUdm) Terminate() {}

func (t *testServerUdm) Context() *udm_context.UDMContext {
	return t.ctx
}

func (t *testServerUdm) Config() *factory.Config {
	return nil
}

func (t *testServerUdm) Consumer() *consumer.Consumer {
	return nil
}

func (t *testServerUdm) Processor() *processor.Processor {
	return nil
}

func (t *testServerUdm) CancelContext() context.Context {
	return context.Background()
}

func TestNewRouterKeepsIndexPublic(t *testing.T) {
	t.Parallel()

	server := &Server{
		ServerUdm: &testServerUdm{
			ctx: &udm_context.UDMContext{OAuth2Required: true},
		},
	}
	router := newRouter(server)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", resp.Code, http.StatusOK)
	}
}

func TestNewRouterRejectsUnauthenticatedCallback(t *testing.T) {
	t.Parallel()

	server := &Server{
		ServerUdm: &testServerUdm{
			ctx: &udm_context.UDMContext{OAuth2Required: true},
		},
	}
	router := newRouter(server)

	body := `{"notifyItems":[{"resourceId":"/nudr-dr/v1/subscription-data/imsi-208930000000003/provisioned-data/am-data","changes":[{"op":"REPLACE","path":"/subscribedUeAmbr"}]}]}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/imsi-208930000000003/sdm-subscriptions",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: got %d want %d", resp.Code, http.StatusUnauthorized)
	}
}
