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
	"free5gc/lib/http_wrapper"
	"free5gc/lib/openapi/models"
	"free5gc/src/udm/logger"
	"free5gc/src/udm/handler"
	"free5gc/src/udm/handler/udm_message"
	"github.com/gin-gonic/gin"
	"net/http"
)

// RegistrationSmfRegistrations - register as SMF
func RegistrationSmfRegistrations(c *gin.Context) {
	var smfRegistration models.SmfRegistration
	if err := c.ShouldBindJSON(&smfRegistration); err != nil {
		logger.UecmLog.Errorln(err)
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	req := http_wrapper.NewRequest(c.Request, smfRegistration)
	req.Params["ueId"] = c.Params.ByName("ueId")
	req.Params["pduSessionId"] = c.Params.ByName("pduSessionId")

	handlerMsg := udm_message.NewHandlerMessage(udm_message.EventRegistrationSmfRegistrations, req)
	handler.SendMessage(handlerMsg)
	rsp := <-handlerMsg.ResponseChan

	HTTPResponse := rsp.HTTPResponse
	c.Header("Location", HTTPResponse.Header.Get("Location"))
	c.JSON(HTTPResponse.Status, HTTPResponse.Body)
	return
}
