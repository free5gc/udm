package processor

import (
	"net/http"

	"github.com/free5gc/udm/internal/logger"
	"github.com/gin-gonic/gin"
)

// UnsubscribeForSharedData - unsubscribe from notifications for shared data
func (p *Processor) HandleUnsubscribeForSharedData(c *gin.Context) {
	logger.SdmLog.Infof("Handle UnsubscribeForSharedData")

	// step 2: retrieve request
	subscriptionID := c.Params.ByName("subscriptionId")
	// step 3: handle the message
	problemDetails := unsubscribeForSharedDataProcedure(subscriptionID)

	// step 4: process the return value from step 3
	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	} else {
		c.Status(http.StatusNoContent)
		return
	}

}
