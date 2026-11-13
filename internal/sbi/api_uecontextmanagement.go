package sbi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	Nudr_DataRepository "github.com/free5gc/openapi/udr/DR"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/util/metrics/sbi"
	"github.com/free5gc/util/validator"
)

func (s *Server) getUEContextManagementRoutes() []Route {
	return []Route{
		{
			Name:        "Index",
			Method:      http.MethodGet,
			Pattern:     "/",
			HandlerFunc: s.HandleIndex,
		},

		{
			Name:        "Get3GppRegistration",
			Method:      http.MethodGet,
			Pattern:     "/:ueId/registrations/amf-3gpp-access",
			HandlerFunc: s.HandleGetAmf3gppAccess,
		},

		{
			Name:        "GetNon3GppRegistration",
			Method:      http.MethodGet,
			Pattern:     "/:ueId/registrations/amf-non-3gpp-access",
			HandlerFunc: s.HandleGetAmfNon3gppAccess,
		},

		{
			Name:        "Call3GppRegistration",
			Method:      http.MethodPut,
			Pattern:     "/:ueId/registrations/amf-3gpp-access",
			HandlerFunc: s.HandleRegistrationAmf3gppAccess,
		},

		{
			Name:        "Non3GppRegistration",
			Method:      http.MethodPut,
			Pattern:     "/:ueId/registrations/amf-non-3gpp-access",
			HandlerFunc: s.HandleRegistrationAmfNon3gppAccess,
		},

		{
			Name:        "Update3GppRegistration",
			Method:      http.MethodPatch,
			Pattern:     "/:ueId/registrations/amf-3gpp-access",
			HandlerFunc: s.HandleUpdateAmf3gppAccess,
		},

		{
			Name:        "UpdateNon3GppRegistration",
			Method:      http.MethodPatch,
			Pattern:     "/:ueId/registrations/amf-non-3gpp-access",
			HandlerFunc: s.HandleUpdateAmfNon3gppAccess,
		},

		{
			Name:        "SmfDeregistration",
			Method:      http.MethodDelete,
			Pattern:     "/:ueId/registrations/smf-registrations/:pduSessionId",
			HandlerFunc: s.HandleDeregistrationSmfRegistrations,
		},

		{
			Name:        "Registration",
			Method:      http.MethodPut,
			Pattern:     "/:ueId/registrations/smf-registrations/:pduSessionId",
			HandlerFunc: s.HandleRegistrationSmfRegistrations,
		},

		{
			Name:        "Get3GppSmsfRegistration",
			Method:      http.MethodGet,
			Pattern:     "/:ueId/registrations/smsf-3gpp-access",
			HandlerFunc: s.HandleGetSmsf3gppAccess,
		},

		{
			Name:        "Call3GppSmsfDeregistration",
			Method:      http.MethodDelete,
			Pattern:     "/:ueId/registrations/smsf-3gpp-access",
			HandlerFunc: s.HandleDeregistrationSmsf3gppAccess,
		},

		{
			Name:        "Non3GppSmsfDeregistration",
			Method:      http.MethodDelete,
			Pattern:     "/:ueId/registrations/smsf-non-3gpp-access",
			HandlerFunc: s.HandleDeregistrationSmsfNon3gppAccess,
		},

		{
			Name:        "GetNon3GppSmsfRegistration",
			Method:      http.MethodGet,
			Pattern:     "/:ueId/registrations/smsf-non-3gpp-access",
			HandlerFunc: s.HandleGetSmsfNon3gppAccess,
		},

		{
			Name:        "Call3GppSmsfRegistration",
			Method:      http.MethodPut,
			Pattern:     "/:ueId/registrations/smsf-3gpp-access",
			HandlerFunc: s.HandleUpdateSMSFReg3GPP,
		},

		{
			Name:        "Non3GppSmsfRegistration",
			Method:      http.MethodPut,
			Pattern:     "/:ueId/registrations/smsf-non-3gpp-access",
			HandlerFunc: s.HandleRegistrationSmsfNon3gppAccess,
		},

		{
			Name:        "DeregAMF",
			Method:      http.MethodPost,
			Pattern:     "/:ueId/registrations/amf-3gpp-access/dereg-amf",
			HandlerFunc: s.HandleDeregAMF,
		},

		{
			Name:        "GetIpSmGwRegistration",
			Method:      http.MethodGet,
			Pattern:     "/:ueId/registrations/ip-sm-gw",
			HandlerFunc: s.HandleGetIpSmGwRegistration,
		},

		{
			Name:        "GetLocationInfo",
			Method:      http.MethodGet,
			Pattern:     "/:ueId/registrations/location",
			HandlerFunc: s.HandleGetLocationInfo,
		},

		{
			Name:        "GetNwdafRegistration",
			Method:      http.MethodGet,
			Pattern:     "/:ueId/registrations/nwdaf-registrations",
			HandlerFunc: s.HandleGetNwdafRegistration,
		},

		{
			Name:        "GetRegistrations",
			Method:      http.MethodGet,
			Pattern:     "/:ueId/registrations",
			HandlerFunc: s.HandleGetRegistrations,
		},

		{
			Name:        "GetSmfRegistration",
			Method:      http.MethodGet,
			Pattern:     "/:ueId/registrations/smf-registrations",
			HandlerFunc: s.HandleGetSmfRegistration,
		},

		{
			Name:        "IpSmGwDeregistration",
			Method:      http.MethodDelete,
			Pattern:     "/:ueId/registrations/ip-sm-gw",
			HandlerFunc: s.HandleIpSmGwDeregistration,
		},

		{
			Name:        "IpSmGwRegistration",
			Method:      http.MethodPut,
			Pattern:     "/:ueId/registrations/ip-sm-gw",
			HandlerFunc: s.HandleIpSmGwRegistration,
		},

		{
			Name:        "NwdafDeregistration",
			Method:      http.MethodDelete,
			Pattern:     "/:ueId/registrations/nwdaf-registrations/:nwdafRegistrationId",
			HandlerFunc: s.HandleNwdafDeregistration,
		},

		{
			Name:        "NwdafRegistration",
			Method:      http.MethodPut,
			Pattern:     "/:ueId/registrations/nwdaf-registrations/:nwdafRegistrationId",
			HandlerFunc: s.HandleNwdafRegistration,
		},

		{
			Name:        "PeiUpdate",
			Method:      http.MethodPost,
			Pattern:     "/:ueId/registrations/amf-3gpp-access/pei-update",
			HandlerFunc: s.HandlePeiUpdate,
		},

		{
			Name:        "RetrieveSmfRegistration",
			Method:      http.MethodGet,
			Pattern:     "/:ueId/registrations/smf-registrations/:pduSessionId",
			HandlerFunc: s.HandleRetrieveSmfRegistration,
		},

		{
			Name:        "SendRoutingInfoSm",
			Method:      http.MethodPost,
			Pattern:     "/:ueId/registrations/send-routing-info-sm",
			HandlerFunc: s.HandleSendRoutingInfoSm,
		},

		{
			Name:        "TriggerPCSCFRestoration",
			Method:      http.MethodPost,
			Pattern:     "/restore-pcscf",
			HandlerFunc: s.HandleTriggerPCSCFRestoration,
		},

		{
			Name:        "UpdateNwdafRegistration",
			Method:      http.MethodPatch,
			Pattern:     "/:ueId/registrations/nwdaf-registrations/:nwdafRegistrationId",
			HandlerFunc: s.HandleUpdateNwdafRegistration,
		},

		{
			Name:        "UpdateRoamingInformation",
			Method:      http.MethodPost,
			Pattern:     "/:ueId/registrations/amf-3gpp-access/roaming-info-update",
			HandlerFunc: s.HandleUpdateRoamingInformation,
		},

		{
			Name:        "UpdateSmfRegistration",
			Method:      http.MethodPatch,
			Pattern:     "/:ueId/registrations/smf-registrations/:pduSessionId",
			HandlerFunc: s.HandleUpdateSmfRegistration,
		},
	}
}

