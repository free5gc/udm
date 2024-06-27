package sbi

import (
	"net/http"
	"strings"

	"github.com/antihax/optional"
	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/Nudr_DataRepository"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
)

func (s *Server) getUEContextManagementRoutes() []Route {
	return []Route{
		{
			"Index",
			"GET",
			"/",
			s.HandleIndex,
		},

		{
			"GetAmf3gppAccess",
			strings.ToUpper("Get"),
			"/:ueId/registrations/amf-3gpp-access",
			s.HandleGetAmf3gppAccess,
		},

		{
			"GetAmfNon3gppAccess",
			strings.ToUpper("Get"),
			"/:ueId/registrations/amf-non-3gpp-access",
			s.HandleGetAmfNon3gppAccess,
		},

		{
			"RegistrationAmf3gppAccess",
			strings.ToUpper("Put"),
			"/:ueId/registrations/amf-3gpp-access",
			s.HandleRegistrationAmf3gppAccess,
		},

		{
			"Register",
			strings.ToUpper("Put"),
			"/:ueId/registrations/amf-non-3gpp-access",
			s.HandleRegistrationAmfNon3gppAccess,
		},

		{
			"UpdateAmf3gppAccess",
			strings.ToUpper("Patch"),
			"/:ueId/registrations/amf-3gpp-access",
			s.HandleUpdateAmf3gppAccess,
		},

		{
			"UpdateAmfNon3gppAccess",
			strings.ToUpper("Patch"),
			"/:ueId/registrations/amf-non-3gpp-access",
			s.HandleUpdateAmfNon3gppAccess,
		},

		{
			"DeregistrationSmfRegistrations",
			strings.ToUpper("Delete"),
			"/:ueId/registrations/smf-registrations/:pduSessionId",
			s.HandleDeregistrationSmfRegistrations,
		},

		{
			"RegistrationSmfRegistrations",
			strings.ToUpper("Put"),
			"/:ueId/registrations/smf-registrations/:pduSessionId",
			s.HandleRegistrationSmfRegistrations,
		},

		{
			"GetSmsf3gppAccess",
			strings.ToUpper("Get"),
			"/:ueId/registrations/smsf-3gpp-access",
			s.HandleGetSmsf3gppAccess,
		},

		{
			"DeregistrationSmsf3gppAccess",
			strings.ToUpper("Delete"),
			"/:ueId/registrations/smsf-3gpp-access",
			s.HandleDeregistrationSmsf3gppAccess,
		},

		{
			"DeregistrationSmsfNon3gppAccess",
			strings.ToUpper("Delete"),
			"/:ueId/registrations/smsf-non-3gpp-access",
			s.HandleDeregistrationSmsfNon3gppAccess,
		},

		{
			"GetSmsfNon3gppAccess",
			strings.ToUpper("Get"),
			"/:ueId/registrations/smsf-non-3gpp-access",
			s.HandleGetSmsfNon3gppAccess,
		},

		{
			"UpdateSMSFReg3GPP",
			strings.ToUpper("Put"),
			"/:ueId/registrations/smsf-3gpp-access",
			s.HandleUpdateSMSFReg3GPP,
		},

		{
			"RegistrationSmsfNon3gppAccess",
			strings.ToUpper("Put"),
			"/:ueId/registrations/smsf-non-3gpp-access",
			s.HandleRegistrationSmsfNon3gppAccess,
		},
	}
}

// GetAmfNon3gppAccess - retrieve the AMF registration for non-3GPP access information
func (s *Server) HandleGetAmfNon3gppAccess(c *gin.Context) {
	logger.UecmLog.Infoln("Handle GetAmfNon3gppAccessRequest")

	ueId := c.Param("ueId")
	supportedFeatures := c.Query("supported-features")

	var queryAmfContextNon3gppParamOpts Nudr_DataRepository.QueryAmfContextNon3gppParamOpts
	queryAmfContextNon3gppParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	s.Processor().GetAmfNon3gppAccessProcedure(c, queryAmfContextNon3gppParamOpts, ueId)
}

// Register - register as AMF for non-3GPP access
func (s *Server) HandleRegistrationAmfNon3gppAccess(c *gin.Context) {
	var amfNon3GppAccessRegistration models.AmfNon3GppAccessRegistration

	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.UecmLog.Errorf("Get Request Body error: %+v", err)
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
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	logger.UecmLog.Infof("Handle RegisterAmfNon3gppAccessRequest")

	ueID := c.Param("ueId")

	s.Processor().RegisterAmfNon3gppAccessProcedure(c, amfNon3GppAccessRegistration, ueID)
}

// RegistrationAmf3gppAccess - register as AMF for 3GPP access
func (s *Server) HandleRegistrationAmf3gppAccess(c *gin.Context) {
	var amf3GppAccessRegistration models.Amf3GppAccessRegistration
	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.UecmLog.Errorf("Get Request Body error: %+v", err)
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
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	logger.UecmLog.Infof("Handle RegistrationAmf3gppAccess")

	ueID := c.Param("ueId")
	logger.UecmLog.Info("UEID: ", ueID)

	s.Processor().RegistrationAmf3gppAccessProcedure(c, amf3GppAccessRegistration, ueID)
}

// UpdateAmfNon3gppAccess - update a parameter in the AMF registration for non-3GPP access
func (s *Server) HandleUpdateAmfNon3gppAccess(c *gin.Context) {
	var amfNon3GppAccessRegistrationModification models.AmfNon3GppAccessRegistrationModification
	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.UecmLog.Errorf("Get Request Body error: %+v", err)
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
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	logger.UecmLog.Infof("Handle UpdateAmfNon3gppAccessRequest")

	ueID := c.Param("ueId")

	s.Processor().UpdateAmfNon3gppAccessProcedure(c, amfNon3GppAccessRegistrationModification, ueID)
}

// UpdateAmf3gppAccess - Update a parameter in the AMF registration for 3GPP access
func (s *Server) HandleUpdateAmf3gppAccess(c *gin.Context) {
	var amf3GppAccessRegistrationModification models.Amf3GppAccessRegistrationModification

	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.UecmLog.Errorf("Get Request Body error: %+v", err)
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
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	logger.UecmLog.Infof("Handle UpdateAmf3gppAccessRequest")

	ueID := c.Param("ueId")

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
	pduSessionID := c.Params.ByName("pduSessionId")

	s.Processor().DeregistrationSmfRegistrationsProcedure(c, ueID, pduSessionID)
}

// RegistrationSmfRegistrations - register as SMF
func (s *Server) HandleRegistrationSmfRegistrations(c *gin.Context) {
	var smfRegistration models.SmfRegistration

	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.UecmLog.Errorf("Get Request Body error: %+v", err)
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
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	logger.UecmLog.Infof("Handle RegistrationSmfRegistrations")

	ueID := c.Params.ByName("ueId")
	pduSessionID := c.Params.ByName("pduSessionId")

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
	supportedFeatures := c.Query("supported-features")

	s.Processor().GetAmf3gppAccessProcedure(c, ueID, supportedFeatures)
}
