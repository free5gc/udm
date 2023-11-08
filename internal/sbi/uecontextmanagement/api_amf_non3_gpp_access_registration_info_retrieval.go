/*
 * Nudm_UECM
 *
 * Nudm Context Management Service
 *
 * API version: 1.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package uecontextmanagement

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/udm/internal/sbi/producer"
	"github.com/free5gc/util/httpwrapper"
)

// GetAmfNon3gppAccess - retrieve the AMF registration for non-3GPP access information
func HTTPGetAmfNon3gppAccess(c *gin.Context) {
	auth_err := authorizationCheck(c)
	if auth_err != nil {
		return
	}

	req := httpwrapper.NewRequest(c.Request, nil)
	req.Params["ueId"] = c.Param("ueId")
	req.Query.Add("supported-features", c.Query("supported-features"))

	rsp := producer.HandleGetAmfNon3gppAccessRequest(req)

	responseBody, err := openapi.Serialize(rsp.Body, "application/json")
	if err != nil {
		logger.UecmLog.Errorln(err)
		problemDetails := models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Cause:  "SYSTEM_FAILURE",
			Detail: err.Error(),
		}
		c.JSON(http.StatusInternalServerError, problemDetails)
	} else {
		c.Data(rsp.Status, "application/json", responseBody)
	}
}
