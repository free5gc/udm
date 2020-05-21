package producer

import (
	"context"
	"free5gc/lib/openapi"
	"free5gc/lib/openapi/models"
	m "free5gc/lib/openapi/models"
	udm_message "free5gc/src/udm/handler/message"
	"net/http"
)

func HandleUpdate(httpChannel chan udm_message.HandlerResponseMessage, gpsi string, body m.PpData) {

	clientAPI := createUDMClientToUDR(gpsi, false)
	res, err := clientAPI.ProvisionedParameterDataDocumentApi.ModifyPpData(context.Background(), gpsi, nil)
	if err != nil {
		var problemDetails m.ProblemDetails
		problemDetails.Cause = err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
		udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		return
	}
	udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusNoContent, nil)
}
