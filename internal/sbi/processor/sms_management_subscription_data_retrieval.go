package processor

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetSmsMngData - retrieve a UE's SMS Management Subscription Data
func (p *Processor) HandleGetSmsMngData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}
