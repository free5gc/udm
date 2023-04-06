package producer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/antihax/optional"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/Nudm_SubscriberDataManagement"
	Nudr "github.com/free5gc/openapi/Nudr_DataRepository"
	"github.com/free5gc/openapi/models"
	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/util/httpwrapper"
)

func HandleGetAmDataRequest(request *httpwrapper.Request) *httpwrapper.Response {
	// step 1: log
	logger.SdmLog.Infof("Handle GetAmData")

	// step 2: retrieve request
	supi := request.Params["supi"]
	plmnIDStruct, problemDetails := getPlmnIDStruct(request.Query)
	if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
	supportedFeatures := request.Query.Get("supported-features")

	// step 3: handle the message
	response, problemDetails := getAmDataProcedure(supi, plmnID, supportedFeatures)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		return httpwrapper.NewResponse(http.StatusOK, nil, response)
	} else if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	problemDetails = &models.ProblemDetails{
		Status: http.StatusForbidden,
		Cause:  "UNSPECIFIED",
	}
	return httpwrapper.NewResponse(http.StatusForbidden, nil, problemDetails)
}

// TS 29.503 5.2.2.2.3
// Access and Mobility Subscription Data Retrieval
func getAmDataProcedure(supi string, plmnID string, supportedFeatures string) (
	response *models.AccessAndMobilitySubscriptionData, problemDetails *models.ProblemDetails,
) {
	var queryAmDataParamOpts Nudr.QueryAmDataParamOpts
	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	clientAPI, err := createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	accessAndMobilitySubscriptionDataResp, res, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.
		QueryAmData(context.Background(), supi, plmnID, &queryAmDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Errorf(err.Error())
		} else if err.Error() != res.Status {
			logger.SdmLog.Errorf("Response State: %+v", err.Error())
		} else {
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryAmData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udmUe, ok := udm_context.Getself().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.Getself().NewUdmUe(supi)
		}
		udmUe.SetAMSubsriptionData(&accessAndMobilitySubscriptionDataResp)
		return &accessAndMobilitySubscriptionDataResp, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		return nil, problemDetails
	}
}

func HandleGetIdTranslationResultRequest(request *httpwrapper.Request) *httpwrapper.Response {
	// step 1: log
	logger.SdmLog.Infof("Handle GetIdTranslationResultRequest")

	// step 2: retrieve request
	gpsi := request.Params["gpsi"]

	// step 3: handle the message
	response, problemDetails := getIdTranslationResultProcedure(gpsi)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		return httpwrapper.NewResponse(http.StatusOK, nil, response)
	} else if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	problemDetails = &models.ProblemDetails{
		Status: http.StatusForbidden,
		Cause:  "UNSPECIFIED",
	}
	return httpwrapper.NewResponse(http.StatusForbidden, nil, problemDetails)
}

// TS 29.503 5.2.2.2.10
// Identifier Translation
func getIdTranslationResultProcedure(gpsi string) (response *models.IdTranslationResult,
	problemDetails *models.ProblemDetails,
) {
	var idTranslationResult models.IdTranslationResult
	var getIdentityDataParamOpts Nudr.GetIdentityDataParamOpts

	clientAPI, err := createUDMClientToUDR(gpsi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	idTranslationResultResp, res, err := clientAPI.QueryIdentityDataBySUPIOrGPSIDocumentApi.GetIdentityData(
		context.Background(), gpsi, &getIdentityDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Errorf(err.Error())
		} else if err.Error() != res.Status {
			logger.SdmLog.Errorf("Response State: %+v", err.Error())
		} else {
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}

			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("GetIdentityData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		if idList := idTranslationResultResp; idList.SupiList != nil {
			// GetCorrespondingSupi get corresponding Supi(here IMSI) matching the given Gpsi from the queried SUPI list from UDR
			idTranslationResult.Supi = udm_context.GetCorrespondingSupi(idList)
			idTranslationResult.Gpsi = gpsi

			return &idTranslationResult, nil
		} else {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "USER_NOT_FOUND",
			}

			return nil, problemDetails
		}
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}

		return nil, problemDetails
	}
}

