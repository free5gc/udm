package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/util/metrics/sbi"
)

func (s *Server) getParameterProvisionRoutes() []Route {
	return []Route{
		{
			Name:        "Index",
			Method:      http.MethodGet,
			Pattern:     "/",
			HandlerFunc: s.HandleIndex,
		},

		{
			Name:        "Update",
			Method:      http.MethodPatch,
			Pattern:     "/:ueId/pp-data",
			HandlerFunc: s.HandleUpdate,
		},

		{
			Name:        "Create5GMBSGroup",
			Method:      http.MethodPut,
			Pattern:     "/mbs-group-membership/:extGroupId",
			HandlerFunc: s.HandleCreate5GMBSGroup,
		},

		{
			Name:        "Create5GVNGroup",
			Method:      http.MethodPut,
			Pattern:     "/5g-vn-groups/:extGroupId",
			HandlerFunc: s.HandleCreate5GVNGroup,
		},

		{
			Name:        "CreatePPDataEntry",
			Method:      http.MethodPut,
			Pattern:     "/:ueId/pp-data-store/:afInstanceId",
			HandlerFunc: s.HandleCreatePPDataEntry,
		},

		{
			Name:        "Delete5GMBSGroup",
			Method:      http.MethodDelete,
			Pattern:     "/mbs-group-membership/:extGroupId",
			HandlerFunc: s.HandleDelete5GMBSGroup,
		},

		{
			Name:        "Delete5GVNGroup",
			Method:      http.MethodDelete,
			Pattern:     "/5g-vn-groups/:extGroupId",
			HandlerFunc: s.HandleDelete5GVNGroup,
		},

		{
			Name:        "DeletePPDataEntry",
			Method:      http.MethodDelete,
			Pattern:     "/:ueId/pp-data-store/:afInstanceId",
			HandlerFunc: s.HandleDeletePPDataEntry,
		},

		{
			Name:        "Get5GMBSGroup",
			Method:      http.MethodGet,
			Pattern:     "/mbs-group-membership/:extGroupId",
			HandlerFunc: s.HandleGet5GMBSGroup,
		},

		{
			Name:        "Get5GVNGroup",
			Method:      http.MethodGet,
			Pattern:     "/5g-vn-groups/:extGroupId",
			HandlerFunc: s.HandleGet5GVNGroup,
		},

		{
			Name:        "GetPPDataEntry",
			Method:      http.MethodGet,
			Pattern:     "/:ueId/pp-data-store/:afInstanceId",
			HandlerFunc: s.HandleGetPPDataEntry,
		},

		{
			Name:        "Modify5GMBSGroup",
			Method:      http.MethodPatch,
			Pattern:     "/mbs-group-membership/:extGroupId",
			HandlerFunc: s.HandleModify5GMBSGroup,
		},

		{
			Name:        "Modify5GVNGroup",
			Method:      http.MethodPatch,
			Pattern:     "/5g-vn-groups/:extGroupId",
			HandlerFunc: s.HandleModify5GVNGroup,
		},
	}
}

func (s *Server) HandleUpdate(c *gin.Context) {
	var ppDataReq models.Udm_PP_PpData

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
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetail.Cause)
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
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(int(rsp.Status)))
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	gpsi := c.Params.ByName("ueId")
	if gpsi == "" {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "NO_GPSI",
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
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
