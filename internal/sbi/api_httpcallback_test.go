package sbi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/sbi/processor"
)

func TestNotifyOAuthRouteBoundary(t *testing.T) {
	const supi = "imsi-001010000000001"
	const notification = `{"notifyItems":[{"resourceId":"/subscription-data/` + supi + `/am-data"}]}`
	udmCtx := &udm_context.UDMContext{
		OAuth2Required:  true,
		NfId:            "11111111-1111-4111-8111-111111111111",
		NrfNfInstanceID: "22222222-2222-4222-8222-222222222222",
	}
	testApp := &traceDataTestApp{udmCtx: udmCtx}
	var err error
	testApp.processor, err = processor.NewProcessor(testApp)
	require.NoError(t, err)
	router := newRouter(&Server{ServerUdm: testApp})

	for _, tt := range []struct {
		name, method, path, body, authorization string
		wantStatus                              int
	}{
		{"notify without token", http.MethodPost, "/" + supi + "/sdm-subscriptions", notification, "", 204},
		{
			"notify ignores invalid token", http.MethodPost, "/" + supi + "/sdm-subscriptions",
			notification, "Bearer invalid", 204,
		},
		{"notify validates JSON", http.MethodPost, "/" + supi + "/sdm-subscriptions", "{", "", 400},
		{"notify requires items", http.MethodPost, "/" + supi + "/sdm-subscriptions", `{}`, "", 400},
		{"notify validates SUPI", http.MethodPost, "/invalid/sdm-subscriptions", notification, "", 400},
		{"subscribe protected", http.MethodPost, "/nudm-sdm/v2/" + supi + "/sdm-subscriptions", `{}`, "", 401},
		{"unsubscribe protected", http.MethodDelete, "/nudm-sdm/v2/" + supi + "/sdm-subscriptions/subscription", "", "", 401},
		{"SDM GET protected", http.MethodGet, "/nudm-sdm/v2/" + supi + "/am-data", "", "", 401},
		{"shared subscribe protected", http.MethodPost, "/nudm-sdm/v2/shared-data-subscriptions", `{}`, "", 401},
		{
			"UECM registration protected", http.MethodPut,
			"/nudm-uecm/v1/" + supi + "/registrations/amf-3gpp-access", `{}`, "", 401,
		},
		{"wrong method", http.MethodGet, "/" + supi + "/sdm-subscriptions", "", "", 404},
		{"wrong prefix", http.MethodPost, "/other/" + supi + "/sdm-subscriptions", notification, "", 404},
		{"wrong suffix", http.MethodPost, "/" + supi + "/sdm-subscriptions-other", notification, "", 404},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			require.Equal(t, tt.wantStatus, recorder.Code, recorder.Body.String())
		})
	}
}