func HandleGetSupiRequest(request *httpwrapper.Request) *httpwrapper.Response {
	// step 1: log
	logger.SdmLog.Infof("Handle GetSupiRequest")

	// step 2: retrieve request
	supi := request.Params["supi"]
	plmnIDStruct, problemDetails := getPlmnIDStruct(request.Query)
	if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
	dataSetNames := strings.Split(request.Query.Get("dataset-names"), ",")
	supportedFeatures := request.Query.Get("supported-features")

	// step 3: handle the message
	response, problemDetails := getSupiProcedure(supi, plmnID, dataSetNames, supportedFeatures)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		return httpwrapper.NewResponse(http.StatusOK, nil, response)
	} else if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	problemDetails = &models.ProblemDetails{
		Status: http.StatusForbidden,
		Cause:  "UNSPECIFIED",
	}
	return httpwrapper.NewResponse(http.StatusForbidden, nil, problemDetails)
}

// TS 29.503 5.2.2.2.9
// Retrieval Of Multiple Data Set
func getSupiProcedure(supi string, plmnID string, dataSetNames []string, supportedFeatures string) (
	response *models.SubscriptionDataSets, problemDetails *models.ProblemDetails,
) {
	if len(dataSetNames) < 2 {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "BAD_REQUEST",
			Detail: "datasetNames must have at least 2 elements",
		}
		return nil, problemDetails
	}

	clientAPI, err := createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var subscriptionDataSets, subsDataSetBody models.SubscriptionDataSets
	var ueContextInSmfDataResp models.UeContextInSmfData
	pduSessionMap := make(map[string]models.PduSession)
	var pgwInfoArray []models.PgwInfo

	var queryAmDataParamOpts Nudr.QueryAmDataParamOpts
	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	var querySmfSelectDataParamOpts Nudr.QuerySmfSelectDataParamOpts
	var queryTraceDataParamOpts Nudr.QueryTraceDataParamOpts
	var querySmDataParamOpts Nudr.QuerySmDataParamOpts

	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	querySmfSelectDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	udm_context.Getself().CreateSubsDataSetsForUe(supi, subsDataSetBody)

	if containDataSetName(dataSetNames, string(models.DataSetName_AM)) {
		var body models.AccessAndMobilitySubscriptionData
		udm_context.Getself().CreateAccessMobilitySubsDataForUe(supi, body)
		amData, res, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.QueryAmData(
			context.Background(), supi, plmnID, &queryAmDataParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorf(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorf("Response State: %+v", err.Error())
			} else {
				problemDetails = &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}

				return nil, problemDetails
			}
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QueryAmData response body cannot close: %+v", rspCloseErr)
			}
		}()
		if res.StatusCode == http.StatusOK {
			udmUe, ok := udm_context.Getself().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.Getself().NewUdmUe(supi)
			}
			udmUe.SetAMSubsriptionData(&amData)
			subscriptionDataSets.AmData = &amData
		} else {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "DATA_NOT_FOUND",
			}

			return nil, problemDetails
		}
	}

	if containDataSetName(dataSetNames, string(models.DataSetName_SMF_SEL)) {
		var smfSelSubsbody models.SmfSelectionSubscriptionData
		udm_context.Getself().CreateSmfSelectionSubsDataforUe(supi, smfSelSubsbody)
		smfSelData, res, err := clientAPI.SMFSelectionSubscriptionDataDocumentApi.QuerySmfSelectData(context.Background(),
			supi, plmnID, &querySmfSelectDataParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorln(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorln(err.Error())
			} else {
				problemDetails = &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}

				return nil, problemDetails
			}
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QuerySmfSelectData response body cannot close: %+v", rspCloseErr)
			}
		}()
		if res.StatusCode == http.StatusOK {
			udmUe, ok := udm_context.Getself().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.Getself().NewUdmUe(supi)
			}
			udmUe.SetSmfSelectionSubsData(&smfSelData)
			subscriptionDataSets.SmfSelData = &smfSelData
		} else {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "DATA_NOT_FOUND",
			}

			return nil, problemDetails
		}
	}

	if containDataSetName(dataSetNames, string(models.DataSetName_UEC_SMF)) {
		var UeContextInSmfbody models.UeContextInSmfData
		var querySmfRegListParamOpts Nudr.QuerySmfRegListParamOpts
		querySmfRegListParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
		udm_context.Getself().CreateUeContextInSmfDataforUe(supi, UeContextInSmfbody)
		pdusess, res, err := clientAPI.SMFRegistrationsCollectionApi.QuerySmfRegList(
			context.Background(), supi, &querySmfRegListParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorf(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorf("Response State: %+v", err.Error())
			} else {
				problemDetails = &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}

				return nil, problemDetails
			}
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QuerySmfRegList response body cannot close: %+v", rspCloseErr)
			}
		}()

		for _, element := range pdusess {
			var pduSession models.PduSession
			pduSession.Dnn = element.Dnn
			pduSession.SmfInstanceId = element.SmfInstanceId
			pduSession.PlmnId = element.PlmnId
			pduSessionMap[strconv.Itoa(int(element.PduSessionId))] = pduSession
		}
		ueContextInSmfDataResp.PduSessions = pduSessionMap

		for _, element := range pdusess {
			var pgwInfo models.PgwInfo
			pgwInfo.Dnn = element.Dnn
			pgwInfo.PgwFqdn = element.PgwFqdn
			pgwInfo.PlmnId = element.PlmnId
			pgwInfoArray = append(pgwInfoArray, pgwInfo)
		}
		ueContextInSmfDataResp.PgwInfo = pgwInfoArray

		if res.StatusCode == http.StatusOK {
			udmUe, ok := udm_context.Getself().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.Getself().NewUdmUe(supi)
			}
			udmUe.UeCtxtInSmfData = &ueContextInSmfDataResp
			subscriptionDataSets.UecSmfData = &ueContextInSmfDataResp
		} else {
			var problemDetails models.ProblemDetails
			problemDetails.Cause = "DATA_NOT_FOUND"
			logger.SdmLog.Errorf(problemDetails.Cause)
		}
	}

	// TODO: UE Context in SMSF Data
	// if containDataSetName(dataSetNames, string(models.DataSetName_UEC_SMSF)) {
	// }

	// TODO: SMS Subscription Data
	// if containDataSetName(dataSetNames, string(models.DataSetName_SMS_SUB)) {
	// }

	if containDataSetName(dataSetNames, string(models.DataSetName_SM)) {
		sessionManagementSubscriptionData, res, err := clientAPI.SessionManagementSubscriptionDataApi.
			QuerySmData(context.Background(), supi, plmnID, &querySmDataParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorf(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorf("Response State: %+v", err.Error())
			} else {
				problemDetails = &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}

				return nil, problemDetails
			}
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QuerySmData response body cannot close: %+v", rspCloseErr)
			}
		}()
		if res.StatusCode == http.StatusOK {
			udmUe, ok := udm_context.Getself().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.Getself().NewUdmUe(supi)
			}
			smData, _, _, _ := udm_context.Getself().ManageSmData(sessionManagementSubscriptionData, "", "")
			udmUe.SetSMSubsData(smData)
			subscriptionDataSets.SmData = sessionManagementSubscriptionData
		} else {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "DATA_NOT_FOUND",
			}

			return nil, problemDetails
		}
	}

	if containDataSetName(dataSetNames, string(models.DataSetName_TRACE)) {
		var TraceDatabody models.TraceData
		udm_context.Getself().CreateTraceDataforUe(supi, TraceDatabody)
		traceData, res, err := clientAPI.TraceDataDocumentApi.QueryTraceData(
			context.Background(), supi, plmnID, &queryTraceDataParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorf(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorf("Response State: %+v", err.Error())
			} else {
				problemDetails = &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}
			}
			return nil, problemDetails
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QueryTraceData response body cannot close: %+v", rspCloseErr)
			}
		}()
		if res.StatusCode == http.StatusOK {
			udmUe, ok := udm_context.Getself().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.Getself().NewUdmUe(supi)
			}
			udmUe.TraceData = &traceData
			udmUe.TraceDataResponse.TraceData = &traceData
			subscriptionDataSets.TraceData = &traceData
		} else {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "DATA_NOT_FOUND",
			}

			return nil, problemDetails
		}
	}

	// TODO: SMS Management Subscription Data
	// if containDataSetName(dataSetNames, string(models.DataSetName_SMS_MNG)) {
	// }

	return &subscriptionDataSets, nil
}

