package processor

import (
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
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDM_SDM, models.Nrf_NFMgmt_NFType_UDM)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	ue, ok := p.Context().UdmUeFindBySupi(supi)
	if !ok {
		c.Status(http.StatusNoContent)
		return
	}

	clientAPI := p.Consumer().GetSDMClient("DataChangeNotification")

	var problemDetails *models.ProblemDetails
	for _, subscriptionDataSubscription := range ue.UdmSubsToNotify {
		onDataChangeNotificationurl := subscriptionDataSubscription.OriginalCallbackReference
		dataChangeNotification := models.Udm_SDM_ModificationNotification{}
		dataChangeNotification.NotifyItems = notifyItems
		var subDataChangeNotificationPostRequest SDM.SubscribeDatachangeNotificationRequest
		subDataChangeNotificationPostRequest.RequestBody = &dataChangeNotification
		_, err = clientAPI.SubscriptionCreationApi.SubscribeDatachangeNotification(
			ctx, onDataChangeNotificationurl, &subDataChangeNotificationPostRequest)
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

func (p *Processor) SendOnDeregistrationNotification(ueId string, onDeregistrationNotificationUrl string,
	deregistData models.Udm_UECM_DeregistrationData,
) *models.ProblemDetails {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDM_UECM, models.Nrf_NFMgmt_NFType_UDM)
	if err != nil {
		return pd
	}

	clientAPI := p.Consumer().GetUECMClient("SendOnDeregistrationNotification")
	var call3GppRegistrationDeregistrationNotificationPostRequest UECM.
		Call3GppRegistrationDeregistrationNotificationRequest
	call3GppRegistrationDeregistrationNotificationPostRequest.RequestBody = &deregistData
	_, err = clientAPI.AMFRegistrationFor3GPPAccessApi.
		Call3GppRegistrationDeregistrationNotification(ctx,
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
