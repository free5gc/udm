package processor

import (
	"net/http"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
	"github.com/gin-gonic/gin"
)

// GetTraceData - retrieve a UE's Trace Configuration Data
func (p *Processor) HandleGetTraceData(c *gin.Context) {
	// step 1: log
	logger.SdmLog.Infof("Handle GetTraceData")

	// step 2: retrieve request
	supi := c.Params.ByName("supi")
	plmnID := c.Query("plmn-id")

	// step 3: handle the message
	response, problemDetails := p.consumer.GetTraceDataProcedure(supi, plmnID)

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