func HandleGetSharedDataRequest(request *httpwrapper.Request) *httpwrapper.Response {
	// step 1: log
	logger.SdmLog.Infof("Handle GetSharedData")

	// step 2: retrieve request
	sharedDataIds := request.Query["sharedDataIds"]
	supportedFeatures := request.Query.Get("supported-features")
	// step 3: handle the message
	response, problemDetails := getSharedDataProcedure(sharedDataIds, supportedFeatures)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		return httpwrapper.NewResponse(http.StatusOK, nil, response)
	} else if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	problemDetails = &models.ProblemDetails{
		Status: http.StatusForbidden,
		Cause:  "UNSPECIFIED",
	}
	return httpwrapper.NewResponse(http.StatusForbidden, nil, problemDetails)
}

// TS 29.503 5.2.2.2.11
// Shared Subscription Data Retrieval
func getSharedDataProcedure(sharedDataIds []string, supportedFeatures string) (
	response []models.SharedData, problemDetails *models.ProblemDetails,
) {
	clientAPI, err := createUDMClientToUDR("")
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var getSharedDataParamOpts Nudr.GetSharedDataParamOpts
	getSharedDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	sharedDataResp, res, err := clientAPI.RetrievalOfSharedDataApi.GetSharedData(context.Background(), sharedDataIds,
		&getSharedDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			logger.SdmLog.Warnln(err)
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}

			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("GetShareData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udm_context.Getself().SharedSubsDataMap = udm_context.MappingSharedData(sharedDataResp)
		sharedData := udm_context.ObtainRequiredSharedData(sharedDataIds, sharedDataResp)
		return sharedData, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		return nil, problemDetails
	}
}

