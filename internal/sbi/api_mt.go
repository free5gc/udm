package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getMTRoutes() []Route {
	return []Route{
		{
			Name:        "Index",
			Method:      http.MethodGet,
			Pattern:     "/",
			HandlerFunc: s.HandleIndex,
		},

		{
			Name:        "ProvideLocationInfo",
			Method:      http.MethodPost,
			Pattern:     "/:supi/loc-info/provide-loc-info",
			HandlerFunc: s.HandleProvideLocationInfo,
		},

		{
			Name:        "QueryUeInfo",
			Method:      http.MethodGet,
			Pattern:     "/:supi",
			HandlerFunc: s.HandleQueryUeInfo,
		},
	}
}

func (s *Server) HandleProvideLocationInfo(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleQueryUeInfo(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}
