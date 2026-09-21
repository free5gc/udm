package consumer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	Nnrf_NFDiscovery "github.com/free5gc/openapi/nrf/NFDisc"
	Nnrf_NFManagement "github.com/free5gc/openapi/nrf/NFMgmt"
	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/udm/internal/util"
	sbi_metrics "github.com/free5gc/util/metrics/sbi"
	"github.com/free5gc/util/nfheartbeat"
)

const registerRetryInterval = 2 * time.Second

type nnrfService struct {
	consumer *Consumer

	nfMngmntMu sync.RWMutex
	nfDiscMu   sync.RWMutex

	nfMngmntClients map[string]*Nnrf_NFManagement.APIClient
	nfDiscClients   map[string]*Nnrf_NFDiscovery.APIClient

	heartbeat *nfheartbeat.Runner

	// Interval in seconds from the last registration response. Written before the heartbeat starts, then only on
	// the heartbeat goroutine.
	heartbeatTimer int32
}

func (s *nnrfService) getNFManagementClient(uri string) *Nnrf_NFManagement.APIClient {
	if uri == "" {
		return nil
	}
	s.nfMngmntMu.RLock()
	client, ok := s.nfMngmntClients[uri]
	s.nfMngmntMu.RUnlock()
	if ok {
		return client
	}

	s.nfMngmntMu.Lock()
	defer s.nfMngmntMu.Unlock()
	// Another caller may have stored a client while the read lock was down.
	if client, ok = s.nfMngmntClients[uri]; ok {
		return client
	}

	configuration := Nnrf_NFManagement.NewConfiguration()
	configuration.SetBasePath(uri)
	configuration.SetMetrics(sbi_metrics.SbiMetricHook)
	client = Nnrf_NFManagement.NewAPIClient(configuration)

	s.nfMngmntClients[uri] = client
	return client
}

func (s *nnrfService) getNFDiscClient(uri string) *Nnrf_NFDiscovery.APIClient {
	if uri == "" {
		return nil
	}
	s.nfDiscMu.RLock()
	client, ok := s.nfDiscClients[uri]
	s.nfDiscMu.RUnlock()
	if ok {
		return client
	}

	s.nfDiscMu.Lock()
	defer s.nfDiscMu.Unlock()
	// Another caller may have stored a client while the read lock was down.
	if client, ok = s.nfDiscClients[uri]; ok {
		return client
	}

	configuration := Nnrf_NFDiscovery.NewConfiguration()
	configuration.SetBasePath(uri)
	configuration.SetMetrics(sbi_metrics.SbiMetricHook)
	client = Nnrf_NFDiscovery.NewAPIClient(configuration)

	s.nfDiscClients[uri] = client
	return client
}

func (s *nnrfService) SendSearchNFInstances(
	nrfUri string, param Nnrf_NFDiscovery.SearchNFInstancesRequest) (
	*models.Nrf_NFDisc_SearchResult, error,
) {
	// Set client and set url
	udmContext := s.consumer.Context()

	client := s.getNFDiscClient(udmContext.NrfUri)

	ctx, _, err := s.consumer.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NNRF_DISC, models.Nrf_NFMgmt_NFType_NRF)
	if err != nil {
		return nil, err
	}

	searchNfInstancesRsp, err1 := client.NFInstancesStoreApi.SearchNFInstances(ctx, &param)
	if err1 != nil {
		logger.ConsumerLog.Errorf("SearchNFInstances failed: %+v", err1)
		return nil, err1
	}
	if searchNfInstancesRsp == nil {
		logger.ConsumerLog.Errorf("SearchNFInstances result nil:%+v", err1)
		return nil, fmt.Errorf("SearchNFInstances result nil:%+v", err1)
	}
	return searchNfInstancesRsp.Nrf_NFDisc_SearchResult, nil
}

