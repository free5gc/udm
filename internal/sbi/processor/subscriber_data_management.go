package processor

import (
	"net/http"
	"strconv"

	"github.com/antihax/optional"
	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/Nudr_DataRepository"
	"github.com/free5gc/openapi/models"
	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/logger"
)

func (p *Processor) GetAmDataProcedure(c *gin.Context, supi string, plmnID string, supportedFeatures string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	var queryAmDataParamOpts Nudr_DataRepository.QueryAmDataParamOpts
	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	accessAndMobilitySubscriptionDataResp, res, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.
		QueryAmData(ctx, supi, plmnID, &queryAmDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Errorf(err.Error())
		} else if err.Error() != res.Status {
			logger.SdmLog.Errorf("Response State: %+v", err.Error())
		} else {
			problemDetails := &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryAmData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udmUe, ok := p.Context().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = p.Context().NewUdmUe(supi)
		}
		udmUe.SetAMSubsriptionData(&accessAndMobilitySubscriptionDataResp)
		c.JSON(http.StatusOK, accessAndMobilitySubscriptionDataResp)
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
	}
}

func (p *Processor) GetIdTranslationResultProcedure(c *gin.Context, gpsi string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
	}
	var idTranslationResult models.IdTranslationResult
	var getIdentityDataParamOpts Nudr_DataRepository.GetIdentityDataParamOpts

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(gpsi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	idTranslationResultResp, res, err := clientAPI.QueryIdentityDataBySUPIOrGPSIDocumentApi.GetIdentityData(
		ctx, gpsi, &getIdentityDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Errorf(err.Error())
		} else if err.Error() != res.Status {
			logger.SdmLog.Errorf("Response State: %+v", err.Error())
		} else {
			problemDetails := &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			c.JSON(int(problemDetails.Status), problemDetails)
			return
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
			c.JSON(http.StatusOK, idTranslationResult)
		} else {
			problemDetails := &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "USER_NOT_FOUND",
			}
			c.JSON(int(problemDetails.Status), problemDetails)
		}
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
	}
}

func (p *Processor) GetSupiProcedure(c *gin.Context,
	supi string,
	plmnID string,
	dataSetNames []string,
	supportedFeatures string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	if len(dataSetNames) < 2 {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "BAD_REQUEST",
			Detail: "datasetNames must have at least 2 elements",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
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
	p.Context().CreateSubsDataSetsForUe(supi, subsDataSetBody)

	if p.containDataSetName(dataSetNames, string(models.DataSetName_AM)) {
		var body models.AccessAndMobilitySubscriptionData
		p.Context().CreateAccessMobilitySubsDataForUe(supi, body)

		amData, res, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.QueryAmData(
			ctx, supi, plmnID, &queryAmDataParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorf(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorf("Response State: %+v", err.Error())
			} else {
				problemDetails := &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}

				c.JSON(int(problemDetails.Status), problemDetails)
				return
			}
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QueryAmData response body cannot close: %+v", rspCloseErr)
			}
		}()
		if res.StatusCode == http.StatusOK {
			udmUe, ok := p.Context().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = p.Context().NewUdmUe(supi)
			}
			udmUe.SetAMSubsriptionData(&amData)
			subscriptionDataSets.AmData = &amData
		} else {
			problemDetails := &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "DATA_NOT_FOUND",
			}

			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}

	if p.containDataSetName(dataSetNames, string(models.DataSetName_SMF_SEL)) {
		var smfSelSubsbody models.SmfSelectionSubscriptionData
		p.Context().CreateSmfSelectionSubsDataforUe(supi, smfSelSubsbody)

		smfSelData, res, err := clientAPI.SMFSelectionSubscriptionDataDocumentApi.QuerySmfSelectData(ctx,
			supi, plmnID, &querySmfSelectDataParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorln(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorln(err.Error())
			} else {
				problemDetails := &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}

				c.JSON(int(problemDetails.Status), problemDetails)
				return
			}
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QuerySmfSelectData response body cannot close: %+v", rspCloseErr)
			}
		}()
		if res.StatusCode == http.StatusOK {
			udmUe, ok := p.Context().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = p.Context().NewUdmUe(supi)
			}
			udmUe.SetSmfSelectionSubsData(&smfSelData)
			subscriptionDataSets.SmfSelData = &smfSelData
		} else {
			problemDetails := &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "DATA_NOT_FOUND",
			}

			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}

	if p.containDataSetName(dataSetNames, string(models.DataSetName_UEC_SMF)) {
		var UeContextInSmfbody models.UeContextInSmfData
		var querySmfRegListParamOpts Nudr_DataRepository.QuerySmfRegListParamOpts
		querySmfRegListParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
		p.Context().CreateUeContextInSmfDataforUe(supi, UeContextInSmfbody)

		pdusess, res, err := clientAPI.SMFRegistrationsCollectionApi.QuerySmfRegList(
			ctx, supi, &querySmfRegListParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorf(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorf("Response State: %+v", err.Error())
			} else {
				problemDetails := &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}

				c.JSON(int(problemDetails.Status), problemDetails)
				return
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
			udmUe, ok := p.Context().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = p.Context().NewUdmUe(supi)
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
				problemDetails := &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}

				c.JSON(int(problemDetails.Status), problemDetails)
				return
			}
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QuerySmData response body cannot close: %+v", rspCloseErr)
			}
		}()
		if res.StatusCode == http.StatusOK {
			udmUe, ok := p.Context().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = p.Context().NewUdmUe(supi)
			}
			smData, _, _, _ := p.Context().ManageSmData(sessionManagementSubscriptionData, "", "")
			udmUe.SetSMSubsData(smData)
			subscriptionDataSets.SmData = sessionManagementSubscriptionData
		} else {
			problemDetails := &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "DATA_NOT_FOUND",
			}

			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}

	if p.containDataSetName(dataSetNames, string(models.DataSetName_TRACE)) {
		var TraceDatabody models.TraceData
		p.Context().CreateTraceDataforUe(supi, TraceDatabody)

		traceData, res, err := clientAPI.TraceDataDocumentApi.QueryTraceData(
			ctx, supi, plmnID, &queryTraceDataParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorf(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorf("Response State: %+v", err.Error())
			} else {
				problemDetails := &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}
				c.JSON(int(problemDetails.Status), problemDetails)
				return
			}
			problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QueryTraceData response body cannot close: %+v", rspCloseErr)
			}
		}()
		if res.StatusCode == http.StatusOK {
			udmUe, ok := p.Context().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = p.Context().NewUdmUe(supi)
			}
			udmUe.TraceData = &traceData
			udmUe.TraceDataResponse.TraceData = &traceData
			subscriptionDataSets.TraceData = &traceData
		} else {
			problemDetails := &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "DATA_NOT_FOUND",
			}

			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}

	// TODO: SMS Management Subscription Data
	// if containDataSetName(dataSetNames, string(models.DataSetName_SMS_MNG)) {
	// }

	c.JSON(http.StatusOK, subscriptionDataSets)
}

