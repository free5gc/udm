package processor

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/free5gc/openapi/models"
)

const serviceNameNAMFCallback models.Nrf_NFMgmt_ServiceName = "namf-callback"

type dataChangeCallbackTarget struct {
	ServiceName  models.Nrf_NFMgmt_ServiceName
	NfType       models.Nrf_NFMgmt_NFType
	NfInstanceID string
}

func resolveDataChangeCallbackTarget(
	subscription *models.Udr_DR_SubscriptionDataSubscriptions,
) (dataChangeCallbackTarget, error) {
	if subscription == nil || subscription.SdmSubscription == nil {
		return dataChangeCallbackTarget{}, fmt.Errorf("missing SDM subscription callback identity")
	}

	callbackURI, err := url.Parse(subscription.OriginalCallbackReference)
	if err != nil || !callbackURI.IsAbs() || callbackURI.Host == "" ||
		(callbackURI.Scheme != "http" && callbackURI.Scheme != "https") {
		return dataChangeCallbackTarget{}, fmt.Errorf("invalid data-change callback URI")
	}
	path := strings.Split(strings.TrimPrefix(callbackURI.EscapedPath(), "/"), "/")
	if len(path) == 0 || path[0] != string(serviceNameNAMFCallback) {
		return dataChangeCallbackTarget{}, fmt.Errorf("unsupported data-change callback service")
	}

	instanceID := strings.TrimSpace(subscription.SdmSubscription.NfInstanceId)
	targetID, err := uuid.Parse(instanceID)
	if err != nil || targetID.Version() != 4 {
		return dataChangeCallbackTarget{}, fmt.Errorf("invalid callback NF instance ID")
	}
	return dataChangeCallbackTarget{
		ServiceName:  serviceNameNAMFCallback,
		NfType:       models.Nrf_NFMgmt_NFType_AMF,
		NfInstanceID: instanceID,
	}, nil
}
