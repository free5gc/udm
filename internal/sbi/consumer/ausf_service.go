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

// SendUEAuthenticationDeregister requests removal of a UE authentication context
// from the original AUSF (TS 29.509 5.2.2.3). The caller must ensure that this
// instance holds a stale context, not the latest Kausf. This is a service request,
// not a notification. Tracking the original AUSF and automatically triggering
// deregistration are outside this sender's responsibility; it must never fall
// back to an arbitrary discovered AUSF.
//
// Reserved interface: UDM does not call this yet. Tracking the original AUSF
// and deciding when a context is stale are left to a follow-up change that
// wires this sender into the authentication flow.
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

	// Ask NRF for the specified original AUSF instance and its nausf-auth service.
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

	// Verify both instance ID and NF type before selecting a service URI, even
	// when NRF returns other profiles. Never substitute another AUSF instance.
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

	// When OAuth is enabled, obtain a token context scoped to nausf-auth and
	// addressed to this specific AUSF instance.
	ctx, _, err := c.Context().GetTokenCtxForNFInstance(
		models.Nrf_NFMgmt_ServiceName_NAUSF_AUTH, target, ausfInstanceID,
	)
	if err != nil {
		return fmt.Errorf("get AUSF deregistration token: %w", err)
	}

	// Use the OpenAPI client to POST the SUPI to /ue-authentications/deregister
	// with the token context, so the client can attach the Authorization header.
	cfg := Nausf_UEAU.NewConfiguration()
	cfg.SetBasePath(uri)
	cfg.SetMetrics(sbi_metrics.SbiMetricHook)
	client := Nausf_UEAU.NewAPIClient(cfg)
	_, err = client.DefaultApi.UeAuthenticationsDeregisterPost(ctx, &Nausf_UEAU.UeAuthenticationsDeregisterPostRequest{
		RequestBody: &models.Ausf_UEAU_DeregistrationInfo{Supi: supi},
	})
	return err
}