func (p *Processor) GetSharedDataProcedure(c *gin.Context, sharedDataIds []string, supportedFeatures string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR("")
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
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
			problemDetails := &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}

			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("GetShareData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		p.Context().SharedSubsDataMap = udm_context.MappingSharedData(sharedDataResp)
		sharedData := udm_context.ObtainRequiredSharedData(sharedDataIds, sharedDataResp)
		c.JSON(http.StatusOK, sharedData)
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
	}
}

func (p *Processor) GetSmDataProcedure(
	c *gin.Context,
	supi string,
	plmnID string,
	Dnn string,
	Snssai string,
	supportedFeatures string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
	}
	logger.SdmLog.Infof("getSmDataProcedure: SUPI[%s] PLMNID[%s] DNN[%s] SNssai[%s]", supi, plmnID, Dnn, Snssai)

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
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
			problemDetails := &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QuerySmData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udmUe, ok := p.Context().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = p.Context().NewUdmUe(supi)
		}
		smData, snssaikey, AllDnnConfigsbyDnn, AllDnns := p.Context().ManageSmData(
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
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
}

func (p *Processor) GetNssaiProcedure(c *gin.Context, supi string, plmnID string, supportedFeatures string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	var queryAmDataParamOpts Nudr_DataRepository.QueryAmDataParamOpts
	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	var nssaiResp models.Nssai
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
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

			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryAmData response body cannot close: %+v", rspCloseErr)
		}
	}()

	nssaiResp = *accessAndMobilitySubscriptionDataResp.Nssai

	if res.StatusCode == http.StatusOK {
		udmUe, ok := p.Context().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = p.Context().NewUdmUe(supi)
		}
		udmUe.Nssai = &nssaiResp
		c.JSON(http.StatusOK, udmUe.Nssai)
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
	}
}

func (p *Processor) GetSmfSelectDataProcedure(c *gin.Context, supi string, plmnID string, supportedFeatures string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	var querySmfSelectDataParamOpts Nudr_DataRepository.QuerySmfSelectDataParamOpts
	querySmfSelectDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	var body models.SmfSelectionSubscriptionData

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	p.Context().CreateSmfSelectionSubsDataforUe(supi, body)

	smfSelectionSubscriptionDataResp, res, err := clientAPI.SMFSelectionSubscriptionDataDocumentApi.
		QuerySmfSelectData(ctx, supi, plmnID, &querySmfSelectDataParamOpts)
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
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QuerySmfSelectData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udmUe, ok := p.Context().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = p.Context().NewUdmUe(supi)
		}
		udmUe.SetSmfSelectionSubsData(&smfSelectionSubscriptionDataResp)
		c.JSON(http.StatusOK, udmUe.SmfSelSubsData)
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
	}
}

