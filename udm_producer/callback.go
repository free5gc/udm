package udm_producer

import (
	"gofree5gc/lib/openapi/models"
	"gofree5gc/src/udm/udm_handler/udm_message"
	"gofree5gc/src/udm/udm_producer/udm_producer_callback"
)

func HandleDataChangeNotificationToNF(httpChannel chan udm_message.HandlerResponseMessage, supi string, dataChangeNotify models.DataChangeNotify) {

	notifyItems := dataChangeNotify.NotifyItems
	go udm_producer_callback.SendOnDataChangeNotification(supi, notifyItems)
}
