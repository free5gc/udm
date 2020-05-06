package producer

import (
	"free5gc/lib/openapi/models"
	"free5gc/src/udm/producer/callback"
	"free5gc/src/udm/udm_handler/udm_message"
)

func HandleDataChangeNotificationToNF(httpChannel chan udm_message.HandlerResponseMessage, supi string, dataChangeNotify models.DataChangeNotify) {

	notifyItems := dataChangeNotify.NotifyItems
	go callback.SendOnDataChangeNotification(supi, notifyItems)
}
