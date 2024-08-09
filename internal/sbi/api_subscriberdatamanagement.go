package sbi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
)

func (s *Server) getSubscriberDataManagementRoutes() []Route {
	return []Route{
		{
			"Index",
			"GET",
			"/",
			s.HandleIndex,
		},
	}
}

// GetAmData - retrieve a UE's Access and Mobility Subscription Data
func (s *Server) HandleGetAmData(c *gin.Context) {
	query := url.Values{}
	query.Set("plmn-id", c.Query("plmn-id"))
	query.Set("supported-features", c.Query("plmn-id"))

	logger.SdmLog.Infof("Handle GetAmData")

	supi := c.Params.ByName("supi")

	plmnIDStruct, problemDetails := s.getPlmnIDStruct(query)
	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
	supportedFeatures := query.Get("supported-features")

	s.Processor().GetAmDataProcedure(c, supi, plmnID, supportedFeatures)
}

func (s *Server) getPlmnIDStruct(
	queryParameters url.Values,
) (plmnIDStruct *models.PlmnId, problemDetails *models.ProblemDetails) {
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
	plmnIDStruct, problemDetails := s.getPlmnIDStruct(query)
	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
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
	plmnIDStruct, problemDetails := s.getPlmnIDStruct(query)
	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
	dataSetNames := strings.Split(query.Get("dataset-names"), ",")
	supportedFeatures := query.Get("supported-features")

	s.Processor().GetSupiProcedure(c, supi, plmnID, dataSetNames, supportedFeatures)
}

// GetSharedData - retrieve shared data
func (s *Server) HandleGetSharedData(c *gin.Context) {
	logger.SdmLog.Infof("Handle GetSharedData")

	sharedDataIds := c.QueryArray("shared-data-ids")
	supportedFeatures := c.QueryArray("supported-features")

	s.Processor().GetSharedDataProcedure(c, sharedDataIds, supportedFeatures[0])
}

// SubscribeToSharedData - subscribe to notifications for shared data
func (s *Server) HandleSubscribeToSharedData(c *gin.Context) {
	var sharedDataSubsReq models.SdmSubscription

	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.SdmLog.Errorf("Get Request Body error: %+v", err)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&sharedDataSubsReq, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.SdmLog.Errorln(problemDetail)
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	logger.SdmLog.Infof("Handle SubscribeToSharedData")

	s.Processor().SubscribeToSharedDataProcedure(c, &sharedDataSubsReq)
}

// Subscribe - subscribe to notifications
func (s *Server) HandleSubscribe(c *gin.Context) {
	var sdmSubscriptionReq models.SdmSubscription

	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.SdmLog.Errorf("Get Request Body error: %+v", err)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&sdmSubscriptionReq, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.SdmLog.Errorln(problemDetail)
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	logger.SdmLog.Infof("Handle Subscribe")

	supi := c.Params.ByName("ueId")

	s.Processor().SubscribeProcedure(c, &sdmSubscriptionReq, supi)
}

// Unsubscribe - unsubscribe from notifications
func (s *Server) HandleUnsubscribe(c *gin.Context) {
	logger.SdmLog.Infof("Handle Unsubscribe")

	supi := c.Params.ByName("ueId")
	subscriptionID := c.Params.ByName("subscriptionId")

	s.Processor().UnsubscribeProcedure(c, supi, subscriptionID)
}

// UnsubscribeForSharedData - unsubscribe from notifications for shared data
func (s *Server) HandleUnsubscribeForSharedData(c *gin.Context) {
	logger.SdmLog.Infof("Handle UnsubscribeForSharedData")

	subscriptionID := c.Params.ByName("subscriptionId")
	s.Processor().UnsubscribeForSharedDataProcedure(c, subscriptionID)
}

// Modify - modify the subscription
func (s *Server) HandleModify(c *gin.Context) {
	var sdmSubsModificationReq models.SdmSubsModification
	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.SdmLog.Errorf("Get Request Body error: %+v", err)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&sdmSubsModificationReq, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.SdmLog.Errorln(problemDetail)
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	logger.SdmLog.Infof("Handle Modify")

	supi := c.Params.ByName("ueId")
	subscriptionID := c.Params.ByName("subscriptionId")

	s.Processor().ModifyProcedure(c, &sdmSubsModificationReq, supi, subscriptionID)
}

// ModifyForSharedData - modify the subscription
func (s *Server) HandleModifyForSharedData(c *gin.Context) {
	var sharedDataSubscriptions models.SdmSubsModification
	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.SdmLog.Errorf("Get Request Body error: %+v", err)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&sharedDataSubscriptions, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.SdmLog.Errorln(problemDetail)
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	logger.SdmLog.Infof("Handle ModifyForSharedData")

	supi := c.Params.ByName("supi")
	subscriptionID := c.Params.ByName("subscriptionId")

	s.Processor().ModifyForSharedDataProcedure(c, &sharedDataSubscriptions, supi, subscriptionID)
}

// GetTraceData - retrieve a UE's Trace Configuration Data
func (s *Server) HandleGetTraceData(c *gin.Context) {
	logger.SdmLog.Infof("Handle GetTraceData")

	supi := c.Params.ByName("supi")
	plmnID := c.Query("plmn-id")

	s.Processor().GetTraceDataProcedure(c, supi, plmnID)
}

