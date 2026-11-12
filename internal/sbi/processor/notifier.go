package processor

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/udm/SDM"
	"github.com/free5gc/openapi/udm/UECM"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/util/metrics/sbi"
)

func (p *Processor) DataChangeNotificationProcedure(c *gin.Context,
	notifyItems []models.NotifyItem,
	supi string,
) {
	ue, ok := p.Context().UdmUeFindBySupi(supi)
	if !ok {
		c.Status(http.StatusNoContent)
		return
	}

	clientAPI := p.Consumer().GetSDMClient("DataChangeNotification")

	var problemDetails *models.ProblemDetails
	for _, subscriptionDataSubscription := range ue.UdmSubsToNotify {
		if subscriptionDataSubscription == nil {
			problemDetails = openapi.ProblemDetailsSystemFailure("missing SDM subscription callback identity")
			continue
		}

		onDataChangeNotificationurl := subscriptionDataSubscription.OriginalCallbackReference
		dataChangeNotification := models.Udm_SDM_ModificationNotification{}
		dataChangeNotification.NotifyItems = notifyItems
		var subDataChangeNotificationPostRequest SDM.SubscribeDatachangeNotificationRequest
		subDataChangeNotificationPostRequest.RequestBody = &dataChangeNotification
		// TS 29.500 6.7.3: use a clean context so no access token is forwarded.
		_, err := clientAPI.SubscriptionCreationApi.SubscribeDatachangeNotification(
			context.TODO(), onDataChangeNotificationurl, &subDataChangeNotificationPostRequest)
		if err != nil {
			if apiErr, ok := err.(openapi.GenericOpenAPIError); ok {
				// API error
				if subDataChangeNoti_err, ok2 := apiErr.
					Model().(SDM.SubscribeDatachangeNotificationError); ok2 {
					problemDetails = subDataChangeNoti_err.ProblemDetails
				}
			} else {
				logger.HttpLog.Error(err.Error())
				problemDetails = openapi.ProblemDetailsSystemFailure(err.Error())
			}
		}
	}
	if problemDetails == nil {
		c.Status(http.StatusNoContent)
		return
	}
	c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
	c.JSON(int(problemDetails.Status), problemDetails)
}

func (p *Processor) SendOnDeregistrationNotification(onDeregistrationNotificationUrl string,
	deregistData models.Udm_UECM_DeregistrationData,
) *models.ProblemDetails {
	clientAPI := p.Consumer().GetUECMClient("SendOnDeregistrationNotification")
	var call3GppRegistrationDeregistrationNotificationPostRequest UECM.
		Call3GppRegistrationDeregistrationNotificationRequest
	call3GppRegistrationDeregistrationNotificationPostRequest.RequestBody = &deregistData
	// This asynchronous notification must not inherit request cancellation or tokens.
	_, err := clientAPI.AMFRegistrationFor3GPPAccessApi.
		Call3GppRegistrationDeregistrationNotification(context.Background(),
			onDeregistrationNotificationUrl,
			&call3GppRegistrationDeregistrationNotificationPostRequest)
	if err != nil {
		if apiErr, ok := err.(openapi.GenericOpenAPIError); ok {
			// API error
			if deregisterNoti_err, ok2 := apiErr.
				Model().(UECM.Call3GppRegistrationDeregistrationNotificationError); ok2 {
				return deregisterNoti_err.ProblemDetails
			}
		}
	}

	return nil
}
