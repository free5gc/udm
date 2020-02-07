package udm_producer

import (
	"context"
	"github.com/antihax/optional"
	Nudr "gofree5gc/lib/Nudr_DataRepository"
	"gofree5gc/lib/openapi/common"
	"gofree5gc/lib/openapi/models"
	"gofree5gc/src/udm/logger"
	"gofree5gc/src/udm/udm_context"
	"gofree5gc/src/udm/udm_handler/udm_message"
	"log"
	"net/http"
	"strconv"
)

func HandleGetAmData(httpChannel chan udm_message.HandlerResponseMessage, supi string, plmnID string, supportedFeatures string) {
	var queryAmDataParamOpts Nudr.QueryAmDataParamOpts
	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	clientAPI := createUDMClientToUDR(supi, false)
	accessAndMobilitySubscriptionDataResp, res, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.QueryAmData(context.Background(),
		supi, plmnID, &queryAmDataParamOpts)
	if err != nil {
		var problemDetails models.ProblemDetails
		if res == nil {
			log.Panic(err)
		} else if err.Error() != res.Status {
			log.Panic(err)
		} else {
			problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
			udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		}
		return
	}

	if res.StatusCode == http.StatusOK {
		udmUe := udm_context.CreateUdmUe(supi)
		udmUe.AccessAndMobilitySubscriptionData = &accessAndMobilitySubscriptionDataResp
		udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, *udmUe.AccessAndMobilitySubscriptionData)
	} else {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = "DATA_NOT_FOUND"
		problemDetails.Status = 404
		udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusNotFound, problemDetails)
	}
}

func HandleGetIdTranslationResult(httpChannel chan udm_message.HandlerResponseMessage, gpsi string) {

	var idTranslationResult models.IdTranslationResult
	var getIdentityDataParamOpts Nudr.GetIdentityDataParamOpts
	clientAPI := createUDMClientToUDR(gpsi, false)
	idTranslationResultResp, res, err := clientAPI.QueryIdentityDataBySUPIOrGPSIDocumentApi.GetIdentityData(context.Background(), gpsi, &getIdentityDataParamOpts)
	if err != nil {
		var problemDetails models.ProblemDetails
		if res == nil {
			log.Panic(err)
		} else if err.Error() != res.Status {
			log.Panic(err)
		} else {
			problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
			udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		}
		return
	}

	if res.StatusCode == http.StatusOK {
		idList := udm_context.UDM_Self().GpsiSupiList
		idList = idTranslationResultResp
		if idList.SupiList != nil {
			idTranslationResult.Supi = udm_context.GetCorrespondingSupi(idList) // GetCorrespondingSupi get corresponding Supi(here IMSI) matching the given Gpsi from the queried SUPI list from UDR
			idTranslationResult.Gpsi = gpsi
			udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, idTranslationResult)
		} else {
			var problemDetail models.ProblemDetails
			problemDetail.Cause = "USER_NOT_FOUND" // GpsiList and SupiList are empty
			udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusNotFound, problemDetail)
		}
	} else {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = "DATA_NOT_FOUND"
		problemDetails.Status = 404
		udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusNotFound, problemDetails)
	}

}

