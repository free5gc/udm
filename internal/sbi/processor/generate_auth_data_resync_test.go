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

// seqInd builds a 48-bit SQN = SEQ || IND with a 5-bit IND.
func seqInd(seq, ind uint64) string {
	return hex.EncodeToString([]byte{
		byte(seq >> 35), byte(seq >> 27), byte(seq >> 19), byte(seq >> 11), byte(seq >> 3),
		byte(seq<<5) | byte(ind),
	})
}

func TestSqnInRange(t *testing.T) {
	testCases := []struct {
		name  string
		sqnHe string
		sqnMs string
		exp   bool
	}{
		{"SQN_HE above SQN_MS", seqInd(129, 0), seqInd(128, 0), true},
		{"same SEQ, higher IND", seqInd(128, 1), seqInd(128, 0), false},
		{"next SEQ, lower IND", seqInd(129, 0), seqInd(128, 31), true},
		{"SQN_HE equal to SQN_MS", seqInd(128, 0), seqInd(128, 0), false},
		{"SQN_HE below SQN_MS", seqInd(127, 31), seqInd(128, 0), false},
		{"SEQ exactly Delta ahead", seqInd(128+1<<28, 31), seqInd(128, 0), true},
		{"SEQ more than Delta ahead", seqInd(128+1<<28+1, 0), seqInd(128, 31), false},
		{"SQN_HE wrapped past 2^48", "000000000010", "ffffffffffe0", false},
		{"SQN_HE at 2^48-1", "ffffffffffff", "000000000000", false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			he, err := hex.DecodeString(tc.sqnHe)
			require.NoError(t, err)
			ms, err := hex.DecodeString(tc.sqnMs)
			require.NoError(t, err)
			require.Equal(t, tc.exp, sqnInRange(he, ms))
		})
	}
}

// TestGenerateAuthDataResync runs a resynchronisation with a valid AUTS
// against a stored SQN and checks the SQN of the returned vector and the SQN
// written back to the UDR.
func TestGenerateAuthDataResync(t *testing.T) {
	const (
		supi = "imsi-208930000000001"
		k    = "8baf473f2f8fd09487cccbd7097c6862"
		opc  = "8e27b6af0e692e750f32667a3b14605d"
		rand = "0123456789abcdef0123456789abcdef"
	)
	testCases := []struct {
		name      string
		storedSqn string
		sqnMs     string
		expSqn    string // SQN of the returned vector
	}{
		{"SQN_HE behind SQN_MS resets", seqInd(2, 0), seqInd(86, 5), seqInd(87, 6)},
		{"SQN_HE equal to SQN_MS resets", seqInd(128, 0), seqInd(128, 0), seqInd(129, 1)},
		{"older AUTS does not roll SQN_HE back", seqInd(256, 0), seqInd(128, 0), seqInd(256, 0)},
		{"SQN_HE exactly Delta ahead is kept", seqInd(128+1<<28, 0), seqInd(128, 0), seqInd(128+1<<28, 0)},
		{"SQN_HE more than Delta ahead resets", seqInd(128+1<<28+1, 0), seqInd(128, 0), seqInd(129, 1)},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer gock.Off()
			openapi.InterceptInnerHttp2Client(t, false)

			gock.New("http://127.0.0.4:8000/nudr-dr/v2").
				Get("/subscription-data/" + supi + "/authentication-data/authentication-subscription").
				Reply(http.StatusOK).
				JSON(models.Udr_DR_AuthenticationSubscription{
					AuthenticationMethod:          models.Udr_DR_AuthMethod_5_G_AKA,
					EncPermanentKey:               k,
					SequenceNumber:                &models.Udr_DR_SequenceNumber{Sqn: tc.storedSqn},
					AuthenticationManagementField: "8000",
					EncOpcKey:                     opc,
				})
			var patched []models.PatchItem
			gock.New("http://127.0.0.4:8000").
				Patch("/nudr-dr/v2/subscription-data/" + supi + "/authentication-data/authentication-subscription").
				AddMatcher(func(r *http.Request, _ *gock.Request) (bool, error) {
					body, err := io.ReadAll(r.Body)
					r.Body = io.NopCloser(bytes.NewReader(body))
					return err == nil && json.Unmarshal(body, &patched) == nil, err
				}).
				Reply(http.StatusNoContent)

			ctrl := gomock.NewController(t)
			mockApp := mockapp.NewMockApp(ctrl)
			testConsumer, err := consumer.NewConsumer(mockApp)
			require.NoError(t, err)
			testProcessor, err := NewProcessor(mockApp)
			require.NoError(t, err)
			ue := new(udm_context.UdmUeContext)
			ue.Init()
			ue.Supi = supi
			ue.UdrUri = "http://127.0.0.4:8000"
			udm_context.GetSelf().UdmUePool.Store(supi, ue)
			mockApp.EXPECT().Consumer().Return(testConsumer).AnyTimes()
			mockApp.EXPECT().Context().Return(&udm_context.UDMContext{NrfUri: "http://127.0.0.10:8000", NfId: "1"}).
				AnyTimes()

			kBytes, err := hex.DecodeString(k)
			require.NoError(t, err)
			opcBytes, err := hex.DecodeString(opc)
			require.NoError(t, err)
			randBytes, err := hex.DecodeString(rand)
			require.NoError(t, err)
			sqnMs, err := hex.DecodeString(tc.sqnMs)
			require.NoError(t, err)
			auts, err := milenage.GenerateAUTS(opcBytes, kBytes, randBytes, sqnMs)
			require.NoError(t, err)

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			testProcessor.GenerateAuthDataProcedure(c, models.Udm_UEAU_AuthenticationInfoRequest{
				ServingNetworkName: "5G:mnc093.mcc208.3gppnetwork.org",
				ResynchronizationInfo: &models.Udm_UEAU_ResynchronizationInfo{
					Rand: rand, Auts: hex.EncodeToString(auts),
				},
			}, supi)

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var res models.Udm_UEAU_AuthenticationInfoResult
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
			avRand, err := hex.DecodeString(res.AuthenticationVector.Rand)
			require.NoError(t, err)
			autn, err := hex.DecodeString(res.AuthenticationVector.Autn)
			require.NoError(t, err)
			avSqn, _, _, _, _, err := milenage.GenerateKeysWithAUTN(opcBytes, kBytes, avRand, autn)
			require.NoError(t, err)
			require.Equal(t, tc.expSqn, hex.EncodeToString(avSqn))

			require.Len(t, patched, 1)
			stored, err := json.Marshal(patched[0].Value)
			require.NoError(t, err)
			expNext, err := hex.DecodeString(tc.expSqn)
			require.NoError(t, err)
			expNext[5]++
			require.JSONEq(t, `{"sqn":"`+hex.EncodeToString(expNext)+`"}`, string(stored))
		})
	}
}