// GetUeContextInSmfData - retrieve a UE's UE Context In SMF Data
func (s *Server) HandleGetUeContextInSmfData(c *gin.Context) {
	logger.SdmLog.Infof("Handle GetUeContextInSmfData")

	supi := c.Params.ByName("supi")
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
	plmnIDStruct, problemDetails := s.getPlmnIDStruct(query)
	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
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
	plmnIDStruct, problemDetails := s.getPlmnIDStruct(query)
	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
	Dnn := query.Get("dnn")
	Snssai := query.Get("single-nssai")
	supportedFeatures := query.Get("supported-features")

	s.Processor().GetSmDataProcedure(c, supi, plmnID, Dnn, Snssai, supportedFeatures)
}

// GetIdTranslationResult - retrieve a UE's SUPI
func (s *Server) HandleGetIdTranslationResult(c *gin.Context) {
	// req.Query.Set("SupportedFeatures", c.Query("supported-features"))

	logger.SdmLog.Infof("Handle GetIdTranslationResultRequest")

	gpsi := c.Params.ByName("ueId")

	s.Processor().GetIdTranslationResultProcedure(c, gpsi)
}

func (s *Server) OneLayerPathHandlerFunc(c *gin.Context) {
	supi := c.Param("supi")
	oneLayerPathRouter := s.getOneLayerRoutes()
	for _, route := range oneLayerPathRouter {
		if strings.Contains(route.Pattern, supi) && route.Method == c.Request.Method {
			route.HandlerFunc(c)
			return
		}
	}

	// special case for :supi
	if c.Request.Method == strings.ToUpper("Get") {
		s.HandleGetSupi(c)
		return
	}

	c.String(http.StatusNotFound, "404 page not found")
}

func (s *Server) TwoLayerPathHandlerFunc(c *gin.Context) {
	supi := c.Param("supi")
	op := c.Param("subscriptionId")

	// for "/shared-data-subscriptions/:subscriptionId"
	if supi == "shared-data-subscriptions" && strings.ToUpper("Delete") == c.Request.Method {
		s.HandleUnsubscribeForSharedData(c)
		return
	}

	// for "/shared-data-subscriptions/:subscriptionId"
	if supi == "shared-data-subscriptions" && strings.ToUpper("Patch") == c.Request.Method {
		s.HandleModifyForSharedData(c)
		return
	}

	// for "/:ueId/id-translation-result"
	if op == "id-translation-result" && strings.ToUpper("Get") == c.Request.Method {
		s.HandleGetIdTranslationResult(c)
		return
	}

	twoLayerPathRouter := s.getTwoLayerRoutes()
	for _, route := range twoLayerPathRouter {
		if strings.Contains(route.Pattern, op) && route.Method == c.Request.Method {
			route.HandlerFunc(c)
			return
		}
	}

	c.String(http.StatusNotFound, "404 page not found")
}

func (s *Server) ThreeLayerPathHandlerFunc(c *gin.Context) {
	op := c.Param("subscriptionId")

	// for "/:ueId/sdm-subscriptions/:subscriptionId"
	if op == "sdm-subscriptions" && strings.ToUpper("Delete") == c.Request.Method {
		var tmpParams gin.Params
		tmpParams = append(tmpParams, gin.Param{Key: "subscriptionId", Value: c.Param("thirdLayer")})
		c.Params = tmpParams
		s.HandleUnsubscribe(c)
		return
	}

	// for "/:supi/am-data/sor-ack"
	if op == "am-data" && strings.ToUpper("Put") == c.Request.Method {
		s.HandleInfo(c)
		return
	}

	// for "/:ueId/sdm-subscriptions/:subscriptionId"
	if op == "sdm-subscriptions" && strings.ToUpper("Patch") == c.Request.Method {
		var tmpParams gin.Params
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
			strings.ToUpper("Get"),
			"/:supi",
			s.HandleGetSupi,
		},

		{
			"GetSharedData",
			strings.ToUpper("Get"),
			"/shared-data",
			s.HandleGetSharedData,
		},

		{
			"SubscribeToSharedData",
			strings.ToUpper("Post"),
			"/shared-data-subscriptions",
			s.HandleSubscribeToSharedData,
		},
	}
}

func (s *Server) getTwoLayerRoutes() []Route {
	return []Route{
		{
			"GetAmData",
			strings.ToUpper("Get"),
			"/:supi/am-data",
			s.HandleGetAmData,
		},

		{
			"GetSmfSelData",
			strings.ToUpper("Get"),
			"/:supi/smf-select-data",
			s.HandleGetSmfSelectData,
		},

		{
			"GetSmsMngtData",
			strings.ToUpper("Get"),
			"/:supi/sms-mng-data",
			s.HandleGetSmsMngData,
		},

		{
			"GetSmsData",
			strings.ToUpper("Get"),
			"/:supi/sms-data",
			s.HandleGetSmsData,
		},

		{
			"GetSmData",
			strings.ToUpper("Get"),
			"/:supi/sm-data",
			s.HandleGetSmData,
		},

		{
			"GetNSSAI",
			strings.ToUpper("Get"),
			"/:supi/nssai",
			s.HandleGetNssai,
		},

		{
			"Subscribe",
			strings.ToUpper("Post"),
			"/:ueId/sdm-subscriptions",
			s.HandleSubscribe,
		},

		{
			"GetTraceConfigData",
			strings.ToUpper("Get"),
			"/:supi/trace-data",
			s.HandleGetTraceData,
		},

		{
			"GetUeCtxInSmfData",
			strings.ToUpper("Get"),
			"/:supi/ue-context-in-smf-data",
			s.HandleGetUeContextInSmfData,
		},

		{
			"GetUeCtxInSmsfData",
			strings.ToUpper("Get"),
			"/:supi/ue-context-in-smsf-data",
			s.HandleGetUeContextInSmsfData,
		},
	}
}