func HandleGetSupi(httpChannel chan udm_message.HandlerResponseMessage, supi string, plmnID string) {

	clientAPI := createUDMClientToUDR(supi, false)
	var subscriptionDataSetsReq models.SubscriptionDataSets
	var ueContextInSmfDataResp models.UeContextInSmfData
	pduSessionMap := make(map[string]models.PduSession)
	var pgwInfoArray []models.PgwInfo

	var queryAmDataParamOpts Nudr.QueryAmDataParamOpts
	var querySmfSelectDataParamOpts Nudr.QuerySmfSelectDataParamOpts

	amData, res, err1 := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.QueryAmData(context.Background(),
		supi, plmnID, &queryAmDataParamOpts)
	subscriptionDataSetsReq.AmData = &amData
	if err1 != nil {
		logger.SdmLog.Panic(err1.Error())
	}

	smfSelData, res, err2 := clientAPI.SMFSelectionSubscriptionDataDocumentApi.QuerySmfSelectData(context.Background(),
		supi, plmnID, &querySmfSelectDataParamOpts)
	subscriptionDataSetsReq.SmfSelData = &smfSelData
	if err2 != nil {
		logger.SdmLog.Panic(err2.Error())
	}

	traceData, res, err3 := clientAPI.TraceDataDocumentApi.QueryTraceData(context.Background(), supi, plmnID, nil)
	subscriptionDataSetsReq.TraceData = &traceData
	if err3 != nil {
		logger.SdmLog.Panic(err3.Error())
	}

	sessionManagementSubscriptionData, res, err4 := clientAPI.SessionManagementSubscriptionDataApi.QuerySmData(context.Background(), supi, plmnID, nil)
	subscriptionDataSetsReq.SmData = sessionManagementSubscriptionData
	if err4 != nil {
		logger.SdmLog.Panic(err4.Error())
	}

	pdusess, res, err := clientAPI.SMFRegistrationsCollectionApi.QuerySmfRegList(context.Background(), supi, nil)
	array := pdusess
	if err != nil {
		logger.SdmLog.Panic(err.Error())
	}

	for _, element := range array {
		var pduSession models.PduSession
		pduSession.Dnn = element.Dnn
		pduSession.SmfInstanceId = element.SmfInstanceId
		pduSession.PlmnId = element.PlmnId
		pduSessionMap[strconv.Itoa(int(element.PduSessionId))] = pduSession
	}
	ueContextInSmfDataResp.PduSessions = pduSessionMap

	for _, element := range array {
		var pgwInfo models.PgwInfo
		pgwInfo.Dnn = element.Dnn
		pgwInfo.PgwFqdn = element.PgwFqdn
		pgwInfo.PlmnId = element.PlmnId
		pgwInfoArray = append(pgwInfoArray, pgwInfo)
	}
	ueContextInSmfDataResp.PgwInfo = pgwInfoArray

	subscriptionDataSetsReq.UecSmfData = &ueContextInSmfDataResp

	if res.StatusCode == http.StatusOK {
		udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, subscriptionDataSetsReq)
	} else {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = "DATA_NOT_FOUND"
		udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusNotFound, problemDetails)

	}
}

func HandleGetSharedData(httpChannel chan udm_message.HandlerResponseMessage, sharedDataIds []string, supportedFeatures string) {

	clientAPI := createUDMClientToUDR("", true)
	var getSharedDataParamOpts Nudr.GetSharedDataParamOpts
	getSharedDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	sharedDataResp, res, err := clientAPI.RetrievalOfSharedDataApi.GetSharedData(context.Background(), sharedDataIds,
		&getSharedDataParamOpts)
	if err != nil {
		var problemDetails models.ProblemDetails
		if res == nil {
			log.Panic(err)
		} else if err.Error() != res.Status {
			log.Panic(err)
		} else {
			problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
			udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		}
		return
	}

	if res.StatusCode == http.StatusOK {
		udm_context.UDM_Self().SharedSubsDataMap = udm_context.MappingSharedData(sharedDataResp)
		sharedData := udm_context.ObtainRequiredSharedData(sharedDataIds, sharedDataResp)
		udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, sharedData)
	} else {
		var problemDetail models.ProblemDetails
		problemDetail.Cause = "DATA_NOT_FOUND"
		udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusNotFound, problemDetail)
	}
}

func HandleGetSmData(httpChannel chan udm_message.HandlerResponseMessage, supi string, plmnID string, Dnn string, Snssai string, supportedFeatures string) {

	clientAPI := createUDMClientToUDR(supi, false)
	var querySmDataParamOpts Nudr.QuerySmDataParamOpts
	querySmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	sessionManagementSubscriptionDataResp, res, err := clientAPI.SessionManagementSubscriptionDataApi.QuerySmData(context.Background(),
		supi, plmnID, &querySmDataParamOpts)
	if err != nil {
		var problemDetails models.ProblemDetails
		if res == nil {
			log.Panic(err)
		} else if err.Error() != res.Status {
			log.Panic(err)
		} else {
			problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
			udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		}
		return
	}

	if res.StatusCode == http.StatusOK {
		udmUe := udm_context.CreateUdmUe(supi)
		var snssaikey string
		var AllDnnConfigsbyDnn []models.DnnConfiguration
		var AllDnns []map[string]models.DnnConfiguration
		udmUe.SessionManagementSubsData, snssaikey, AllDnnConfigsbyDnn, AllDnns = udm_context.ManageSmData(sessionManagementSubscriptionDataResp, Snssai, Dnn)

		switch {
		case Snssai == "" && Dnn == "":
			udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, AllDnns)
		case Snssai != "" && Dnn == "":
			udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, udmUe.SessionManagementSubsData[snssaikey].DnnConfigurations)
		case Snssai == "" && Dnn != "":
			udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, AllDnnConfigsbyDnn)
		case Snssai != "" && Dnn != "":
			udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, udmUe.SessionManagementSubsData[snssaikey].DnnConfigurations[Dnn])
		default:
			udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, udmUe.SessionManagementSubsData)
		}
	} else {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = "DATA_NOT_FOUND"
		udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusNotFound, problemDetails)
	}
}