func HandleGetSmDataRequest(request *httpwrapper.Request) *httpwrapper.Response {
	// step 1: log
	logger.SdmLog.Infof("Handle GetSmData")

	// step 2: retrieve request
	supi := request.Params["supi"]
	plmnIDStruct, problemDetails := getPlmnIDStruct(request.Query)
	if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
	Dnn := request.Query.Get("dnn")
	Snssai := request.Query.Get("single-nssai")
	supportedFeatures := request.Query.Get("supported-features")

	// step 3: handle the message
	response, problemDetails := getSmDataProcedure(supi, plmnID, Dnn, Snssai, supportedFeatures)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		return httpwrapper.NewResponse(http.StatusOK, nil, response)
	} else if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	problemDetails = &models.ProblemDetails{
		Status: http.StatusForbidden,
		Cause:  "UNSPECIFIED",
	}
	return httpwrapper.NewResponse(http.StatusForbidden, nil, problemDetails)
}

// TS 29.503 5.2.2.2.5
// Session Management Subscription Data Retrieval
func getSmDataProcedure(supi string, plmnID string, Dnn string, Snssai string, supportedFeatures string) (
	response interface{}, problemDetails *models.ProblemDetails,
) {
	logger.SdmLog.Infof("getSmDataProcedure: SUPI[%s] PLMNID[%s] DNN[%s] SNssai[%s]", supi, plmnID, Dnn, Snssai)

	clientAPI, err := createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var querySmDataParamOpts Nudr.QuerySmDataParamOpts
	querySmDataParamOpts.SingleNssai = optional.NewInterface(Snssai)

	sessionManagementSubscriptionDataResp, res, err := clientAPI.SessionManagementSubscriptionDataApi.
		QuerySmData(context.Background(), supi, plmnID, &querySmDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			logger.SdmLog.Warnln(err)
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}

			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QuerySmData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udmUe, ok := udm_context.Getself().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.Getself().NewUdmUe(supi)
		}
		smData, snssaikey, AllDnnConfigsbyDnn, AllDnns := udm_context.Getself().ManageSmData(
			sessionManagementSubscriptionDataResp, Snssai, Dnn)
		udmUe.SetSMSubsData(smData)

		rspSMSubDataList := make([]models.SessionManagementSubscriptionData, 0, 4)

		udmUe.SmSubsDataLock.RLock()
		for _, eachSMSubData := range udmUe.SessionManagementSubsData {
			rspSMSubDataList = append(rspSMSubDataList, eachSMSubData)
		}
		udmUe.SmSubsDataLock.RUnlock()

		switch {
		case Snssai == "" && Dnn == "":
			return AllDnns, nil
		case Snssai != "" && Dnn == "":
			udmUe.SmSubsDataLock.RLock()
			defer udmUe.SmSubsDataLock.RUnlock()
			return udmUe.SessionManagementSubsData[snssaikey].DnnConfigurations, nil
		case Snssai == "" && Dnn != "":
			return AllDnnConfigsbyDnn, nil
		case Snssai != "" && Dnn != "":
			return rspSMSubDataList, nil
		default:
			udmUe.SmSubsDataLock.RLock()
			defer udmUe.SmSubsDataLock.RUnlock()
			return udmUe.SessionManagementSubsData, nil
		}
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}

		return nil, problemDetails
	}
}

