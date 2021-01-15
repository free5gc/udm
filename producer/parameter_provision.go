package producer

import (
	"context"
	"free5gc/lib/http_wrapper"
	"free5gc/lib/openapi"
	"free5gc/lib/openapi/models"
	"free5gc/src/udm/logger"
	"net/http"
)

func HandleUpdateRequest(request *http_wrapper.Request) *http_wrapper.Response {
	// step 1: log
	logger.PpLop.Infoln("Handle UpdateRequest")

	// step 2: retrieve request
	updateRequest := request.Body.(models.PpData)
	gpsi := request.Params["gpsi"]

	// step 3: handle the message
	problemDetails := UpdateProcedure(updateRequest, gpsi)

	// step 4: process the return value from step 3
	if problemDetails != nil {
		return http_wrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	} else {
		return http_wrapper.NewResponse(http.StatusNoContent, nil, nil)
	}
}

func UpdateProcedure(updateRequest models.PpData, gpsi string) (problemDetails *models.ProblemDetails) {
	clientAPI := createUDMClientToUDR(gpsi, false)
	res, err := clientAPI.ProvisionedParameterDataDocumentApi.ModifyPpData(context.Background(), gpsi, nil)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: int32(res.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		return problemDetails
	}
	return nil
}