func (p *Processor) SubscribeToSharedDataProcedure(c *gin.Context, sdmSubscription *models.SdmSubscription) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDM_SDM, models.NfType_UDM)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}

	udmClientAPI := p.Consumer().GetSDMClient("subscribeToSharedData")

	sdmSubscriptionResp, res, err := udmClientAPI.SubscriptionCreationForSharedDataApi.SubscribeToSharedData(
		ctx, *sdmSubscription)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			problemDetails := &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("SubscribeToSharedData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusCreated {
		p.Context().CreateSubstoNotifSharedData(sdmSubscriptionResp.SubscriptionId, &sdmSubscriptionResp)
		reourceUri := p.Context().
			GetSDMUri() +
			"//shared-data-subscriptions/" + sdmSubscriptionResp.SubscriptionId
		c.Header("Location", reourceUri)
		c.JSON(http.StatusOK, sdmSubscriptionResp)
	} else if res.StatusCode == http.StatusNotFound {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}

		c.JSON(int(problemDetails.Status), problemDetails)
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotImplemented,
			Cause:  "UNSUPPORTED_RESOURCE_URI",
		}

		c.JSON(int(problemDetails.Status), problemDetails)
	}
}

func (p *Processor) SubscribeProcedure(c *gin.Context, sdmSubscription *models.SdmSubscription, supi string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
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
			problemDetails := &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("CreateSdmSubscriptions response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusCreated {
		udmUe, _ := p.Context().UdmUeFindBySupi(supi)
		if udmUe == nil {
			udmUe = p.Context().NewUdmUe(supi)
		}
		udmUe.CreateSubscriptiontoNotifChange(sdmSubscriptionResp.SubscriptionId, &sdmSubscriptionResp)
		c.Header("Location", udmUe.GetLocationURI2(udm_context.LocationUriSdmSubscription, supi))
		c.JSON(http.StatusCreated, sdmSubscriptionResp)
	} else if res.StatusCode == http.StatusNotFound {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotImplemented,
			Cause:  "UNSUPPORTED_RESOURCE_URI",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
	}
}

func (p *Processor) UnsubscribeForSharedDataProcedure(c *gin.Context, subscriptionID string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDM_SDM, models.NfType_UDM)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}

	udmClientAPI := p.Consumer().GetSDMClient("unsubscribeForSharedData")

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
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("UnsubscribeForSharedData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusNoContent {
		c.Status(http.StatusNoContent)
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
	}
}

func (p *Processor) UnsubscribeProcedure(c *gin.Context, supi string, subscriptionID string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
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
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("RemovesdmSubscriptions response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusNoContent {
		c.Status(http.StatusNoContent)
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "USER_NOT_FOUND",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
	}
}

func (p *Processor) ModifyProcedure(c *gin.Context,
	sdmSubsModification *models.SdmSubsModification,
	supi string,
	subscriptionID string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
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
			problemDetails := &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("Updatesdmsubscriptions response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		c.JSON(http.StatusOK, sdmSubscription)
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "USER_NOT_FOUND",
		}

		c.JSON(int(problemDetails.Status), problemDetails)
	}
}

// TS 29.503 5.2.2.7.3
// Modification of a subscription to notifications of shared data change
func (p *Processor) ModifyForSharedDataProcedure(c *gin.Context,
	sdmSubsModification *models.SdmSubsModification,
	supi string,
	subscriptionID string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
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
			problemDetails := &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("Updatesdmsubscriptions response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		c.JSON(http.StatusOK, sdmSubscription)
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "USER_NOT_FOUND",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
	}
}

func (p *Processor) GetTraceDataProcedure(c *gin.Context, supi string, plmnID string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	var body models.TraceData
	var queryTraceDataParamOpts Nudr_DataRepository.QueryTraceDataParamOpts

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	p.Context().CreateTraceDataforUe(supi, body)

	traceDataRes, res, err := clientAPI.TraceDataDocumentApi.QueryTraceData(
		ctx, supi, plmnID, &queryTraceDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			problemDetails := &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryTraceData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udmUe, ok := p.Context().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = p.Context().NewUdmUe(supi)
		}
		udmUe.TraceData = &traceDataRes
		udmUe.TraceDataResponse.TraceData = &traceDataRes

		c.JSON(http.StatusOK, udmUe.TraceDataResponse.TraceData)
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "USER_NOT_FOUND",
		}

		c.JSON(int(problemDetails.Status), problemDetails)
	}
}

func (p *Processor) GetUeContextInSmfDataProcedure(c *gin.Context, supi string, supportedFeatures string) {
	var body models.UeContextInSmfData
	var ueContextInSmfData models.UeContextInSmfData
	var pgwInfoArray []models.PgwInfo
	var querySmfRegListParamOpts Nudr_DataRepository.QuerySmfRegListParamOpts
	querySmfRegListParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	pduSessionMap := make(map[string]models.PduSession)
	p.Context().CreateUeContextInSmfDataforUe(supi, body)

	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
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
			problemDetails := &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}

			c.JSON(int(problemDetails.Status), problemDetails)
			return
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
		udmUe, ok := p.Context().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = p.Context().NewUdmUe(supi)
		}
		udmUe.UeCtxtInSmfData = &ueContextInSmfData
		c.JSON(http.StatusOK, udmUe.UeCtxtInSmfData)
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
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
