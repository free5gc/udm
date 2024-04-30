package processor

import (
	"net/http"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
	"github.com/gin-gonic/gin"
)

// GetUeContextInSmfData - retrieve a UE's UE Context In SMF Data
func HTTPGetUeContextInSmfData(c *gin.Context) {

	// step 1: log
	logger.SdmLog.Infof("Handle GetUeContextInSmfData")

	// step 2: retrieve request
	supi := c.Params.ByName("supi")
	supportedFeatures := c.Query("supported-features")

	// step 3: handle the message
	response, problemDetails := getUeContextInSmfDataProcedure(supi, supportedFeatures)

	// step 4: process the return value from step 3
	if response != nil {
		// status code is based on SPEC, and option headers
		c.JSON(http.StatusOK, response)
		return
	} else if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusForbidden,
			Cause:  "UNSPECIFIED",
		}
		c.JSON(http.StatusForbidden, problemDetails)
		return
	}

}
