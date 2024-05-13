package processor

import (
	"net/http"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
	"github.com/gin-gonic/gin"
)

// RegistrationSmfRegistrations - register as SMF
func (p *Processor) HandleRegistrationSmfRegistrations(c *gin.Context) {
	var smfRegistration models.SmfRegistration

	// step 1: retrieve http request body
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

	// step 2: convert requestBody to openapi models
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

	// step 1: log
	logger.UecmLog.Infof("Handle RegistrationSmfRegistrations")

	// step 2: retrieve request
	ueID := c.Params.ByName("ueId")
	pduSessionID := c.Params.ByName("pduSessionId")

	// step 3: handle the message
	header, response, problemDetails := p.consumer.RegistrationSmfRegistrationsProcedure(&smfRegistration, ueID, pduSessionID)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		for key, val := range header { // header response is optional
			c.Header(key, val[0])
		}
		c.JSON(http.StatusCreated, response)
		return
	} else if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	} else {
		// all nil
		c.Status(http.StatusNoContent)
		return
	}

}
