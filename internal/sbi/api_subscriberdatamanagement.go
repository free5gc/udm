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

	// step 1: log
	logger.SdmLog.Infof("Handle GetAmData")

	// step 2: retrieve request
	supi := c.Params.ByName("supi")

	plmnIDStruct, problemDetails := s.getPlmnIDStruct(query)
	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
	supportedFeatures := query.Get("supported-features")

	// step 3: handle the message
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
	c.JSON(http.StatusOK, gin.H{})
}

// PutUpuAck - Nudm_Sdm Info for UPU service operation
func (s *Server) HandlePutUpuAck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// GetSmfSelectData - retrieve a UE's SMF Selection Subscription Data
func (s *Server) HandleGetSmfSelectData(c *gin.Context) {
	query := url.Values{}
	query.Set("plmn-id", c.Query("plmn-id"))
	query.Set("supported-features", c.Query("supported-features"))

	// step 1: log
	logger.SdmLog.Infof("Handle GetSmfSelectData")

	// step 2: retrieve request
	supi := c.Params.ByName("supi")
	plmnIDStruct, problemDetails := s.getPlmnIDStruct(query)
	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
	supportedFeatures := query.Get("supported-features")

	// step 3: handle the message
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

	// step 1: log
	logger.SdmLog.Infof("Handle GetSupiRequest")

	// step 2: retrieve request
	supi := c.Params.ByName("supi")
	plmnIDStruct, problemDetails := s.getPlmnIDStruct(query)
	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
	dataSetNames := strings.Split(query.Get("dataset-names"), ",")
	supportedFeatures := query.Get("supported-features")

	// step 3: handle the message
	s.Processor().GetSupiProcedure(c, supi, plmnID, dataSetNames, supportedFeatures)
}

// GetSharedData - retrieve shared data
func (s *Server) HandleGetSharedData(c *gin.Context) {
	// step 1: log
	logger.SdmLog.Infof("Handle GetSharedData")

	// step 2: retrieve request
	sharedDataIds := c.QueryArray("shared-data-ids")
	supportedFeatures := c.QueryArray("supported-features")
	// step 3: handle the message
	s.Processor().GetSharedDataProcedure(c, sharedDataIds, supportedFeatures[0])
}

// SubscribeToSharedData - subscribe to notifications for shared data
func (s *Server) HandleSubscribeToSharedData(c *gin.Context) {
	var sharedDataSubsReq models.SdmSubscription
	// step 1: retrieve http request body
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

	// step 2: convert requestBody to openapi models
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

	// step 1: log
	logger.SdmLog.Infof("Handle SubscribeToSharedData")

	// step 2: retrieve request

	// step 3: handle the message
	s.Processor().SubscribeToSharedDataProcedure(c, &sharedDataSubsReq)
}

// Subscribe - subscribe to notifications
func (s *Server) HandleSubscribe(c *gin.Context) {
	var sdmSubscriptionReq models.SdmSubscription

	// step 1: retrieve http request body
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

	// step 2: convert requestBody to openapi models
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

	// step 1: log
	logger.SdmLog.Infof("Handle Subscribe")

	// step 2: retrieve request
	supi := c.Params.ByName("supi")

	// step 3: handle the message
	s.Processor().SubscribeProcedure(c, &sdmSubscriptionReq, supi)
}

// Unsubscribe - unsubscribe from notifications
func (s *Server) HandleUnsubscribe(c *gin.Context) {
	logger.SdmLog.Infof("Handle Unsubscribe")

	// step 2: retrieve request
	supi := c.Params.ByName("supi")
	subscriptionID := c.Params.ByName("subscriptionId")

	// step 3: handle the message
	s.Processor().UnsubscribeProcedure(c, supi, subscriptionID)
}

// UnsubscribeForSharedData - unsubscribe from notifications for shared data
func (s *Server) HandleUnsubscribeForSharedData(c *gin.Context) {
	logger.SdmLog.Infof("Handle UnsubscribeForSharedData")

	// step 2: retrieve request
	subscriptionID := c.Params.ByName("subscriptionId")
	// step 3: handle the message
	s.Processor().UnsubscribeForSharedDataProcedure(c, subscriptionID)
}

// Modify - modify the subscription
func (s *Server) HandleModify(c *gin.Context) {
	var sdmSubsModificationReq models.SdmSubsModification
	// step 1: retrieve http request body
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

	// step 2: convert requestBody to openapi models
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

	// step 1: log
	logger.SdmLog.Infof("Handle Modify")

	// step 2: retrieve request
	supi := c.Params.ByName("supi")
	subscriptionID := c.Params.ByName("subscriptionId")

	// step 3: handle the message
	s.Processor().ModifyProcedure(c, &sdmSubsModificationReq, supi, subscriptionID)
}