func HandleGetNssai(httpChannel chan udm_message.HandlerResponseMessage, supi string, plmnID string, supportedFeatures string) {

	var queryAmDataParamOpts Nudr.QueryAmDataParamOpts
	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	var nssaiResp models.Nssai
	clientAPI := createUDMClientToUDR(supi, false)

	accessAndMobilitySubscriptionDataResp, res, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.QueryAmData(context.Background(),
		supi, plmnID, &queryAmDataParamOpts)
	if err != nil {
		var problemDetails models.ProblemDetails
		if res == nil {
			log.Panic(err)
		} else if err.Error() != res.Status {
			log.Panic(err)
		} else {
			problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
			udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		}
		return
	}
	nssaiResp = *accessAndMobilitySubscriptionDataResp.Nssai

	if res.StatusCode == http.StatusOK {
		udmUe := udm_context.CreateUdmUe(supi)
		udmUe.Nssai = &nssaiResp
		if plmnID != "" {
			udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, udmUe.Nssai)
		} else {
			udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, udmUe.Nssai)
		}
	} else {
		var problemDetail models.ProblemDetails
		problemDetail.Cause = "DATA_NOT_FOUND"
		udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusNotFound, problemDetail)
	}
}

func HandleGetSmfSelectData(httpChannel chan udm_message.HandlerResponseMessage, supi string, plmnID string) {

	clientAPI := createUDMClientToUDR(supi, false)
	var querySmfSelectDataParamOpts Nudr.QuerySmfSelectDataParamOpts
	smfSelectionSubscriptionDataResp, res, err := clientAPI.SMFSelectionSubscriptionDataDocumentApi.QuerySmfSelectData(context.Background(),
		supi, plmnID, &querySmfSelectDataParamOpts)
	if err != nil {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
		udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		return
	}
	udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, smfSelectionSubscriptionDataResp)
}

func HandleSubscribeToSharedData(httpChannel chan udm_message.HandlerResponseMessage, sdmSubscription models.SdmSubscription) {

	// TODO
	clientAPI := createUDMClientToUDR("", true)
	sdmSubscriptionResp, res, err := clientAPI.SDMSubscriptionsCollectionApi.CreateSdmSubscriptions(context.Background(), "===", sdmSubscription)
	if err != nil {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
		udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		return
	}
	udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusCreated, sdmSubscriptionResp)
}

func HandleSubscribe(httpChannel chan udm_message.HandlerResponseMessage, supi string, subscriptionID string, sdmSubscription models.SdmSubscription) {

	clientAPI := createUDMClientToUDR(supi, false)
	sdmSubscriptionResp, res, err := clientAPI.SDMSubscriptionsCollectionApi.CreateSdmSubscriptions(context.Background(), supi, sdmSubscription)
	if err != nil {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
		udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		return
	}
	udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusCreated, sdmSubscriptionResp)
}

func HandleUnsubscribeForSharedData(httpChannel chan udm_message.HandlerResponseMessage, subscriptionID string) {

	// TODO
	clientAPI := createUDMClientToUDR("", true)
	res, err := clientAPI.SDMSubscriptionDocumentApi.RemovesdmSubscriptions(context.Background(), "====", subscriptionID)
	if err != nil {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
		udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		return
	}
	udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusNoContent, nil)
}

func HandleUnsubscribe(httpChannel chan udm_message.HandlerResponseMessage, supi string, subscriptionID string) {

	clientAPI := createUDMClientToUDR(supi, false)
	res, err := clientAPI.SDMSubscriptionDocumentApi.RemovesdmSubscriptions(context.Background(), supi, subscriptionID)
	if err != nil {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
		udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		return
	}
	udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusNoContent, nil)
}

