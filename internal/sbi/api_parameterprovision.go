package sbi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
)

func (s *Server) getParameterProvisionRoutes() []Route {
	return []Route{
		{
			"Index",
			"GET",
			"/",
			s.HandleIndex,
		},

		{
			"Update",
			strings.ToUpper("Patch"),
			"/:ueId/pp-data",
			s.HandleUpdate,
		},

		{
			"Create5GMBSGroup",
			strings.ToUpper("Put"),
			"/mbs-group-membership/:extGroupId",
			s.HandleCreate5GMBSGroup,
		},

		{
			"Create5GVNGroup",
			strings.ToUpper("Put"),
			"/5g-vn-groups/:extGroupId",
			s.HandleCreate5GVNGroup,
		},

		{
			"CreatePPDataEntry",
			strings.ToUpper("Put"),
			"/:ueId/pp-data-store/:afInstanceId",
			s.HandleCreatePPDataEntry,
		},

		{
			"Delete5GMBSGroup",
			strings.ToUpper("Delete"),
			"/mbs-group-membership/:extGroupId",
			s.HandleDelete5GMBSGroup,
		},

		{
			"Delete5GVNGroup",
			strings.ToUpper("Delete"),
			"/5g-vn-groups/:extGroupId",
			s.HandleDelete5GVNGroup,
		},

		{
			"DeletePPDataEntry",
			strings.ToUpper("Delete"),
			"/:ueId/pp-data-store/:afInstanceId",
			s.HandleDeletePPDataEntry,
		},

		{
			"Get5GMBSGroup",
			strings.ToUpper("Get"),
			"/mbs-group-membership/:extGroupId",
			s.HandleGet5GMBSGroup,
		},

		{
			"Get5GVNGroup",
			strings.ToUpper("Get"),
			"/5g-vn-groups/:extGroupId",
			s.HandleGet5GVNGroup,
		},

		{
			"GetPPDataEntry",
			strings.ToUpper("Get"),
			"/:ueId/pp-data-store/:afInstanceId",
			s.HandleGetPPDataEntry,
		},

		{
			"Modify5GMBSGroup",
			strings.ToUpper("Patch"),
			"/mbs-group-membership/:extGroupId",
			s.HandleModify5GMBSGroup,
		},

		{
			"Modify5GVNGroup",
			strings.ToUpper("Patch"),
			"/5g-vn-groups/:extGroupId",
			s.HandleModify5GVNGroup,
		},
	}
}

func (s *Server) HandleUpdate(c *gin.Context) {
	var ppDataReq models.PpData

	// step 1: retrieve http request body
	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.PpLog.Errorf("Get Request Body error: %+v", err)
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	// step 2: convert requestBody to openapi models
	err = openapi.Deserialize(&ppDataReq, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.PpLog.Errorln(problemDetail)
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	gpsi := c.Params.ByName("ueId")
	if gpsi == "" {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "NO_GPSI",
		}
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	logger.PpLog.Infoln("Handle UpdateRequest")

	// step 3: handle the message
	s.Processor().UpdateProcedure(c, ppDataReq, gpsi)
}

func (s *Server) HandleCreate5GMBSGroup(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleCreate5GVNGroup(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleCreatePPDataEntry(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleDelete5GMBSGroup(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleDelete5GVNGroup(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleDeletePPDataEntry(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGet5GMBSGroup(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGet5GVNGroup(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleGetPPDataEntry(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleModify5GMBSGroup(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}

func (s *Server) HandleModify5GVNGroup(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{})
}
