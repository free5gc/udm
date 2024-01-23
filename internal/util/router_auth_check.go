package util

import (
	"net/http"

	"github.com/free5gc/udm/internal/logger"
	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/gin-gonic/gin"
)

type NFContextGetter func() *udm_context.UDMContext

type RouterAuthorizationCheck struct {
	serviceName string
}

func NewRouterAuthorizationCheck(serviceName string) *RouterAuthorizationCheck {
	return &RouterAuthorizationCheck{
		serviceName: serviceName,
	}
}

func (rac *RouterAuthorizationCheck) Check(c *gin.Context, udmContext udm_context.NFContext) {
	token := c.Request.Header.Get("Authorization")
	err := udmContext.AuthorizationCheck(token, rac.serviceName)
	if err != nil {
		logger.UtilLog.Debugf("RouterAuthorizationCheck::Check Unauthorized: %s", err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		c.Abort()
		return
	}

	logger.UtilLog.Debugf("RouterAuthorizationCheck::Check Authorized")
}
