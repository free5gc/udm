package sbi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/util/metrics/sbi"
	"github.com/free5gc/util/validator"
)

func (s *Server) getSubscriberDataManagementRoutes() []Route {
	return []Route{
		{
			"Index",
			http.MethodGet,
			"/",
			s.HandleIndex,
		},
	}
}

func isValidSDMSubscriptionID(subscriptionID string) bool {
	id, err := strconv.ParseUint(subscriptionID, 10, 64)
	return err == nil && id > 0 && strconv.FormatUint(id, 10) == subscriptionID
}

func isValidSharedDataSubscriptionID(subscriptionID string) bool {
	if len(subscriptionID) == 0 || len(subscriptionID) > 128 {
		return false
	}

	for i := 0; i < len(subscriptionID); i++ {
		ch := subscriptionID[i]
		if (ch < 'a' || ch > 'z') &&
			(ch < 'A' || ch > 'Z') &&
			(ch < '0' || ch > '9') &&
			ch != '-' && ch != '.' && ch != '_' && ch != '~' {
			return false
		}
	}
	return true
}

func validateSubscriptionID(c *gin.Context, sharedData bool) (string, bool) {
	subscriptionID := c.Params.ByName("subscriptionId")
	valid := isValidSDMSubscriptionID(subscriptionID)
	if sharedData {
		valid = isValidSharedDataSubscriptionID(subscriptionID)
	}
	if !valid {
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "Subscription ID is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.SdmLog.Warnln("Subscription ID is invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return "", false
	}
	return subscriptionID, true
}

// GetAmData - retrieve a UE's Access and Mobility Subscription Data
func (s *Server) HandleGetAmData(c *gin.Context) {
	query := url.Values{}
	query.Set("plmn-id", c.Query("plmn-id"))
	query.Set("supported-features", c.Query("supported-features"))

	logger.SdmLog.Infof("Handle GetAmData")

	// TS 29.503 6.1.3.5.2
	// Validate SUPI format
	supi := c.Params.ByName("supi")
	if !validator.IsValidSupi(supi) {
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "Supi is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.SdmLog.Warnln("Supi is invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}

	// use c.Request.URL.Query() only for getPlmnIDStruct
	plmnIDStruct, problemDetails := s.getPlmnIDStruct(c.Request.URL.Query())
	if problemDetails != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.Header("Content-Type", "application/problem+json")
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	var plmnID string
	if plmnIDStruct != nil {
		plmnID = plmnIDStruct.Mcc + plmnIDStruct.Mnc
	}
	supportedFeatures := query.Get("supported-features")

	s.Processor().GetAmDataProcedure(c, supi, plmnID, supportedFeatures)
}

func (s *Server) getPlmnIDStruct(
	queryParameters url.Values,
) (plmnIDStruct *models.PlmnId, problemDetails *models.ProblemDetails) {
	values, exists := queryParameters["plmn-id"]
	if !exists {
		// not exist like: http:{ip:port}/api/.../
		return nil, nil
	}
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		// exist but it is empty like: http:{ip:port}/api/.../?plmn-id=
		problemDetails = &models.ProblemDetails{
			Title:  "Invalid Parameter",
			Status: http.StatusBadRequest,
			Cause:  "OPTIONAL_QUERY_PARAM_INCORRECT",
			InvalidParams: []models.InvalidParam{{
				Param:  "query plmn-id",
				Reason: "cannot be empty",
			}},
		}
		return nil, problemDetails
	}

	// exist and not empty link: http:{ip:port}/api/.../?plmn-id=xxx
	plmnIDJson := values[0]
	plmnIDStruct = &models.PlmnId{}
	err := json.Unmarshal([]byte(plmnIDJson), plmnIDStruct)
	if err != nil {
		logger.SdmLog.Warnln("Unmarshal Error in targetPlmnListtruct: ", err)
		problemDetails = &models.ProblemDetails{
			Title:  "Invalid Parameter",
			Status: http.StatusBadRequest,
			Cause:  "OPTIONAL_QUERY_PARAM_INCORRECT",
			InvalidParams: []models.InvalidParam{{
				Param:  "query plmn-id",
				Reason: "must be a valid PlmnId JSON object",
			}},
		}
		return nil, problemDetails
	}
	if !validator.IsValidPlmnIdParts(plmnIDStruct.Mcc, plmnIDStruct.Mnc) {
		problemDetails = &models.ProblemDetails{
			Title:  "Invalid Parameter",
			Status: http.StatusBadRequest,
			Cause:  "OPTIONAL_QUERY_PARAM_INCORRECT",
			InvalidParams: []models.InvalidParam{{
				Param:  "query plmn-id",
				Reason: "MCC must contain 3 digits and MNC must contain 2 or 3 digits",
			}},
		}
		return nil, problemDetails
	}
	return plmnIDStruct, nil
}