func HandleGetNssaiRequest(request *httpwrapper.Request) *httpwrapper.Response {
	// step 1: log
	logger.SdmLog.Infof("Handle GetNssai")

	// step 2: retrieve request
	supi := request.Params["supi"]
	plmnIDStruct, problemDetails := getPlmnIDStruct(request.Query)
	if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
	supportedFeatures := request.Query.Get("supported-features")

	// step 3: handle the message
	response, problemDetails := getNssaiProcedure(supi, plmnID, supportedFeatures)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		return httpwrapper.NewResponse(http.StatusOK, nil, response)
	} else if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	problemDetails = &models.ProblemDetails{
		Status: http.StatusForbidden,
		Cause:  "UNSPECIFIED",
	}
	return httpwrapper.NewResponse(http.StatusForbidden, nil, problemDetails)
}

// TS 29.503 5.2.2.2.2
// Slice Selection Subscription Data Retrieval
func getNssaiProcedure(supi string, plmnID string, supportedFeatures string) (
	*models.Nssai, *models.ProblemDetails,
) {
	var queryAmDataParamOpts Nudr.QueryAmDataParamOpts
	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	var nssaiResp models.Nssai
	clientAPI, err := createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	accessAndMobilitySubscriptionDataResp, res, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.
		QueryAmData(context.Background(), supi, plmnID, &queryAmDataParamOpts)
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

			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryAmData response body cannot close: %+v", rspCloseErr)
		}
	}()

	nssaiResp = *accessAndMobilitySubscriptionDataResp.Nssai

	if res.StatusCode == http.StatusOK {
		udmUe, ok := udm_context.Getself().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.Getself().NewUdmUe(supi)
		}
		udmUe.Nssai = &nssaiResp
		return udmUe.Nssai, nil
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		return nil, problemDetails
	}
}

func HandleGetSmfSelectDataRequest(request *httpwrapper.Request) *httpwrapper.Response {
	// step 1: log
	logger.SdmLog.Infof("Handle GetSmfSelectData")

	// step 2: retrieve request
	supi := request.Params["supi"]
	plmnIDStruct, problemDetails := getPlmnIDStruct(request.Query)
	if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
	supportedFeatures := request.Query.Get("supported-features")

	// step 3: handle the message
	response, problemDetails := getSmfSelectDataProcedure(supi, plmnID, supportedFeatures)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		return httpwrapper.NewResponse(http.StatusOK, nil, response)
	} else if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	problemDetails = &models.ProblemDetails{
		Status: http.StatusForbidden,
		Cause:  "UNSPECIFIED",
	}
	return httpwrapper.NewResponse(http.StatusForbidden, nil, problemDetails)
}

// TS 29.503 5.2.2.2.4
// SMF Selection Subscription Data Retrieval
func getSmfSelectDataProcedure(supi string, plmnID string, supportedFeatures string) (
	response *models.SmfSelectionSubscriptionData, problemDetails *models.ProblemDetails,
) {
	var querySmfSelectDataParamOpts Nudr.QuerySmfSelectDataParamOpts
	querySmfSelectDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	var body models.SmfSelectionSubscriptionData

	clientAPI, err := createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	udm_context.Getself().CreateSmfSelectionSubsDataforUe(supi, body)

	smfSelectionSubscriptionDataResp, res, err := clientAPI.SMFSelectionSubscriptionDataDocumentApi.
		QuerySmfSelectData(context.Background(), supi, plmnID, &querySmfSelectDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			logger.SdmLog.Warnln(err)
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			return nil, problemDetails
		}
		return
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QuerySmfSelectData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udmUe, ok := udm_context.Getself().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.Getself().NewUdmUe(supi)
		}
		udmUe.SetSmfSelectionSubsData(&smfSelectionSubscriptionDataResp)
		return udmUe.SmfSelSubsData, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		return nil, problemDetails
	}
}

func HandleSubscribeToSharedDataRequest(request *httpwrapper.Request) *httpwrapper.Response {
	// step 1: log
	logger.SdmLog.Infof("Handle SubscribeToSharedData")

	// step 2: retrieve request
	sdmSubscription := request.Body.(models.SdmSubscription)

	// step 3: handle the message
	header, response, problemDetails := subscribeToSharedDataProcedure(&sdmSubscription)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		return httpwrapper.NewResponse(http.StatusCreated, header, response)
	} else if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	} else {
		return httpwrapper.NewResponse(http.StatusNotFound, nil, nil)
	}
}

