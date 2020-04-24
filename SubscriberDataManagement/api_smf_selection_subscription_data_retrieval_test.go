/*
 * Nudm_SDM
 *
 * Nudm Subscriber Data Management Service
 *
 * API version: 2.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package SubscriberDataManagement_test

import (
	"context"
	"fmt"
	Nudm_SDM_Client "free5gc/lib/Nudm_SubscriberDataManagement"
	"free5gc/lib/http2_util"
	"free5gc/lib/openapi/models"
	"free5gc/lib/path_util"
	Nudm_SDM_Server "free5gc/src/udm/SubscriberDataManagement"
	"free5gc/src/udm/logger"
	"free5gc/src/udm/udm_context"
	"free5gc/src/udm/udm_handler"

	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// GetSmfSelectData - retrieve a UE's SMF Selection Subscription Data
func TestGetSmfSelectData(t *testing.T) {
	udm_context.TestInit()
	go udm_handler.Handle()
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

	go func() { // fake udr server
		router := gin.Default()

		router.GET("/nudr-dr/v1/subscription-data/:ueId/provisioned-data/smf-selection-subscription-data", func(c *gin.Context) { // :servingPlmnId/
			supi := c.Param("supi")
			fmt.Println("==========SMF selection subscription data==========")
			fmt.Println("supi: ", supi)
			var testsmfSelectionSubscriptionData models.SmfSelectionSubscriptionData
			testsmfSelectionSubscriptionData.SharedSnssaiInfosId = "Test001"
			testsmfSelectionSubscriptionData.SupportedFeatures = "test002"
			c.JSON(http.StatusOK, testsmfSelectionSubscriptionData)
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
	smfSelectionSubscriptionData, resp, err := clientAPI.SMFSelectionSubscriptionDataRetrievalApi.GetSmfSelectData(context.Background(), supi, nil)

	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("resp: ", resp)
		fmt.Println("smfSelectionSubscriptionData: ", smfSelectionSubscriptionData)
	}
}
