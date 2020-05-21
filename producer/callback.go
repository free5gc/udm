package producer

import (
	"free5gc/lib/openapi/models"
	"free5gc/src/udm/handler/udm_message"
	"free5gc/src/udm/producer/callback"
)

func HandleDataChangeNotificationToNF(httpChannel chan udm_message.HandlerResponseMessage, supi string, dataChangeNotify models.DataChangeNotify) {

	notifyItems := dataChangeNotify.NotifyItems
	go callback.SendOnDataChangeNotification(supi, notifyItems)
}
