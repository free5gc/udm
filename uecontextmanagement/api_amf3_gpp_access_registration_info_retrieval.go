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
	// "fmt"
	"free5gc/lib/http_wrapper"
	"free5gc/src/udm/handler"
	"free5gc/src/udm/handler/udm_message"
	"github.com/gin-gonic/gin"
)

// GetAmf3gppAccess - retrieve the AMF registration for 3GPP access information
func GetAmf3gppAccess(c *gin.Context) {
	req := http_wrapper.NewRequest(c.Request, nil)
	req.Params["ueId"] = c.Param("ueId")
	req.Query.Add("supported-features", c.Query("supported-features"))

	handlerMsg := udm_message.NewHandlerMessage(udm_message.EventGetAmf3gppAccess, req)
	handler.SendMessage(handlerMsg)
	rsp := <-handlerMsg.ResponseChan

	HTTPResponse := rsp.HTTPResponse
	c.JSON(HTTPResponse.Status, HTTPResponse.Body)
	return
}
