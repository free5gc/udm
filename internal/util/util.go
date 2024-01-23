package util

import (
	"net/http"

	"github.com/free5gc/openapi/models"
)

const (
	UdmDefaultKeyLogPath = "./log/udmsslkey.log"
	UdmDefaultPemPath    = "./config/TLS/udm.pem"
	UdmDefaultKeyPath    = "./config/TLS/udm.key"
	UdmDefaultConfigPath = "./config/udmcfg.yaml"
)

const (
	ServiceNameNnrfDisc = string(models.ServiceName_NNRF_DISC)
	ServiceNameNnrfNfm  = string(models.ServiceName_NNRF_NFM)
	ServiceNameNudmEe   = string(models.ServiceName_NUDM_EE)
	ServiceNameNudmPp   = string(models.ServiceName_NUDM_PP)
	ServiceNameNudmSdm  = string(models.ServiceName_NUDM_SDM)
	ServiceNameNudmUeau = string(models.ServiceName_NUDM_UEAU)
	ServiceNameNudmUecm = string(models.ServiceName_NUDM_UECM)
	ServiceNameNudrDr   = string(models.ServiceName_NUDR_DR)
)

const (
	NfTypeNRF = models.NfType_NRF
	NfTypeUDM = models.NfType_UDM
	NfTypeUDR = models.NfType_UDR
)

func ProblemDetailsSystemFailure(detail string) *models.ProblemDetails {
	return &models.ProblemDetails{
		Title:  "System failure",
		Status: http.StatusInternalServerError,
		Detail: detail,
		Cause:  "SYSTEM_FAILURE",
	}
}
