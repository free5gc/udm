/*
 * Nudm_SDM
 *
 * Nudm Subscriber Data Management Service
 *
 * API version: 2.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package subscriberdatamanagement_test

import (
	"context"
	"fmt"
	"free5gc/lib/http2_util"
	Nudm_SDM_Client "free5gc/lib/openapi/Nudm_SubscriberDataManagement"
	"free5gc/lib/openapi/models"
	"free5gc/lib/path_util"
	udm_context "free5gc/src/udm/context"
	"free5gc/src/udm/logger"
	Nudm_SDM_Server "free5gc/src/udm/subscriberdatamanagement"
	"free5gc/src/udm/udm_handler"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Subscribe - subscribe to notifications
func TestSubscribe(t *testing.T) {

	go func() { // udm server

		router := gin.Default()
		Nudm_SDM_Server.AddService(router)

		udmLogPath := path_util.Gofree5gcPath("free5gc/udmsslkey.log")
		udmPemPath := path_util.Gofree5gcPath("free5gc/support/TLS/udm.pem")
		udmKeyPath := path_util.Gofree5gcPath("free5gc/support/TLS/udm.key")

		server, err := http2_util.NewServer(":29503", udmLogPath, router)
		if err == nil && server != nil {
			logger.InitLog.Infoln(server.ListenAndServeTLS(udmPemPath, udmKeyPath))
			assert.True(t, err == nil)
		}
	}()
	udm_context.TestInit()
	go udm_handler.Handle()

	go func() { // fake udr server
		router := gin.Default()

		router.POST("/nudr-dr/v1/subscription-data/:ueId/context-data/sdm-subscriptions", func(c *gin.Context) {
			supi := c.Param("supi")
			fmt.Println("==========Subscribe - subscribe to notifications==========")
			fmt.Println("supi: ", supi)

			var sdmSubscription models.SdmSubscription
			if err := c.ShouldBindJSON(&sdmSubscription); err != nil {
				fmt.Println("fake udr server error")
				c.JSON(http.StatusInternalServerError, gin.H{})
				return
			}
			var sdmSubscriptionResponse models.SdmSubscription
			sdmSubscriptionResponse.NfInstanceId = sdmSubscription.NfInstanceId
			fmt.Println("sdmSubscription : ", sdmSubscriptionResponse.NfInstanceId)
			c.JSON(http.StatusCreated, sdmSubscription)
		})

		udrLogPath := path_util.Gofree5gcPath("free5gc/udrsslkey.log")
		udrPemPath := path_util.Gofree5gcPath("free5gc/support/TLS/udr.pem")
		udrKeyPath := path_util.Gofree5gcPath("free5gc/support/TLS/udr.key")

		server, err := http2_util.NewServer(":29504", udrLogPath, router)
		if err == nil && server != nil {
			logger.InitLog.Infoln(server.ListenAndServeTLS(udrPemPath, udrKeyPath))
			assert.True(t, err == nil)
		}
	}()

	udm_context.Init()
	cfg := Nudm_SDM_Client.NewConfiguration()
	cfg.SetBasePath("https://localhost:29503")
	clientAPI := Nudm_SDM_Client.NewAPIClient(cfg)

	supi := "SDM1234"
	var sdmSubscription models.SdmSubscription
	sdmSubscription.NfInstanceId = "Test_NfinstanceId"
	sdmSubscription.Dnn = "3"
	_, resp, err := clientAPI.SubscriptionCreationApi.Subscribe(context.Background(), supi, sdmSubscription)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("resp: ", resp)
	}
}
