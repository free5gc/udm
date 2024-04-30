package processor

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetSmsf3gppAccess - retrieve the SMSF registration for 3GPP access information
func HTTPGetSmsf3gppAccess(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}
