package processor

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
)

// GetIdTranslationResult - retrieve a UE's SUPI
func (p *Processor) HandleGetIdTranslationResult(c *gin.Context) {
	// req.Query.Set("SupportedFeatures", c.Query("supported-features"))

	// step 1: log
	logger.SdmLog.Infof("Handle GetIdTranslationResultRequest")

	// step 2: retrieve request
	gpsi := c.Params.ByName("gpsi")

	// step 3: handle the message
	response, problemDetails := p.consumer.GetIdTranslationResultProcedure(gpsi)

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