// GetAmfNon3gppAccess - retrieve the AMF registration for non-3GPP access information
func (s *Server) HandleGetAmfNon3gppAccess(c *gin.Context) {
	logger.UecmLog.Infoln("Handle GetAmfNon3gppAccessRequest")

	ueId := c.Param("ueId")
	// TS 29.503 5.3.2.5.3
	// Validate SUPI and GPSI format the UE ID (SUPI or GPSI) shall be in the format defined in 3GPP TS 23.003 & 29.571
	valid := validator.IsValidSupi(ueId) || validator.IsValidGpsi(ueId)
	if !valid {
		problemDetail := models.ProblemDetails{
			Title:  "Invalid ueID format",
			Status: http.StatusBadRequest,
			Detail: "The ueID format is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.UecmLog.Warnf("Registration Reject: Invalid ueID format [%s]", ueId)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}
	supportedFeatures := c.Query("supported-features")
	var queryAmfContextNon3gppRequest Nudr_DataRepository.QueryAmfContextNon3gppRequest
	queryAmfContextNon3gppRequest.SupportedFeatures = &supportedFeatures
	queryAmfContextNon3gppRequest.UeId = &ueId
	s.Processor().GetAmfNon3gppAccessProcedure(c, queryAmfContextNon3gppRequest, ueId)
}

// Register - register as AMF for non-3GPP access
func (s *Server) HandleRegistrationAmfNon3gppAccess(c *gin.Context) {
	var amfNon3GppAccessRegistration models.Udm_UECM_AmfNon3GppAccessRegistration

	ueID := c.Param("ueId")
	// TS 29.503 5.3.2.2.3
	// Validate SUPI format the UE ID (SUPI) shall be in the format defined in 3GPP TS 23.003 & 29.571
	valid := validator.IsValidSupi(ueID)
	if !valid {
		problemDetail := models.ProblemDetails{
			Title:  "Invalid ueID format",
			Status: http.StatusBadRequest,
			Detail: "The ueID format is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.UecmLog.Warnf("Registration Reject: Invalid ueID format [%s]", ueID)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
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
		logger.UecmLog.Errorf("Get Request Body error: %+v", err)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&amfNon3GppAccessRegistration, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.UecmLog.Errorln(problemDetail)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(rsp.Status)))
		c.JSON(int(rsp.Status), rsp)
		return
	}

	// TS 29.503 6.2.6.2.3 requirements check
	missingIEList := make([]string, 0)
	if amfNon3GppAccessRegistration.AmfInstanceId == "" {
		missingIEList = append(missingIEList, "AmfInstanceId")
	}
	if amfNon3GppAccessRegistration.Guami == nil {
		missingIEList = append(missingIEList, "Guami")
	}
	if amfNon3GppAccessRegistration.DeregCallbackUri == "" {
		missingIEList = append(missingIEList, "DeregCallbackUri")
	}
	if amfNon3GppAccessRegistration.RatType == "" {
		missingIEList = append(missingIEList, "RatType")
	}

	if len(missingIEList) > 0 {
		missingIEs := strings.Join(missingIEList, ", ")
		problemDetail := models.ProblemDetails{
			Title:  "Missing or invalid parameter",
			Status: http.StatusBadRequest,
			Detail: "Mandatory IE [" + missingIEs + "] is missing or invalid",
			Cause:  "MANDATORY_IE_MISSING",
		}
		logger.UecmLog.Warnln("Mandatory IE [" + missingIEs + "] is missing or invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}

	logger.UecmLog.Infof("Handle RegisterAmfNon3gppAccessRequest")

	s.Processor().RegisterAmfNon3gppAccessProcedure(c, amfNon3GppAccessRegistration, ueID)
}

// RegistrationAmf3gppAccess - register as AMF for 3GPP access
func (s *Server) HandleRegistrationAmf3gppAccess(c *gin.Context) {
	var amf3GppAccessRegistration models.Udm_UECM_Amf3GppAccessRegistration

	ueID := c.Param("ueId")
	// TS 29.503 5.3.2.2.2
	// Validate SUPI format the UE ID (SUPI) shall be in the format defined in 3GPP TS 23.003 & 29.571
	valid := validator.IsValidSupi(ueID)
	if !valid {
		problemDetail := models.ProblemDetails{
			Title:  "Invalid ueID format",
			Status: http.StatusBadRequest,
			Detail: "The ueID format is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.UecmLog.Warnf("Registration Reject: Invalid ueID format [%s]", ueID)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
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
		logger.UecmLog.Errorf("Get Request Body error: %+v", err)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&amf3GppAccessRegistration, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.UecmLog.Errorln(problemDetail)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(rsp.Status)))
		c.JSON(int(rsp.Status), rsp)
		return
	}
	// TS 29.503 6.2.6.2.2 requirements check
	missingIEList := make([]string, 0)
	if amf3GppAccessRegistration.AmfInstanceId == "" {
		missingIEList = append(missingIEList, "AmfInstanceId")
	}
	if amf3GppAccessRegistration.Guami == nil {
		missingIEList = append(missingIEList, "Guami")
	}
	if amf3GppAccessRegistration.DeregCallbackUri == "" {
		missingIEList = append(missingIEList, "DeregCallbackUri")
	}
	if amf3GppAccessRegistration.RatType == "" {
		missingIEList = append(missingIEList, "RatType")
	}

	if len(missingIEList) > 0 {
		missingIEs := strings.Join(missingIEList, ", ")
		problemDetail := models.ProblemDetails{
			Title:  "Missing or invalid parameter",
			Status: http.StatusBadRequest,
			Detail: "Mandatory IE [" + missingIEs + "] is missing or invalid",
			Cause:  "MANDATORY_IE_MISSING",
		}
		logger.UecmLog.Warnln("Mandatory IE [" + missingIEs + "] is missing or invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}

	logger.UecmLog.Infof("Handle RegistrationAmf3gppAccess")

	logger.UecmLog.Info("UEID: ", ueID)

	s.Processor().RegistrationAmf3gppAccessProcedure(c, amf3GppAccessRegistration, ueID)
}

// UpdateAmfNon3gppAccess - update a parameter in the AMF registration for non-3GPP access
func (s *Server) HandleUpdateAmfNon3gppAccess(c *gin.Context) {
	var amfNon3GppAccessRegistrationModification models.Udm_UECM_AmfNon3GppAccessRegistrationModification

	ueID := c.Param("ueId")
	// TS 29.503 5.3.2.6.3
	// Validate SUPI format the UE ID (SUPI) shall be in the format defined in 3GPP TS 23.003 & 29.571
	valid := validator.IsValidSupi(ueID)
	if !valid {
		problemDetail := models.ProblemDetails{
			Title:  "Invalid ueID format",
			Status: http.StatusBadRequest,
			Detail: "The ueID format is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.UecmLog.Warnf("Registration Reject: Invalid ueID format [%s]", ueID)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
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
		logger.UecmLog.Errorf("Get Request Body error: %+v", err)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&amfNon3GppAccessRegistrationModification, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.UecmLog.Errorln(problemDetail)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(rsp.Status)))
		c.JSON(int(rsp.Status), rsp)
		return
	}

	// TS 29.503 6.2.6.2.8 requirements check
	if amfNon3GppAccessRegistrationModification.Guami == nil {
		problemDetail := models.ProblemDetails{
			Title:  "Missing or invalid parameter",
			Status: http.StatusBadRequest,
			Detail: "Mandatory IE Guami is missing or invalid",
			Cause:  "MANDATORY_IE_MISSING",
		}
		logger.UecmLog.Warnln("Mandatory IE Guami is missing or invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}

	logger.UecmLog.Infof("Handle UpdateAmfNon3gppAccessRequest")

	s.Processor().UpdateAmfNon3gppAccessProcedure(c, amfNon3GppAccessRegistrationModification, ueID)
}

// UpdateAmf3gppAccess - Update a parameter in the AMF registration for 3GPP access
func (s *Server) HandleUpdateAmf3gppAccess(c *gin.Context) {
	var amf3GppAccessRegistrationModification models.Udm_UECM_Amf3GppAccessRegistrationModification

	ueID := c.Param("ueId")
	// TS 29.503 5.3.2.6.2
	// Validate SUPI format the UE ID (SUPI) shall be in the format defined in 3GPP TS 23.003 & 29.571
	valid := validator.IsValidSupi(ueID)
	if !valid {
		problemDetail := models.ProblemDetails{
			Title:  "Invalid ueID format",
			Status: http.StatusBadRequest,
			Detail: "The ueID format is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.UecmLog.Warnf("Registration Reject: Invalid ueID format [%s]", ueID)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
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
		logger.UecmLog.Errorf("Get Request Body error: %+v", err)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&amf3GppAccessRegistrationModification, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.UecmLog.Errorln(problemDetail)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(rsp.Status)))
		c.JSON(int(rsp.Status), rsp)
		return
	}

	// TS 29.503 6.2.6.2.7 requirements check
	if amf3GppAccessRegistrationModification.Guami == nil {
		problemDetail := models.ProblemDetails{
			Title:  "Missing or invalid parameter",
			Status: http.StatusBadRequest,
			Detail: "Mandatory IE Guami  is missing or invalid",
			Cause:  "MANDATORY_IE_MISSING",
		}
		logger.UecmLog.Warnln("Mandatory IE Guami is missing or invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}

	logger.UecmLog.Infof("Handle UpdateAmf3gppAccessRequest")

	s.Processor().UpdateAmf3gppAccessProcedure(c, amf3GppAccessRegistrationModification, ueID)
}

// DeregistrationSmsfNon3gppAccess - delete SMSF registration for non 3GPP access
func (s *Server) HandleDeregistrationSmsfNon3gppAccess(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

// DeregistrationSmsf3gppAccess - delete the SMSF registration for 3GPP access
func (s *Server) HandleDeregistrationSmsf3gppAccess(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

// GetSmsfNon3gppAccess - retrieve the SMSF registration for non-3GPP access information
func (s *Server) HandleGetSmsfNon3gppAccess(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

// RegistrationSmsfNon3gppAccess - register as SMSF for non-3GPP access
func (s *Server) HandleRegistrationSmsfNon3gppAccess(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

// UpdateSMSFReg3GPP - register as SMSF for 3GPP access
func (s *Server) HandleUpdateSMSFReg3GPP(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

// GetSmsf3gppAccess - retrieve the SMSF registration for 3GPP access information
func (s *Server) HandleGetSmsf3gppAccess(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

// DeregistrationSmfRegistrations - delete an SMF registration
func (s *Server) HandleDeregistrationSmfRegistrations(c *gin.Context) {
	logger.UecmLog.Infof("Handle DeregistrationSmfRegistrations")

	ueID := c.Params.ByName("ueId")
	// TS 29.503 5.3.2.4.4
	// Validate SUPI format the UE ID (SUPI) shall be in the format defined in 3GPP TS 23.003 & 29.571
	valid := validator.IsValidSupi(ueID)
	if !valid {
		problemDetail := models.ProblemDetails{
			Title:  "Invalid ueID format",
			Status: http.StatusBadRequest,
			Detail: "The ueID format is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.UecmLog.Warnf("Registration Reject: Invalid ueID format [%s]", ueID)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}
	pduSessionID := c.Params.ByName("pduSessionId")
	// TS 29.571 5.4.2 valid PDU Session ID is an integer in the range 0 to 255
	if !validator.IsValidPduSessionID(pduSessionID) {
		problemDetail := models.ProblemDetails{
			Title:  "Missing or invalid parameter",
			Status: http.StatusBadRequest,
			Detail: "Mandatory IE PduSessionId is missing or invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.UecmLog.Warnln("Mandatory IE PduSessionId is missing or invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}

	s.Processor().DeregistrationSmfRegistrationsProcedure(c, ueID, pduSessionID)
}

// RegistrationSmfRegistrations - register as SMF
func (s *Server) HandleRegistrationSmfRegistrations(c *gin.Context) {
	var smfRegistration models.Udm_UECM_SmfRegistration

	ueID := c.Params.ByName("ueId")
	// TS 29.503 5.3.2.2.4
	// Validate SUPI format the UE ID (SUPI) shall be in the format defined in 3GPP TS 23.003 & 29.571
	valid := validator.IsValidSupi(ueID)
	if !valid {
		problemDetail := models.ProblemDetails{
			Title:  "Invalid ueID format",
			Status: http.StatusBadRequest,
			Detail: "The ueID format is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.UecmLog.Warnf("Registration Reject: Invalid ueID format [%s]", ueID)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
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
		logger.UecmLog.Errorf("Get Request Body error: %+v", err)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&smfRegistration, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.UecmLog.Errorln(problemDetail)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(rsp.Status)))
		c.JSON(int(rsp.Status), rsp)
		return
	}

	// TS 29.503 6.2.6.2.4 requirements check
	missingIEList := make([]string, 0)
	if smfRegistration.SmfInstanceId == "" {
		missingIEList = append(missingIEList, "SmfInstanceId")
	}
	if smfRegistration.SingleNssai == nil {
		missingIEList = append(missingIEList, "SingleNssai")
	}
	if smfRegistration.PlmnId == nil {
		missingIEList = append(missingIEList, "PlmnId")
	}

	if len(missingIEList) > 0 {
		missingIEs := strings.Join(missingIEList, ", ")
		problemDetail := models.ProblemDetails{
			Title:  "Missing or invalid parameter",
			Status: http.StatusBadRequest,
			Detail: "Mandatory IE [" + missingIEs + "] is missing or invalid",
			Cause:  "MANDATORY_IE_MISSING",
		}
		logger.UecmLog.Warnln("Mandatory IE [" + missingIEs + "] is missing or invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}

	logger.UecmLog.Infof("Handle RegistrationSmfRegistrations")

	pduSessionID := c.Params.ByName("pduSessionId")
	// TS 29.571 5.4.2 valid PDU Session ID is an integer in the range 0 to 255
	if !validator.IsValidPduSessionID(pduSessionID) {
		problemDetail := models.ProblemDetails{
			Title:  "Missing or invalid parameter",
			Status: http.StatusBadRequest,
			Detail: "Mandatory IE PduSessionId is missing or invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.UecmLog.Warnln("Mandatory IE PduSessionId is missing or invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}

	s.Processor().RegistrationSmfRegistrationsProcedure(
		c,
		&smfRegistration,
		ueID,
		pduSessionID,
	)
}

// GetAmf3gppAccess - retrieve the AMF registration for 3GPP access information
func (s *Server) HandleGetAmf3gppAccess(c *gin.Context) {
	logger.UecmLog.Infof("Handle HandleGetAmf3gppAccessRequest")

	ueID := c.Param("ueId")
	// TS 29.503 5.3.2.5.2
	// Validate SUPI and GPSI format the UE ID (SUPI or GPSI) shall be in the format defined in 3GPP TS 23.003 & 29.571
	valid := validator.IsValidSupi(ueID) || validator.IsValidGpsi(ueID)
	if !valid {
		problemDetail := models.ProblemDetails{
			Title:  "Invalid ueID format",
			Status: http.StatusBadRequest,
			Detail: "The ueID format is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.UecmLog.Warnf("Registration Reject: Invalid ueID format [%s]", ueID)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}
	supportedFeatures := c.Query("supported-features")

	s.Processor().GetAmf3gppAccessProcedure(c, ueID, supportedFeatures)
}

func (s *Server) HandleDeregAMF(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetIpSmGwRegistration(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetLocationInfo(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetNwdafRegistration(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetRegistrations(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetSmfRegistration(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleIpSmGwDeregistration(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleIpSmGwRegistration(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleNwdafDeregistration(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleNwdafRegistration(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandlePeiUpdate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleRetrieveSmfRegistration(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleSendRoutingInfoSm(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleTriggerPCSCFRestoration(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleUpdateNwdafRegistration(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleUpdateRoamingInformation(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleUpdateSmfRegistration(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}
