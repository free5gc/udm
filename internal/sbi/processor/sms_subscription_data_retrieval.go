package processor

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetSmsData - retrieve a UE's SMS Subscription Data
func (p *Processor) HandleGetSmsData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}
