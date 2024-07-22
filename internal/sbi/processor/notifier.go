package processor

import (
	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/udm/SubscriberDataManagement"
	"github.com/free5gc/openapi/udm/UEContextManagement"
	"github.com/free5gc/udm/internal/logger"
)

func (p *Processor) DataChangeNotificationProcedure(c *gin.Context,
	notifyItems []models.NotifyItem,
	supi string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDM_SDM, models.NrfNfManagementNfType_UDM)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}

	ue, _ := p.Context().UdmUeFindBySupi(supi)

	clientAPI := p.Consumer().GetSDMClient("DataChangeNotification")

	var problemDetails *models.ProblemDetails
	for _, subscriptionDataSubscription := range ue.UdmSubsToNotify {
		onDataChangeNotificationurl := subscriptionDataSubscription.OriginalCallbackReference
		dataChangeNotification := models.ModificationNotification{}
		dataChangeNotification.NotifyItems = notifyItems
		var subDataChangeNotificationPostRequest SubscriberDataManagement.SubscribeDatachangeNotificationPostRequest
		subDataChangeNotificationPostRequest.ModificationNotification = &dataChangeNotification
		_, err = clientAPI.SubscriptionCreationApi.SubscribeDatachangeNotificationPost(
			ctx, onDataChangeNotificationurl, &subDataChangeNotificationPostRequest)
		if err != nil {
			problem, ok := err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails)
			if !ok {
				logger.HttpLog.Error(err.Error())
				problemDetails = openapi.ProblemDetailsSystemFailure(err.Error())
			} else {
				logger.HttpLog.Errorln(err.Error())
				problemDetails = &problem
			}
		}
	}

	c.JSON(int(problemDetails.Status), problemDetails)
}

func (p *Processor) SendOnDeregistrationNotification(ueId string, onDeregistrationNotificationUrl string,
	deregistData models.UdmUecmDeregistrationData,
) *models.ProblemDetails {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDM_UECM, models.NrfNfManagementNfType_UDM)
	if err != nil {
		return pd
	}

	clientAPI := p.Consumer().GetUECMClient("SendOnDeregistrationNotification")
	var call3GppRegistrationDeregistrationNotificationPostRequest UEContextManagement.Call3GppRegistrationDeregistrationNotificationPostRequest
	call3GppRegistrationDeregistrationNotificationPostRequest.UdmUecmDeregistrationData = &deregistData
	_, err = clientAPI.AMFRegistrationFor3GPPAccessApi.
		Call3GppRegistrationDeregistrationNotificationPost(ctx,
			onDeregistrationNotificationUrl,
			&call3GppRegistrationDeregistrationNotificationPostRequest)

	if err != nil {
		problem, ok := err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails)
		if !ok {
			problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
			return problemDetails
		}
		return &problem
	}

	return nil
}
