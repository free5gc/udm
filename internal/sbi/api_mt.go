package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getMTRoutes() []Route {
	return []Route{
		{
			"Index",
			"GET",
			"/",
			s.HandleIndex,
		},

		{
			"ProvideLocationInfo",
			"Post",
			"/:supi/loc-info/provide-loc-info",
			s.HandleProvideLocationInfo,
		},

		{
			"QueryUeInfo",
			"GET",
			"/:supi",
			s.HandleQueryUeInfo,
		},
	}
}

func (s *Server) HandleProvideLocationInfo(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleQueryUeInfo(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}
