package consumer

import (
	"context"
	tlsX509 "crypto/x509"
	tlsErrors "errors"
	tlsIO "io"
	tlsLog "log"
	tlsHTTP "net/http"
	tlsServer "net/http/httptest"
	tlsAtomic "sync/atomic"
	"testing"

	"github.com/h2non/gock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	tlsYAML "gopkg.in/yaml.v2"

	"github.com/free5gc/openapi"
	tlsNFM "github.com/free5gc/openapi/nrf/NFMgmt"
	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/pkg/app"
	tlsFactory "github.com/free5gc/udm/pkg/factory"
)

func TestSendRegisterNFInstance(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	openapi.InterceptInnerHttp2Client(t, false)

	gock.New("http://127.0.0.10:8000").
		Put("/nnrf-nfm/v1/nf-instances/1").
		Reply(200).
		JSON(map[string]string{})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockApp := app.NewMockApp(ctrl)
	consumer, err := NewConsumer(mockApp)
	require.NoError(t, err)

	mockApp.EXPECT().Context().Times(1).Return(
		&udm_context.UDMContext{
			NrfUri: "http://127.0.0.10:8000",
			NfId:   "1",
		},
	)

	_, _, err = consumer.RegisterNFInstance(context.TODO())
	require.NoError(t, err)
}

// Exercise parsed NF config through the same startup configuration API and
// generated NRF client used in production. No deployment files are modified.
func TestSBIConfigTLSVerification(t *testing.T) {
	t.Cleanup(func() {
		if err := openapi.ConfigureHTTP2Clients(
			openapi.TransportReadIdleTimeout, openapi.TransportPingTimeout, false,
		); err != nil {
			t.Errorf("restore HTTP clients: %v", err)
		}
	})
	var requests tlsAtomic.Int32
	server := tlsServer.NewUnstartedServer(tlsHTTP.HandlerFunc(func(w tlsHTTP.ResponseWriter, r *tlsHTTP.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if _, err := tlsIO.WriteString(w, `{}`); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	server.EnableHTTP2 = true
	server.Config.ErrorLog = tlsLog.New(tlsIO.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	for _, value := range []string{"", "false", "true"} {
		t.Run("verify="+value, func(t *testing.T) {
			data := "configuration:\n  sbi:\n    scheme: https\n"
			if value != "" {
				data += "    tls:\n      verify: " + value + "\n"
			}
			var cfg tlsFactory.Config
			if err := tlsYAML.Unmarshal([]byte(data), &cfg); err != nil {
				t.Fatal(err)
			}
			if cfg.GetTLSVerify() != (value == "true") {
				t.Fatal("verify parsing mismatch")
			}
			if err := openapi.ConfigureHTTP2Clients(
				cfg.GetReadIdleTimeout(), cfg.GetPingTimeout(), cfg.GetTLSVerify(),
			); err != nil {
				t.Fatal(err)
			}
			apiCfg := tlsNFM.NewConfiguration()
			apiCfg.SetBasePath(server.URL)
			client := tlsNFM.NewAPIClient(apiCfg)
			req := &tlsNFM.GetNFInstanceRequest{}
			req.SetNfInstanceID("tls-test")
			before := requests.Load()
			_, err := client.NFInstanceIDDocumentApi.GetNFInstance(context.Background(), req)
			if value == "true" {
				var unknown tlsX509.UnknownAuthorityError
				if !tlsErrors.As(err, &unknown) {
					t.Fatalf("expected unknown authority, got %v", err)
				}
				if requests.Load() != before {
					t.Fatal("HTTP handler reached despite rejected certificate")
				}
			} else if err != nil {
				t.Fatal(err)
			} else if requests.Load() != before+1 {
				t.Fatal("request not received")
			}
		})
	}
}