// TS 29.503 5.2.2.3.3
// Subscription to notifications of shared data change
func subscribeToSharedDataProcedure(sdmSubscription *models.SdmSubscription) (
	header http.Header, response *models.SdmSubscription, problemDetails *models.ProblemDetails,
) {
	cfg := Nudm_SubscriberDataManagement.NewConfiguration()
	udmClientAPI := Nudm_SubscriberDataManagement.NewAPIClient(cfg)

	sdmSubscriptionResp, res, err := udmClientAPI.SubscriptionCreationForSharedDataApi.SubscribeToSharedData(
		context.Background(), *sdmSubscription)
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
		udm_context.Getself().CreateSubstoNotifSharedData(sdmSubscriptionResp.SubscriptionId, &sdmSubscriptionResp)
		reourceUri := udm_context.Getself().GetSDMUri() + "//shared-data-subscriptions/" + sdmSubscriptionResp.SubscriptionId
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

func HandleSubscribeRequest(request *httpwrapper.Request) *httpwrapper.Response {
	// step 1: log
	logger.SdmLog.Infof("Handle Subscribe")

	// step 2: retrieve request
	sdmSubscription := request.Body.(models.SdmSubscription)
	supi := request.Params["supi"]

	// step 3: handle the message
	header, response, problemDetails := subscribeProcedure(&sdmSubscription, supi)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		return httpwrapper.NewResponse(http.StatusCreated, header, response)
	} else if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	} else {
		return httpwrapper.NewResponse(http.StatusNotFound, nil, nil)
	}
}

// TS 29.503 5.2.2.3.2
// Subscription to notifications of data change
func subscribeProcedure(sdmSubscription *models.SdmSubscription, supi string) (
	header http.Header, response *models.SdmSubscription, problemDetails *models.ProblemDetails,
) {
	clientAPI, err := createUDMClientToUDR(supi)
	if err != nil {
		return nil, nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	sdmSubscriptionResp, res, err := clientAPI.SDMSubscriptionsCollectionApi.CreateSdmSubscriptions(
		context.Background(), supi, *sdmSubscription)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			logger.SdmLog.Warnln(err)
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
			logger.SdmLog.Errorf("CreateSdmSubscriptions response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusCreated {
		header = make(http.Header)
		udmUe, _ := udm_context.Getself().UdmUeFindBySupi(supi)
		if udmUe == nil {
			udmUe = udm_context.Getself().NewUdmUe(supi)
		}
		udmUe.CreateSubscriptiontoNotifChange(sdmSubscriptionResp.SubscriptionId, &sdmSubscriptionResp)
		header.Set("Location", udmUe.GetLocationURI2(udm_context.LocationUriSdmSubscription, supi))
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

func HandleUnsubscribeForSharedDataRequest(request *httpwrapper.Request) *httpwrapper.Response {
	logger.SdmLog.Infof("Handle UnsubscribeForSharedData")

	// step 2: retrieve request
	subscriptionID := request.Params["subscriptionId"]
	// step 3: handle the message
	problemDetails := unsubscribeForSharedDataProcedure(subscriptionID)

	// step 4: process the return value from step 3
	if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}

	return httpwrapper.NewResponse(http.StatusNoContent, nil, nil)
}

// TS 29.503 5.2.2.4.3
// Unsubscribe to notifications of data change
func unsubscribeForSharedDataProcedure(subscriptionID string) *models.ProblemDetails {
	cfg := Nudm_SubscriberDataManagement.NewConfiguration()
	udmClientAPI := Nudm_SubscriberDataManagement.NewAPIClient(cfg)

	res, err := udmClientAPI.SubscriptionDeletionForSharedDataApi.UnsubscribeForSharedData(
		context.Background(), subscriptionID)
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

func HandleUnsubscribeRequest(request *httpwrapper.Request) *httpwrapper.Response {
	logger.SdmLog.Infof("Handle Unsubscribe")

	// step 2: retrieve request
	supi := request.Params["supi"]
	subscriptionID := request.Params["subscriptionId"]

	// step 3: handle the message
	problemDetails := unsubscribeProcedure(supi, subscriptionID)

	// step 4: process the return value from step 3
	if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}

	return httpwrapper.NewResponse(http.StatusNoContent, nil, nil)
}

// TS 29.503 5.2.2.4.2
// Unsubscribe to notifications of data change
func unsubscribeProcedure(supi string, subscriptionID string) *models.ProblemDetails {
	clientAPI, err := createUDMClientToUDR(supi)
	if err != nil {
		return openapi.ProblemDetailsSystemFailure(err.Error())
	}

	res, err := clientAPI.SDMSubscriptionDocumentApi.RemovesdmSubscriptions(context.Background(), supi, subscriptionID)
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
			logger.SdmLog.Errorf("RemovesdmSubscriptions response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusNoContent {
		return nil
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "USER_NOT_FOUND",
		}
		return problemDetails
	}
}

func HandleModifyRequest(request *httpwrapper.Request) *httpwrapper.Response {
	// step 1: log
	logger.SdmLog.Infof("Handle Modify")

	// step 2: retrieve request
	sdmSubsModification := request.Body.(models.SdmSubsModification)
	supi := request.Params["supi"]
	subscriptionID := request.Params["subscriptionId"]

	// step 3: handle the message
	response, problemDetails := modifyProcedure(&sdmSubsModification, supi, subscriptionID)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		return httpwrapper.NewResponse(http.StatusOK, nil, response)
	} else if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	problemDetails = &models.ProblemDetails{
		Status: http.StatusForbidden,
		Cause:  "UNSPECIFIED",
	}
	return httpwrapper.NewResponse(http.StatusForbidden, nil, problemDetails)
}

