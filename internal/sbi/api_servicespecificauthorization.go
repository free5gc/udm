package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getServiceSpecificAuthorizationRoutes() []Route {
	return []Route{
		{
			"Index",
			"GET",
			"/",
			s.HandleIndex,
		},

		{
			"ServiceSpecificAuthorization",
			"Post",
			"/:ueIdentity/:serviceType/authorize",
			s.HandleServiceSpecificAuthorization,
		},

		{
			"ServiceSpecificAuthorizationRemoval",
			"Post",
			"/:ueIdentity/:serviceType/remove",
			s.HandleServiceSpecificAuthorizationRemoval,
		},
	}
}

func (s *Server) HandleServiceSpecificAuthorization(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleServiceSpecificAuthorizationRemoval(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}
