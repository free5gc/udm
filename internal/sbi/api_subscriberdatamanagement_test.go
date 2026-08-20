package sbi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/free5gc/openapi/models"
	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/sbi/consumer"
	"github.com/free5gc/udm/internal/sbi/processor"
	"github.com/free5gc/udm/pkg/app"
	"github.com/free5gc/util/validator"
)

type traceDataTestApp struct {
	app.App
	udmCtx    *udm_context.UDMContext
	consumer  *consumer.Consumer
	processor *processor.Processor
}

func (a *traceDataTestApp) Context() *udm_context.UDMContext {
	return a.udmCtx
}

func (a *traceDataTestApp) Consumer() *consumer.Consumer {
	return a.consumer
}

func (a *traceDataTestApp) Processor() *processor.Processor {
	return a.processor
}

func (a *traceDataTestApp) CancelContext() context.Context {
	return context.Background()
}

func TestOneLayerPathHandlerDoesNotMatchSubstrings(t *testing.T) {
	server := &Server{}
	for _, supi := range []string{"imsi-208930000000001", "imsi-208930000000002", "imsi-208930000000003"} {
		t.Run(supi, func(t *testing.T) {
			target := "/" + supi + "?plmn-id=not-json"
			recorder, c := newSDMTestContext(t, http.MethodGet, target, "")
			c.Params = gin.Params{{Key: "supi", Value: supi}}

			require.NotPanics(t, func() {
				server.OneLayerPathHandlerFunc(c)
			})
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), "OPTIONAL_QUERY_PARAM_INCORRECT")
		})
	}
}

func TestPathPatternMatchesAllTwoLayerRoutes(t *testing.T) {
	s := &Server{}
	routes := s.getTwoLayerRoutes()
	require.NotEmpty(t, routes)

	for _, route := range routes {
		t.Run(route.Name, func(t *testing.T) {
			patternParts := strings.Split(strings.Trim(route.Pattern, "/"), "/")
			pathParts := append([]string(nil), patternParts...)
			staticPartIndex := -1
			for i, part := range pathParts {
				if strings.HasPrefix(part, ":") {
					pathParts[i] = "test-value"
				} else if staticPartIndex == -1 {
					staticPartIndex = i
				}
			}

			path := "/" + strings.Join(pathParts, "/")
			require.True(t, pathPatternMatches(route.Pattern, path),
				"route pattern %q should match %q", route.Pattern, path)

			require.NotEqual(t, -1, staticPartIndex,
				"route pattern %q should contain a fixed path segment", route.Pattern)
			pathParts[staticPartIndex] += "-other"
			mismatchedPath := "/" + strings.Join(pathParts, "/")
			require.False(t, pathPatternMatches(route.Pattern, mismatchedPath),
				"route pattern %q should not match %q", route.Pattern, mismatchedPath)
		})
	}
}

func TestPathPatternMatchesRejectsInvalidPaths(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{name: "substring does not match", pattern: "/:supi/sm-data", path: "/imsi-208930000000001/data", want: false},
		{
			name: "wrong static prefix", pattern: "/group-data/group-identifiers",
			path: "/other/group-identifiers", want: false,
		},
		{
			name: "missing segment", pattern: "/:supi/sm-data",
			path: "/sm-data", want: false,
		},
		{
			name: "extra segment", pattern: "/:supi/sm-data",
			path: "/imsi-208930000000001/sm-data/extra", want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, pathPatternMatches(tt.pattern, tt.path))
		})
	}
}

func TestTwoLayerPathHandlerMatchesPathAndMethod(t *testing.T) {
	server := &Server{}
	tests := []struct {
		name       string
		method     string
		operation  string
		wantStatus int
	}{
		{
			name: "matching path and method", method: http.MethodGet,
			operation: "uc-data", wantStatus: http.StatusNotImplemented,
		},
		{
			name: "substring is not a match", method: http.MethodGet,
			operation: "data", wantStatus: http.StatusNotFound,
		},
		{
			name: "wrong method is not a match", method: http.MethodPost,
			operation: "uc-data", wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder, c := newSDMTestContext(t, tt.method, "/test-supi/"+tt.operation, "")
			c.Params = gin.Params{
				{Key: "supi", Value: "test-supi"},
				{Key: "subscriptionId", Value: tt.operation},
			}

			require.NotPanics(t, func() {
				server.TwoLayerPathHandlerFunc(c)
			})
			require.Equal(t, tt.wantStatus, recorder.Code)
		})
	}
}

