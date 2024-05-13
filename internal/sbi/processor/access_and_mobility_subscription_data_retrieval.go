package processor

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
)

// GetAmData - retrieve a UE's Access and Mobility Subscription Data
func (p *Processor) HandleGetAmData(c *gin.Context) {
	var query url.Values
	query.Set("plmn-id", c.Query("plmn-id"))
	query.Set("supported-features", c.Query("plmn-id"))

	// step 1: log
	logger.SdmLog.Infof("Handle GetAmData")

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
	response, problemDetails := p.consumer.GetAmDataProcedure(supi, plmnID, supportedFeatures)

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

func (p *Processor) getPlmnIDStruct(queryParameters url.Values) (plmnIDStruct *models.PlmnId, problemDetails *models.ProblemDetails) {
	if queryParameters["plmn-id"] != nil {
		plmnIDJson := queryParameters["plmn-id"][0]
		plmnIDStruct := &models.PlmnId{}
		err := json.Unmarshal([]byte(plmnIDJson), plmnIDStruct)
		if err != nil {
			logger.SdmLog.Warnln("Unmarshal Error in targetPlmnListtruct: ", err)
		}
		return plmnIDStruct, nil
	} else {
		problemDetails := &models.ProblemDetails{
			Title:  "Invalid Parameter",
			Status: http.StatusBadRequest,
			Cause:  "No get plmn-id",
		}
		return nil, problemDetails
	}
}
