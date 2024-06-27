package sbi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
)

func (s *Server) getUEAuthenticationRoutes() []Route {
	return []Route{
		{
			"Index",
			"GET",
			"/",
			s.HandleIndex,
		},

		{
			"ConfirmAuth",
			strings.ToUpper("Post"),
			"/:supi/auth-events",
			s.HandleConfirmAuth,
		},
	}
}

// ConfirmAuth - Create a new confirmation event
func (s *Server) HandleConfirmAuth(c *gin.Context) {
	var authEvent models.AuthEvent
	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.UeauLog.Errorf("Get Request Body error: %+v", err)
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
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	supi := c.Params.ByName("supi")

	logger.UeauLog.Infoln("Handle ConfirmAuthDataRequest")

	s.Processor().ConfirmAuthDataProcedure(c, authEvent, supi)
}

// GenerateAuthData - Generate authentication data for the UE
func (s *Server) HandleGenerateAuthData(c *gin.Context) {
	var authInfoReq models.AuthenticationInfoRequest

	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.UeauLog.Errorf("Get Request Body error: %+v", err)
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
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	logger.UeauLog.Infoln("Handle GenerateAuthDataRequest")

	supiOrSuci := c.Param("supiOrSuci")

	s.Processor().GenerateAuthDataProcedure(c, authInfoReq, supiOrSuci)
}

func (s *Server) GenAuthDataHandlerFunc(c *gin.Context) {
	c.Params = append(c.Params, gin.Param{Key: "supiOrSuci", Value: c.Param("supi")})
	if strings.ToUpper("Post") == c.Request.Method {
		s.HandleGenerateAuthData(c)
		return
	}

	c.String(http.StatusNotFound, "404 page not found")
}
