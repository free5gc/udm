package processor

import (
	"net/http"
	"strconv"

	"github.com/antihax/optional"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/Nudr_DataRepository"
	"github.com/free5gc/openapi/models"
	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/logger"
)

func (p *Processor) GetAmDataProcedure(supi string, plmnID string, supportedFeatures string) (
	response *models.AccessAndMobilitySubscriptionData, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	var queryAmDataParamOpts Nudr_DataRepository.QueryAmDataParamOpts
	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	clientAPI, err := p.consumer.CreateUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	accessAndMobilitySubscriptionDataResp, res, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.
		QueryAmData(ctx, supi, plmnID, &queryAmDataParamOpts)
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
		udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
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

func (p *Processor) GetIdTranslationResultProcedure(gpsi string) (response *models.IdTranslationResult,
	problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	var idTranslationResult models.IdTranslationResult
	var getIdentityDataParamOpts Nudr_DataRepository.GetIdentityDataParamOpts

	clientAPI, err := p.consumer.CreateUDMClientToUDR(gpsi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	idTranslationResultResp, res, err := clientAPI.QueryIdentityDataBySUPIOrGPSIDocumentApi.GetIdentityData(
		ctx, gpsi, &getIdentityDataParamOpts)
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

func (p *Processor) GetSupiProcedure(supi string, plmnID string, dataSetNames []string, supportedFeatures string) (
	response *models.SubscriptionDataSets, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	if len(dataSetNames) < 2 {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "BAD_REQUEST",
			Detail: "datasetNames must have at least 2 elements",
		}
		return nil, problemDetails
	}

	clientAPI, err := p.consumer.CreateUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var subscriptionDataSets, subsDataSetBody models.SubscriptionDataSets
	var ueContextInSmfDataResp models.UeContextInSmfData
	pduSessionMap := make(map[string]models.PduSession)
	var pgwInfoArray []models.PgwInfo

	var queryAmDataParamOpts Nudr_DataRepository.QueryAmDataParamOpts
	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	var querySmfSelectDataParamOpts Nudr_DataRepository.QuerySmfSelectDataParamOpts
	var queryTraceDataParamOpts Nudr_DataRepository.QueryTraceDataParamOpts
	var querySmDataParamOpts Nudr_DataRepository.QuerySmDataParamOpts

	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	querySmfSelectDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	udm_context.GetSelf().CreateSubsDataSetsForUe(supi, subsDataSetBody)

	if p.containDataSetName(dataSetNames, string(models.DataSetName_AM)) {
		var body models.AccessAndMobilitySubscriptionData
		udm_context.GetSelf().CreateAccessMobilitySubsDataForUe(supi, body)

		amData, res, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.QueryAmData(
			ctx, supi, plmnID, &queryAmDataParamOpts)
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
			udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.GetSelf().NewUdmUe(supi)
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

	if p.containDataSetName(dataSetNames, string(models.DataSetName_SMF_SEL)) {
		var smfSelSubsbody models.SmfSelectionSubscriptionData
		udm_context.GetSelf().CreateSmfSelectionSubsDataforUe(supi, smfSelSubsbody)

		smfSelData, res, err := clientAPI.SMFSelectionSubscriptionDataDocumentApi.QuerySmfSelectData(ctx,
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
			udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.GetSelf().NewUdmUe(supi)
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

	if p.containDataSetName(dataSetNames, string(models.DataSetName_UEC_SMF)) {
		var UeContextInSmfbody models.UeContextInSmfData
		var querySmfRegListParamOpts Nudr_DataRepository.QuerySmfRegListParamOpts
		querySmfRegListParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
		udm_context.GetSelf().CreateUeContextInSmfDataforUe(supi, UeContextInSmfbody)

		pdusess, res, err := clientAPI.SMFRegistrationsCollectionApi.QuerySmfRegList(
			ctx, supi, &querySmfRegListParamOpts)
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
			udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.GetSelf().NewUdmUe(supi)
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

	if p.containDataSetName(dataSetNames, string(models.DataSetName_SM)) {
		sessionManagementSubscriptionData, res, err := clientAPI.SessionManagementSubscriptionDataApi.
			QuerySmData(ctx, supi, plmnID, &querySmDataParamOpts)
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
			udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.GetSelf().NewUdmUe(supi)
			}
			smData, _, _, _ := udm_context.GetSelf().ManageSmData(sessionManagementSubscriptionData, "", "")
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

	if p.containDataSetName(dataSetNames, string(models.DataSetName_TRACE)) {
		var TraceDatabody models.TraceData
		udm_context.GetSelf().CreateTraceDataforUe(supi, TraceDatabody)

		traceData, res, err := clientAPI.TraceDataDocumentApi.QueryTraceData(
			ctx, supi, plmnID, &queryTraceDataParamOpts)
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
			udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.GetSelf().NewUdmUe(supi)
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

func (p *Processor) GetSharedDataProcedure(sharedDataIds []string, supportedFeatures string) (
	response []models.SharedData, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	clientAPI, err := p.consumer.CreateUDMClientToUDR("")
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var getSharedDataParamOpts Nudr_DataRepository.GetSharedDataParamOpts
	getSharedDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	sharedDataResp, res, err := clientAPI.RetrievalOfSharedDataApi.GetSharedData(ctx, sharedDataIds,
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
		udm_context.GetSelf().SharedSubsDataMap = udm_context.MappingSharedData(sharedDataResp)
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

func (p *Processor) GetSmDataProcedure(
	supi string,
	plmnID string,
	Dnn string,
	Snssai string,
	supportedFeatures string,
) (
	response interface{}, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	logger.SdmLog.Infof("getSmDataProcedure: SUPI[%s] PLMNID[%s] DNN[%s] SNssai[%s]", supi, plmnID, Dnn, Snssai)

	clientAPI, err := p.consumer.CreateUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var querySmDataParamOpts Nudr_DataRepository.QuerySmDataParamOpts
	querySmDataParamOpts.SingleNssai = optional.NewInterface(Snssai)

	sessionManagementSubscriptionDataResp, res, err := clientAPI.SessionManagementSubscriptionDataApi.
		QuerySmData(ctx, supi, plmnID, &querySmDataParamOpts)
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
		udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
		}
		smData, snssaikey, AllDnnConfigsbyDnn, AllDnns := udm_context.GetSelf().ManageSmData(
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

func (p *Processor) GetNssaiProcedure(supi string, plmnID string, supportedFeatures string) (
	*models.Nssai, *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	var queryAmDataParamOpts Nudr_DataRepository.QueryAmDataParamOpts
	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	var nssaiResp models.Nssai
	clientAPI, err := p.consumer.CreateUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	accessAndMobilitySubscriptionDataResp, res, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.
		QueryAmData(ctx, supi, plmnID, &queryAmDataParamOpts)
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
		udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
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

func (p *Processor) GetSmfSelectDataProcedure(supi string, plmnID string, supportedFeatures string) (
	response *models.SmfSelectionSubscriptionData, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	var querySmfSelectDataParamOpts Nudr_DataRepository.QuerySmfSelectDataParamOpts
	querySmfSelectDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	var body models.SmfSelectionSubscriptionData

	clientAPI, err := p.consumer.CreateUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	udm_context.GetSelf().CreateSmfSelectionSubsDataforUe(supi, body)

	smfSelectionSubscriptionDataResp, res, err := clientAPI.SMFSelectionSubscriptionDataDocumentApi.
		QuerySmfSelectData(ctx, supi, plmnID, &querySmfSelectDataParamOpts)
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
		return nil, problemDetails
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QuerySmfSelectData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
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

func (p *Processor) SubscribeToSharedDataProcedure(sdmSubscription *models.SdmSubscription) (
	header http.Header, response *models.SdmSubscription, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDM_SDM, models.NfType_UDM)
	if err != nil {
		return nil, nil, pd
	}

	udmClientAPI := p.consumer.GetSDMClient("subscribeToSharedData")

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
		reourceUri := udm_context.GetSelf().
			GetSDMUri() +
			"//shared-data-subscriptions/" + sdmSubscriptionResp.SubscriptionId
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

func (p *Processor) SubscribeProcedure(sdmSubscription *models.SdmSubscription, supi string) (
	header http.Header, response *models.SdmSubscription, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, nil, pd
	}
	clientAPI, err := p.consumer.CreateUDMClientToUDR(supi)
	if err != nil {
		return nil, nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	sdmSubscriptionResp, res, err := clientAPI.SDMSubscriptionsCollectionApi.CreateSdmSubscriptions(
		ctx, supi, *sdmSubscription)
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
		udmUe, _ := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if udmUe == nil {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
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

func (p *Processor) UnsubscribeForSharedDataProcedure(subscriptionID string) *models.ProblemDetails {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDM_SDM, models.NfType_UDM)
	if err != nil {
		return pd
	}

	udmClientAPI := p.consumer.GetSDMClient("unsubscribeForSharedData")

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

func (p *Processor) UnsubscribeProcedure(supi string, subscriptionID string) *models.ProblemDetails {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return pd
	}
	clientAPI, err := p.consumer.CreateUDMClientToUDR(supi)
	if err != nil {
		return openapi.ProblemDetailsSystemFailure(err.Error())
	}

	res, err := clientAPI.SDMSubscriptionDocumentApi.RemovesdmSubscriptions(ctx, supi, subscriptionID)
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

func (p *Processor) ModifyProcedure(
	sdmSubsModification *models.SdmSubsModification,
	supi string,
	subscriptionID string,
) (
	response *models.SdmSubscription, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	clientAPI, err := p.consumer.CreateUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	sdmSubscription := models.SdmSubscription{}
	body := Nudr_DataRepository.UpdatesdmsubscriptionsParamOpts{
		SdmSubscription: optional.NewInterface(sdmSubscription),
	}

	res, err := clientAPI.SDMSubscriptionDocumentApi.Updatesdmsubscriptions(
		ctx, supi, subscriptionID, &body)
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

// TS 29.503 5.2.2.7.3
// Modification of a subscription to notifications of shared data change
func (p *Processor) ModifyForSharedDataProcedure(sdmSubsModification *models.SdmSubsModification, supi string,
	subscriptionID string,
) (response *models.SdmSubscription, problemDetails *models.ProblemDetails) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	clientAPI, err := p.consumer.CreateUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var sdmSubscription models.SdmSubscription
	sdmSubs := models.SdmSubscription{}
	body := Nudr_DataRepository.UpdatesdmsubscriptionsParamOpts{
		SdmSubscription: optional.NewInterface(sdmSubs),
	}

	res, err := clientAPI.SDMSubscriptionDocumentApi.Updatesdmsubscriptions(
		ctx, supi, subscriptionID, &body)
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

func (p *Processor) GetTraceDataProcedure(supi string, plmnID string) (
	response *models.TraceData, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	var body models.TraceData
	var queryTraceDataParamOpts Nudr_DataRepository.QueryTraceDataParamOpts

	clientAPI, err := p.consumer.CreateUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	udm_context.GetSelf().CreateTraceDataforUe(supi, body)

	traceDataRes, res, err := clientAPI.TraceDataDocumentApi.QueryTraceData(
		ctx, supi, plmnID, &queryTraceDataParamOpts)
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
		udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
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

func (p *Processor) GetUeContextInSmfDataProcedure(supi string, supportedFeatures string) (
	response *models.UeContextInSmfData, problemDetails *models.ProblemDetails,
) {
	var body models.UeContextInSmfData
	var ueContextInSmfData models.UeContextInSmfData
	var pgwInfoArray []models.PgwInfo
	var querySmfRegListParamOpts Nudr_DataRepository.QuerySmfRegListParamOpts
	querySmfRegListParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	clientAPI, err := p.consumer.CreateUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	pduSessionMap := make(map[string]models.PduSession)
	udm_context.GetSelf().CreateUeContextInSmfDataforUe(supi, body)

	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}

	pdusess, res, err := clientAPI.SMFRegistrationsCollectionApi.QuerySmfRegList(
		ctx, supi, &querySmfRegListParamOpts)
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
		udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
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

func (p *Processor) containDataSetName(dataSetNames []string, target string) bool {
	for _, dataSetName := range dataSetNames {
		if dataSetName == target {
			return true
		}
	}
	return false
}
