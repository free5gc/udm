package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/udm/SDM"
	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/sbi/consumer"
	"github.com/free5gc/udm/pkg/app"
	"github.com/free5gc/udm/pkg/factory"
)

type notifierTestApp struct {
	app.App
	consumer *consumer.Consumer
}

func (a *notifierTestApp) Consumer() *consumer.Consumer { return a.consumer }

func newNotifyTestProcessor(t *testing.T, ctx *udm_context.UDMContext) *Processor {
	t.Helper()
	mockApp := app.NewMockApp(gomock.NewController(t))
	mockApp.EXPECT().Context().Return(ctx).AnyTimes()
	mockApp.EXPECT().Config().Return(&factory.Config{Configuration: &factory.Configuration{}}).AnyTimes()
	a := &notifierTestApp{App: mockApp}
	var err error
	a.consumer, err = consumer.NewConsumer(a)
	require.NoError(t, err)
	p, err := NewProcessor(a)
	require.NoError(t, err)
	return p
}

func TestDataChangeNotifyWithoutOAuth(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(map[bool]string{true: "OAuth enabled", false: "OAuth disabled"}[enabled], func(t *testing.T) {
			const supi = "imsi-001010000000001"
			items := []models.NotifyItem{{ResourceId: "/subscription-data/" + supi + "/am-data"}}
			var tokens, callbacks atomic.Int32
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/oauth2/token" {
					tokens.Add(1)
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				callbacks.Add(1)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.NotContains(t, r.Header, "Authorization")
				assert.Contains(t, []string{"/namf-callback/v1/data-change", "/smf-custom/data-change"}, r.URL.Path)
				var body models.Udm_SDM_ModificationNotification
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, items, body.NotifyItems)
				w.WriteHeader(http.StatusNoContent)
			})
			server := httptest.NewServer(h2c.NewHandler(handler, &http2.Server{}))
			defer server.Close()
			ctx := &udm_context.UDMContext{OAuth2Required: enabled, NfId: uuid.NewString(), NrfUri: server.URL}
			ue := ctx.NewUdmUe(supi)
			ue.UdmSubsToNotify["amf"] = &models.Udr_DR_SubscriptionDataSubscriptions{
				OriginalCallbackReference: server.URL + "/namf-callback/v1/data-change",
				SdmSubscription:           &models.Udm_SDM_SdmSubscription{NfInstanceId: uuid.NewString()},
			}
			// A valid non-AMF callback needs no OAuth targeting identity.
			ue.UdmSubsToNotify["smf"] = &models.Udr_DR_SubscriptionDataSubscriptions{
				OriginalCallbackReference: server.URL + "/smf-custom/data-change",
			}
			p := newNotifyTestProcessor(t, ctx)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/"+supi+"/sdm-subscriptions", nil)
			c.Request.Header.Set("Authorization", "Bearer inbound-token")
			inboundCtx := context.WithValue(c.Request.Context(), openapi.ContextAccessToken, "inbound-token")
			c.Request = c.Request.WithContext(inboundCtx)
			p.DataChangeNotificationProcedure(c, items, supi)
			c.Writer.WriteHeaderNow()
			require.Equal(t, http.StatusNoContent, recorder.Code, recorder.Body.String())
			require.EqualValues(t, 2, callbacks.Load())
			require.Zero(t, tokens.Load())
		})
	}
}

func TestDataChangeNotifyNilSubscription(t *testing.T) {
	ctx := &udm_context.UDMContext{OAuth2Required: true}
	ctx.NewUdmUe("imsi-001010000000001").UdmSubsToNotify["nil"] = nil
	p := newNotifyTestProcessor(t, ctx)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	require.NotPanics(t, func() {
		p.DataChangeNotificationProcedure(c, nil, "imsi-001010000000001")
	})
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Contains(t, recorder.Body.String(), "missing SDM subscription callback identity")
}

func TestDeregistrationNotifyWithoutOAuth(t *testing.T) {
	for _, access := range []models.AccessType{models.AccessType_3_GPP_ACCESS, models.AccessType_NON_3_GPP_ACCESS} {
		t.Run(string(access), func(t *testing.T) {
			var tokens, callbacks atomic.Int32
			data := models.Udm_UECM_DeregistrationData{
				AccessType: access, DeregReason: models.Udm_UECM_DeregistrationReason_UE_INITIAL_REGISTRATION,
			}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/oauth2/token" {
					tokens.Add(1)
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				callbacks.Add(1)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/namf-callback/v1/deregistration/imsi-001010000000001", r.URL.Path)
				assert.NotContains(t, r.Header, "Authorization")
				var body models.Udm_UECM_DeregistrationData
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, data, body)
				w.WriteHeader(http.StatusNoContent)
			})
			server := httptest.NewServer(h2c.NewHandler(handler, &http2.Server{}))
			defer server.Close()
			ctx := &udm_context.UDMContext{OAuth2Required: true, NfId: uuid.NewString(), NrfUri: server.URL}
			p := newNotifyTestProcessor(t, ctx)
			pd := p.SendOnDeregistrationNotification(
				server.URL+"/namf-callback/v1/deregistration/imsi-001010000000001", data)
			require.Nil(t, pd)
			require.EqualValues(t, 1, callbacks.Load())
			require.Zero(t, tokens.Load())
		})
	}
}

