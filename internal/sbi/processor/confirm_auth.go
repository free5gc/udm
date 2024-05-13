package processor

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
)

// ConfirmAuth - Create a new confirmation event
func (p *Processor) HandleConfirmAuth(c *gin.Context) {
	var authEvent models.AuthEvent
	// step 1: retrieve http request body
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

	// step 2: convert requestBody to openapi models
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

	problemDetails := p.consumer.ConfirmAuthDataProcedure(authEvent, supi)

	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	} else {
		c.Status(http.StatusCreated)
		return
	}
}
