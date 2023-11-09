package util

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi/oauth"
	"github.com/free5gc/udm/pkg/factory"
)

// This function would check the OAuth2 token, and the requestNF is in ServiceAllowNfType
func AuthorizationCheck(c *gin.Context, serviceName string) error {
	if factory.UdmConfig.GetOAuth() {
		oauth_err := oauth.VerifyOAuth(c.Request.Header.Get("Authorization"), serviceName,
			factory.UdmConfig.GetNrfCertPemPath())
		if oauth_err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": oauth_err.Error()})
			return oauth_err
		}
	}
	return nil
}
