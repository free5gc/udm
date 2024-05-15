package processor

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/udm/internal/logger"
)

// DeregistrationSmfRegistrations - delete an SMF registration
func (p *Processor) HandleDeregistrationSmfRegistrations(c *gin.Context) {
	// step 1: log
	logger.UecmLog.Infof("Handle DeregistrationSmfRegistrations")

	// step 2: retrieve request
	ueID := c.Params.ByName("ueId")
	pduSessionID := c.Params.ByName("pduSessionId")

	// step 3: handle the message
	problemDetails := p.consumer.DeregistrationSmfRegistrationsProcedure(ueID, pduSessionID)

	// step 4: process the return value from step 3
	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	} else {
		c.Status(http.StatusNoContent)
		return
	}
}
