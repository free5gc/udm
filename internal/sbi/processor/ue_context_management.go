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

// ue_context_managemanet_service
func (p *Processor) GetAmf3gppAccessProcedure(c *gin.Context, ueID string, supportedFeatures string) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	var queryAmfContext3gppParamOpts Nudr_DataRepository.QueryAmfContext3gppParamOpts
	queryAmfContext3gppParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(ueID)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	amf3GppAccessRegistration, resp, err := clientAPI.AMF3GPPAccessRegistrationDocumentApi.
		QueryAmfContext3gpp(ctx, ueID, &queryAmfContext3gppParamOpts)
	if err != nil {
		problemDetails := &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryAmfContext3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	c.JSON(http.StatusOK, amf3GppAccessRegistration)
}

func (p *Processor) GetAmfNon3gppAccessProcedure(c *gin.Context, queryAmfContextNon3gppParamOpts Nudr_DataRepository.
	QueryAmfContextNon3gppParamOpts, ueID string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(ueID)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	amfNon3GppAccessRegistration, resp, err := clientAPI.AMFNon3GPPAccessRegistrationDocumentApi.
		QueryAmfContextNon3gpp(ctx, ueID, &queryAmfContextNon3gppParamOpts)
	if err != nil {
		problemDetails := &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryAmfContext3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	c.JSON(http.StatusOK, amfNon3GppAccessRegistration)
}

func (p *Processor) RegistrationAmf3gppAccessProcedure(c *gin.Context,
	registerRequest models.Amf3GppAccessRegistration,
	ueID string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	// TODO: EPS interworking with N26 is not supported yet in this stage
	var oldAmf3GppAccessRegContext *models.Amf3GppAccessRegistration
	var ue *udm_context.UdmUeContext

	if p.Context().UdmAmf3gppRegContextExists(ueID) {
		ue, _ = p.Context().UdmUeFindBySupi(ueID)
		oldAmf3GppAccessRegContext = ue.Amf3GppAccessRegistration
	}

	p.Context().CreateAmf3gppRegContext(ueID, registerRequest)

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(ueID)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	var createAmfContext3gppParamOpts Nudr_DataRepository.CreateAmfContext3gppParamOpts
	optInterface := optional.NewInterface(registerRequest)
	createAmfContext3gppParamOpts.Amf3GppAccessRegistration = optInterface
	resp, err := clientAPI.AMF3GPPAccessRegistrationDocumentApi.CreateAmfContext3gpp(ctx,
		ueID, &createAmfContext3gppParamOpts)
	if err != nil {
		logger.UecmLog.Errorln("CreateAmfContext3gpp error : ", err)
		problemDetails := &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.UecmLog.Errorf("CreateAmfContext3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	// TS 23.502 4.2.2.2.2 14d: UDM initiate a Nudm_UECM_DeregistrationNotification to the old AMF
	// corresponding to the same (e.g. 3GPP) access, if one exists
	if oldAmf3GppAccessRegContext != nil {
		if !ue.SameAsStoredGUAMI3gpp(*oldAmf3GppAccessRegContext.Guami) {
			// Based on TS 23.502 4.2.2.2.2, If the serving NF removal reason indicated by the UDM is Initial Registration,
			// the old AMF invokes the Nsmf_PDUSession_ReleaseSMContext (SM Context ID). Thus we give different
			// dereg cause based on registration parameter from serving AMF
			deregReason := models.DeregistrationReason_UE_REGISTRATION_AREA_CHANGE
			if registerRequest.InitialRegistrationInd {
				deregReason = models.DeregistrationReason_UE_INITIAL_REGISTRATION
			}
			deregistData := models.DeregistrationData{
				DeregReason: deregReason,
				AccessType:  models.AccessType__3_GPP_ACCESS,
			}

			go func() {
				logger.UecmLog.Infof("Send DeregNotify to old AMF GUAMI=%v", oldAmf3GppAccessRegContext.Guami)
				pd := p.SendOnDeregistrationNotification(ueID,
					oldAmf3GppAccessRegContext.DeregCallbackUri,
					deregistData) // Deregistration Notify Triggered
				if pd != nil {
					logger.UecmLog.Errorf("RegistrationAmf3gppAccess: send DeregNotify fail %v", pd)
				}
			}()
		}

		c.JSON(http.StatusOK, registerRequest)
	} else {
		udmUe, _ := p.Context().UdmUeFindBySupi(ueID)
		c.Header("Location", udmUe.GetLocationURI(udm_context.LocationUriAmf3GppAccessRegistration))
		c.JSON(http.StatusCreated, registerRequest)
	}
}

func (p *Processor) RegisterAmfNon3gppAccessProcedure(c *gin.Context,
	registerRequest models.AmfNon3GppAccessRegistration,
	ueID string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	var oldAmfNon3GppAccessRegContext *models.AmfNon3GppAccessRegistration
	if p.Context().UdmAmfNon3gppRegContextExists(ueID) {
		ue, _ := p.Context().UdmUeFindBySupi(ueID)
		oldAmfNon3GppAccessRegContext = ue.AmfNon3GppAccessRegistration
	}

	p.Context().CreateAmfNon3gppRegContext(ueID, registerRequest)

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(ueID)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	var createAmfContextNon3gppParamOpts Nudr_DataRepository.CreateAmfContextNon3gppParamOpts
	optInterface := optional.NewInterface(registerRequest)
	createAmfContextNon3gppParamOpts.AmfNon3GppAccessRegistration = optInterface

	resp, err := clientAPI.AMFNon3GPPAccessRegistrationDocumentApi.CreateAmfContextNon3gpp(
		ctx, ueID, &createAmfContextNon3gppParamOpts)
	if err != nil {
		problemDetails := &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}

		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.UecmLog.Errorf("CreateAmfContext3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	// TS 23.502 4.2.2.2.2 14d: UDM initiate a Nudm_UECM_DeregistrationNotification to the old AMF
	// corresponding to the same (e.g. 3GPP) access, if one exists
	if oldAmfNon3GppAccessRegContext != nil {
		deregistData := models.DeregistrationData{
			DeregReason: models.DeregistrationReason_UE_INITIAL_REGISTRATION,
			AccessType:  models.AccessType_NON_3_GPP_ACCESS,
		}
		p.SendOnDeregistrationNotification(ueID, oldAmfNon3GppAccessRegContext.DeregCallbackUri,
			deregistData) // Deregistration Notify Triggered

		return
	} else {
		udmUe, _ := p.Context().UdmUeFindBySupi(ueID)
		c.Header("Location", udmUe.GetLocationURI(udm_context.LocationUriAmfNon3GppAccessRegistration))
		c.JSON(http.StatusCreated, registerRequest)
	}
}

func (p *Processor) UpdateAmf3gppAccessProcedure(c *gin.Context,
	request models.Amf3GppAccessRegistrationModification,
	ueID string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	var patchItemReqArray []models.PatchItem
	currentContext := p.Context().GetAmf3gppRegContext(ueID)
	if currentContext == nil {
		logger.UecmLog.Errorln("[UpdateAmf3gppAccess] Empty Amf3gppRegContext")
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "CONTEXT_NOT_FOUND",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	if request.Guami != nil {
		udmUe, _ := p.Context().UdmUeFindBySupi(ueID)
		if udmUe.SameAsStoredGUAMI3gpp(*request.Guami) { // deregistration
			logger.UecmLog.Infoln("UpdateAmf3gppAccess - deregistration")
			request.PurgeFlag = true
		} else {
			logger.UecmLog.Errorln("INVALID_GUAMI")
			problemDetails := &models.ProblemDetails{
				Status: http.StatusForbidden,
				Cause:  "INVALID_GUAMI",
			}
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}

		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "guami"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = *request.Guami
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.PurgeFlag {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "purgeFlag"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.PurgeFlag
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.Pei != "" {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "pei"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.Pei
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.ImsVoPs != "" {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "imsVoPs"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.ImsVoPs
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.BackupAmfInfo != nil {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "backupAmfInfo"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.BackupAmfInfo
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(ueID)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	resp, err := clientAPI.AMF3GPPAccessRegistrationDocumentApi.AmfContext3gpp(ctx, ueID,
		patchItemReqArray)
	if err != nil {
		problemDetails := &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	if request.PurgeFlag {
		udmUe, _ := p.Context().UdmUeFindBySupi(ueID)
		udmUe.Amf3GppAccessRegistration = nil
	}

	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.UecmLog.Errorf("AmfContext3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	c.Status(http.StatusNoContent)
}

func (p *Processor) UpdateAmfNon3gppAccessProcedure(c *gin.Context,
	request models.AmfNon3GppAccessRegistrationModification,
	ueID string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	var patchItemReqArray []models.PatchItem
	currentContext := p.Context().GetAmfNon3gppRegContext(ueID)
	if currentContext == nil {
		logger.UecmLog.Errorln("[UpdateAmfNon3gppAccess] Empty AmfNon3gppRegContext")
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "CONTEXT_NOT_FOUND",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	if request.Guami != nil {
		udmUe, _ := p.Context().UdmUeFindBySupi(ueID)
		if udmUe.SameAsStoredGUAMINon3gpp(*request.Guami) { // deregistration
			logger.UecmLog.Infoln("UpdateAmfNon3gppAccess - deregistration")
			request.PurgeFlag = true
		} else {
			logger.UecmLog.Errorln("INVALID_GUAMI")
			problemDetails := &models.ProblemDetails{
				Status: http.StatusForbidden,
				Cause:  "INVALID_GUAMI",
			}
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}

		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "guami"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = *request.Guami
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.PurgeFlag {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "purgeFlag"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.PurgeFlag
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.Pei != "" {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "pei"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.Pei
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.ImsVoPs != "" {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "imsVoPs"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.ImsVoPs
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.BackupAmfInfo != nil {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "backupAmfInfo"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.BackupAmfInfo
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(ueID)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	resp, err := clientAPI.AMFNon3GPPAccessRegistrationDocumentApi.AmfContextNon3gpp(ctx,
		ueID, patchItemReqArray)
	if err != nil {
		problemDetails := &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.UecmLog.Errorf("AmfContextNon3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	c.Status(http.StatusNoContent)
}

func (p *Processor) DeregistrationSmfRegistrationsProcedure(c *gin.Context,
	ueID string,
	pduSessionID string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(ueID)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	resp, err := clientAPI.SMFRegistrationDocumentApi.DeleteSmfContext(ctx, ueID, pduSessionID)
	if err != nil {
		problemDetails := &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.UecmLog.Errorf("DeleteSmfContext response body cannot close: %+v", rspCloseErr)
		}
	}()

	c.Status(http.StatusNoContent)
}

func (p *Processor) RegistrationSmfRegistrationsProcedure(
	c *gin.Context,
	smfRegistration *models.SmfRegistration,
	ueID string,
	pduSessionID string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	contextExisted := false
	p.Context().CreateSmfRegContext(ueID, pduSessionID)
	if !p.Context().UdmSmfRegContextNotExists(ueID) {
		contextExisted = true
	}

	pduID64, err := strconv.ParseInt(pduSessionID, 10, 32)
	if err != nil {
		logger.UecmLog.Errorln(err.Error())
	}
	pduID32 := int32(pduID64)

	var createSmfContextNon3gppParamOpts Nudr_DataRepository.CreateSmfContextNon3gppParamOpts
	optInterface := optional.NewInterface(*smfRegistration)
	createSmfContextNon3gppParamOpts.SmfRegistration = optInterface

	clientAPI, err := p.Consumer().CreateUDMClientToUDR(ueID)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	resp, err := clientAPI.SMFRegistrationDocumentApi.CreateSmfContextNon3gpp(ctx, ueID,
		pduID32, &createSmfContextNon3gppParamOpts)
	if err != nil {
		problemDetails := &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.UecmLog.Errorf("CreateSmfContextNon3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	if contextExisted {
		c.Status(http.StatusNoContent)
	} else {
		udmUe, _ := p.Context().UdmUeFindBySupi(ueID)
		c.Header("Location", udmUe.GetLocationURI(udm_context.LocationUriSmfRegistration))
		c.JSON(http.StatusCreated, smfRegistration)
	}
}
