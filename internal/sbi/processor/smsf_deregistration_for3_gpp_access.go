package processor

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DeregistrationSmsf3gppAccess - delete the SMSF registration for 3GPP access
func (p *Processor) HandleDeregistrationSmsf3gppAccess(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}
