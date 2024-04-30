package processor

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DeregistrationSmsfNon3gppAccess - delete SMSF registration for non 3GPP access
func HTTPDeregistrationSmsfNon3gppAccess(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}
