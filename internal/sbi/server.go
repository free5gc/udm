package sbi

import (
	"context"
	"net/http"

	"github.com/free5gc/udm/pkg/factory"

	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/logger"
	logger_util "github.com/free5gc/util/logger"
	"github.com/gin-gonic/gin"
)

const (
	CorsConfigMaxAge = 86400
)

type Endpoint struct {
	Method  string
	Pattern string
	APIFunc gin.HandlerFunc
}

func applyEndpoints(group *gin.RouterGroup, endpoints []Endpoint) {
	for _, endpoint := range endpoints {
		switch endpoint.Method {
		case "GET":
			group.GET(endpoint.Pattern, endpoint.APIFunc)
		case "POST":
			group.POST(endpoint.Pattern, endpoint.APIFunc)
		case "PUT":
			group.PUT(endpoint.Pattern, endpoint.APIFunc)
		case "PATCH":
			group.PATCH(endpoint.Pattern, endpoint.APIFunc)
		case "DELETE":
			group.DELETE(endpoint.Pattern, endpoint.APIFunc)
		}
	}
}

type udm interface {
	Config() *factory.Config
	Context() *udm_context.UdmNFContext
	CancelContext() context.Context
	//Consumer() *consumer.Consumer
	//Processor() *processor.Processor
}

type Server struct {
	udm

	httpServer *http.Server
	router     *gin.Engine
}

func NewServer(udm udm, tlsKeyLogPath string) (*Server, error) {
	s := &Server{
		udm:    udm,
		router: logger_util.NewGinWithLogrus(logger.GinLog),
	}

	//endpoints := s.getConvergenChargingEndpoints()

}