func HandleModify(httpChannel chan udm_message.HandlerResponseMessage, supi string, subscriptionID string, sdmSubsModification models.SdmSubsModification) {

	clientAPI := createUDMClientToUDR(supi, false)
	sdmSubscription := models.SdmSubscription{}
	body := Nudr.UpdatesdmsubscriptionsParamOpts{
		SdmSubscription: optional.NewInterface(sdmSubscription),
	}
	res, err := clientAPI.SDMSubscriptionDocumentApi.Updatesdmsubscriptions(context.Background(), supi, subscriptionID, &body)
	if err != nil {
		var problemDetails models.ProblemDetails
		if res == nil {
			problemDetails.Status = http.StatusServiceUnavailable
			problemDetails.Cause = "Service_Unavailable"
			udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusServiceUnavailable, problemDetails)
			return
		}
		if _, ok := err.(common.GenericOpenAPIError).Model().(models.ProblemDetails); !ok {
			problemDetails.Status = http.StatusServiceUnavailable
			problemDetails.Cause = "Service_Unavailable"
			udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusServiceUnavailable, problemDetails)
			return
		}
		problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
		udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		return
	}
	udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, sdmSubscription)
}

func HandleModifyForSharedData(httpChannel chan udm_message.HandlerResponseMessage, supi string, subscriptionID string, sdmSubsModification models.SdmSubsModification) {

	// TODO
	var sdmSubscription models.SdmSubscription
	clientAPI := createUDMClientToUDR(supi, false)
	sdmSubs := models.SdmSubscription{}
	body := Nudr.UpdatesdmsubscriptionsParamOpts{
		SdmSubscription: optional.NewInterface(sdmSubs),
	}

	res, err := clientAPI.SDMSubscriptionDocumentApi.Updatesdmsubscriptions(context.Background(), supi, subscriptionID, &body)
	if err != nil {
		var problemDetails models.ProblemDetails
		if res == nil {
			problemDetails.Status = http.StatusServiceUnavailable
			problemDetails.Cause = "Service_Unavailable"
			udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusServiceUnavailable, problemDetails)
			return
		}
		if _, ok := err.(common.GenericOpenAPIError).Model().(models.ProblemDetails); !ok {
			problemDetails.Status = http.StatusServiceUnavailable
			problemDetails.Cause = "Service_Unavailable"
			udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusServiceUnavailable, problemDetails)
			return
		}
		problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
		udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		return
	}
	udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, sdmSubscription)
}

func HandleGetTraceData(httpChannel chan udm_message.HandlerResponseMessage, supi string, plmnID string) {

	clientAPI := createUDMClientToUDR(supi, false)
	traceDataResp, res, err := clientAPI.TraceDataDocumentApi.QueryTraceData(context.Background(), supi, plmnID, nil)
	if err != nil {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
		udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		return
	}
	udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, traceDataResp)
}

func HandleGetUeContextInSmfData(httpChannel chan udm_message.HandlerResponseMessage, supi string) {

	clientAPI := createUDMClientToUDR(supi, false)
	var ueContextInSmfData models.UeContextInSmfData
	pduSessionMap := make(map[string]models.PduSession)
	var pgwInfoArray []models.PgwInfo

	pdusess, res, err := clientAPI.SMFRegistrationsCollectionApi.QuerySmfRegList(context.Background(), supi, nil)
	if err != nil {
		var problemDetails models.ProblemDetails
		problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
		udm_message.SendHttpResponseMessage(httpChannel, nil, res.StatusCode, problemDetails)
		return
	}
	array := pdusess

	for _, element := range array {
		var pduSession models.PduSession
		pduSession.Dnn = element.Dnn
		pduSession.SmfInstanceId = element.SmfInstanceId
		pduSession.PlmnId = element.PlmnId
		pduSessionMap[strconv.Itoa(int(element.PduSessionId))] = pduSession
	}
	ueContextInSmfData.PduSessions = pduSessionMap

	for _, element := range array {
		var pgwInfo models.PgwInfo
		pgwInfo.Dnn = element.Dnn
		pgwInfo.PgwFqdn = element.PgwFqdn
		pgwInfo.PlmnId = element.PlmnId
		pgwInfoArray = append(pgwInfoArray, pgwInfo)
	}
	ueContextInSmfData.PgwInfo = pgwInfoArray

	udm_message.SendHttpResponseMessage(httpChannel, nil, http.StatusOK, ueContextInSmfData)
}
