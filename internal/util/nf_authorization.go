package util

import (
	"net/http"

	"github.com/free5gc/openapi"
	"github.com/free5gc/udm/pkg/factory"
	"github.com/gin-gonic/gin"
)

// This function would check the OAuth2 token, and the requestNF is in ServiceAllowNfType
func AuthorizationCheck(c *gin.Context, serviceName string) error {
	if factory.UdmConfig.GetOAuth() {
		oauth_err := openapi.VerifyOAuth(c.Request.Header.Get("Authorization"), serviceName,
			factory.UdmConfig.GetNrfCertPemPath())
		if oauth_err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": oauth_err.Error()})
			return oauth_err
		}
	}
	allowNf_err := factory.UdmConfig.VerifyServiceAllowType(c.Request.Header.Get("requestNF"), serviceName)
	if allowNf_err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": allowNf_err.Error()})
		return allowNf_err
	}
	return nil
}
