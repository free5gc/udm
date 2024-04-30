package processor

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegistrationSmsfNon3gppAccess - register as SMSF for non-3GPP access
func HTTPRegistrationSmsfNon3gppAccess(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}
