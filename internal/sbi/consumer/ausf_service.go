package consumer

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/free5gc/openapi"
	Nausf_UEAU "github.com/free5gc/openapi/ausf/UEAU"
	"github.com/free5gc/openapi/models"
	Nnrf_NFDiscovery "github.com/free5gc/openapi/nrf/NFDisc"
	sbi_metrics "github.com/free5gc/util/metrics/sbi"
	"github.com/free5gc/util/validator"
)

// SendUEAuthenticationDeregister clears the UE security context at the original
// AUSF (TS 29.509 5.2.2.3). The caller must ensure that this instance holds a stale
// context, not the latest Kausf. This is a service request, not a notification.
// Selection/tracking of that original AUSF and automatic triggers are separate
// from this sender; never fall back to an arbitrary discovered AUSF.
func (c *Consumer) SendUEAuthenticationDeregister(supi, ausfInstanceID string) error {
	if !validator.IsValidSupi(supi) {
		return fmt.Errorf("invalid deregistration SUPI")
	}
	id, err := uuid.Parse(ausfInstanceID)
	if err != nil || id.Version() != 4 {
		return fmt.Errorf("invalid original AUSF instance ID: UUID v4 required")
	}
	if strings.TrimSpace(c.Context().NrfUri) == "" {
		return fmt.Errorf("NRF URI is required to discover the original AUSF")
	}
	target, requester := models.Nrf_NFMgmt_NFType_AUSF, models.Nrf_NFMgmt_NFType_UDM
	result, err := c.SendSearchNFInstances(c.Context().NrfUri, Nnrf_NFDiscovery.SearchNFInstancesRequest{
		TargetNfType: &target, RequesterNfType: &requester,
		TargetNfInstanceId: &ausfInstanceID,
		ServiceNames:       []models.Nrf_NFMgmt_ServiceName{models.Nrf_NFMgmt_ServiceName_NAUSF_AUTH},
	})
	if err != nil {
		return fmt.Errorf("discover original AUSF: %w", err)
	}
	if result == nil {
		return fmt.Errorf("empty original AUSF discovery result")
	}
	var profiles []models.Nrf_NFDisc_NFProfile
	for _, profile := range result.NfInstances {
		if profile.NfInstanceId == ausfInstanceID && profile.NfType == target {
			profiles = append(profiles, profile)
		}
	}
	_, uri, err := openapi.GetServiceNfProfileAndUri(profiles, models.Nrf_NFMgmt_ServiceName_NAUSF_AUTH)
	if err != nil || uri == "" {
		return fmt.Errorf("original AUSF %s has no nausf-auth service", ausfInstanceID)
	}
	ctx, _, err := c.Context().GetTokenCtxForNFInstance(
		models.Nrf_NFMgmt_ServiceName_NAUSF_AUTH, target, ausfInstanceID,
	)
	if err != nil {
		return fmt.Errorf("get AUSF deregistration token: %w", err)
	}
	cfg := Nausf_UEAU.NewConfiguration()
	cfg.SetBasePath(uri)
	cfg.SetMetrics(sbi_metrics.SbiMetricHook)
	client := Nausf_UEAU.NewAPIClient(cfg)
	_, err = client.DefaultApi.UeAuthenticationsDeregisterPost(ctx, &Nausf_UEAU.UeAuthenticationsDeregisterPostRequest{
		RequestBody: &models.Ausf_UEAU_DeregistrationInfo{Supi: supi},
	})
	return err
}
