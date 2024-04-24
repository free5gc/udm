package consumer

import (
	"context"

	"github.com/free5gc/openapi/Nnrf_NFDiscovery"
	"github.com/free5gc/openapi/Nnrf_NFManagement"
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
	return c, nil
}
