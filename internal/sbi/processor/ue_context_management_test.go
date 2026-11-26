package processor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/sbi/consumer"
	mockapp "github.com/free5gc/udm/pkg/mockapp"
)

func TestUpdateAmfAccessPreservesPurgeIntent(t *testing.T) {
	const pei = "imeisv-0123456789012345"
	tests := []struct {
		name      string
		body      string
		udrStatus int
		wantPurge bool
	}{
		{"PEI update without purgeFlag", `{"pei":"` + pei + `"}`, http.StatusNoContent, false},
		{"PEI update with false purgeFlag", `{"pei":"` + pei + `","purgeFlag":false}`, http.StatusNoContent, false},
		{"explicit deregistration", `{"purgeFlag":true}`, http.StatusNoContent, true},
		{"failed deregistration", `{"purgeFlag":true}`, http.StatusInternalServerError, true},
	}

	for _, access := range []string{"amf-3gpp-access", "amf-non-3gpp-access"} {
		t.Run(access, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					p, ue, guami := newAmfUpdateTestProcessor(t)
					defer gock.Off()
					openapi.InterceptInnerHttp2Client(t, false)

					var patches []models.PatchItem
					gock.New(ue.UdrUri).
						Patch("/nudr-dr/v2/subscription-data/" + ue.Supi + "/context-data/" + access).
						AddMatcher(func(r *http.Request, _ *gock.Request) (bool, error) {
							err := json.NewDecoder(r.Body).Decode(&patches)
							return err == nil, err
						}).
						Reply(tt.udrStatus).
						JSON(models.ProblemDetails{Status: int32(tt.udrStatus), Cause: "SYSTEM_FAILURE"})

					recorder := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(recorder)
					if access == "amf-3gpp-access" {
						request := models.Udm_UECM_Amf3GppAccessRegistrationModification{Guami: guami}
						require.NoError(t, json.Unmarshal([]byte(tt.body), &request))
						p.UpdateAmf3gppAccessProcedure(c, request, ue.Supi)
					} else {
						request := models.Udm_UECM_AmfNon3GppAccessRegistrationModification{Guami: guami}
						require.NoError(t, json.Unmarshal([]byte(tt.body), &request))
						p.UpdateAmfNon3gppAccessProcedure(c, request, ue.Supi)
					}
					c.Writer.WriteHeaderNow()
					require.Equal(t, tt.udrStatus, recorder.Code, recorder.Body.String())
					require.True(t, gock.IsDone(), "expected a PATCH to UDR")
					byPath := make(map[string]models.PatchItem)
					for _, item := range patches {
						byPath[item.Path] = item
					}
					if tt.wantPurge {
						require.Equal(t, models.PatchOperation_ADD, byPath["/purgeFlag"].Op)
						require.Equal(t, true, byPath["/purgeFlag"].Value)
					} else {
						require.NotContains(t, byPath, "/purgeFlag")
						require.Equal(t, models.PatchOperation_REPLACE, byPath["/pei"].Op)
						require.Equal(t, pei, byPath["/pei"].Value)
					}
					cleared := tt.wantPurge && tt.udrStatus == http.StatusNoContent
					require.Equal(t, cleared && access == "amf-3gpp-access", ue.Amf3GppAccessRegistration == nil)
					require.Equal(t, cleared && access == "amf-non-3gpp-access", ue.AmfNon3GppAccessRegistration == nil)
				})
			}
		})
	}
}

func newAmfUpdateTestProcessor(t *testing.T) (*Processor, *udm_context.UdmUeContext, *models.Guami) {
	t.Helper()

	guami := &models.Guami{PlmnId: &models.PlmnIdNid{Mcc: "001", Mnc: "01"}, AmfId: "000001"}
	ue := &udm_context.UdmUeContext{
		Supi:                         "imsi-001010000001150",
		UdrUri:                       "http://127.0.0.4:8000",
		Amf3GppAccessRegistration:    &models.Udm_UECM_Amf3GppAccessRegistration{Guami: guami},
		AmfNon3GppAccessRegistration: &models.Udm_UECM_AmfNon3GppAccessRegistration{Guami: guami},
	}
	// The UDR client looks up the cached URI in the global UE pool.
	ctx := udm_context.GetSelf()
	old, exists := ctx.UdmUePool.Load(ue.Supi)
	ctx.UdmUePool.Store(ue.Supi, ue)
	t.Cleanup(func() {
		if exists {
			ctx.UdmUePool.Store(ue.Supi, old)
		} else {
			ctx.UdmUePool.Delete(ue.Supi)
		}
	})
	mockApp := mockapp.NewMockApp(gomock.NewController(t))
	mockApp.EXPECT().Context().Return(ctx).AnyTimes()
	testConsumer, err := consumer.NewConsumer(mockApp)
	require.NoError(t, err)
	mockApp.EXPECT().Consumer().Return(testConsumer).AnyTimes()
	p, err := NewProcessor(mockApp)
	require.NoError(t, err)
	return p, ue, guami
}