// TS 29.503 5.2.2.7.2
// Modification of a subscription to notifications of data change
func modifyProcedure(sdmSubsModification *models.SdmSubsModification, supi string, subscriptionID string) (
	response *models.SdmSubscription, problemDetails *models.ProblemDetails,
) {
	clientAPI, err := createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	sdmSubscription := models.SdmSubscription{}
	body := Nudr.UpdatesdmsubscriptionsParamOpts{
		SdmSubscription: optional.NewInterface(sdmSubscription),
	}
	res, err := clientAPI.SDMSubscriptionDocumentApi.Updatesdmsubscriptions(
		context.Background(), supi, subscriptionID, &body)
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
			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("Updatesdmsubscriptions response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		return &sdmSubscription, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "USER_NOT_FOUND",
		}

		return nil, problemDetails
	}
}

func HandleModifyForSharedDataRequest(request *httpwrapper.Request) *httpwrapper.Response {
	// step 1: log
	logger.SdmLog.Infof("Handle ModifyForSharedData")

	// step 2: retrieve request
	sdmSubsModification := request.Body.(models.SdmSubsModification)
	supi := request.Params["supi"]
	subscriptionID := request.Params["subscriptionId"]

	// step 3: handle the message
	response, problemDetails := modifyForSharedDataProcedure(&sdmSubsModification, supi, subscriptionID)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		return httpwrapper.NewResponse(http.StatusOK, nil, response)
	} else if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	problemDetails = &models.ProblemDetails{
		Status: http.StatusForbidden,
		Cause:  "UNSPECIFIED",
	}
	return httpwrapper.NewResponse(http.StatusForbidden, nil, problemDetails)
}

// TS 29.503 5.2.2.7.3
// Modification of a subscription to notifications of shared data change
func modifyForSharedDataProcedure(sdmSubsModification *models.SdmSubsModification, supi string,
	subscriptionID string,
) (response *models.SdmSubscription, problemDetails *models.ProblemDetails) {
	clientAPI, err := createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var sdmSubscription models.SdmSubscription
	sdmSubs := models.SdmSubscription{}
	body := Nudr.UpdatesdmsubscriptionsParamOpts{
		SdmSubscription: optional.NewInterface(sdmSubs),
	}

	res, err := clientAPI.SDMSubscriptionDocumentApi.Updatesdmsubscriptions(
		context.Background(), supi, subscriptionID, &body)
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
			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("Updatesdmsubscriptions response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		return &sdmSubscription, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "USER_NOT_FOUND",
		}

		return nil, problemDetails
	}
}

func HandleGetTraceDataRequest(request *httpwrapper.Request) *httpwrapper.Response {
	// step 1: log
	logger.SdmLog.Infof("Handle GetTraceData")

	// step 2: retrieve request
	supi := request.Params["supi"]
	plmnID := request.Query.Get("plmn-id")

	// step 3: handle the message
	response, problemDetails := getTraceDataProcedure(supi, plmnID)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		return httpwrapper.NewResponse(http.StatusOK, nil, response)
	} else if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	problemDetails = &models.ProblemDetails{
		Status: http.StatusForbidden,
		Cause:  "UNSPECIFIED",
	}
	return httpwrapper.NewResponse(http.StatusForbidden, nil, problemDetails)
}

