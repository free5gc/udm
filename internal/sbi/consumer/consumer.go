package consumer

import (
	Nnrf_NFDiscovery "github.com/free5gc/openapi/nrf/NFDisc"
	Nnrf_NFManagement "github.com/free5gc/openapi/nrf/NFMgmt"
	Nudm_SubscriberDataManagement "github.com/free5gc/openapi/udm/SDM"
	Nudm_UEContextManagement "github.com/free5gc/openapi/udm/UECM"
	Nudr_DataRepository "github.com/free5gc/openapi/udr/DR"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/udm/pkg/app"
	"github.com/free5gc/util/nfheartbeat"
)

type ConsumerUdm interface {
	app.App
}

type Consumer struct {
	ConsumerUdm

	// consumer services
	*nnrfService
	*nudrService
	*nudmService
}

func NewConsumer(udm ConsumerUdm) (*Consumer, error) {
	c := &Consumer{
		ConsumerUdm: udm,
	}

	nrfService := &nnrfService{
		consumer:        c,
		nfMngmntClients: make(map[string]*Nnrf_NFManagement.APIClient),
		nfDiscClients:   make(map[string]*Nnrf_NFDiscovery.APIClient),
	}
	c.nnrfService = nrfService
	heartbeat, err := nfheartbeat.NewRunner(
		nrfRegistrar{nrfService},
		func() int32 { return c.Config().GetNfHeartBeatTimer() },
		logger.ConsumerLog,
	)
	if err != nil {
		return nil, err
	}
	nrfService.heartbeat = heartbeat

	c.nudrService = &nudrService{
		consumer:    c,
		nfDRClients: make(map[string]*Nudr_DataRepository.APIClient),
	}

	c.nudmService = &nudmService{
		consumer:      c,
		nfSDMClients:  make(map[string]*Nudm_SubscriberDataManagement.APIClient),
		nfUECMClients: make(map[string]*Nudm_UEContextManagement.APIClient),
	}
	return c, nil
}