func (s *nnrfService) SendNFInstancesUDR(id string, types int) string {
	self := udm_context.GetSelf()
	targetNfType := models.Nrf_NFMgmt_NFType_UDR
	requestNfType := models.Nrf_NFMgmt_NFType_UDM
	searchNFinstanceRequest := Nnrf_NFDiscovery.SearchNFInstancesRequest{
		// 	DataSet: optional.NewInterface(models.DataSetId_SUBSCRIPTION),
	}
	searchNFinstanceRequest.RequesterNfType = &requestNfType
	searchNFinstanceRequest.TargetNfType = &targetNfType

	result, err := s.SendSearchNFInstances(self.NrfUri, searchNFinstanceRequest)
	if err != nil {
		logger.ConsumerLog.Error(err.Error())
		return ""
	}
	for _, profile := range result.NfInstances {
		return util.SearchNFServiceUri(
			profile,
			models.Nrf_NFMgmt_ServiceName_NUDR_DR,
			models.Nrf_NFMgmt_NFServiceStatus_REGISTERED,
		)
	}
	return ""
}

func (s *nnrfService) SendDeregisterNFInstance() (err error) {
	logger.ConsumerLog.Infof("Send Deregister NFInstance")

	ctx, _, err := s.consumer.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NNRF_NFM, models.Nrf_NFMgmt_NFType_NRF)
	if err != nil {
		return err
	}

	udmContext := s.consumer.Context()
	client := s.getNFManagementClient(udmContext.NrfUri)

	var derigisterNfInstanceRequest Nnrf_NFManagement.DeregisterNFInstanceRequest
	derigisterNfInstanceRequest.NfInstanceID = &udmContext.NfId
	_, err = client.NFInstanceIDDocumentApi.DeregisterNFInstance(ctx, &derigisterNfInstanceRequest)

	return err
}

// RegisterNFInstance PUTs the profile on the NF's own instance ID (3GPP TS 29.510 clause 5.2.2.2.2) until it
// succeeds or ctx is done. Only the startup call may pass applyOAuth2: it writes OAuth2Required unsynchronized.
func (s *nnrfService) RegisterNFInstance(ctx context.Context, applyOAuth2 bool) error {
	udmContext := s.consumer.Context()
	client := s.getNFManagementClient(udmContext.NrfUri)
	if client == nil {
		return errors.Errorf("RegisterNFInstance: nrf not found")
	}

	nfProfile, err := s.buildNfProfile(udmContext)
	if err != nil {
		return errors.Wrap(err, "RegisterNFInstance buildNfProfile()")
	}

	var res *Nnrf_NFManagement.RegisterNFInstanceResponse
	registerNfInstanceRequest := &Nnrf_NFManagement.RegisterNFInstanceRequest{
		NfInstanceID: &udmContext.NfId,
		RequestBody:  &nfProfile,
	}
	for ctx.Err() == nil {
		res, err = client.NFInstanceIDDocumentApi.RegisterNFInstance(ctx, registerNfInstanceRequest)
		if err == nil && res != nil {
			var nf models.Nrf_NFMgmt_NFProfile
			if res.Nrf_NFMgmt_NFProfile != nil {
				nf = *res.Nrf_NFMgmt_NFProfile
			}
			s.processRegisterResponse(udmContext, nf, applyOAuth2)
			return nil
		}
		logger.ConsumerLog.Errorf("UDM register to NRF Error[%v]", err)
		select {
		case <-ctx.Done():
		case <-time.After(registerRetryInterval):
		}
	}
	return fmt.Errorf("NFRegister aborted: %w (last error: %v)", ctx.Err(), err)
}

