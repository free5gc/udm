package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/util/metrics/sbi"
	"github.com/free5gc/util/validator"
)

func (s *Server) getHttpCallBackRoutes() []Route {
	return []Route{
		{
			"Index",
			http.MethodGet,
			"/",
			s.HandleIndex,
		},

		{
			"DataChangeNotificationToNF",
			http.MethodPost,
			"/sdm-subscriptions",
			s.HandleDataChangeNotificationToNF,
		},
	}
}

func (s *Server) HandleDataChangeNotificationToNF(c *gin.Context) {
	var dataChangeNotify models.DataChangeNotify
	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.CallbackLog.Errorf("Get Request Body error: %+v", err)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&dataChangeNotify, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.CallbackLog.Errorln(problemDetail)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(rsp.Status)))
		c.JSON(int(rsp.Status), rsp)
		return
	}

	// TS 29.503 6.1.6.2.21
	if len(dataChangeNotify.NotifyItems) == 0 {
		problemDetail := models.ProblemDetails{
			Title:  "Missing or invalid parameter",
			Status: http.StatusBadRequest,
			Detail: "Mandatory IE NotifyItems is missing or invalid",
			Cause:  "MANDATORY_IE_MISSING",
		}
		logger.CallbackLog.Warnln("Mandatory IE NotifyItems is missing or invalid")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}
	
	supi := dataChangeNotify.UeId
	if !validator.IsValidSupi(supi) {
		problemDetail := models.ProblemDetails{
			Title:  "Invalid Supi format",
			Status: http.StatusBadRequest,
			Detail: "The Supi format is invalid",
			Cause:  "MANDATORY_IE_INCORRECT",
		}
		logger.UecmLog.Warnf("Registration Reject: Invalid Supi format [%s]", supi)
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(problemDetail.Status)))
		c.JSON(int(problemDetail.Status), problemDetail)
		return
	}

	logger.CallbackLog.Infof("Handle DataChangeNotificationToNF")

	s.Processor().DataChangeNotificationProcedure(c, dataChangeNotify.NotifyItems, supi)
}
