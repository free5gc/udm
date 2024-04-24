package consumer

import (
	"testing"

	"github.com/h2non/gock"
	"github.com/stretchr/testify/require"

	"github.com/free5gc/openapi"
)

func TestSendRegisterNFInstance(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	gock.InterceptClient(openapi.GetHttpClient())
	defer gock.RestoreClient(openapi.GetHttpClient())
	gock.New("http://127.0.0.10:8000").
		Put("/nnrf-nfm/v1/nf-instances/1").
		Reply(200).
		JSON(map[string]string{})

	//_, _, err := SendRegisterNFInstance("http://127.0.0.10:8000", "1", models.NfProfile{})
	require.NoError(t, nil)
}
