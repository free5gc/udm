package context

import (
	"fmt"
	"free5gc/lib/openapi/models"
	"free5gc/lib/path_util"
	"free5gc/src/udm/factory"
	"free5gc/src/udm/logger"
	"os"

	"github.com/google/uuid"
)

func TestInit() {
	DefaultUDMConfigPath := path_util.Gofree5gcPath("free5gc/config/udmcfg.conf")
	factory.InitConfigFactory(DefaultUDMConfigPath)
	Init()
}

func InitUDMContext(context *UDMContext) {
	config := factory.UdmConfig
	logger.UtilLog.Info("udmconfig Info: Version[", config.Info.Version, "] Description[", config.Info.Description, "]")
	configuration := config.Configuration
	context.NfId = uuid.New().String()
	if configuration.UdmName != "" {
		context.Name = configuration.UdmName
	}
	nrfclient := config.Configuration.Nrfclient
	context.NrfUri = fmt.Sprintf("%s://%s:%d", nrfclient.Scheme, nrfclient.Ipv4Addr, nrfclient.Port)
	sbi := configuration.Sbi
	context.UriScheme = models.UriScheme(sbi.Scheme)
	context.HttpIpv4Port = 29503
	context.HttpIPv4Address = "127.0.0.1"
	if sbi != nil {
		if sbi.RegisterIPv4 != "" {
			context.HttpIPv4Address = sbi.RegisterIPv4
		}
		if sbi.Port != 0 {
			context.HttpIpv4Port = sbi.Port
		}
		context.BindingIPv4 = os.Getenv(sbi.BindingIPv4)
		if context.BindingIPv4 != "" {
			logger.UtilLog.Info("Parsing ServerIPv4 address from ENV Variable.")
		} else {
			context.BindingIPv4 = sbi.BindingIPv4
			if context.BindingIPv4 == "" {
				logger.UtilLog.Info("Error parsing ServerIPv4 address as string. Using the 0.0.0.0 address as default.")
				context.BindingIPv4 = "0.0.0.0"
			}
		}
	}
	if configuration.NrfUri != "" {
		context.NrfUri = configuration.NrfUri
	} else {
		logger.UtilLog.Info("NRF Uri is empty! Using localhost as NRF IPv4 address.")
		context.NrfUri = fmt.Sprintf("%s://%s:%d", context.UriScheme, "127.0.0.1", 29510)
	}
	servingNameList := configuration.ServiceNameList

	context.Keys = configuration.Keys

	context.InitNFService(servingNameList, config.Info.Version)
}
