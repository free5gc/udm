package udm_consumer

import (
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	Nudr "gofree5gc/lib/Nudr_DataRepository"
	"gofree5gc/lib/openapi/common"
	"gofree5gc/lib/openapi/models"
	"gofree5gc/src/udm/factory"
	"gofree5gc/src/udm/logger"
	"gofree5gc/src/udm/udm_context"
	"gofree5gc/src/udm/udm_handler/udm_message"
	"net/http"
	"strings"
)

func UDMClientToUDR(id string, nonUe bool) *Nudr.APIClient {
	var addr string
	if !nonUe {
		addr = getUdrUri(id)
	}
	if addr == "" {
		// dafault
		if !nonUe {
			logger.Handlelog.Warnf("Use default UDR Uri bacause ID[%s] does not match any UDR", id)
		}
		config := factory.UdmConfig
		udrclient := config.Configuration.Udrclient
		addr = fmt.Sprintf("%s://%s:%d", udrclient.Scheme, udrclient.Ipv4Addr, udrclient.Port)
	}
	cfg := Nudr.NewConfiguration()
	cfg.SetBasePath(addr)
	clientAPI := Nudr.NewAPIClient(cfg)
	return clientAPI
}

func getUdrUri(id string) string {
	// supi
	if strings.Contains(id, "imsi") || strings.Contains(id, "nai") {
		udmUe := udm_context.UDM_Self().UdmUePool[id]
		if udmUe != nil {
			if udmUe.UdrUri == "" {
				udmUe.UdrUri = SendNFIntancesUDR(id, NFDiscoveryToUDRParamSupi)
			}
			return udmUe.UdrUri
		} else {
			udmUe = udm_context.CreateUdmUe(id)
			udmUe.UdrUri = SendNFIntancesUDR(id, NFDiscoveryToUDRParamSupi)
			return udmUe.UdrUri
		}
	} else if strings.Contains(id, "pei") {
		for _, udmUe := range udm_context.UDM_Self().UdmUePool {
			if udmUe.Amf3GppAccessRegistration != nil && udmUe.Amf3GppAccessRegistration.Pei == id {
				if udmUe.UdrUri != "" {
					udmUe.UdrUri = SendNFIntancesUDR(udmUe.Supi, NFDiscoveryToUDRParamSupi)
				}
				return udmUe.UdrUri
			} else if udmUe.AmfNon3GppAccessRegistration != nil && udmUe.AmfNon3GppAccessRegistration.Pei == id {
				if udmUe.UdrUri != "" {
					udmUe.UdrUri = SendNFIntancesUDR(udmUe.Supi, NFDiscoveryToUDRParamSupi)
				}
				return udmUe.UdrUri
			}
		}
	} else if strings.Contains(id, "extgroupid") {
		// extra group id
		return SendNFIntancesUDR(id, NFDiscoveryToUDRParamExtGroupId)
	} else if strings.Contains(id, "msisdn") || strings.Contains(id, "extid") {
		// gpsi
		return SendNFIntancesUDR(id, NFDiscoveryToUDRParamGpsi)
	}
	return ""
}

func SubscriptionToNotify(ueID string, subscriptionDataSubscriptions models.SubscriptionDataSubscriptions) {

	clientAPI := UDMClientToUDR(ueID, false)

	SubData, res, err := clientAPI.SubsToNofifyCollectionApi.PostSubscriptionDataSubscriptions(context.Background(), subscriptionDataSubscriptions)
	if err != nil {
		var problemDetails models.ProblemDetails
		if res == nil {
			fmt.Println(err.Error())
		} else if err.Error() != res.Status {
			fmt.Println(err.Error())
		} else {
			problemDetails.Cause = err.(common.GenericOpenAPIError).Model().(models.ProblemDetails).Cause
			udm_message.SendHttpResponseMessage(nil, nil, res.StatusCode, problemDetails)
		}
		return
	}

	if res.StatusCode == http.StatusCreated {
		subsUri := res.Header.Get("Location")
		spew.Printf("[subsUri_Header_Location] %s\n", subsUri)
		subsId := subsUri[strings.LastIndex(subsUri, "/")+1:]
		// spew.Printf("[subsId] %s\n", subsId)
		udmUe := udm_context.CreateUdmUe(ueID)
		udmUe.UdmSubsToNotify[subsId] = &SubData
	} else {
		fmt.Println(res.Status)
	}
}