// Info - Nudm_Sdm Info service operation
func (s *Server) HandleInfo(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

// PutUpuAck - Nudm_Sdm Info for UPU service operation
func (s *Server) HandlePutUpuAck(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

// GetSmfSelectData - retrieve a UE's SMF Selection Subscription Data
func (s *Server) HandleGetSmfSelectData(c *gin.Context) {
	query := url.Values{}
	query.Set("plmn-id", c.Query("plmn-id"))
	query.Set("supported-features", c.Query("supported-features"))

	logger.SdmLog.Infof("Handle GetSmfSelectData")

	supi := c.Params.ByName("supi")
	// TS 29.503 6.1.3.5.2
	// Validate SUPI format
	if !validator.IsValidSupi(supi) {
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "Supi is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.SdmLog.Warnln("Supi is invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}
	// use c.Request.URL.Query() only for getPlmnIDStruct
	plmnIDStruct, problemDetails := s.getPlmnIDStruct(c.Request.URL.Query())
	if problemDetails != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.Header("Content-Type", "application/problem+json")
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	var plmnID string
	if plmnIDStruct != nil {
		plmnID = plmnIDStruct.Mcc + plmnIDStruct.Mnc
	}
	supportedFeatures := query.Get("supported-features")

	s.Processor().GetSmfSelectDataProcedure(c, supi, plmnID, supportedFeatures)
}

// GetSmsMngData - retrieve a UE's SMS Management Subscription Data
func (s *Server) HandleGetSmsMngData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// GetSmsData - retrieve a UE's SMS Subscription Data
func (s *Server) HandleGetSmsData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// GetSupi - retrieve multiple data sets
func (s *Server) HandleGetSupi(c *gin.Context) {
	query := url.Values{}
	query.Set("plmn-id", c.Query("plmn-id"))
	query.Set("dataset-names", c.Query("dataset-names"))
	query.Set("supported-features", c.Query("supported-features"))

	logger.SdmLog.Infof("Handle GetSupiRequest")

	supi := c.Params.ByName("supi")
	// TS 29.503 6.1.3.5.2
	// Validate SUPI format
	if !validator.IsValidSupi(supi) {
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "Supi is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.SdmLog.Warnln("Supi is invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}
	// use c.Request.URL.Query() only for getPlmnIDStruct
	plmnIDStruct, problemDetails := s.getPlmnIDStruct(c.Request.URL.Query())
	if problemDetails != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.Header("Content-Type", "application/problem+json")
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	var plmnID string
	if plmnIDStruct != nil {
		plmnID = plmnIDStruct.Mcc + plmnIDStruct.Mnc
	}
	dataSetNames := strings.Split(query.Get("dataset-names"), ",")
	supportedFeatures := query.Get("supported-features")

	s.Processor().GetSupiProcedure(c, supi, plmnID, dataSetNames, supportedFeatures)
}

// GetSharedData - retrieve shared data
func (s *Server) HandleGetSharedData(c *gin.Context) {
	logger.SdmLog.Infof("Handle GetSharedData")

	sharedDataIds := c.QueryArray("shared-data-ids")
	supportedFeatures := c.QueryArray("supported-features")

	supportedFeature := ""
	if len(supportedFeatures) > 0 {
		supportedFeature = supportedFeatures[0]
	}

	s.Processor().GetSharedDataProcedure(c, sharedDataIds, supportedFeature)
}

// SubscribeToSharedData - subscribe to notifications for shared data
func (s *Server) HandleSubscribeToSharedData(c *gin.Context) {
	var sharedDataSubsReq models.Udm_SDM_SdmSubscription

	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.SdmLog.Errorf("Get Request Body error: %+v", err)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&sharedDataSubsReq, requestBody, "application/json")
	if err != nil {
		logger.SdmLog.Errorf("[Request Body] %v", err)
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "The request body is malformed or does not match the expected schema.",
			Cause:  "INVALID_MSG_FORMAT",
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, rsp.Cause)
		c.Header("Content-Type", "application/problem+json")
		c.JSON(int(rsp.Status), rsp)
		return
	}

	logger.SdmLog.Infof("Handle SubscribeToSharedData")

	s.Processor().SubscribeToSharedDataProcedure(c, &sharedDataSubsReq)
}

