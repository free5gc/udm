package consumer

import (
	"context"

	"github.com/free5gc/openapi/Nnrf_NFDiscovery"
	"github.com/free5gc/openapi/Nnrf_NFManagement"
	"github.com/free5gc/openapi/Nudm_SubscriberDataManagement"
	"github.com/free5gc/openapi/Nudm_UEContextManagement"
	"github.com/free5gc/openapi/Nudr_DataRepository"
	"github.com/free5gc/udm/pkg/factory"

	udm_context "github.com/free5gc/udm/internal/context"
)

type udm interface {
	Config() *factory.Config
	Context() *udm_context.UDMContext
	CancelContext() context.Context
}

type Consumer struct {
	udm

	// consumer services
	*nnrfService
	*nudrService
	*nudmService
}

func NewConsumer(udm udm) (*Consumer, error) {
	c := &Consumer{
		udm: udm,
	}

	c.nnrfService = &nnrfService{
		consumer:        c,
		nfMngmntClients: make(map[string]*Nnrf_NFManagement.APIClient),
		nfDiscClients:   make(map[string]*Nnrf_NFDiscovery.APIClient),
	}

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
