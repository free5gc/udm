package processor

import (
	"net/http"
	"net/url"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
	"github.com/gin-gonic/gin"
)

// GetNssai - retrieve a UE's subscribed NSSAI
func (p *Processor) HandleGetNssai(c *gin.Context) {
	var query url.Values
	query.Set("plmn-id", c.Query("plmn-id"))
	query.Set("supported-features", c.Query("supported-features"))

	// step 1: log
	logger.SdmLog.Infof("Handle GetNssai")

	// step 2: retrieve request
	supi := c.Params.ByName("supi")
	plmnIDStruct, problemDetails := p.getPlmnIDStruct(query)
	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	plmnID := plmnIDStruct.Mcc + plmnIDStruct.Mnc
	supportedFeatures := query.Get("supported-features")

	// step 3: handle the message
	response, problemDetails := p.consumer.GetNssaiProcedure(supi, plmnID, supportedFeatures)

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
