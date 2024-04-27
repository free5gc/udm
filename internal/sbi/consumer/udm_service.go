package consumer

import (
	"net/http"
	"sync"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/Nudm_SubscriberDataManagement"
	"github.com/free5gc/openapi/Nudm_UEContextManagement"
	"github.com/free5gc/openapi/models"
	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/logger"
)

type nudmService struct {
	consumer *Consumer

	nfSDMMu  sync.RWMutex
	nfUECMMu sync.RWMutex

	nfSDMClients  map[string]*Nudm_SubscriberDataManagement.APIClient
	nfUECMClients map[string]*Nudm_UEContextManagement.APIClient
}

func (s *nudmService) getSDMClient(uri string) *Nudm_SubscriberDataManagement.APIClient {
	if uri == "" {
		return nil
	}
	s.nfSDMMu.RLock()
	client, ok := s.nfSDMClients[uri]
	if ok {
		defer s.nfSDMMu.RUnlock()
		return client
	}

	configuration := Nudm_SubscriberDataManagement.NewConfiguration()
	configuration.SetBasePath(uri)
	client = Nudm_SubscriberDataManagement.NewAPIClient(configuration)

	s.nfSDMMu.RUnlock()
	s.nfSDMMu.Lock()
	defer s.nfSDMMu.Unlock()
	s.nfSDMClients[uri] = client
	return client
}

func (s *nudmService) getUECMClient(uri string) *Nudm_UEContextManagement.APIClient {
	if uri == "" {
		return nil
	}
	s.nfUECMMu.RLock()
	client, ok := s.nfUECMClients[uri]
	if ok {
		defer s.nfUECMMu.RUnlock()
		return client
	}

	configuration := Nudm_UEContextManagement.NewConfiguration()
	configuration.SetBasePath(uri)
	client = Nudm_UEContextManagement.NewAPIClient(configuration)

	s.nfUECMMu.RUnlock()
	s.nfUECMMu.Lock()
	defer s.nfUECMMu.Unlock()
	s.nfUECMClients[uri] = client
	return client
}

func (s *nudmService) subscribeToSharedDataProcedure(sdmSubscription *models.SdmSubscription) (
	header http.Header, response *models.SdmSubscription, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDM_SDM, models.NfType_UDM)
	if err != nil {
		return nil, nil, pd
	}

	udmClientAPI := s.getSDMClient("subscribeToSharedData")

	sdmSubscriptionResp, res, err := udmClientAPI.SubscriptionCreationForSharedDataApi.SubscribeToSharedData(
		ctx, *sdmSubscription)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			return nil, nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("SubscribeToSharedData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusCreated {
		header = make(http.Header)
		udm_context.GetSelf().CreateSubstoNotifSharedData(sdmSubscriptionResp.SubscriptionId, &sdmSubscriptionResp)
		reourceUri := udm_context.GetSelf().GetSDMUri() + "//shared-data-subscriptions/" + sdmSubscriptionResp.SubscriptionId
		header.Set("Location", reourceUri)
		return header, &sdmSubscriptionResp, nil
	} else if res.StatusCode == http.StatusNotFound {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}

		return nil, nil, problemDetails
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotImplemented,
			Cause:  "UNSUPPORTED_RESOURCE_URI",
		}

		return nil, nil, problemDetails
	}
}

func (s *nudmService) unsubscribeForSharedDataProcedure(subscriptionID string) *models.ProblemDetails {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDM_SDM, models.NfType_UDM)
	if err != nil {
		return pd
	}

	udmClientAPI := s.getSDMClient("unsubscribeForSharedData")

	res, err := udmClientAPI.SubscriptionDeletionForSharedDataApi.UnsubscribeForSharedData(
		ctx, subscriptionID)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			logger.SdmLog.Warnln(err)
			problemDetails := &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			return problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("UnsubscribeForSharedData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusNoContent {
		return nil
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		return problemDetails
	}
}

func (s *nudmService) DataChangeNotificationProcedure(notifyItems []models.NotifyItem, supi string) *models.ProblemDetails {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDM_SDM, models.NfType_UDM)
	if err != nil {
		return pd
	}

	ue, _ := udm_context.GetSelf().UdmUeFindBySupi(supi)

	clientAPI := s.getSDMClient("DataChangeNotification")

	var problemDetails *models.ProblemDetails
	for _, subscriptionDataSubscription := range ue.UdmSubsToNotify {
		onDataChangeNotificationurl := subscriptionDataSubscription.OriginalCallbackReference
		dataChangeNotification := models.ModificationNotification{}
		dataChangeNotification.NotifyItems = notifyItems

		httpResponse, err := clientAPI.DataChangeNotificationCallbackDocumentApi.OnDataChangeNotification(
			ctx, onDataChangeNotificationurl, dataChangeNotification)
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
		defer func() {
			if rspCloseErr := httpResponse.Body.Close(); rspCloseErr != nil {
				logger.HttpLog.Errorf("OnDataChangeNotification response body cannot close: %+v", rspCloseErr)
			}
		}()
	}

	return problemDetails
}

func (s *nudmService) SendOnDeregistrationNotification(ueId string, onDeregistrationNotificationUrl string,
	deregistData models.DeregistrationData,
) *models.ProblemDetails {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDM_UECM, models.NfType_UDM)
	if err != nil {
		return pd
	}

	clientAPI := s.getUECMClient("SendOnDeregistrationNotification")

	httpResponse, err := clientAPI.DeregistrationNotificationCallbackApi.DeregistrationNotify(
		ctx, onDeregistrationNotificationUrl, deregistData)
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
	defer func() {
		if rspCloseErr := httpResponse.Body.Close(); rspCloseErr != nil {
			logger.HttpLog.Errorf("DeregistrationNotify response body cannot close: %+v", rspCloseErr)
		}
	}()

	return nil
}
