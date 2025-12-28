package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/udm/internal/util"
	"github.com/free5gc/util/metrics/sbi"
)

func (s *Server) getUEAuthenticationRoutes() []Route {
	return []Route{
		{
			"Index",
			http.MethodGet,
			"/",
			s.HandleIndex,
		},
	}
}

// ConfirmAuth - Create a new confirmation event
func (s *Server) HandleConfirmAuth(c *gin.Context) {
	var authEvent models.AuthEvent
	// TS 29.503 6.3.6.2.3
	// Validate SUPI format
	supi := c.Params.ByName("supi")
	if !util.IsValidSupi(supi) {
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "Supi is invalid",
			Cause:  "INVALID_KEY",
		}
		logger.UeauLog.Warnln("Supi is invalid")
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
		logger.UeauLog.Errorf("Get Request Body error: %+v", err)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&authEvent, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.UeauLog.Errorln(problemDetail)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(rsp.Status)))
		c.JSON(int(rsp.Status), rsp)
		return
	}

	// TS 29.503 6.3.6.2.7 requirements check
	missingIE := ""
	if authEvent.NfInstanceId == "" {
		missingIE = "nfInstanceId"
	} else if authEvent.TimeStamp == nil {
		missingIE = "timestamp"
	} else if authEvent.AuthType == "" {
		missingIE = "authtype"
	} else if authEvent.ServingNetworkName == "" {
		missingIE = "servingNetworkName"
	}

	if missingIE != "" {
		problemDetail := models.ProblemDetails{
			Title:  "Missing or invalid parameter",
			Status: http.StatusBadRequest,
			Detail: "Mandatory IE " + missingIE + " is missing or invalid",
			Cause:  "MISSING_OR_INVALID_PARAMETER",
		}
		logger.UeauLog.Warnln("Mandatory IE " + missingIE + "is missing or invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}

	logger.UeauLog.Infoln("Handle ConfirmAuthDataRequest")

	s.Processor().ConfirmAuthDataProcedure(c, authEvent, supi)
}

// GenerateAuthData - Generate authentication data for the UE
func (s *Server) HandleGenerateAuthData(c *gin.Context) {
	var authInfoReq models.AuthenticationInfoRequest
	// TS 29.503 6.3.3.2.2
	// Validate SUPI or SUCI format
	supiOrSuci := c.Param("supiOrSuci")
	if !util.IsValidSupi(supiOrSuci) && !util.IsValidSuci(supiOrSuci) {
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: "Supi or Suci is invalid",
			Cause:  "INVALID_KEY",
		}
		logger.UeauLog.Warnln("Supi or Suci is invalid")
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
		logger.UeauLog.Errorf("Get Request Body error: %+v", err)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&authInfoReq, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.UeauLog.Errorln(problemDetail)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(rsp.Status)))
		c.JSON(int(rsp.Status), rsp)
		return
	}

	// TS 29.503 6.3.6.2.2 requirements check
	missingIE := ""
	if authInfoReq.ServingNetworkName == "" {
		missingIE = "servingNetworkName"
	} else if authInfoReq.AusfInstanceId == "" {
		missingIE = "ausfInstanceId"
	}

	if missingIE != "" {
		problemDetail := models.ProblemDetails{
			Title:  "Missing or invalid parameter",
			Status: http.StatusBadRequest,
			Detail: "Mandatory IE " + missingIE + " is missing or invalid",
			Cause:  "MISSING_OR_INVALID_PARAMETER",
		}
		logger.UeauLog.Warnln("Mandatory IE " + missingIE + "is missing or invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}

	logger.UeauLog.Infoln("Handle GenerateAuthDataRequest")

	s.Processor().GenerateAuthDataProcedure(c, authInfoReq, supiOrSuci)
}

func (s *Server) HandleDeleteAuth(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGenerateAv(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGenerateGbaAv(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGenerateProseAV(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetRgAuthData(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) UEAUTwoLayerPathHandlerFunc(c *gin.Context) {
	twoLayer := c.Param("twoLayer")

	// for "/:supi/auth-events"
	if twoLayer == "auth-events" && http.MethodPost == c.Request.Method {
		s.HandleConfirmAuth(c)
		return
	}

	// for "/:supiOrSuci/security-information-rg"
	if twoLayer == "security-information-rg" && http.MethodGet == c.Request.Method {
		var tmpParams gin.Params
		tmpParams = append(tmpParams, gin.Param{Key: "supiOrSuci", Value: c.Param("supi")})
		c.Params = tmpParams
		s.HandleGetRgAuthData(c)
		return
	}

	c.String(http.StatusNotFound, "404 page not found")
}

func (s *Server) UEAUThreeLayerPathHandlerFunc(c *gin.Context) {
	twoLayer := c.Param("twoLayer")

	// for "/:supi/auth-events/:authEventId"
	if twoLayer == "auth-events" && http.MethodPut == c.Request.Method {
		s.HandleDeleteAuth(c)
		return
	}

	// for "/:supi/gba-security-information/generate-av"
	if twoLayer == "gba-security-information" && http.MethodPost == c.Request.Method {
		s.HandleGenerateGbaAv(c)
		return
	}

	// for "/:supiOrSuci/prose-security-information/generate-av"
	if twoLayer == "prose-security-information" && http.MethodPost == c.Request.Method {
		var tmpParams gin.Params
		tmpParams = append(tmpParams, gin.Param{Key: "supiOrSuci", Value: c.Param("supi")})
		c.Params = tmpParams
		s.HandleGenerateProseAV(c)
		return
	}

	// for "/:supiOrSuci/security-information/generate-auth-data"
	if twoLayer == "security-information" && http.MethodPost == c.Request.Method {
		var tmpParams gin.Params
		tmpParams = append(tmpParams, gin.Param{Key: "supiOrSuci", Value: c.Param("supi")})
		c.Params = tmpParams
		s.HandleGenerateAuthData(c)
		return
	}

	c.String(http.StatusNotFound, "404 page not found")
}
