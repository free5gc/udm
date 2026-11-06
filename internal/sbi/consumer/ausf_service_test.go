package consumer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/free5gc/openapi/models"
	udm_context "github.com/free5gc/udm/internal/context"
)

// These are component tests over real HTTP/2, using simulated NRF/AUSF peers.
// Actual NRF policy enforcement and multi-AUSF cleanup triggers need integration
// testing; no NF-type claim is invented in the producer's access token.
func TestSendUEAuthenticationDeregister(t *testing.T) {
	for _, tt := range []struct {
		name                               string
		tokenStatus, ausfStatus            int
		wrongInstance, wrongType, disabled bool
		wantError                          bool
		wantCalls                          int
	}{
		{name: "authenticated deregistration", ausfStatus: 204, wantCalls: 1},
		{name: "OAuth disabled deployment", disabled: true, ausfStatus: 204, wantCalls: 1},
		{name: "token request rejected", tokenStatus: 400, wantError: true},
		{name: "AUSF authorization error propagated", ausfStatus: 401, wantError: true, wantCalls: 1},
		{name: "context not found", ausfStatus: 404, wantError: true, wantCalls: 1},
		{name: "never select another AUSF", wrongInstance: true, wantError: true},
		{name: "never select another NF type", wrongType: true, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			udmID, ausfID, nrfID := uuid.NewString(), uuid.NewString(), uuid.NewString()
			const supi = "imsi-001010000000001"
			var mu sync.Mutex
			var tokenForms []url.Values
			var discovery url.Values
			var authorization string
			var info models.Ausf_UEAU_DeregistrationInfo
			calls := 0
			var server *httptest.Server
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/oauth2/token":
					if r.Method != http.MethodPost {
						t.Errorf("token method: %s", r.Method)
					}
					if err := r.ParseForm(); err != nil {
						t.Errorf("token form: %v", err)
					}
					tokenForms = append(tokenForms, r.PostForm)
					// Discovery has a separate scope and target; only reject the
					// token acquisition for the AUSF service in the error case.
					if r.PostForm.Get("scope") == "nausf-auth" && tt.tokenStatus != 0 {
						w.WriteHeader(tt.tokenStatus)
						_, _ = w.Write([]byte(`{"error":"unauthorized_client"}`))
						return
					}
					_, _ = w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
				case "/nnrf-disc/v1/nf-instances":
					if r.Method != http.MethodGet {
						t.Errorf("discovery method: %s", r.Method)
					}
					discovery = r.URL.Query()
					if !tt.disabled && r.Header.Get("Authorization") != "Bearer test-token" {
						t.Error("discovery token missing")
					}
					profileID := ausfID
					if tt.wrongInstance {
						profileID = uuid.NewString()
					}
					nfType := models.Nrf_NFMgmt_NFType_AUSF
					if tt.wrongType {
						nfType = models.Nrf_NFMgmt_NFType_SMF
					}
					err := json.NewEncoder(w).Encode(models.Nrf_NFDisc_SearchResult{
						NfInstances: []models.Nrf_NFDisc_NFProfile{{
							NfInstanceId: profileID, NfType: nfType,
							NfServices: []models.Nrf_NFDisc_NFService{{
								ServiceName:     models.Nrf_NFMgmt_ServiceName_NAUSF_AUTH,
								NfServiceStatus: models.Nrf_NFMgmt_NFServiceStatus_REGISTERED,
								ApiPrefix:       server.URL,
							}},
						}},
					})
					if err != nil {
						t.Errorf("discovery response: %v", err)
					}
				case "/nausf-auth/v1/ue-authentications/deregister":
					calls++
					if r.Method != http.MethodPost {
						t.Errorf("deregister method: %s", r.Method)
					}
					if r.Header.Get("Content-Type") != "application/json" {
						t.Error("deregister content type")
					}
					authorization = r.Header.Get("Authorization")
					if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
						t.Errorf("deregistration body: %v", err)
					}
					if tt.ausfStatus == 404 {
						w.Header().Set("Content-Type", "application/problem+json")
						w.WriteHeader(404)
						_, _ = w.Write([]byte(`{"status":404,"cause":"CONTEXT_NOT_FOUND"}`))
						return
					}
					w.WriteHeader(tt.ausfStatus)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
				}
			})
			server = httptest.NewServer(h2c.NewHandler(handler, &http2.Server{}))
			t.Cleanup(server.Close)
			c := newTestConsumer(t, &udm_context.UDMContext{
				NfId: udmID, NrfUri: server.URL, NrfNfInstanceID: nrfID, OAuth2Required: !tt.disabled,
			})
			err := c.SendUEAuthenticationDeregister(supi, ausfID)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			mu.Lock()
			defer mu.Unlock()
			require.Equal(t, tt.wantCalls, calls)
			require.Equal(t, "UDM", discovery.Get("requester-nf-type"))
			require.Equal(t, "AUSF", discovery.Get("target-nf-type"))
			require.Equal(t, ausfID, discovery.Get("target-nf-instance-id"))
			require.Equal(t, "nausf-auth", discovery.Get("service-names"))
			if tt.wantCalls > 0 {
				require.Equal(t, supi, info.Supi)
				wantAuthorization := "Bearer test-token"
				if tt.disabled {
					wantAuthorization = ""
				}
				require.Equal(t, wantAuthorization, authorization)
			}
			if tt.disabled {
				require.Empty(t, tokenForms)
			} else {
				wantTokens := 2
				if tt.wrongInstance || tt.wrongType {
					wantTokens = 1
				}
				require.Len(t, tokenForms, wantTokens)
				for i, form := range tokenForms {
					require.Equal(t, "client_credentials", form.Get("grant_type"))
					require.Equal(t, "UDM", form.Get("nfType"))
					require.Equal(t, udmID, form.Get("nfInstanceId"))
					scope, target, targetID := "nnrf-disc", "NRF", nrfID
					if i == 1 {
						scope, target, targetID = "nausf-auth", "AUSF", ausfID
					}
					require.Equal(t, scope, form.Get("scope"))
					require.Equal(t, target, form.Get("targetNfType"))
					require.Equal(t, targetID, form.Get("targetNfInstanceId"))
				}
			}
		})
	}
}

func TestSendUEAuthenticationDeregisterRejectsInvalidInput(t *testing.T) {
	c := newTestConsumer(t, &udm_context.UDMContext{})
	for _, input := range []struct{ supi, target string }{
		{"", uuid.NewString()}, {"suci-0-001-01-0000-0-0-0000000001", uuid.NewString()},
		{"imsi-001010000000001", ""}, {"imsi-001010000000001", "not-a-uuid"},
		{"imsi-001010000000001", uuid.NewString()}, // No NRF URI: fail before making an HTTP request.
	} {
		require.Error(t, c.SendUEAuthenticationDeregister(input.supi, input.target))
	}
}
