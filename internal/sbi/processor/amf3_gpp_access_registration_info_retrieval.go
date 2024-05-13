package processor

import (
	"net/http"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
	"github.com/gin-gonic/gin"
)

// GetAmf3gppAccess - retrieve the AMF registration for 3GPP access information
func (p *Processor) HandleGetAmf3gppAccess(c *gin.Context) {

	// step 1: log
	logger.UecmLog.Infof("Handle HandleGetAmf3gppAccessRequest")

	// step 2: retrieve request
	ueID := c.Param("ueId")
	supportedFeatures := c.Query("supported-features")

	// step 3: handle the message
	response, problemDetails := p.consumer.GetAmf3gppAccessProcedure(ueID, supportedFeatures)

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
