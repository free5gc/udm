package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getNIDDAuthenticationRoutes() []Route {
	return []Route{
		{
			Name:        "Index",
			Method:      http.MethodGet,
			Pattern:     "/",
			HandlerFunc: s.HandleIndex,
		},

		{
			Name:        "AuthorizeNiddData",
			Method:      http.MethodPost,
			Pattern:     "/:ueIdentity/authorize",
			HandlerFunc: s.HandleAuthorizeNiddData,
		},
	}
}

func (s *Server) HandleAuthorizeNiddData(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}
