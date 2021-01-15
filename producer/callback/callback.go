package callback

import (
	"context"
	"free5gc/lib/openapi/Nudm_SubscriberDataManagement"
	"free5gc/lib/openapi/Nudm_UEContextManagement"
	"free5gc/lib/openapi/models"
	udm_context "free5gc/src/udm/context"
	"free5gc/src/udm/logger"
	"net/http"
)

func DataChangeNotificationProcedure(notifyItems []models.NotifyItem, supi string) *models.ProblemDetails {
	ue, _ := udm_context.UDM_Self().UdmUeFindBySupi(supi)
	configuration := Nudm_SubscriberDataManagement.NewConfiguration()
	clientAPI := Nudm_SubscriberDataManagement.NewAPIClient(configuration)

	var problemDetails *models.ProblemDetails
	for _, subscriptionDataSubscription := range ue.UdmSubsToNotify {
		onDataChangeNotificationurl := subscriptionDataSubscription.OriginalCallbackReference
		dataChangeNotification := models.ModificationNotification{}
		dataChangeNotification.NotifyItems = notifyItems
		httpResponse, err := clientAPI.DataChangeNotificationCallbackDocumentApi.OnDataChangeNotification(
			context.TODO(), onDataChangeNotificationurl, dataChangeNotification)
		if err != nil {
			if httpResponse == nil {
				logger.HttpLog.Error(err.Error())
				problemDetails = &models.ProblemDetails{
					Status: http.StatusForbidden,
					Detail: err.Error(),
				}
			} else {
				logger.HttpLog.Errorln(err.Error())

				problemDetails = &models.ProblemDetails{
					Status: int32(httpResponse.StatusCode),
					Detail: err.Error(),
				}
			}
		}
	}

	return problemDetails
}

func SendOnDeregistrationNotification(ueId string, onDeregistrationNotificationUrl string,
	deregistData models.DeregistrationData) *models.ProblemDetails {
	configuration := Nudm_UEContextManagement.NewConfiguration()
	clientAPI := Nudm_UEContextManagement.NewAPIClient(configuration)

	httpResponse, err := clientAPI.DeregistrationNotificationCallbackApi.DeregistrationNotify(
		context.TODO(), onDeregistrationNotificationUrl, deregistData)
	if err != nil {
		if httpResponse == nil {
			logger.HttpLog.Error(err.Error())
			problemDetails := &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Cause:  "DEREGISTRATION_NOTIFICATION_ERROR",
				Detail: err.Error(),
			}

			return problemDetails
		} else {
			logger.HttpLog.Errorln(err.Error())
			problemDetails := &models.ProblemDetails{
				Status: int32(httpResponse.StatusCode),
				Cause:  "DEREGISTRATION_NOTIFICATION_ERROR",
				Detail: err.Error(),
			}

			return problemDetails
		}
	}

	return nil
}
