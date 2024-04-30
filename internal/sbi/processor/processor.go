package processor

import (
	"github.com/free5gc/udm/internal/repository"
)

type Processor struct {
}

func NewProcessor(runtimeRepo *repository.RuntimeRepository) *Processor {
	return &Processor{}
}