// processRegisterResponse adopts what the NRF answered to the NFRegister PUT: the
// heartbeat interval and the oauth2 custom info.
func (s *nnrfService) processRegisterResponse(
	udmContext *udm_context.UDMContext,
	nf models.Nrf_NFMgmt_NFProfile,
	applyOAuth2 bool,
) {
	s.heartbeatTimer = nf.HeartBeatTimer

	oauth2 := false
	if customInfo, ok := nf.CustomInfo.(map[string]interface{}); ok {
		if v, isBool := customInfo["oauth2"].(bool); isBool {
			oauth2 = v
			logger.MainLog.Infoln("OAuth2 setting receive from NRF:", oauth2)
		}
	}
	if applyOAuth2 {
		udmContext.OAuth2Required = oauth2
		if oauth2 && udmContext.NrfCertPem == "" {
			logger.CfgLog.Error("OAuth2 enable but no nrfCertPem provided in config.")
		}
	} else if oauth2 != udmContext.OAuth2Required {
		logger.ConsumerLog.Warnf("NRF OAuth2 setting changed to %v, restart UDM to apply it", oauth2)
	}
}

// SendUpdateNFInstance sends an NFUpdate PATCH to the NRF, honoring ctx. The
// raw err comes back alongside any ProblemDetails so callers can read its
// GenericOpenAPIError status.
func (s *nnrfService) SendUpdateNFInstance(ctx context.Context, patchItems []models.PatchItem) (
	nf models.Nrf_NFMgmt_NFProfile, problemDetails *models.ProblemDetails, err error,
) {
	udmContext := s.consumer.Context()
	tokCtx, pd, err := udmContext.GetTokenCtx(
		models.Nrf_NFMgmt_ServiceName_NNRF_NFM,
		models.Nrf_NFMgmt_NFType_NRF,
	)
	if err != nil {
		return nf, pd, err
	}
	// GetTokenCtx takes no parent, so the token request stays uncancellable;
	// transplanting the token lets at least the PATCH honor ctx.
	if tok := tokCtx.Value(openapi.ContextOAuth2); tok != nil {
		ctx = context.WithValue(ctx, openapi.ContextOAuth2, tok)
	}

	client := s.getNFManagementClient(udmContext.NrfUri)
	if client == nil {
		return nf, nil, errors.Errorf("SendUpdateNFInstance: nrf not found")
	}

	request := &Nnrf_NFManagement.UpdateNFInstanceRequest{
		NfInstanceID: &udmContext.NfId,
		RequestBody:  patchItems,
	}

	res, err := client.NFInstanceIDDocumentApi.UpdateNFInstance(ctx, request)
	if err != nil {
		var apiErr openapi.GenericOpenAPIError
		if errors.As(err, &apiErr) {
			if updateErr, okModel := apiErr.Model().(Nnrf_NFManagement.UpdateNFInstanceError); okModel {
				return nf, updateErr.ProblemDetails, err
			}
		}
		return nf, nil, err
	}
	if res == nil {
		return nf, nil, errors.Errorf("empty NFUpdate response")
	}
	if res.Nrf_NFMgmt_NFProfile != nil {
		nf = *res.Nrf_NFMgmt_NFProfile
	}
	return nf, nil, nil
}

func (s *nnrfService) buildNfProfile(udmContext *udm_context.UDMContext) (
	profile models.Nrf_NFMgmt_NFProfile, err error,
) {
	profile.NfInstanceId = udmContext.NfId
	profile.NfType = models.Nrf_NFMgmt_NFType_UDM
	profile.NfStatus = models.Nrf_NFMgmt_NFStatus_REGISTERED
	profile.Ipv4Addresses = append(profile.Ipv4Addresses, udmContext.RegisterIPv4)
	for _, nfService := range udmContext.NfService {
		profile.NfServices = append(profile.NfServices, nfService)
	}
	profile.UdmInfo = &models.Nrf_NFMgmt_UdmInfo{
		// Todo
		// SupiRanges: &[]models.Nrf_NFMgmt_SupiRange{
		// 	{
		// 		//from TS 29.510 6.1.6.2.9 example2
		//		//no need to set supirange in this moment 2019/10/4
		// 		Start:   "123456789040000",
		// 		End:     "123456789059999",
		// 		Pattern: "^imsi-12345678904[0-9]{4}$",
		// 	},
		// },
	}
	return
}
