package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getUEIDRoutes() []Route {
	return []Route{
		{
			Name:        "Index",
			Method:      http.MethodGet,
			Pattern:     "/",
			HandlerFunc: s.HandleIndex,
		},

		{
			Name:        "Deconceal",
			Method:      http.MethodPost,
			Pattern:     "/deconceal",
			HandlerFunc: s.HandleDeconceal,
		},
	}
}

func (s *Server) HandleDeconceal(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}
