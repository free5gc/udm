package processor

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
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
	"github.com/free5gc/util/milenage"
)

// TS 35.208 test set 1 subscriber keys
const (
	hssAvTestSupi = "imsi-208930000000003"
	hssAvTestK    = "465b5ce8b199b49faa5f0a2ee238a6bc"
	hssAvTestOpc  = "cd63cb71954a9f4e48a5994e37a02baf"
	hssAvUdrPath  = "/nudr-dr/v2/subscription-data/" + hssAvTestSupi +
		"/authentication-data/authentication-subscription"
)

func hssAvTestProcessor(t *testing.T) *Processor {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockApp := mockapp.NewMockApp(ctrl)
	testConsumer, err := consumer.NewConsumer(mockApp)
	require.NoError(t, err)
	testProcessor, err := NewProcessor(mockApp)
	require.NoError(t, err)
	udm_context.GetSelf().NrfUri = "http://127.0.0.10:8000"
	ue := new(udm_context.UdmUeContext)
	ue.Init()
	ue.Supi = hssAvTestSupi
	ue.UdrUri = "http://127.0.0.4:8000"
	udm_context.GetSelf().UdmUePool.Store(hssAvTestSupi, ue)
	mockApp.EXPECT().Consumer().Return(testConsumer).AnyTimes()
	mockApp.EXPECT().Context().Return(&udm_context.UDMContext{
		OAuth2Required: false,
		NrfUri:         "http://127.0.0.10:8000",
		NfId:           "1",
	}).AnyTimes()
	return testProcessor
}

func hssAvMockUdr(sqn, storedSqn string) {
	gock.New("http://127.0.0.4:8000").
		Get(hssAvUdrPath).
		Reply(200).
		AddHeader("Content-Type", "application/json").
		JSON(models.Udr_DR_AuthenticationSubscription{
			AuthenticationMethod:          models.Udr_DR_AuthMethod_5_G_AKA,
			EncPermanentKey:               hssAvTestK,
			EncOpcKey:                     hssAvTestOpc,
			SequenceNumber:                &models.Udr_DR_SequenceNumber{Sqn: sqn},
			AuthenticationManagementField: "8000",
		})
	gock.New("http://127.0.0.4:8000").
		Patch(hssAvUdrPath).
		AddMatcher(func(req *http.Request, _ *gock.Request) (bool, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return false, err
			}
			req.Body = io.NopCloser(bytes.NewReader(body))
			var patch []struct {
				Op    string                       `json:"op"`
				Path  string                       `json:"path"`
				Value models.Udr_DR_SequenceNumber `json:"value"`
			}
			if err = json.Unmarshal(body, &patch); err != nil {
				return false, nil
			}
			return len(patch) == 1 && patch[0].Op == "replace" &&
				patch[0].Path == "/sequenceNumber" && patch[0].Value.Sqn == storedSqn, nil
		}).
		Reply(204)
}

func hssAvCall(t *testing.T, p *Processor, authType string,
	req models.Udm_UEAU_HssAuthenticationInfoRequest,
) (int, []byte) {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	p.GenerateHssAvProcedure(c, req, hssAvTestSupi, authType)
	resp := rec.Result()
	defer func() { require.NoError(t, resp.Body.Close()) }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, body
}

func TestGenerateHssAvProcedure_ImsAka(t *testing.T) {
	defer gock.Off()
	openapi.InterceptInnerHttp2Client(t, false)
	p := hssAvTestProcessor(t)
	// two vectors from SQN 0x23: SQN 0x23 and 0x24 are used, 0x25 is stored
	hssAvMockUdr("000000000023", "000000000025")

	status, body := hssAvCall(t, p, string(models.Udm_UEAU_HssAuthTypeInUri_IMS_AKA),
		models.Udm_UEAU_HssAuthenticationInfoRequest{
			HssAuthType:           models.Udm_UEAU_HssAuthType_IMS_AKA,
			NumOfRequestedVectors: 2,
		})
	require.Equal(t, http.StatusOK, status, string(body))
	require.True(t, gock.IsDone(), "UDR SQN update not sent as expected")

	var res struct {
		HssAuthenticationVectors []models.Udm_UEAU_AvImsGbaEapAka `json:"hssAuthenticationVectors"`
	}
	require.NoError(t, json.Unmarshal(body, &res))
	require.Len(t, res.HssAuthenticationVectors, 2)

	k, _ := hex.DecodeString(hssAvTestK)
	opc, _ := hex.DecodeString(hssAvTestOpc)
	for i, av := range res.HssAuthenticationVectors {
		require.Equal(t, models.Udm_UEAU_HssAvType_IMS_AKA, av.AvType)
		rnd, err := hex.DecodeString(av.Rand)
		require.NoError(t, err)
		autn, err := hex.DecodeString(av.Autn)
		require.NoError(t, err)
		// What the USIM does: check MAC-A and recover SQN, then derive keys.
		sqn, _, ik, ck, res, err := milenage.GenerateKeysWithAUTN(opc, k, rnd, autn)
		require.NoError(t, err, "vector %d: AUTN does not verify", i)
		require.Equal(t, []byte{0, 0, 0, 0, 0, byte(0x23 + i)}, sqn, "vector %d SQN", i)
		require.Equal(t, hex.EncodeToString(res), av.Xres)
		require.Equal(t, hex.EncodeToString(ck), av.Ck)
		require.Equal(t, hex.EncodeToString(ik), av.Ik)
		// AMF separation bit cleared for UMTS AVs (TS 33.102 Annex H)
		require.True(t, bytes.Equal(autn[6:8], []byte{0x00, 0x00}), "vector %d AMF %x", i, autn[6:8])
	}
	require.NotEqual(t, res.HssAuthenticationVectors[0].Rand, res.HssAuthenticationVectors[1].Rand)
}

