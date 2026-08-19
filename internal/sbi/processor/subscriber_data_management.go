package processor

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/udm/SDM"
	Nudr_DataRepository "github.com/free5gc/openapi/udr/DR"
	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/util/metrics/sbi"
)

func (p *Processor) GetAmDataProcedure(c *gin.Context, supi string, plmnID string, supportedFeatures string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	var queryAmDataRequest Nudr_DataRepository.QueryAmDataRequest
	queryAmDataRequest.SupportedFeatures = &supportedFeatures
	queryAmDataRequest.UeId = &supi
	queryAmDataRequest.ServingPlmnId = &plmnID

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	accessAndMobilitySubscriptionDataResp, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.
		QueryAmData(ctx, &queryAmDataRequest)
	if err != nil {
		apiError, ok := err.(openapi.GenericOpenAPIError)
		if ok {
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(apiError.ErrorStatus))
			c.Data(apiError.ErrorStatus, "application/json", apiError.RawBody)
			return
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	if accessAndMobilitySubscriptionDataResp != nil &&
		accessAndMobilitySubscriptionDataResp.Udm_SDM_AccessAndMobilitySubscriptionData != nil {
		udmUe, ok := p.Context().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = p.Context().NewUdmUe(supi)
		}
		udmUe.SetAMSubsriptionData(accessAndMobilitySubscriptionDataResp.Udm_SDM_AccessAndMobilitySubscriptionData)
		c.JSON(http.StatusOK, accessAndMobilitySubscriptionDataResp.Udm_SDM_AccessAndMobilitySubscriptionData)
		return
	}
	c.String(http.StatusInternalServerError, "accessAndMobilitySubscriptionDataResp is nil")
}

