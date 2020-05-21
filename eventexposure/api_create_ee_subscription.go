/*
 * Nudm_EE
 *
 * Nudm Event Exposure Service
 *
 * API version: 1.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package eventexposure

import (
	"free5gc/lib/http_wrapper"
	"free5gc/lib/openapi/models"
	"free5gc/src/udm/handler"
	"free5gc/src/udm/handler/udm_message"
	"free5gc/src/udm/logger"
	"github.com/gin-gonic/gin"
	"net/http"
)

// CreateEeSubscription - Subscribe
func CreateEeSubscription(c *gin.Context) {

	var eeSubscriptionReq models.EeSubscription

	err := c.ShouldBindJSON(&eeSubscriptionReq)
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.EeLog.Errorln(problemDetail)
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	req := http_wrapper.NewRequest(c.Request, eeSubscriptionReq)
	req.Params["ueIdentity"] = c.Params.ByName("ueIdentity")
	req.Params["subscriptionID"] = c.Params.ByName("subscriptionId")

	handlerMsg := udm_message.NewHandlerMessage(udm_message.EventCreateEeSubscription, req)
	handler.SendMessage(handlerMsg)

	rsp := <-handlerMsg.ResponseChan

	HTTPResponse := rsp.HTTPResponse

	c.JSON(HTTPResponse.Status, HTTPResponse.Body)

}