func getTraceDataProcedure(supi string, plmnID string) (
	response *models.TraceData, problemDetails *models.ProblemDetails,
) {
	var body models.TraceData
	var queryTraceDataParamOpts Nudr.QueryTraceDataParamOpts

	clientAPI, err := createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	udm_context.Getself().CreateTraceDataforUe(supi, body)

	traceDataRes, res, err := clientAPI.TraceDataDocumentApi.QueryTraceData(
		context.Background(), supi, plmnID, &queryTraceDataParamOpts)
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

			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryTraceData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udmUe, ok := udm_context.Getself().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.Getself().NewUdmUe(supi)
		}
		udmUe.TraceData = &traceDataRes
		udmUe.TraceDataResponse.TraceData = &traceDataRes

		return udmUe.TraceDataResponse.TraceData, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "USER_NOT_FOUND",
		}

		return nil, problemDetails
	}
}

func HandleGetUeContextInSmfDataRequest(request *httpwrapper.Request) *httpwrapper.Response {
	// step 1: log
	logger.SdmLog.Infof("Handle GetUeContextInSmfData")

	// step 2: retrieve request
	supi := request.Params["supi"]
	supportedFeatures := request.Query.Get("supported-features")

	// step 3: handle the message
	response, problemDetails := getUeContextInSmfDataProcedure(supi, supportedFeatures)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		return httpwrapper.NewResponse(http.StatusOK, nil, response)
	} else if problemDetails != nil {
		return httpwrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	problemDetails = &models.ProblemDetails{
		Status: http.StatusForbidden,
		Cause:  "UNSPECIFIED",
	}
	return httpwrapper.NewResponse(http.StatusForbidden, nil, problemDetails)
}

func getUeContextInSmfDataProcedure(supi string, supportedFeatures string) (
	response *models.UeContextInSmfData, problemDetails *models.ProblemDetails,
) {
	var body models.UeContextInSmfData
	var ueContextInSmfData models.UeContextInSmfData
	var pgwInfoArray []models.PgwInfo
	var querySmfRegListParamOpts Nudr.QuerySmfRegListParamOpts
	querySmfRegListParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	clientAPI, err := createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	pduSessionMap := make(map[string]models.PduSession)
	udm_context.Getself().CreateUeContextInSmfDataforUe(supi, body)

	pdusess, res, err := clientAPI.SMFRegistrationsCollectionApi.QuerySmfRegList(
		context.Background(), supi, &querySmfRegListParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Infoln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Infoln(err)
		} else {
			logger.SdmLog.Infoln(err)
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}

			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QuerySmfRegList response body cannot close: %+v", rspCloseErr)
		}
	}()

	for _, element := range pdusess {
		var pduSession models.PduSession
		pduSession.Dnn = element.Dnn
		pduSession.SmfInstanceId = element.SmfInstanceId
		pduSession.PlmnId = element.PlmnId
		pduSessionMap[strconv.Itoa(int(element.PduSessionId))] = pduSession
	}
	ueContextInSmfData.PduSessions = pduSessionMap

	for _, element := range pdusess {
		var pgwInfo models.PgwInfo
		pgwInfo.Dnn = element.Dnn
		pgwInfo.PgwFqdn = element.PgwFqdn
		pgwInfo.PlmnId = element.PlmnId
		pgwInfoArray = append(pgwInfoArray, pgwInfo)
	}
	ueContextInSmfData.PgwInfo = pgwInfoArray

	if res.StatusCode == http.StatusOK {
		udmUe, ok := udm_context.Getself().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.Getself().NewUdmUe(supi)
		}
		udmUe.UeCtxtInSmfData = &ueContextInSmfData
		return udmUe.UeCtxtInSmfData, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		return nil, problemDetails
	}
}

func getPlmnIDStruct(queryParameters url.Values) (plmnIDStruct *models.PlmnId, problemDetails *models.ProblemDetails) {
	if queryParameters["plmn-id"] != nil {
		plmnIDJson := queryParameters["plmn-id"][0]
		plmnIDStruct := &models.PlmnId{}
		err := json.Unmarshal([]byte(plmnIDJson), plmnIDStruct)
		if err != nil {
			logger.SdmLog.Warnln("Unmarshal Error in targetPlmnListtruct: ", err)
		}
		return plmnIDStruct, nil
	} else {
		problemDetails := &models.ProblemDetails{
			Title:  "Invalid Parameter",
			Status: http.StatusBadRequest,
			Cause:  "No get plmn-id",
		}
		return nil, problemDetails
	}
}

func containDataSetName(dataSetNames []string, target string) bool {
	for _, dataSetName := range dataSetNames {
		if dataSetName == target {
			return true
		}
	}
	return false
}
