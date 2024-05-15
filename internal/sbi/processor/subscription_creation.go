package processor

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
)

// Subscribe - subscribe to notifications
func (p *Processor) HandleSubscribe(c *gin.Context) {
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
	header, response, problemDetails := p.consumer.SubscribeProcedure(&sdmSubscriptionReq, supi)

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
		c.Status(http.StatusNotFound)
		return
	}
}
