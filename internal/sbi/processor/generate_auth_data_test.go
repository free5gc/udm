package processor

import (
	"net/http/httptest"
	"testing"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	Nudr_DataRepository "github.com/free5gc/openapi/udr/DataRepository"
	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/sbi/consumer"
	mockapp "github.com/free5gc/udm/pkg/mockapp"
	"github.com/gin-gonic/gin"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGenerateAuthDataProcedure(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	gock.InterceptClient(openapi.GetHttpClient())
	defer gock.RestoreClient(openapi.GetHttpClient())

	queryRes := Nudr_DataRepository.QueryAuthSubsDataResponse{
		AuthenticationSubscription: models.AuthenticationSubscription{
			AuthenticationMethod:          models.AuthMethod__5_G_AKA,
			EncPermanentKey:               "8baf473f2f8fd09487cccbd7097c6862",
			ProtectionParameterId:         "8baf473f2f8fd09487cccbd7097c6862",
			SequenceNumber:                &models.SequenceNumber{Sqn: "000000000023"},
			AuthenticationManagementField: "8000",
			AlgorithmId:                   "128-EEA0",
			EncOpcKey:                     "8e27b6af0e692e750f32667a3b14605d",
			EncTopcKey:                    "8e27b6",
		},
	}

	gock.New("http://127.0.0.4:8000/nudr-dr/v2").
		Get("/subscription-data/imsi-208930000000001/authentication-data/authentication-subscription").
		Reply(200).
		AddHeader("Content-Type", "application/json").
		JSON(queryRes)

	gock.New("http://127.0.0.4:8000").
		Patch("/nudr-dr/v2/subscription-data/imsi-208930000000001/authentication-data/authentication-subscription").
		Reply(204).
		JSON(map[string]string{})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockApp := mockapp.NewMockApp(ctrl)
	testConsumer, err := consumer.NewConsumer(mockApp)
	require.NoError(t, err)
	testProcessor, err := NewProcessor(mockApp)
	require.NoError(t, err)
	udm_context.GetSelf().NrfUri = "http://127.0.0.10:8000"
	ue := new(udm_context.UdmUeContext)
	ue.Init()
	ue.Supi = "imsi-208930000000001"
	ue.UdrUri = "http://127.0.0.4:8000"
	udm_context.GetSelf().UdmUePool.Store("imsi-208930000000001", ue)

	mockApp.EXPECT().Consumer().Return(testConsumer).AnyTimes()
	mockApp.EXPECT().Context().Return(
		&udm_context.UDMContext{
			OAuth2Required: false,
			NrfUri:         "http://127.0.0.10:8000",
			NfId:           "1",
		},
	).AnyTimes()

	authInfoReq := models.AuthenticationInfoRequest{
		ServingNetworkName: "internet",
	}
	httpRecorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(httpRecorder)
	testProcessor.GenerateAuthDataProcedure(c, authInfoReq, "imsi-208930000000001")
}
