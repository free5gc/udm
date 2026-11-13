package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getReportSMDeliveryStatusRoutes() []Route {
	return []Route{
		{
			Name:        "Index",
			Method:      http.MethodGet,
			Pattern:     "/",
			HandlerFunc: s.HandleIndex,
		},

		{
			Name:        "ReportSMDeliveryStatus",
			Method:      http.MethodPost,
			Pattern:     "/:ueIdentity/sm-delivery-status",
			HandlerFunc: s.HandleReportSMDeliveryStatus,
		},
	}
}

func (s *Server) HandleReportSMDeliveryStatus(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}
