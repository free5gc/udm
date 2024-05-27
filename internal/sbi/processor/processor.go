package processor

import (
	"github.com/free5gc/udm/internal/sbi/consumer"
	"github.com/free5gc/udm/pkg/app"
)

type ProcessorUdm interface {
	app.App

	Consumer() *consumer.Consumer
}

type Processor struct {
	ProcessorUdm
	consumer *consumer.Consumer
}

type HandlerResponse struct {
	Status  int
	Headers map[string][]string
	Body    interface{}
}

func NewProcessor(udm ProcessorUdm) (*Processor, error) {
	p := &Processor{
		ProcessorUdm: udm,
		consumer:     udm.Consumer(),
	}
	return p, nil
}

func (p *Processor) Consumer() *consumer.Consumer {
	return p.consumer
}