func TestGenerateHssAvProcedure_UnsupportedAuthType(t *testing.T) {
	p := hssAvTestProcessor(t)
	status, _ := hssAvCall(t, p, string(models.Udm_UEAU_HssAuthTypeInUri_EAP_AKA),
		models.Udm_UEAU_HssAuthenticationInfoRequest{NumOfRequestedVectors: 1})
	require.Equal(t, http.StatusNotImplemented, status)
}

func TestGenerateHssAvProcedure_Resynchronisation(t *testing.T) {
	defer gock.Off()
	openapi.InterceptInnerHttp2Client(t, false)
	p := hssAvTestProcessor(t)

	k, _ := hex.DecodeString(hssAvTestK)
	opc, _ := hex.DecodeString(hssAvTestOpc)
	rnd, _ := hex.DecodeString("23553cbe9637a89d218ae64dae47bf35")
	sqnMS := []byte{0, 0, 0, 0, 0x01, 0x00} // the USIM is at 0x100
	auts, err := milenage.GenerateAUTS(opc, k, rnd, sqnMS)
	require.NoError(t, err)
	// network SQN restarts at SQN_MS + IND + 1 = 0x121; one vector, 0x122 stored
	hssAvMockUdr("000000000023", "000000000122")

	status, body := hssAvCall(t, p, string(models.Udm_UEAU_HssAuthTypeInUri_IMS_AKA),
		models.Udm_UEAU_HssAuthenticationInfoRequest{
			NumOfRequestedVectors: 1,
			ResynchronizationInfo: &models.Udm_UEAU_ResynchronizationInfo{
				Rand: hex.EncodeToString(rnd), Auts: hex.EncodeToString(auts),
			},
		})
	require.Equal(t, http.StatusOK, status, string(body))
	require.True(t, gock.IsDone())

	var res struct {
		HssAuthenticationVectors []models.Udm_UEAU_AvImsGbaEapAka `json:"hssAuthenticationVectors"`
	}
	require.NoError(t, json.Unmarshal(body, &res))
	require.Len(t, res.HssAuthenticationVectors, 1)
	av := res.HssAuthenticationVectors[0]
	r, _ := hex.DecodeString(av.Rand)
	autn, _ := hex.DecodeString(av.Autn)
	sqn, _, _, _, _, err := milenage.GenerateKeysWithAUTN(opc, k, r, autn)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 0, 0, 0x01, 0x21}, sqn)
}

func TestGenerateHssAvProcedure_ResynchronisationBadMac(t *testing.T) {
	defer gock.Off()
	openapi.InterceptInnerHttp2Client(t, false)
	p := hssAvTestProcessor(t)
	gock.New("http://127.0.0.4:8000").
		Get(hssAvUdrPath).
		Reply(200).
		AddHeader("Content-Type", "application/json").
		JSON(models.Udr_DR_AuthenticationSubscription{
			EncPermanentKey:               hssAvTestK,
			EncOpcKey:                     hssAvTestOpc,
			SequenceNumber:                &models.Udr_DR_SequenceNumber{Sqn: "000000000023"},
			AuthenticationManagementField: "8000",
		})

	status, _ := hssAvCall(t, p, string(models.Udm_UEAU_HssAuthTypeInUri_IMS_AKA),
		models.Udm_UEAU_HssAuthenticationInfoRequest{
			NumOfRequestedVectors: 1,
			ResynchronizationInfo: &models.Udm_UEAU_ResynchronizationInfo{
				Rand: "23553cbe9637a89d218ae64dae47bf35",
				Auts: "000000000000000000000000000000"[:28],
			},
		})
	require.Equal(t, http.StatusForbidden, status)
	require.False(t, gock.HasUnmatchedRequest(), "no SQN update may be sent")
}
