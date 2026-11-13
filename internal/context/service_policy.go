package context

import (
	"slices"

	"github.com/free5gc/openapi/models"
)

var servicePolicies = map[models.Nrf_NFMgmt_ServiceName][]models.Nrf_NFMgmt_NFType{
	models.Nrf_NFMgmt_ServiceName_NUDM_SDM: {
		models.Nrf_NFMgmt_NFType_AMF,
		models.Nrf_NFMgmt_NFType_SMF,
		// UDR sends SDM data-change callbacks to UDM.
		models.Nrf_NFMgmt_NFType_UDR,
		// UDM uses Nudm_SDM for shared-data self-calls.
		models.Nrf_NFMgmt_NFType_UDM,
		models.Nrf_NFMgmt_NFType_SMSF,
		models.Nrf_NFMgmt_NFType_GMLC,
		models.Nrf_NFMgmt_NFType_NEF,
		models.Nrf_NFMgmt_NFType_5_G_DDNMF,
		models.Nrf_NFMgmt_NFType_NWDAF,
		models.Nrf_NFMgmt_NFType_DCCF,
		models.Nrf_NFMgmt_NFType_AF,
	},
	models.Nrf_NFMgmt_ServiceName_NUDM_UECM: {
		models.Nrf_NFMgmt_NFType_AMF,
		models.Nrf_NFMgmt_NFType_SMF,
		models.Nrf_NFMgmt_NFType_SMSF,
		models.Nrf_NFMgmt_NFType_NWDAF,
		models.Nrf_NFMgmt_NFType_NEF,
		models.Nrf_NFMgmt_NFType_NSSAAF,
		models.Nrf_NFMgmt_NFType_DCCF,
		models.Nrf_NFMgmt_NFType_HSS,
	},
	models.Nrf_NFMgmt_ServiceName_NUDM_UEAU: {
		models.Nrf_NFMgmt_NFType_AUSF,
	},
	models.Nrf_NFMgmt_ServiceName_NUDM_EE: {
		models.Nrf_NFMgmt_NFType_NEF,
	},
	models.Nrf_NFMgmt_ServiceName_NUDM_PP: {
		models.Nrf_NFMgmt_NFType_NEF,
		models.Nrf_NFMgmt_NFType_SOR_AF,
		models.Nrf_NFMgmt_NFType_AMF,
	},
	// TS 29.503 Nudm_NIDDAuthorization: NF Service Consumer is NEF.
	models.Nrf_NFMgmt_ServiceName_NUDM_NIDDAU: {
		models.Nrf_NFMgmt_NFType_NEF,
	},
	// TS 29.503 Nudm_ReportSMDeliveryStatus: NF Service Consumers are
	// SMS-GMSC, IP-SM-GW and SMS Router.
	models.Nrf_NFMgmt_ServiceName_NUDM_RSDS: {
		models.Nrf_NFMgmt_NFType_SMS_GMSC,
		models.Nrf_NFMgmt_NFType_IP_SM_GW,
		models.Nrf_NFMgmt_NFType_SMS_ROUTER,
	},
	// TS 29.503 Nudm_ServiceSpecificAuthorization: NF Service Consumer is NEF.
	models.Nrf_NFMgmt_ServiceName_NUDM_SSAU: {
		models.Nrf_NFMgmt_NFType_NEF,
	},
}

func AllowedNfTypesForService(
	serviceName models.Nrf_NFMgmt_ServiceName,
) ([]models.Nrf_NFMgmt_NFType, bool) {
	allowed, known := servicePolicies[serviceName]
	return slices.Clone(allowed), known
}