func (p *Processor) GetIdTranslationResultProcedure(c *gin.Context, gpsi string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
	}
	var idTranslationResult models.Udm_SDM_IdTranslationResult
	var getIdentityDataRequest Nudr_DataRepository.GetIdentityDataRequest

	getIdentityDataRequest.UeId = &gpsi

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(gpsi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	idTranslationResultResp, err := clientAPI.QueryIdentityDataBySUPIOrGPSIDocumentApi.GetIdentityData(
		ctx, &getIdentityDataRequest)
	if err != nil {
		if apiErr, ok := err.(openapi.GenericOpenAPIError); ok {
			if getIdTransError, ok2 := apiErr.Model().(Nudr_DataRepository.GetIdentityDataError); ok2 {
				problem := getIdTransError.ProblemDetails
				c.Set(sbi.IN_PB_DETAILS_CTX_STR, problem.Cause)
				c.JSON(int(problem.Status), problem)
				return
			}
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	if idList := idTranslationResultResp.Udr_DR_IdentityData; idList != nil && idList.SupiList != nil {
		// GetCorrespondingSupi get corresponding Supi(here IMSI) matching the given Gpsi from the queried SUPI list from UDR
		idTranslationResult.Supi = udm_context.GetCorrespondingSupi(*idList)
		idTranslationResult.Gpsi = gpsi
		c.JSON(http.StatusOK, idTranslationResult)
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
	}
}

func (p *Processor) GetSupiProcedure(c *gin.Context,
	supi string,
	plmnID string,
	dataSetNames []string,
	supportedFeatures string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	if len(dataSetNames) < 2 {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "BAD_REQUEST",
			Detail: "datasetNames must have at least 2 elements",
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	var subscriptionDataSets, subsDataSetBody models.Udm_SDM_SubscriptionDataSets
	var ueContextInSmfDataResp models.Udm_SDM_UeContextInSmfData
	pduSessionMap := make(map[string]models.Udm_SDM_PduSession)
	var pgwInfoArray []models.Udm_SDM_PgwInfo

	var queryAmDataRequest Nudr_DataRepository.QueryAmDataRequest
	var querySmfSelectDataRequest Nudr_DataRepository.QuerySmfSelectDataRequest
	var queryTraceDataRequest Nudr_DataRepository.QueryTraceDataRequest
	var querySmDataRequest Nudr_DataRepository.QuerySmDataRequest

	queryAmDataRequest.SupportedFeatures = &supportedFeatures
	queryAmDataRequest.UeId = &supi
	queryAmDataRequest.ServingPlmnId = &plmnID

	querySmfSelectDataRequest.SupportedFeatures = &supportedFeatures
	querySmfSelectDataRequest.UeId = &supi
	querySmfSelectDataRequest.ServingPlmnId = &plmnID
	p.Context().CreateSubsDataSetsForUe(supi, subsDataSetBody)

	if p.containDataSetName(dataSetNames, string(models.Udm_SDM_DataSetName_AM)) {
		var body models.Udm_SDM_AccessAndMobilitySubscriptionData
		p.Context().CreateAccessMobilitySubsDataForUe(supi, body)

		amDataRsp, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.QueryAmData(
			ctx, &queryAmDataRequest)
		if err != nil {
			apiError, ok := err.(openapi.GenericOpenAPIError)
			if ok {
				c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(apiError.ErrorStatus))
				c.Data(apiError.ErrorStatus, "application/json", apiError.RawBody)
				return
			}
			problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}

		udmUe, ok := p.Context().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = p.Context().NewUdmUe(supi)
		}
		udmUe.SetAMSubsriptionData(amDataRsp.Udm_SDM_AccessAndMobilitySubscriptionData)
		subscriptionDataSets.AmData = amDataRsp.Udm_SDM_AccessAndMobilitySubscriptionData
	}

	if p.containDataSetName(dataSetNames, string(models.Udm_SDM_DataSetName_SMF_SEL)) {
		var smfSelSubsbody models.Udm_SDM_SmfSelectionSubscriptionData
		p.Context().CreateSmfSelectionSubsDataforUe(supi, smfSelSubsbody)

		smfSelDataRsp, err := clientAPI.SMFSelectionSubscriptionDataDocumentApi.QuerySmfSelectData(ctx,
			&querySmfSelectDataRequest)
		if err != nil {
			apiError, ok := err.(openapi.GenericOpenAPIError)
			if ok {
				c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(apiError.ErrorStatus))
				c.Data(apiError.ErrorStatus, "application/json", apiError.RawBody)
				return
			}
			problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}

		udmUe, ok := p.Context().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = p.Context().NewUdmUe(supi)
		}
		udmUe.SetSmfSelectionSubsData(smfSelDataRsp.Udm_SDM_SmfSelectionSubscriptionData)
		subscriptionDataSets.SmfSelData = smfSelDataRsp.Udm_SDM_SmfSelectionSubscriptionData
	}
	if p.containDataSetName(dataSetNames, string(models.Udm_SDM_DataSetName_UEC_SMF)) {
		var UeContextInSmfbody models.Udm_SDM_UeContextInSmfData
		var querySmfRegListRequest Nudr_DataRepository.QuerySmfRegListRequest
		querySmfRegListRequest.SupportedFeatures = &supportedFeatures
		querySmfRegListRequest.UeId = &supi
		p.Context().CreateUeContextInSmfDataforUe(supi, UeContextInSmfbody)

		pdusess, err := clientAPI.SMFRegistrationsCollectionApi.QuerySmfRegList(
			ctx, &querySmfRegListRequest)
		if err != nil {
			apiError, ok := err.(openapi.GenericOpenAPIError)
			if ok {
				c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(apiError.ErrorStatus))
				c.Data(apiError.ErrorStatus, "application/json", apiError.RawBody)
				return
			}
			problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}

		for _, element := range pdusess.Udm_UECM_SmfRegistration {
			var pduSession models.Udm_SDM_PduSession
			pduSession.Dnn = element.Dnn
			pduSession.SmfInstanceId = element.SmfInstanceId
			pduSession.PlmnId = element.PlmnId
			pduSessionMap[strconv.Itoa(int(element.PduSessionId))] = pduSession
		}
		ueContextInSmfDataResp.PduSessions = pduSessionMap

		for _, element := range pdusess.Udm_UECM_SmfRegistration {
			var pgwInfo models.Udm_SDM_PgwInfo
			pgwInfo.Dnn = element.Dnn
			pgwInfo.PgwFqdn = element.PgwFqdn
			pgwInfo.PlmnId = element.PlmnId
			pgwInfoArray = append(pgwInfoArray, pgwInfo)
		}
		ueContextInSmfDataResp.PgwInfo = pgwInfoArray

		udmUe, ok := p.Context().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = p.Context().NewUdmUe(supi)
		}
		udmUe.UeCtxtInSmfData = &ueContextInSmfDataResp
		subscriptionDataSets.UecSmfData = &ueContextInSmfDataResp
	}

	// TODO: UE Context in SMSF Data
	// if containDataSetName(dataSetNames, string(models.Udm_SDM_DataSetName_UEC_SMSF)) {
	// }

	// TODO: SMS Subscription Data
	// if containDataSetName(dataSetNames, string(models.Udm_SDM_DataSetName_SMS_SUB)) {
	// }

	if p.containDataSetName(dataSetNames, string(models.Udm_SDM_DataSetName_SM)) {
		querySmDataRequest.UeId = &supi
		querySmDataRequest.ServingPlmnId = &plmnID
		sessionManagementSubscriptionDataRsp, err := clientAPI.SessionManagementSubscriptionDataApi.
			QuerySmData(ctx, &querySmDataRequest)
		if err != nil {
			apiError, ok := err.(openapi.GenericOpenAPIError)
			if ok {
				c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(apiError.ErrorStatus))
				c.Data(apiError.ErrorStatus, "application/json", apiError.RawBody)
				return
			}
			problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}

		udmUe, ok := p.Context().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = p.Context().NewUdmUe(supi)
		}
		smData, _, _, _ := p.Context().
			ManageSmData(sessionManagementSubscriptionDataRsp.Udm_SDM_SmSubsData.IndividualSmSubsData, "", "")
		udmUe.SetSMSubsData(smData)
		subscriptionDataSets.SmData = sessionManagementSubscriptionDataRsp.Udm_SDM_SmSubsData
	}

	if p.containDataSetName(dataSetNames, string(models.Udm_SDM_DataSetName_TRACE)) {
		var TraceDatabody models.TraceData
		p.Context().CreateTraceDataforUe(supi, TraceDatabody)

		queryTraceDataRequest.UeId = &supi
		queryTraceDataRequest.ServingPlmnId = &plmnID
		traceDataRsp, err := clientAPI.TraceDataDocumentApi.QueryTraceData(
			ctx, &queryTraceDataRequest)
		if err != nil {
			apiError, ok := err.(openapi.GenericOpenAPIError)
			if ok {
				c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(apiError.ErrorStatus))
				c.Data(apiError.ErrorStatus, "application/json", apiError.RawBody)
				return
			}
			problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}

		udmUe, ok := p.Context().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = p.Context().NewUdmUe(supi)
		}
		udmUe.TraceData = traceDataRsp.TraceData
		udmUe.TraceDataResponse.TraceData = traceDataRsp.TraceData
		subscriptionDataSets.TraceData = traceDataRsp.TraceData
	}

	// TODO: SMS Management Subscription Data
	// if containDataSetName(dataSetNames, string(models.Udm_SDM_DataSetName_SMS_MNG)) {
	// }

	c.JSON(http.StatusOK, subscriptionDataSets)
}

