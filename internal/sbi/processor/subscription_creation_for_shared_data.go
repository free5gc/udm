package processor

import (
	"net/http"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/util/httpwrapper"
	"github.com/gin-gonic/gin"
)

// SubscribeToSharedData - subscribe to notifications for shared data
func HTTPSubscribeToSharedData(c *gin.Context) {
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
	header, response, problemDetails := subscribeToSharedDataProcedure(&sharedDataSubsReq)

	var rsp *httpwrapper.Response
	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		// step 5: response
		for key, val := range header { // header response is optional
			c.Header(key, val[0])
		}
		c.JSON(http.StatusOK, response)
		return
	} else if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	} else {
		problemDetails := models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Cause:  "SYSTEM_FAILURE",
			Detail: err.Error(),
		}
		c.JSON(http.StatusInternalServerError, problemDetails)
		return
	}

}