func TestGetPlmnIDStruct(t *testing.T) {
	server := &Server{}
	tests := []struct {
		name        string
		query       url.Values
		want        *models.PlmnId
		wantProblem bool
	}{
		{name: "omitted", query: url.Values{}},
		{
			name:  "valid",
			query: url.Values{"plmn-id": {`{"mcc":"208","mnc":"93"}`}},
			want:  &models.PlmnId{Mcc: "208", Mnc: "93"},
		},
		{name: "malformed JSON", query: url.Values{"plmn-id": {`20893`}}, wantProblem: true},
		{name: "invalid MCC", query: url.Values{"plmn-id": {`{"mcc":"AAAA","mnc":"93"}`}}, wantProblem: true},
		{name: "invalid MNC", query: url.Values{"plmn-id": {`{"mcc":"208","mnc":"9"}`}}, wantProblem: true},
		{
			name:        "oversized MCC",
			query:       url.Values{"plmn-id": {`{"mcc":"` + strings.Repeat("1", 10_000) + `","mnc":"93"}`}},
			wantProblem: true,
		},
		{
			name:        "oversized MNC",
			query:       url.Values{"plmn-id": {`{"mcc":"208","mnc":"` + strings.Repeat("1", 10_000) + `"}`}},
			wantProblem: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, problem := server.getPlmnIDStruct(tt.query)
			if tt.wantProblem {
				require.NotNil(t, problem)
				require.Equal(t, int32(http.StatusBadRequest), problem.Status)
				require.Equal(t, "OPTIONAL_QUERY_PARAM_INCORRECT", problem.Cause)
				require.Equal(t, "query plmn-id", problem.InvalidParams[0].Param)
				return
			}
			require.Nil(t, problem)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestHandleGetTraceDataForwardsServingPlmnID(t *testing.T) {
	const supi = "imsi-222771234567890"

	requestPath := make(chan string, 1)
	udrHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath <- r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{}`)); err != nil {
			t.Errorf("write fake UDR response: %v", err)
		}
	})
	udr := httptest.NewServer(h2c.NewHandler(udrHandler, &http2.Server{}))
	t.Cleanup(udr.Close)

	udmCtx := udm_context.GetSelf()
	previousOAuth2Required := udmCtx.OAuth2Required
	udmCtx.OAuth2Required = false
	t.Cleanup(func() {
		udmCtx.OAuth2Required = previousOAuth2Required
	})
	ue := udmCtx.NewUdmUe(supi)
	ue.UdrUri = udr.URL
	t.Cleanup(func() {
		udmCtx.UdmUePool.Delete(supi)
	})

	testApp := &traceDataTestApp{udmCtx: udmCtx}
	var err error
	testApp.consumer, err = consumer.NewConsumer(testApp)
	require.NoError(t, err)
	testApp.processor, err = processor.NewProcessor(testApp)
	require.NoError(t, err)

	server := &Server{ServerUdm: testApp}
	plmnID := url.QueryEscape(`{"mcc":"222","mnc":"77"}`)
	recorder, c := newSDMTestContext(t, http.MethodGet, "/trace-data?plmn-id="+plmnID, "")
	c.Params = gin.Params{{Key: "supi", Value: supi}}
	server.HandleGetTraceData(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t,
		"/nudr-dr/v2/subscription-data/"+supi+"/22277/provisioned-data/trace-data",
		<-requestPath,
	)
}

func TestHandleGetTraceDataRejectsMalformedPlmnID(t *testing.T) {
	recorder, c := newSDMTestContext(t, http.MethodGet, "/trace-data?plmn-id=not-json", "")
	c.Params = gin.Params{{Key: "supi", Value: "imsi-208930000000001"}}

	server := &Server{}
	server.HandleGetTraceData(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, "application/problem+json", recorder.Header().Get("Content-Type"))
	require.Contains(t, recorder.Body.String(), "OPTIONAL_QUERY_PARAM_INCORRECT")
}

func TestDeserializationErrorsAreSanitized(t *testing.T) {
	server := &Server{}
	tests := []struct {
		name    string
		method  string
		path    string
		params  gin.Params
		handler func(*gin.Context)
	}{
		{
			name: "subscribe to shared data", method: http.MethodPost, path: "/shared-data-subscriptions",
			handler: server.HandleSubscribeToSharedData,
		},
		{
			name: "subscribe", method: http.MethodPost, path: "/imsi-208930000000001/sdm-subscriptions",
			params: gin.Params{{Key: "supi", Value: "imsi-208930000000001"}}, handler: server.HandleSubscribe,
		},
		{
			name: "modify", method: http.MethodPatch, path: "/imsi-208930000000001/sdm-subscriptions/1",
			params:  gin.Params{{Key: "ueId", Value: "imsi-208930000000001"}, {Key: "subscriptionId", Value: "1"}},
			handler: server.HandleModify,
		},
		{
			name: "modify shared data", method: http.MethodPatch, path: "/shared-data-subscriptions/1",
			params: gin.Params{{Key: "subscriptionId", Value: "1"}}, handler: server.HandleModifyForSharedData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder, c := newSDMTestContext(t, tt.method, tt.path, `[1,2,3]`)
			c.Params = tt.params
			tt.handler(c)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			var problem models.ProblemDetails
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &problem))
			require.Equal(t,
				"The request body is malformed or does not match the expected schema.", problem.Detail)
			require.Equal(t, "INVALID_MSG_FORMAT", problem.Cause)
			require.Equal(t, "application/problem+json", recorder.Header().Get("Content-Type"))
			require.NotContains(t, recorder.Body.String(), "models.")
		})
	}
}

func newSDMTestContext(t *testing.T, method, target, body string) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req, err := http.NewRequestWithContext(context.Background(), method, target, strings.NewReader(body))
	require.NoError(t, err)
	c.Request = req
	return recorder, c
}

// setupSdmTestRouter registers the six nudm-sdm GET handlers under test
// (plus HandleGetAmData as the reference handler that already validates).
// The handlers are served by a zero-value Server: invalid-supi requests are
// rejected before any Processor call, so no Processor stub is needed.
func setupSdmTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	s := &Server{}
	r := gin.New()
	group := r.Group("/nudm-sdm/v2")
	AddService(group, []Route{
		{"GetSmfSelectData", http.MethodGet, "/:supi/smf-select-data", s.HandleGetSmfSelectData},
		{"GetSupi", http.MethodGet, "/:supi", s.HandleGetSupi},
		{"GetTraceData", http.MethodGet, "/:supi/trace-data", s.HandleGetTraceData},
		{"GetUeContextInSmfData", http.MethodGet, "/:supi/ue-context-in-smf-data", s.HandleGetUeContextInSmfData},
		{"GetNssai", http.MethodGet, "/:supi/nssai", s.HandleGetNssai},
		{"GetSmData", http.MethodGet, "/:supi/sm-data", s.HandleGetSmData},
		{"GetAmData", http.MethodGet, "/:supi/am-data", s.HandleGetAmData},
	})
	return r
}

func TestSdmGetHandlersRejectInvalidSupi(t *testing.T) {
	router := setupSdmTestRouter()

	// CVE-2026-42459 PoC: control characters in supi make UDM build an
	// invalid internal UDR URL and leak it in a 500 response.
	const invalidSupi = "imsi-22277%00INJECTED"

	cases := []struct {
		name string
		path string
	}{
		{"smf-select-data", "/nudm-sdm/v2/" + invalidSupi + "/smf-select-data"},
		{"supi", "/nudm-sdm/v2/" + invalidSupi},
		{"trace-data", "/nudm-sdm/v2/" + invalidSupi + "/trace-data"},
		{"ue-context-in-smf-data", "/nudm-sdm/v2/" + invalidSupi + "/ue-context-in-smf-data"},
		{"nssai", "/nudm-sdm/v2/" + invalidSupi + "/nssai"},
		{"sm-data", "/nudm-sdm/v2/" + invalidSupi + "/sm-data"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request, got %d", w.Code)
			}
			var pd models.ProblemDetails
			if err := json.Unmarshal(w.Body.Bytes(), &pd); err != nil {
				t.Fatalf("unmarshal problem details: %v", err)
			}
			if pd.Detail != "Supi is invalid" {
				t.Fatalf("expected detail %q, got %q", "Supi is invalid", pd.Detail)
			}
		})
	}
}

func TestSdmValidSupiPassesValidation(t *testing.T) {
	// Well-formed SUPIs must pass validator.IsValidSupi so the new guard
	// blocks in the six handlers never reject legitimate requests.
	for _, supi := range []string{
		"imsi-222770000000001",
		"nai-username@example.com",
	} {
		if !validator.IsValidSupi(supi) {
			t.Errorf("expected %q to be a valid SUPI", supi)
		}
	}
}