// ModifyForSharedData - modify the subscription
func (s *Server) HandleModifyForSharedData(c *gin.Context) {
	var sharedDataSubscriptions models.SdmSubsModification
	// step 1: retrieve http request body
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

	// step 2: convert requestBody to openapi models
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

	// step 1: log
	logger.SdmLog.Infof("Handle ModifyForSharedData")

	// step 2: retrieve request
	supi := c.Params.ByName("supi")
	subscriptionID := c.Params.ByName("subscriptionId")

	// step 3: handle the message
	s.Processor().ModifyForSharedDataProcedure(c, &sharedDataSubscriptions, supi, subscriptionID)
}

// GetTraceData - retrieve a UE's Trace Configuration Data
func (s *Server) HandleGetTraceData(c *gin.Context) {
	// step 1: log
	logger.SdmLog.Infof("Handle GetTraceData")

	// step 2: retrieve request
	supi := c.Params.ByName("supi")
	plmnID := c.Query("plmn-id")

	// step 3: handle the message
	s.Processor().GetTraceDataProcedure(c, supi, plmnID)
}

// GetUeContextInSmfData - retrieve a UE's UE Context In SMF Data
func (s *Server) HandleGetUeContextInSmfData(c *gin.Context) {
	// step 1: log
	logger.SdmLog.Infof("Handle GetUeContextInSmfData")

	// step 2: retrieve request
	supi := c.Params.ByName("supi")
	supportedFeatures := c.Query("supported-features")

	// step 3: handle the message
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

	// step 1: log
	logger.SdmLog.Infof("Handle GetNssai")

	// step 2: retrieve request
	supi := c.Params.ByName("supi")
	plmnIDStruct, problemDetails := s.getPlmnIDStruct(query)
	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
	supportedFeatures := query.Get("supported-features")

	// step 3: handle the message
	s.Processor().GetNssaiProcedure(c, supi, plmnID, supportedFeatures)
}

// GetSmData - retrieve a UE's Session Management Subscription Data
func (s *Server) HandleGetSmData(c *gin.Context) {
	query := url.Values{}
	query.Set("plmn-id", c.Query("plmn-id"))
	query.Set("dnn", c.Query("dnn"))
	query.Set("single-nssai", c.Query("single-nssai"))
	query.Set("supported-features", c.Query("supported-features"))

	// step 1: log
	logger.SdmLog.Infof("Handle GetSmData")

	// step 2: retrieve request
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

	// step 3: handle the message
	s.Processor().GetSmDataProcedure(c, supi, plmnID, Dnn, Snssai, supportedFeatures)
}

// GetIdTranslationResult - retrieve a UE's SUPI
func (s *Server) HandleGetIdTranslationResult(c *gin.Context) {
	// req.Query.Set("SupportedFeatures", c.Query("supported-features"))

	// step 1: log
	logger.SdmLog.Infof("Handle GetIdTranslationResultRequest")

	// step 2: retrieve request
	gpsi := c.Params.ByName("gpsi")

	// step 3: handle the message
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

	// for "/:gpsi/id-translation-result"
	if op == "id-translation-result" && strings.ToUpper("Get") == c.Request.Method {
		c.Params = append(c.Params, gin.Param{Key: "gpsi", Value: c.Param("supi")})
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

	// for "/:supi/sdm-subscriptions/:subscriptionId"
	if op == "sdm-subscriptions" && strings.ToUpper("Delete") == c.Request.Method {
		var tmpParams gin.Params
		tmpParams = append(tmpParams, gin.Param{Key: "supi", Value: c.Param("supi")})
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

	// for "/:supi/sdm-subscriptions/:subscriptionId"
	if op == "sdm-subscriptions" && strings.ToUpper("Patch") == c.Request.Method {
		var tmpParams gin.Params
		tmpParams = append(tmpParams, gin.Param{Key: "supi", Value: c.Param("supi")})
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
			"GetSupi",
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
			"GetSmfSelectData",
			strings.ToUpper("Get"),
			"/:supi/smf-select-data",
			s.HandleGetSmfSelectData,
		},

		{
			"GetSmsMngData",
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
			"GetNssai",
			strings.ToUpper("Get"),
			"/:supi/nssai",
			s.HandleGetNssai,
		},

		{
			"Subscribe",
			strings.ToUpper("Post"),
			"/:supi/sdm-subscriptions",
			s.HandleSubscribe,
		},

		{
			"GetTraceData",
			strings.ToUpper("Get"),
			"/:supi/trace-data",
			s.HandleGetTraceData,
		},

		{
			"GetUeContextInSmfData",
			strings.ToUpper("Get"),
			"/:supi/ue-context-in-smf-data",
			s.HandleGetUeContextInSmfData,
		},

		{
			"GetUeContextInSmsfData",
			strings.ToUpper("Get"),
			"/:supi/ue-context-in-smsf-data",
			s.HandleGetUeContextInSmsfData,
		},
	}
}