func (p *Processor) GetSharedDataProcedure(c *gin.Context, sharedDataIds []string, supportedFeatures string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR("")
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	var getSharedDataRequest Nudr_DataRepository.GetSharedDataRequest
	getSharedDataRequest.SupportedFeatures = &supportedFeatures
	getSharedDataRequest.SharedDataIds = sharedDataIds

	sharedDataResp, err := clientAPI.RetrievalOfSharedDataApi.GetSharedData(ctx,
		&getSharedDataRequest)
	if err != nil {
		if apiErr, ok := err.(openapi.GenericOpenAPIError); ok {
			if getShareDataError, ok2 := apiErr.Model().(Nudr_DataRepository.GetSharedDataError); ok2 {
				problem := getShareDataError.ProblemDetails
				c.Set(sbi.IN_PB_DETAILS_CTX_STR, problem.Cause)
				c.JSON(int(problem.Status), problem)
				return
			}
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	p.Context().SharedSubsDataMap = udm_context.MappingSharedData(sharedDataResp.Udm_SDM_SharedData)
	sharedData := udm_context.ObtainRequiredSharedData(sharedDataIds, sharedDataResp.Udm_SDM_SharedData)
	c.JSON(http.StatusOK, sharedData)
}

func (p *Processor) GetSmDataProcedure(
	c *gin.Context,
	supi string,
	plmnID string,
	Dnn string,
	Snssai string,
	supportedFeatures string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	logger.SdmLog.Infof("getSmDataProcedure: SUPI[%s] PLMNID[%s] DNN[%s] SNssai[%s]", supi, plmnID, Dnn, Snssai)

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		logger.ProcLog.Errorf("CreateUDMClientToUDR Error: %+v", err)
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	var modelSnassai models.Snssai
	if Snssai != "" {
		if errUnmarshal := json.Unmarshal([]byte(Snssai), &modelSnassai); errUnmarshal != nil {
			logger.ProcLog.Errorf("modelSnassai Unmarshal Error: %+v", errUnmarshal)
			problemDetail := models.ProblemDetails{
				Status: http.StatusBadRequest,
				Detail: "The 'single-nssai' parameter is malformed.",
				Cause:  "INVALID_IE_VALUE",
			}
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
			c.JSON(int(problemDetail.Status), problemDetail)
			return
		}
	}

	var querySmDataRequest Nudr_DataRepository.QuerySmDataRequest
	querySmDataRequest.SingleNssai = &modelSnassai
	querySmDataRequest.UeId = &supi
	querySmDataRequest.ServingPlmnId = &plmnID

	sessionManagementSubscriptionDataResp, err := clientAPI.SessionManagementSubscriptionDataApi.
		QuerySmData(ctx, &querySmDataRequest)
	if err != nil {
		logger.ProcLog.Errorf("QuerySmData Error: %+v", err)
		apiError, ok := err.(openapi.GenericOpenAPIError)
		if ok {
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(apiError.ErrorStatus))
			c.Data(apiError.ErrorStatus, "application/json", apiError.RawBody)
			return
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	udmUe, ok := p.Context().UdmUeFindBySupi(supi)
	if !ok {
		udmUe = p.Context().NewUdmUe(supi)
	}
	smData, snssaikey, AllDnnConfigsbyDnn, AllDnns := p.Context().ManageSmData(
		sessionManagementSubscriptionDataResp.Udm_SDM_SmSubsData.IndividualSmSubsData, Snssai, Dnn)
	udmUe.SetSMSubsData(smData)

	rspSMSubDataList := make([]models.Udm_SDM_SessionManagementSubscriptionData, 0, 4)

	udmUe.SmSubsDataLock.RLock()
	for _, eachSMSubData := range udmUe.SessionManagementSubsData {
		rspSMSubDataList = append(rspSMSubDataList, eachSMSubData)
	}
	udmUe.SmSubsDataLock.RUnlock()

	switch {
	case Snssai == "" && Dnn == "":
		c.JSON(http.StatusOK, AllDnns)
	case Snssai != "" && Dnn == "":
		udmUe.SmSubsDataLock.RLock()
		defer udmUe.SmSubsDataLock.RUnlock()
		c.JSON(http.StatusOK, udmUe.SessionManagementSubsData[snssaikey].DnnConfigurations)
	case Snssai == "" && Dnn != "":
		c.JSON(http.StatusOK, AllDnnConfigsbyDnn)
	case Snssai != "" && Dnn != "":
		c.JSON(http.StatusOK, rspSMSubDataList)
	default:
		udmUe.SmSubsDataLock.RLock()
		defer udmUe.SmSubsDataLock.RUnlock()
		c.JSON(http.StatusOK, udmUe.SessionManagementSubsData)
	}
}

func (p *Processor) GetNssaiProcedure(c *gin.Context, supi string, plmnID string, supportedFeatures string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	var queryAmDataRequest Nudr_DataRepository.QueryAmDataRequest
	queryAmDataRequest.SupportedFeatures = &supportedFeatures
	queryAmDataRequest.UeId = &supi
	queryAmDataRequest.ServingPlmnId = &plmnID

	var nssaiResp models.Udm_SDM_Nssai
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	accessAndMobilitySubscriptionDataResp, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.
		QueryAmData(ctx, &queryAmDataRequest)
	if err != nil {
		apiError, ok := err.(openapi.GenericOpenAPIError)
		if ok {
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(apiError.ErrorStatus))
			c.Data(apiError.ErrorStatus, "application/json", apiError.RawBody)
			return
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	nssaiResp = *accessAndMobilitySubscriptionDataResp.Udm_SDM_AccessAndMobilitySubscriptionData.Nssai

	udmUe, ok := p.Context().UdmUeFindBySupi(supi)
	if !ok {
		udmUe = p.Context().NewUdmUe(supi)
	}
	udmUe.Nssai = &nssaiResp
	c.JSON(http.StatusOK, udmUe.Nssai)
}

func (p *Processor) GetSmfSelectDataProcedure(c *gin.Context, supi string, plmnID string, supportedFeatures string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	var querySmfSelectDataRequest Nudr_DataRepository.QuerySmfSelectDataRequest
	querySmfSelectDataRequest.SupportedFeatures = &supportedFeatures
	querySmfSelectDataRequest.UeId = &supi
	querySmfSelectDataRequest.ServingPlmnId = &plmnID

	var body models.Udm_SDM_SmfSelectionSubscriptionData

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	p.Context().CreateSmfSelectionSubsDataforUe(supi, body)

	smfSelectionSubscriptionDataResp, err := clientAPI.SMFSelectionSubscriptionDataDocumentApi.
		QuerySmfSelectData(ctx, &querySmfSelectDataRequest)
	if err != nil {
		apiError, ok := err.(openapi.GenericOpenAPIError)
		if ok {
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(apiError.ErrorStatus))
			c.Data(apiError.ErrorStatus, "application/json", apiError.RawBody)
			return
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	udmUe, ok := p.Context().UdmUeFindBySupi(supi)
	if !ok {
		udmUe = p.Context().NewUdmUe(supi)
	}
	udmUe.SetSmfSelectionSubsData(smfSelectionSubscriptionDataResp.Udm_SDM_SmfSelectionSubscriptionData)
	c.JSON(http.StatusOK, udmUe.SmfSelSubsData)
}

func (p *Processor) SubscribeToSharedDataProcedure(c *gin.Context, sdmSubscription *models.Udm_SDM_SdmSubscription) {
	if sdmSubscription.NfInstanceId == "" {
		logger.SdmLog.Warnf("Missing mandatory parameter: nfInstanceId")
		problemDetails := models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "MANDATORY_IE_MISSING",
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(http.StatusBadRequest, problemDetails)
		return
	}

	if _, err := uuid.Parse(sdmSubscription.NfInstanceId); err != nil {
		logger.SdmLog.Warnf("Invalid nfInstanceId format: %s", sdmSubscription.NfInstanceId)
		problemDetails := models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "INVALID_IE_VALUE",
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(http.StatusBadRequest, problemDetails)
		return
	}

	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDM_SDM, models.Nrf_NFMgmt_NFType_UDM)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	var subscibeToShareDataRequest SDM.SubscribeToSharedDataRequest
	subscibeToShareDataRequest.RequestBody = sdmSubscription
	udmClientAPI := p.Consumer().GetSDMClient("subscribeToSharedData")

	sdmSubscriptionResp, err := udmClientAPI.SubscriptionCreationForSharedDataApi.SubscribeToSharedData(
		ctx, &subscibeToShareDataRequest)
	if err != nil {
		if apiErr, ok := err.(openapi.GenericOpenAPIError); ok {
			if subToShareDataErr, ok2 := apiErr.Model().(SDM.SubscribeToSharedDataError); ok2 {
				problem := subToShareDataErr.ProblemDetails
				c.Set(sbi.IN_PB_DETAILS_CTX_STR, problem.Cause)
				c.JSON(int(problem.Status), problem)
				return
			}
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	if sdmSubscriptionResp == nil || sdmSubscriptionResp.Udm_SDM_SdmSubscription == nil {
		problemDetails := openapi.ProblemDetailsSystemFailure("UDM returned an empty shared-data subscription")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	p.Context().CreateSubstoNotifSharedData(sdmSubscriptionResp.Udm_SDM_SdmSubscription.SubscriptionId,
		sdmSubscriptionResp.Udm_SDM_SdmSubscription)
	reourceUri := p.Context().
		GetSDMUri() +
		"//shared-data-subscriptions/" + sdmSubscriptionResp.Udm_SDM_SdmSubscription.SubscriptionId
	c.Header("Location", reourceUri)
	c.JSON(http.StatusOK, sdmSubscriptionResp.Udm_SDM_SdmSubscription)
}

func (p *Processor) SubscribeProcedure(c *gin.Context, sdmSubscription *models.Udm_SDM_SdmSubscription, supi string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	var createSdmSubscriptionsRequest Nudr_DataRepository.CreateSdmSubscriptionsRequest
	createSdmSubscriptionsRequest.RequestBody = sdmSubscription
	createSdmSubscriptionsRequest.UeId = &supi
	sdmSubscriptionResp, err := clientAPI.SDMSubscriptionsCollectionApi.CreateSdmSubscriptions(
		ctx, &createSdmSubscriptionsRequest)
	if err != nil {
		apiError, ok := err.(openapi.GenericOpenAPIError)
		if ok {
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(apiError.ErrorStatus))
			c.Data(apiError.ErrorStatus, "application/json", apiError.RawBody)
			return
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	if sdmSubscriptionResp == nil || sdmSubscriptionResp.Udm_SDM_SdmSubscription == nil {
		problemDetails := openapi.ProblemDetailsSystemFailure("UDR returned an empty SDM subscription")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	udmUe, _ := p.Context().UdmUeFindBySupi(supi)
	if udmUe == nil {
		udmUe = p.Context().NewUdmUe(supi)
	}
	udmUe.CreateSubscriptiontoNotifChange(sdmSubscriptionResp.Udm_SDM_SdmSubscription.SubscriptionId,
		sdmSubscriptionResp.Udm_SDM_SdmSubscription)
	c.Header("Location", udmUe.GetLocationURI2(udm_context.LocationUriSdmSubscription, supi))
	c.JSON(http.StatusCreated, sdmSubscriptionResp.Udm_SDM_SdmSubscription)
}

func (p *Processor) UnsubscribeForSharedDataProcedure(c *gin.Context, subscriptionID string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDM_SDM, models.Nrf_NFMgmt_NFType_UDM)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	udmClientAPI := p.Consumer().GetSDMClient("unsubscribeForSharedData")
	var unsubscribeForSharedDataRequest SDM.UnsubscribeForSharedDataRequest
	unsubscribeForSharedDataRequest.SubscriptionId = &subscriptionID
	_, err = udmClientAPI.SubscriptionDeletionForSharedDataApi.UnsubscribeForSharedData(
		ctx, &unsubscribeForSharedDataRequest)
	if err != nil {
		if apiErr, ok := err.(openapi.GenericOpenAPIError); ok {
			if subToShareDataErr, ok2 := apiErr.Model().(SDM.UnsubscribeForSharedDataError); ok2 {
				problem := subToShareDataErr.ProblemDetails
				c.Set(sbi.IN_PB_DETAILS_CTX_STR, problem.Cause)
				c.JSON(int(problem.Status), problem)
				return
			}
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	c.Status(http.StatusNoContent)
}

func (p *Processor) UnsubscribeProcedure(c *gin.Context, supi string, subscriptionID string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	var removesdmSubscriptionRequest Nudr_DataRepository.RemovesdmSubscriptionsRequest
	removesdmSubscriptionRequest.UeId = &supi
	removesdmSubscriptionRequest.SubsId = &subscriptionID
	_, err = clientAPI.SDMSubscriptionDocumentApi.RemovesdmSubscriptions(ctx, &removesdmSubscriptionRequest)
	if err != nil {
		if apiErr, ok := err.(openapi.GenericOpenAPIError); ok {
			if removeSubErr, ok2 := apiErr.Model().(Nudr_DataRepository.RemovesdmSubscriptionsError); ok2 {
				problem := removeSubErr.ProblemDetails
				c.Set(sbi.IN_PB_DETAILS_CTX_STR, problem.Cause)
				c.JSON(int(problem.Status), problem)
				return
			}
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	c.Status(http.StatusNoContent)
}

func (p *Processor) ModifyProcedure(c *gin.Context,
	sdmSubsModification *models.Udm_SDM_SdmSubsModification,
	supi string,
	subscriptionID string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	sdmSubscription := models.Udm_SDM_SdmSubscription{}
	var updatesdmsubscriptionsRequest Nudr_DataRepository.UpdatesdmsubscriptionsRequest
	updatesdmsubscriptionsRequest.RequestBody = &sdmSubscription
	updatesdmsubscriptionsRequest.SubsId = &subscriptionID
	updatesdmsubscriptionsRequest.UeId = &supi

	_, err = clientAPI.SDMSubscriptionDocumentApi.Updatesdmsubscriptions(
		ctx, &updatesdmsubscriptionsRequest)
	if err != nil {
		if apiErr, ok := err.(openapi.GenericOpenAPIError); ok {
			if updateSubErr, ok2 := apiErr.Model().(Nudr_DataRepository.UpdatesdmsubscriptionsError); ok2 {
				problem := updateSubErr.ProblemDetails
				c.Set(sbi.IN_PB_DETAILS_CTX_STR, problem.Cause)
				c.JSON(int(problem.Status), problem)
				return
			}
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	c.JSON(http.StatusOK, sdmSubscription)
}

// TS 29.503 5.2.2.7.3
// Modification of a subscription to notifications of shared data change
func (p *Processor) ModifyForSharedDataProcedure(c *gin.Context,
	sdmSubsModification *models.Udm_SDM_SdmSubsModification,
	supi string,
	subscriptionID string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	var sdmSubscription models.Udm_SDM_SdmSubscription
	sdmSubs := models.Udm_SDM_SdmSubscription{}
	var updatesdmsubscriptionsRequest Nudr_DataRepository.UpdatesdmsubscriptionsRequest
	updatesdmsubscriptionsRequest.SubsId = &subscriptionID
	updatesdmsubscriptionsRequest.UeId = &supi
	updatesdmsubscriptionsRequest.RequestBody = &sdmSubs

	_, err = clientAPI.SDMSubscriptionDocumentApi.Updatesdmsubscriptions(
		ctx, &updatesdmsubscriptionsRequest)
	if err != nil {
		if apiErr, ok := err.(openapi.GenericOpenAPIError); ok {
			if updateShareSubErr, ok2 := apiErr.Model().(Nudr_DataRepository.UpdatesdmsubscriptionsError); ok2 {
				problem := updateShareSubErr.ProblemDetails
				c.Set(sbi.IN_PB_DETAILS_CTX_STR, problem.Cause)
				c.JSON(int(problem.Status), problem)
				return
			}
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	c.JSON(http.StatusOK, sdmSubscription)
}

func (p *Processor) GetTraceDataProcedure(c *gin.Context, supi string, plmnID string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	var body models.TraceData
	var queryTraceDataRequest Nudr_DataRepository.QueryTraceDataRequest
	queryTraceDataRequest.UeId = &supi
	queryTraceDataRequest.ServingPlmnId = &plmnID

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		logger.SdmLog.Errorf("Create UDR client for trace-data query failed: %v", err)
		problemDetails := openapi.ProblemDetailsSystemFailure("Upstream request failed")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	p.Context().CreateTraceDataforUe(supi, body)

	traceDataRes, err := clientAPI.TraceDataDocumentApi.QueryTraceData(
		ctx, &queryTraceDataRequest)
	if err != nil {
		apiError, ok := err.(openapi.GenericOpenAPIError)
		if ok {
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(apiError.ErrorStatus))
			c.Data(apiError.ErrorStatus, "application/json", apiError.RawBody)
			return
		}
		logger.SdmLog.Errorf("Trace-data query failed: %v", err)
		problemDetails := openapi.ProblemDetailsSystemFailure("Upstream request failed")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	udmUe, ok := p.Context().UdmUeFindBySupi(supi)
	if !ok {
		udmUe = p.Context().NewUdmUe(supi)
	}
	udmUe.TraceData = traceDataRes.TraceData
	udmUe.TraceDataResponse.TraceData = traceDataRes.TraceData

	c.JSON(http.StatusOK, udmUe.TraceDataResponse.TraceData)
}

func (p *Processor) GetUeContextInSmfDataProcedure(c *gin.Context, supi string, supportedFeatures string) {
	var body models.Udm_SDM_UeContextInSmfData
	var ueContextInSmfData models.Udm_SDM_UeContextInSmfData
	var pgwInfoArray []models.Udm_SDM_PgwInfo
	var querySmfRegListRequest Nudr_DataRepository.QuerySmfRegListRequest
	querySmfRegListRequest.SupportedFeatures = &supportedFeatures
	querySmfRegListRequest.UeId = &supi

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	pduSessionMap := make(map[string]models.Udm_SDM_PduSession)
	p.Context().CreateUeContextInSmfDataforUe(supi, body)

	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	pdusessRes, err := clientAPI.SMFRegistrationsCollectionApi.QuerySmfRegList(
		ctx, &querySmfRegListRequest)
	if err != nil {
		apiError, ok := err.(openapi.GenericOpenAPIError)
		if ok {
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(apiError.ErrorStatus))
			c.Data(apiError.ErrorStatus, "application/json", apiError.RawBody)
			return
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	for _, element := range pdusessRes.Udm_UECM_SmfRegistration {
		var pduSession models.Udm_SDM_PduSession
		pduSession.Dnn = element.Dnn
		pduSession.SmfInstanceId = element.SmfInstanceId
		pduSession.PlmnId = element.PlmnId
		pduSessionMap[strconv.Itoa(int(element.PduSessionId))] = pduSession
	}
	ueContextInSmfData.PduSessions = pduSessionMap

	for _, element := range pdusessRes.Udm_UECM_SmfRegistration {
		var pgwInfo models.Udm_SDM_PgwInfo
		pgwInfo.Dnn = element.Dnn
		pgwInfo.PgwFqdn = element.PgwFqdn
		pgwInfo.PlmnId = element.PlmnId
		pgwInfoArray = append(pgwInfoArray, pgwInfo)
	}
	ueContextInSmfData.PgwInfo = pgwInfoArray

	udmUe, ok := p.Context().UdmUeFindBySupi(supi)
	if !ok {
		udmUe = p.Context().NewUdmUe(supi)
	}
	udmUe.UeCtxtInSmfData = &ueContextInSmfData
	c.JSON(http.StatusOK, udmUe.UeCtxtInSmfData)
}

func (p *Processor) containDataSetName(dataSetNames []string, target string) bool {
	for _, dataSetName := range dataSetNames {
		if dataSetName == target {
			return true
		}
	}
	return false
}
