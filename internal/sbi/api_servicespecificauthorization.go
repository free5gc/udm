package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getServiceSpecificAuthorizationRoutes() []Route {
	return []Route{
		{
			Name:        "Index",
			Method:      http.MethodGet,
			Pattern:     "/",
			HandlerFunc: s.HandleIndex,
		},

		{
			Name:        "ServiceSpecificAuthorization",
			Method:      http.MethodPost,
			Pattern:     "/:ueIdentity/:serviceType/authorize",
			HandlerFunc: s.HandleServiceSpecificAuthorization,
		},

		{
			Name:        "ServiceSpecificAuthorizationRemoval",
			Method:      http.MethodPost,
			Pattern:     "/:ueIdentity/:serviceType/remove",
			HandlerFunc: s.HandleServiceSpecificAuthorizationRemoval,
		},
	}
}

func (s *Server) HandleServiceSpecificAuthorization(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleServiceSpecificAuthorizationRemoval(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}