// Subscribe - subscribe to notifications
func (s *Server) HandleSubscribe(c *gin.Context) {
	var sdmSubscriptionReq models.Udm_SDM_SdmSubscription

	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.SdmLog.Errorf("Get Request Body error: %+v", err)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&sdmSubscriptionReq, requestBody, "application/json")
	if err != nil {
		logger.SdmLog.Errorf("[Request Body] %v", err)
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "The request body is malformed or does not match the expected schema.",
			Cause:  "INVALID_MSG_FORMAT",
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, rsp.Cause)
		c.Header("Content-Type", "application/problem+json")
		c.JSON(int(rsp.Status), rsp)
		return
	}

	logger.SdmLog.Infof("Handle Subscribe")

	supi := c.Params.ByName("supi")
	s.Processor().SubscribeProcedure(c, &sdmSubscriptionReq, supi)
}

// Unsubscribe - unsubscribe from notifications
func (s *Server) HandleUnsubscribe(c *gin.Context) {
	logger.SdmLog.Infof("Handle Unsubscribe")

	// TS 29.503 6.1.3.4.2
	// Validate SUPI and GPSI format the UE ID (SUPI or GPSI)
	ueId := c.Params.ByName("ueId")
	valid := validator.IsValidGpsi(ueId) || validator.IsValidSupi(ueId)
	if !valid {
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "UE ID is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.SdmLog.Warnln("UE ID is invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}
	subscriptionID, valid := validateSubscriptionID(c, false)
	if !valid {
		return
	}

	s.Processor().UnsubscribeProcedure(c, ueId, subscriptionID)
}

// UnsubscribeForSharedData - unsubscribe from notifications for shared data
func (s *Server) HandleUnsubscribeForSharedData(c *gin.Context) {
	logger.SdmLog.Infof("Handle UnsubscribeForSharedData")

	subscriptionID, valid := validateSubscriptionID(c, true)
	if !valid {
		return
	}
	s.Processor().UnsubscribeForSharedDataProcedure(c, subscriptionID)
}