func TestNotifyHTTPErrorHandling(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			const supi = "imsi-001010000000001"
			var callbacks atomic.Int32
			server := httptest.NewServer(h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.NotContains(t, r.Header, "Authorization")
				callbacks.Add(1)
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(status)
				assert.NoError(t, json.NewEncoder(w).Encode(models.ProblemDetails{
					Status: int32(status), Cause: "SYSTEM_FAILURE",
				}))
			}), &http2.Server{}))
			defer server.Close()
			ctx := &udm_context.UDMContext{OAuth2Required: true}
			ue := ctx.NewUdmUe(supi)
			for i := range 2 {
				ue.UdmSubsToNotify[fmt.Sprint(i)] = &models.Udr_DR_SubscriptionDataSubscriptions{
					OriginalCallbackReference: server.URL + "/subscriber/data-change",
				}
			}
			p := newNotifyTestProcessor(t, ctx)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			p.DataChangeNotificationProcedure(c, []models.NotifyItem{{ResourceId: "/am-data"}}, supi)
			require.Equal(t, status, recorder.Code)
			require.EqualValues(t, 2, callbacks.Load(), "failure must not stop subsequent notifications")
			pd := p.SendOnDeregistrationNotification(server.URL+"/deregistration/"+supi,
				models.Udm_UECM_DeregistrationData{AccessType: models.AccessType_3_GPP_ACCESS})
			require.NotNil(t, pd)
			require.EqualValues(t, status, pd.Status)
			require.Equal(t, "SYSTEM_FAILURE", pd.Cause)
			require.EqualValues(t, 3, callbacks.Load())
		})
	}
}

func TestNotifyIgnoresCachedOAuthToken(t *testing.T) {
	for _, tt := range []struct {
		name       string
		accessType models.AccessType
	}{
		{name: "data change"},
		{name: "3GPP deregistration", accessType: models.AccessType_3_GPP_ACCESS},
		{name: "non-3GPP deregistration", accessType: models.AccessType_NON_3_GPP_ACCESS},
	} {
		t.Run(tt.name, func(t *testing.T) {
			const supi = "imsi-001010000000001"
			const accessToken = "cached-notify-token"
			var tokenRequests, controls, notifications atomic.Int32
			var tokenEndpointAvailable atomic.Bool
			tokenEndpointAvailable.Store(true)
			server := httptest.NewServer(h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/oauth2/token":
					tokenRequests.Add(1)
					if !tokenEndpointAvailable.Load() {
						w.WriteHeader(http.StatusServiceUnavailable)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{
						"access_token": accessToken, "token_type": "Bearer", "expires_in": 3600,
					}))
				case "/control":
					controls.Add(1)
					assert.Equal(t, "Bearer "+accessToken, r.Header.Get("Authorization"))
					w.WriteHeader(http.StatusNoContent)
				case "/namf-callback/v1/notify":
					notifications.Add(1)
					assert.Equal(t, http.MethodPost, r.Method)
					assert.NotContains(t, r.Header, "Authorization")
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}), &http2.Server{}))
			defer server.Close()
			// Unique IDs isolate this test from the process-wide OAuth token cache.
			targetID := uuid.NewString()
			ctx := &udm_context.UDMContext{OAuth2Required: true, NfId: uuid.NewString(), NrfUri: server.URL}
			p := newNotifyTestProcessor(t, ctx)
			// Use exactly the target and scope requested by the former Notify code.
			serviceName := models.Nrf_NFMgmt_ServiceName("namf-callback")
			_, pd, err := ctx.GetTokenCtxForNFInstance(serviceName, models.Nrf_NFMgmt_NFType_AMF, targetID)
			require.NoError(t, err)
			require.Nil(t, pd)
			require.EqualValues(t, 1, tokenRequests.Load())
			tokenEndpointAvailable.Store(false)
			cachedCtx, pd, err := ctx.GetTokenCtxForNFInstance(serviceName, models.Nrf_NFMgmt_NFType_AMF, targetID)
			require.NoError(t, err)
			require.Nil(t, pd)
			require.EqualValues(t, 1, tokenRequests.Load(), "second acquisition must use the valid cached token")
			// A positive control proves the cached token can still reach the wire.
			_, err = p.Consumer().GetSDMClient("cache-control").SubscriptionCreationApi.SubscribeDatachangeNotification(
				cachedCtx, server.URL+"/control", &SDM.SubscribeDatachangeNotificationRequest{
					RequestBody: &models.Udm_SDM_ModificationNotification{},
				})
			require.NoError(t, err)
			require.EqualValues(t, 1, controls.Load())

			callbackURI := server.URL + "/namf-callback/v1/notify"
			if tt.accessType == "" {
				ue := ctx.NewUdmUe(supi)
				ue.UdmSubsToNotify["cached-target"] = &models.Udr_DR_SubscriptionDataSubscriptions{
					OriginalCallbackReference: callbackURI,
					SdmSubscription:           &models.Udm_SDM_SdmSubscription{NfInstanceId: targetID},
				}
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				// Even an incoming context holding the cached token must not be forwarded.
				c.Request = httptest.NewRequestWithContext(cachedCtx, http.MethodPost, "/"+supi+"/sdm-subscriptions", nil)
				p.DataChangeNotificationProcedure(c, []models.NotifyItem{{ResourceId: "/am-data"}}, supi)
				c.Writer.WriteHeaderNow()
				require.Equal(t, http.StatusNoContent, recorder.Code, recorder.Body.String())
			} else {
				require.Nil(t, p.SendOnDeregistrationNotification(callbackURI, models.Udm_UECM_DeregistrationData{
					AccessType: tt.accessType, DeregReason: models.Udm_UECM_DeregistrationReason_UE_INITIAL_REGISTRATION,
				}))
			}
			require.EqualValues(t, 1, notifications.Load())
			require.EqualValues(t, 1, tokenRequests.Load(), "Notify must not request another token")
		})
	}
}
