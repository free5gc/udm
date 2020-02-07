package udm_producer

import (
	"context"
	"gofree5gc/lib/Nudr_DataRepository"
	"gofree5gc/lib/openapi/common"
	"gofree5gc/lib/openapi/models"
	"gofree5gc/src/udm/udm_handler/udm_message"
	"net/http"

	"github.com/antihax/optional"
)

func HandleCreateEeSubscription(httpChannel chan udm_message.HandlerResponseMessage, ueIdentity string, subscriptionID string, eesubscription models.EeSubscription) {

	clientAPI := createUDMClientToUDR(ueIdentity, false)
	eeSubscriptionResp, res, err := clientAPI.EventExposureSubscriptionsCollectionApi.CreateEeSubscriptions(context.Background(),
		ueIdentity, eesubscription)
	if err != nil {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
		udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		return
	}
	udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusCreated, eeSubscriptionResp)
}
func HandleDeleteEeSubscription(httpChannel chan udm_message.HandlerResponseMessage, ueIdentity string, subscriptionID string) {

	clientAPI := createUDMClientToUDR(ueIdentity, false)
	res, err := clientAPI.EventExposureSubscriptionDocumentApi.RemoveeeSubscriptions(context.Background(), ueIdentity, subscriptionID)
	if err != nil {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
		udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		return
	}
	udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusNoContent, nil)
}

func HandleUpdateEeSubscription(httpChannel chan udm_message.HandlerResponseMessage, ueIdentity string, subscriptionID string) {

	clientAPI := createUDMClientToUDR(ueIdentity, false)
	patchItem := models.PatchItem{}
	body := Nudr_DataRepository.UpdateEesubscriptionsParamOpts{
		EeSubscription: optional.NewInterface(patchItem),
	}
	res, err := clientAPI.EventExposureSubscriptionDocumentApi.UpdateEesubscriptions(context.Background(), ueIdentity, subscriptionID, &body)
	if err != nil {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
		udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		return
	}
	udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusNoContent, nil)
}