// Modify - modify the subscription
func (s *Server) HandleModify(c *gin.Context) {
	var sdmSubsModificationReq models.Udm_SDM_SdmSubsModification

	// TS 29.503 6.1.3.4.2
	// Validate SUPI and GPSI format the UE ID (SUPI or GPSI)
	ueId := c.Params.ByName("ueId")
	valid := validator.IsValidGpsi(ueId) || validator.IsValidSupi(ueId)
	if !valid {
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "UE ID is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.SdmLog.Warnln("UE ID is invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}
	subscriptionID, valid := validateSubscriptionID(c, false)
	if !valid {
		return
	}

	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.SdmLog.Errorf("Get Request Body error: %+v", err)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&sdmSubsModificationReq, requestBody, "application/json")
	if err != nil {
		logger.SdmLog.Errorf("[Request Body] %v", err)
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "The request body is malformed or does not match the expected schema.",
			Cause:  "INVALID_MSG_FORMAT",
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, rsp.Cause)
		c.Header("Content-Type", "application/problem+json")
		c.JSON(int(rsp.Status), rsp)
		return
	}

	logger.SdmLog.Infof("Handle Modify")

	s.Processor().ModifyProcedure(c, &sdmSubsModificationReq, ueId, subscriptionID)
}

// ModifyForSharedData - modify the subscription
func (s *Server) HandleModifyForSharedData(c *gin.Context) {
	subscriptionID, valid := validateSubscriptionID(c, true)
	if !valid {
		return
	}

	var sharedDataSubscriptions models.Udm_SDM_SdmSubsModification
	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.SdmLog.Errorf("Get Request Body error: %+v", err)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&sharedDataSubscriptions, requestBody, "application/json")
	if err != nil {
		logger.SdmLog.Errorf("[Request Body] %v", err)
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "The request body is malformed or does not match the expected schema.",
			Cause:  "INVALID_MSG_FORMAT",
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, rsp.Cause)
		c.Header("Content-Type", "application/problem+json")
		c.JSON(int(rsp.Status), rsp)
		return
	}

	logger.SdmLog.Infof("Handle ModifyForSharedData")

	supi := c.Params.ByName("supi")

	s.Processor().ModifyForSharedDataProcedure(c, &sharedDataSubscriptions, supi, subscriptionID)
}

// GetTraceData - retrieve a UE's Trace Configuration Data
func (s *Server) HandleGetTraceData(c *gin.Context) {
	logger.SdmLog.Infof("Handle GetTraceData")

	supi := c.Params.ByName("supi")
	// TS 29.503 6.1.3.5.2
	// Validate SUPI format
	if !validator.IsValidSupi(supi) {
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "Supi is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.SdmLog.Warnln("Supi is invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}
	plmnIDStruct, problemDetails := s.getPlmnIDStruct(c.Request.URL.Query())
	if problemDetails != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.Header("Content-Type", "application/problem+json")
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	var plmnID string
	if plmnIDStruct != nil {
		plmnID = plmnIDStruct.Mcc + plmnIDStruct.Mnc
	}

	s.Processor().GetTraceDataProcedure(c, supi, plmnID)
}

// GetUeContextInSmfData - retrieve a UE's UE Context In SMF Data
func (s *Server) HandleGetUeContextInSmfData(c *gin.Context) {
	logger.SdmLog.Infof("Handle GetUeContextInSmfData")

	supi := c.Params.ByName("supi")
	// TS 29.503 6.1.3.5.2
	// Validate SUPI format
	if !validator.IsValidSupi(supi) {
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "Supi is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.SdmLog.Warnln("Supi is invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}
	supportedFeatures := c.Query("supported-features")

	s.Processor().GetUeContextInSmfDataProcedure(c, supi, supportedFeatures)
}

// GetUeContextInSmsfData - retrieve a UE's UE Context In SMSF Data
func (s *Server) HandleGetUeContextInSmsfData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// GetNssai - retrieve a UE's subscribed NSSAI
func (s *Server) HandleGetNssai(c *gin.Context) {
	query := url.Values{}
	query.Set("plmn-id", c.Query("plmn-id"))
	query.Set("supported-features", c.Query("supported-features"))

	logger.SdmLog.Infof("Handle GetNssai")

	supi := c.Params.ByName("supi")
	// TS 29.503 6.1.3.5.2
	// Validate SUPI format
	if !validator.IsValidSupi(supi) {
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "Supi is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.SdmLog.Warnln("Supi is invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}
	// use c.Request.URL.Query() only for getPlmnIDStruct
	plmnIDStruct, problemDetails := s.getPlmnIDStruct(c.Request.URL.Query())
	if problemDetails != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.Header("Content-Type", "application/problem+json")
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	var plmnID string
	if plmnIDStruct != nil {
		plmnID = plmnIDStruct.Mcc + plmnIDStruct.Mnc
	}
	supportedFeatures := query.Get("supported-features")

	s.Processor().GetNssaiProcedure(c, supi, plmnID, supportedFeatures)
}

