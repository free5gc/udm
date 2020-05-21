package handler

import (
	"free5gc/lib/openapi/models"
	"free5gc/src/udm/logger"
	"free5gc/src/udm/producer"
	udm_message "free5gc/src/udm/handler/message"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	MaxChannel int = 100000
)

var udmChannel chan udm_message.HandlerMessage
var HandlerLog *logrus.Entry

func init() {

	HandlerLog = logger.Handlelog
	udmChannel = make(chan udm_message.HandlerMessage, MaxChannel)
}

func SendMessage(msg udm_message.HandlerMessage) {
	udmChannel <- msg
}

func Handle() {
	for {
		select {
		case msg, ok := <-udmChannel:
			if ok {
				switch msg.Event {
				case udm_message.EventGenerateAuthData:
					supiOrSuci := msg.HTTPRequest.Params["supiOrSuci"]
					producer.HandleGenerateAuthData(msg.ResponseChan, supiOrSuci,
						msg.HTTPRequest.Body.(models.AuthenticationInfoRequest))
				case udm_message.EventConfirmAuth:
					supi := msg.HTTPRequest.Params["supi"]
					producer.HandleConfirmAuthData(msg.ResponseChan, supi,
						msg.HTTPRequest.Body.(models.AuthEvent))
				case udm_message.EventGetAmData:
					supi := msg.HTTPRequest.Params["supi"]
					plmnID := msg.HTTPRequest.Query.Get("plmn-id")
					supportedFeatures := msg.HTTPRequest.Query.Get("supported-features")
					producer.HandleGetAmData(msg.ResponseChan, supi, plmnID, supportedFeatures)
				case udm_message.EventGetIdTranslationResult:
					gpsi := msg.HTTPRequest.Params["gpsi"]
					supportedFeatures := msg.HTTPRequest.Query.Get("supported-features")
					_ = supportedFeatures
					producer.HandleGetIdTranslationResult(msg.ResponseChan, gpsi)
				case udm_message.EventGetNssai:
					supi := msg.HTTPRequest.Params["supi"]
					plmnID := msg.HTTPRequest.Query.Get("plmn-id")
					supportedFeatures := msg.HTTPRequest.Query.Get("supported-features")
					producer.HandleGetNssai(msg.ResponseChan, supi, plmnID, supportedFeatures)

				case udm_message.EventGetSharedData:
					sharedDataIds := msg.HTTPRequest.Query["sharedDataIds"]
					supportedFeatures := msg.HTTPRequest.Query.Get("supported-features")
					producer.HandleGetSharedData(msg.ResponseChan, sharedDataIds, supportedFeatures)

				case udm_message.EventGetSmData:
					supi := msg.HTTPRequest.Params["supi"]
					plmnID := msg.HTTPRequest.Query.Get("plmn-id")
					Dnn := msg.HTTPRequest.Query.Get("dnn")
					Snssai := msg.HTTPRequest.Query.Get("single-nssai")
					supportedFeatures := msg.HTTPRequest.Query.Get("supported-features")
					producer.HandleGetSmData(msg.ResponseChan, supi, plmnID, Dnn, Snssai, supportedFeatures)
				case udm_message.EventGetSmfSelectData:
					supi := msg.HTTPRequest.Params["supi"]
					plmnID := msg.HTTPRequest.Query.Get("plmn-id")
					supportedFeatures := msg.HTTPRequest.Query.Get("supported-features")
					producer.HandleGetSmfSelectData(msg.ResponseChan, supi, plmnID, supportedFeatures)

				case udm_message.EventGetSupi:
					supi := msg.HTTPRequest.Params["supi"]
					plmnID := msg.HTTPRequest.Query.Get("plmn-id")
					dataSetNames := msg.HTTPRequest.Query["dataset-names"]
					supportedFeatures := msg.HTTPRequest.Query.Get("supported-features")
					producer.HandleGetSupi(msg.ResponseChan, supi, plmnID, dataSetNames, supportedFeatures)
				case udm_message.EventGetTraceData:
					supi := msg.HTTPRequest.Params["supi"]
					plmnID := msg.HTTPRequest.Query.Get("plmn-id")
					producer.HandleGetTraceData(msg.ResponseChan, supi, plmnID)
				case udm_message.EventGetUeContextInSmfData:
					supi := msg.HTTPRequest.Params["supi"]
					supportedFeatures := msg.HTTPRequest.Query.Get("supported-features")
					producer.HandleGetUeContextInSmfData(msg.ResponseChan, supi, supportedFeatures)
				case udm_message.EventSubscribe:
					supi := msg.HTTPRequest.Params["supi"]
					subscriptionID := msg.HTTPRequest.Params["subscriptionId"]
					producer.HandleSubscribe(msg.ResponseChan, supi, subscriptionID, msg.HTTPRequest.Body.(models.SdmSubscription))
				case udm_message.EventSubscribeToSharedData:
					producer.HandleSubscribeToSharedData(msg.ResponseChan, msg.HTTPRequest.Body.(models.SdmSubscription))
				case udm_message.EventUnsubscribe:
					supi := msg.HTTPRequest.Params["supi"]
					subscriptionID := msg.HTTPRequest.Params["subscriptionId"]
					producer.HandleUnsubscribe(msg.ResponseChan, supi, subscriptionID)
				case udm_message.EventUnsubscribeForSharedData:
					subscriptionID := msg.HTTPRequest.Params["subscriptionId"]
					producer.HandleUnsubscribeForSharedData(msg.ResponseChan, subscriptionID)
				case udm_message.EventModify:
					supi := msg.HTTPRequest.Params["supi"]
					subscriptionID := msg.HTTPRequest.Params["subscriptionId"]
					producer.HandleModify(msg.ResponseChan, supi, subscriptionID, msg.HTTPRequest.Body.(models.SdmSubsModification))
				case udm_message.EventModifyForSharedData:
					supi := msg.HTTPRequest.Params["supi"]
					subscriptionID := msg.HTTPRequest.Params["subscriptionId"]
					producer.HandleModifyForSharedData(msg.ResponseChan, supi, subscriptionID, msg.HTTPRequest.Body.(models.SdmSubsModification))
				case udm_message.EventCreateEeSubscription:
					ueIdentity := msg.HTTPRequest.Params["ueIdentity"]
					subscriptionID := msg.HTTPRequest.Params["subscriptionId"]
					producer.HandleCreateEeSubscription(msg.ResponseChan, ueIdentity, subscriptionID, msg.HTTPRequest.Body.(models.EeSubscription))
				case udm_message.EventDeleteEeSubscription:
					ueIdentity := msg.HTTPRequest.Params["ueIdentity"]
					subscriptionID := msg.HTTPRequest.Params["subscriptionId"]
					producer.HandleDeleteEeSubscription(msg.ResponseChan, ueIdentity, subscriptionID)
				case udm_message.EventUpdateEeSubscription:
					ueIdentity := msg.HTTPRequest.Params["ueIdentity"]
					subscriptionID := msg.HTTPRequest.Params["subscriptionId"]
					producer.HandleUpdateEeSubscription(msg.ResponseChan, ueIdentity, subscriptionID)
				case udm_message.EventGetAmf3gppAccess:
					ueID := msg.HTTPRequest.Params["ueId"]
					supportedFeatures := msg.HTTPRequest.Query.Get("supported-features")
					producer.HandleGetAmf3gppAccess(msg.ResponseChan, ueID, supportedFeatures)
				case udm_message.EventGetAmfNon3gppAccess:
					ueID := msg.HTTPRequest.Params["ueId"]
					supportedFeatures := msg.HTTPRequest.Query.Get("supported-features")
					producer.HandleGetAmfNon3gppAccess(msg.ResponseChan, ueID, supportedFeatures)
				case udm_message.EventRegistrationAmf3gppAccess:
					ueID := msg.HTTPRequest.Params["ueId"]
					producer.HandleRegistrationAmf3gppAccess(msg.ResponseChan, ueID, msg.HTTPRequest.Body.(models.Amf3GppAccessRegistration))
				case udm_message.EventRegisterAmfNon3gppAccess:
					ueID := msg.HTTPRequest.Params["ueId"]
					producer.HandleRegisterAmfNon3gppAccess(msg.ResponseChan, ueID, msg.HTTPRequest.Body.(models.AmfNon3GppAccessRegistration))
				case udm_message.EventUpdateAmf3gppAccess:
					ueID := msg.HTTPRequest.Params["ueId"]
					producer.HandleUpdateAmf3gppAccess(msg.ResponseChan, ueID, msg.HTTPRequest.Body.(models.Amf3GppAccessRegistrationModification))
				case udm_message.EventUpdateAmfNon3gppAccess:
					ueID := msg.HTTPRequest.Params["ueId"]
					producer.HandleUpdateAmfNon3gppAccess(msg.ResponseChan, ueID, msg.HTTPRequest.Body.(models.AmfNon3GppAccessRegistrationModification))
				case udm_message.EventDeregistrationSmfRegistrations:
					ueID := msg.HTTPRequest.Params["ueId"]
					pduSessionID := msg.HTTPRequest.Params["pduSessionId"]
					producer.HandleDeregistrationSmfRegistrations(msg.ResponseChan, ueID, pduSessionID)
				case udm_message.EventRegistrationSmfRegistrations:
					ueID := msg.HTTPRequest.Params["ueId"]
					pduSessionID := msg.HTTPRequest.Params["pduSessionId"]
					producer.HandleRegistrationSmfRegistrations(msg.ResponseChan, ueID, pduSessionID, msg.HTTPRequest.Body.(models.SmfRegistration))
				case udm_message.EventUpdate:
					gpsi := msg.HTTPRequest.Params["gpsi"]
					producer.HandleUpdate(msg.ResponseChan, gpsi, msg.HTTPRequest.Body.(models.PpData))
				case udm_message.EventDataChangeNotificationToNF:
					supi := msg.HTTPRequest.Params["supi"]
					producer.HandleDataChangeNotificationToNF(msg.ResponseChan, supi, msg.HTTPRequest.Body.(models.DataChangeNotify))
				default:
					HandlerLog.Warnf("Event[%d] has not implemented", msg.Event)
				}

			} else {
				HandlerLog.Errorln("UDM Channel closed!")
			}

		case <-time.After(time.Second * 1):
		}
	}
}