// GetSmData - retrieve a UE's Session Management Subscription Data
func (s *Server) HandleGetSmData(c *gin.Context) {
	query := url.Values{}
	query.Set("plmn-id", c.Query("plmn-id"))
	query.Set("dnn", c.Query("dnn"))
	query.Set("single-nssai", c.Query("single-nssai"))
	query.Set("supported-features", c.Query("supported-features"))

	logger.SdmLog.Infof("Handle GetSmData")

	supi := c.Params.ByName("supi")
	// TS 29.503 6.1.3.5.2
	// Validate SUPI format
	if !validator.IsValidSupi(supi) {
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "Supi is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.SdmLog.Warnln("Supi is invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}
	// use c.Request.URL.Query() only for getPlmnIDStruct
	plmnIDStruct, problemDetails := s.getPlmnIDStruct(c.Request.URL.Query())
	if problemDetails != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.Header("Content-Type", "application/problem+json")
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	var plmnID string
	if plmnIDStruct != nil {
		plmnID = plmnIDStruct.Mcc + plmnIDStruct.Mnc
	}
	Dnn := query.Get("dnn")
	Snssai := query.Get("single-nssai")
	supportedFeatures := query.Get("supported-features")

	s.Processor().GetSmDataProcedure(c, supi, plmnID, Dnn, Snssai, supportedFeatures)
}

// GetIdTranslationResult - retrieve a UE's SUPI
func (s *Server) HandleGetIdTranslationResult(c *gin.Context) {
	// req.Query.Set("SupportedFeatures", c.Query("supported-features"))

	logger.SdmLog.Infof("Handle GetIdTranslationResultRequest")

	// TS 29.503 6.1.3.12.2
	// Validate SUPI and GPSI format the UE ID (SUPI or GPSI)
	ueId := c.Params.ByName("ueId")
	valid := validator.IsValidGpsi(ueId) || validator.IsValidSupi(ueId)
	if !valid {
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "UE ID is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.SdmLog.Warnln("UE ID is invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}

	s.Processor().GetIdTranslationResultProcedure(c, ueId)
}

func (s *Server) HandleGetMultipleIdentifiers(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetGroupIdentifiers(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetLcsBcaData(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetLcsMoData(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetLcsPrivacyData(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetMbsData(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetProseData(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetUcData(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetUeCtxInAmfData(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetV2xData(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetIndividualSharedData(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleCAGAck(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetEcrData(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleSNSSAIsAck(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleUpdateSORInfo(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleUpuAck(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) OneLayerPathHandlerFunc(c *gin.Context) {
	supi := c.Param("supi")
	oneLayerPathRouter := s.getOneLayerRoutes()
	for _, route := range oneLayerPathRouter {
		if route.Pattern == "/"+supi && route.Method == c.Request.Method {
			route.HandlerFunc(c)
			return
		}
	}

	// special case for :supi
	if c.Request.Method == http.MethodGet {
		s.HandleGetSupi(c)
		return
	}

	c.String(http.StatusNotFound, "404 page not found")
}

func (s *Server) TwoLayerPathHandlerFunc(c *gin.Context) {
	supi := c.Param("supi")
	op := c.Param("subscriptionId")

	logger.ConsumerLog.Infoln("TwoLayerPathHandlerFunc, ", supi, op)

	// for "/shared-data-subscriptions/:subscriptionId"
	if supi == "shared-data-subscriptions" && http.MethodDelete == c.Request.Method {
		s.HandleUnsubscribeForSharedData(c)
		return
	}

	// for "/shared-data-subscriptions/:subscriptionId"
	if supi == "shared-data-subscriptions" && http.MethodPatch == c.Request.Method {
		s.HandleModifyForSharedData(c)
		return
	}

	// for "/:ueId/id-translation-result"
	if op == "id-translation-result" && http.MethodGet == c.Request.Method {
		c.Params = append(c.Params, gin.Param{Key: "ueId", Value: c.Param("supi")})
		s.HandleGetIdTranslationResult(c)
		return
	}

	// for "/shared-data/:sharedDataId"
	if supi == "shared-data" && http.MethodGet == c.Request.Method {
		s.HandleGetIndividualSharedData(c)
		return
	}

	twoLayerPathRouter := s.getTwoLayerRoutes()
	requestPath := "/" + supi + "/" + op
	for _, route := range twoLayerPathRouter {
		if pathPatternMatches(route.Pattern, requestPath) && route.Method == c.Request.Method {
			route.HandlerFunc(c)
			return
		}
	}

	c.String(http.StatusNotFound, "404 page not found")
}

func pathPatternMatches(pattern, path string) bool {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(patternParts) != len(pathParts) {
		return false
	}

	for i := range patternParts {
		if strings.HasPrefix(patternParts[i], ":") {
			continue
		}
		if patternParts[i] != pathParts[i] {
			return false
		}
	}
	return true
}

func (s *Server) ThreeLayerPathHandlerFunc(c *gin.Context) {
	op := c.Param("subscriptionId")
	thirdLayer := c.Param("thirdLayer")

	// for "/:ueId/sdm-subscriptions/:subscriptionId"
	if op == "sdm-subscriptions" && http.MethodDelete == c.Request.Method {
		var tmpParams gin.Params
		tmpParams = append(tmpParams, gin.Param{Key: "ueId", Value: c.Param("supi")})
		tmpParams = append(tmpParams, gin.Param{Key: "subscriptionId", Value: c.Param("thirdLayer")})
		c.Params = tmpParams
		s.HandleUnsubscribe(c)
		return
	}

	// for "/:supi/am-data/sor-ack"
	if op == "am-data" && http.MethodPut == c.Request.Method && thirdLayer == "sor-ack" {
		s.HandleInfo(c)
		return
	}

	// for "/:supi/am-data/cag-ack"
	if op == "am-data" && http.MethodPut == c.Request.Method && thirdLayer == "cag-ack" {
		s.HandleCAGAck(c)
		return
	}

	// for "/:supi/am-data/ecr-data"
	if op == "am-data" && http.MethodGet == c.Request.Method && thirdLayer == "ecr-data" {
		s.HandleGetEcrData(c)
		return
	}

	// for "/:supi/am-data/subscribed-snssais-ack"
	if op == "am-data" && http.MethodPut == c.Request.Method &&
		thirdLayer == "subscribed-snssais-ack" {
		s.HandleSNSSAIsAck(c)
		return
	}

	// for "/:supi/am-data/update-sor"
	if op == "am-data" && http.MethodPost == c.Request.Method && thirdLayer == "update-sor" {
		s.HandleUpdateSORInfo(c)
		return
	}

	// for "/:supi/am-data/upu-ack"
	if op == "am-data" && http.MethodPut == c.Request.Method && thirdLayer == "upu-ack" {
		s.HandleUpuAck(c)
		return
	}

	// for "/:ueId/sdm-subscriptions/:subscriptionId"
	if op == "sdm-subscriptions" && http.MethodPatch == c.Request.Method {
		var tmpParams gin.Params
		tmpParams = append(tmpParams, gin.Param{Key: "ueId", Value: c.Param("supi")})
		tmpParams = append(tmpParams, gin.Param{Key: "subscriptionId", Value: c.Param("thirdLayer")})
		c.Params = tmpParams
		s.HandleModify(c)
		return
	}

	c.String(http.StatusNotFound, "404 page not found")
}

func (s *Server) getOneLayerRoutes() []Route {
	return []Route{
		{
			"GetDataSets",
			http.MethodGet,
			"/:supi",
			s.HandleGetSupi,
		},

		{
			"GetSharedData",
			http.MethodGet,
			"/shared-data",
			s.HandleGetSharedData,
		},

		{
			"SubscribeToSharedData",
			http.MethodPost,
			"/shared-data-subscriptions",
			s.HandleSubscribeToSharedData,
		},

		{
			"GetMultipleIdentifiers",
			http.MethodGet,
			"/multiple-identifiers",
			s.HandleGetMultipleIdentifiers,
		},
	}
}

func (s *Server) getTwoLayerRoutes() []Route {
	return []Route{
		{
			"GetAmData",
			http.MethodGet,
			"/:supi/am-data",
			s.HandleGetAmData,
		},

		{
			"GetSmfSelData",
			http.MethodGet,
			"/:supi/smf-select-data",
			s.HandleGetSmfSelectData,
		},

		{
			"GetSmsMngtData",
			http.MethodGet,
			"/:supi/sms-mng-data",
			s.HandleGetSmsMngData,
		},

		{
			"GetSmsData",
			http.MethodGet,
			"/:supi/sms-data",
			s.HandleGetSmsData,
		},

		{
			"GetSmData",
			http.MethodGet,
			"/:supi/sm-data",
			s.HandleGetSmData,
		},

		{
			"GetNSSAI",
			http.MethodGet,
			"/:supi/nssai",
			s.HandleGetNssai,
		},

		{
			"Subscribe",
			http.MethodPost,
			"/:ueId/sdm-subscriptions",
			s.HandleSubscribe,
		},

		{
			"GetTraceConfigData",
			http.MethodGet,
			"/:supi/trace-data",
			s.HandleGetTraceData,
		},

		{
			"GetUeCtxInSmfData",
			http.MethodGet,
			"/:supi/ue-context-in-smf-data",
			s.HandleGetUeContextInSmfData,
		},

		{
			"GetUeCtxInSmsfData",
			http.MethodGet,
			"/:supi/ue-context-in-smsf-data",
			s.HandleGetUeContextInSmsfData,
		},

		{
			"GetGroupIdentifiers",
			http.MethodGet,
			"/group-data/group-identifiers",
			s.HandleGetGroupIdentifiers,
		},

		{
			"GetLcsBcaData",
			http.MethodGet,
			"/:supi/lcs-bca-data",
			s.HandleGetLcsBcaData,
		},

		{
			"GetLcsMoData",
			http.MethodGet,
			"/:supi/lcs-mo-data",
			s.HandleGetLcsMoData,
		},

		{
			"GetLcsPrivacyData",
			http.MethodGet,
			"/:ueId/lcs-privacy-data",
			s.HandleGetLcsPrivacyData,
		},

		{
			"GetMbsData",
			http.MethodGet,
			"/:supi/5mbs-data",
			s.HandleGetMbsData,
		},

		{
			"GetProseData",
			http.MethodGet,
			"/:supi/prose-data",
			s.HandleGetProseData,
		},

		{
			"GetUcData",
			http.MethodGet,
			"/:supi/uc-data",
			s.HandleGetUcData,
		},

		{
			"GetUeCtxInAmfData",
			http.MethodGet,
			"/:supi/ue-context-in-amf-data",
			s.HandleGetUeCtxInAmfData,
		},

		{
			"GetV2xData",
			http.MethodGet,
			"/:supi/v2x-data",
			s.HandleGetV2xData,
		},
	}
}
