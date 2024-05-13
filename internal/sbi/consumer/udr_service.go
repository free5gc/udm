package consumer

import (
	cryptoRand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/antihax/optional"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/Nudr_DataRepository"
	"github.com/free5gc/openapi/models"
	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/udm/internal/sbi/producer/callback"
	"github.com/free5gc/udm/pkg/suci"
	"github.com/free5gc/util/milenage"
	"github.com/free5gc/util/ueauth"
)

type nudrService struct {
	consumer *Consumer

	nfDRMu sync.RWMutex

	nfDRClients map[string]*Nudr_DataRepository.APIClient
}

const (
	NFDiscoveryToUDRParamNone int = iota
	NFDiscoveryToUDRParamSupi
	NFDiscoveryToUDRParamExtGroupId
	NFDiscoveryToUDRParamGpsi
)

const (
	SqnMAx    int64 = 0xFFFFFFFFFFFF
	ind       int64 = 32
	keyStrLen int   = 32
	opStrLen  int   = 32
	opcStrLen int   = 32
)

const (
	authenticationRejected string = "AUTHENTICATION_REJECTED"
	resyncAMF              string = "0000"
)

func (s *nudrService) createUDMClientToUDR(id string) (*Nudr_DataRepository.APIClient, error) {
	uri := s.getUdrURI(id)
	if uri == "" {
		logger.ProcLog.Errorf("ID[%s] does not match any UDR", id)
		return nil, fmt.Errorf("No UDR URI found")
	}
	s.nfDRMu.RLock()
	client, ok := s.nfDRClients[uri]
	if ok {
		defer s.nfDRMu.RUnlock()
		return client, nil
	}

	cfg := Nudr_DataRepository.NewConfiguration()
	cfg.SetBasePath(uri)
	client = Nudr_DataRepository.NewAPIClient(cfg)

	s.nfDRMu.RUnlock()
	s.nfDRMu.Lock()
	defer s.nfDRMu.Unlock()
	s.nfDRClients[uri] = client
	return client, nil
}

func (s *nudrService) containDataSetName(dataSetNames []string, target string) bool {
	for _, dataSetName := range dataSetNames {
		if dataSetName == target {
			return true
		}
	}
	return false
}

func (s *nudrService) getUdrURI(id string) string {
	if strings.Contains(id, "imsi") || strings.Contains(id, "nai") { // supi
		ue, ok := udm_context.GetSelf().UdmUeFindBySupi(id)
		if ok {
			if ue.UdrUri == "" {
				ue.UdrUri = SendNFIntancesUDR(id, NFDiscoveryToUDRParamSupi)
			}
			return ue.UdrUri
		} else {
			ue = udm_context.GetSelf().NewUdmUe(id)
			ue.UdrUri = SendNFIntancesUDR(id, NFDiscoveryToUDRParamSupi)
			return ue.UdrUri
		}
	} else if strings.Contains(id, "pei") {
		var udrURI string
		udm_context.GetSelf().UdmUePool.Range(func(key, value interface{}) bool {
			ue := value.(*udm_context.UdmUeContext)
			if ue.Amf3GppAccessRegistration != nil && ue.Amf3GppAccessRegistration.Pei == id {
				if ue.UdrUri == "" {
					ue.UdrUri = SendNFIntancesUDR(ue.Supi, NFDiscoveryToUDRParamSupi)
				}
				udrURI = ue.UdrUri
				return false
			} else if ue.AmfNon3GppAccessRegistration != nil && ue.AmfNon3GppAccessRegistration.Pei == id {
				if ue.UdrUri == "" {
					ue.UdrUri = SendNFIntancesUDR(ue.Supi, NFDiscoveryToUDRParamSupi)
				}
				udrURI = ue.UdrUri
				return false
			}
			return true
		})
		return udrURI
	} else if strings.Contains(id, "extgroupid") {
		// extra group id
		return SendNFIntancesUDR(id, NFDiscoveryToUDRParamExtGroupId)
	} else if strings.Contains(id, "msisdn") || strings.Contains(id, "extid") {
		// gpsi
		return SendNFIntancesUDR(id, NFDiscoveryToUDRParamGpsi)
	}
	return SendNFIntancesUDR("", NFDiscoveryToUDRParamNone)
}

func (s *nudrService) aucSQN(opc, k, auts, rand []byte) ([]byte, []byte) {
	AK, SQNms := make([]byte, 6), make([]byte, 6)
	macS := make([]byte, 8)
	ConcSQNms := auts[:6]
	AMF, err := hex.DecodeString(resyncAMF)
	if err != nil {
		return nil, nil
	}

	logger.UeauLog.Tracef("aucSQN: ConcSQNms=[%x]", ConcSQNms)

	err = milenage.F2345(opc, k, rand, nil, nil, nil, nil, AK)
	if err != nil {
		logger.UeauLog.Errorln("aucSQN milenage F2345 err:", err)
	}

	for i := 0; i < 6; i++ {
		SQNms[i] = AK[i] ^ ConcSQNms[i]
	}

	logger.UeauLog.Tracef("aucSQN: opc=[%x], k=[%x], rand=[%x], AMF=[%x], SQNms=[%x]\n", opc, k, rand, AMF, SQNms)
	// The AMF used to calculate MAC-S assumes a dummy value of all zeros
	err = milenage.F1(opc, k, rand, SQNms, AMF, nil, macS)
	if err != nil {
		logger.UeauLog.Errorln("aucSQN milenage F1 err:", err)
	}
	logger.UeauLog.Tracef("aucSQN: macS=[%x]\n", macS)
	return SQNms, macS
}

func (s *nudrService) strictHex(ss string, n int) string {
	l := len(ss)
	if l < n {
		return strings.Repeat("0", n-l) + ss
	} else {
		return ss[l-n : l]
	}
}

// EE service
func (s *nudrService) CreateEeSubscriptionProcedure(ueIdentity string,
	eesubscription models.EeSubscription,
) (*models.CreatedEeSubscription, *models.ProblemDetails) {
	udmSelf := udm_context.GetSelf()
	logger.EeLog.Debugf("udIdentity: %s", ueIdentity)
	switch {
	// GPSI (MSISDN identifier) represents a single UE
	case strings.HasPrefix(ueIdentity, "msisdn-"):
		fallthrough
	// GPSI (External identifier) represents a single UE
	case strings.HasPrefix(ueIdentity, "extid-"):
		if ue, ok := udmSelf.UdmUeFindByGpsi(ueIdentity); ok {
			id, err := udmSelf.EeSubscriptionIDGenerator.Allocate()
			if err != nil {
				problemDetails := &models.ProblemDetails{
					Status: http.StatusInternalServerError,
					Cause:  "UNSPECIFIED_NF_FAILURE",
				}
				return nil, problemDetails
			}

			subscriptionID := strconv.Itoa(int(id))
			ue.EeSubscriptions[subscriptionID] = &eesubscription
			createdEeSubscription := &models.CreatedEeSubscription{
				EeSubscription: &eesubscription,
			}
			return createdEeSubscription, nil
		} else {
			problemDetails := &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "USER_NOT_FOUND",
			}
			return nil, problemDetails
		}
	// external groupID represents a group of UEs
	case strings.HasPrefix(ueIdentity, "extgroupid-"):
		id, err := udmSelf.EeSubscriptionIDGenerator.Allocate()
		if err != nil {
			problemDetails := &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Cause:  "UNSPECIFIED_NF_FAILURE",
			}
			return nil, problemDetails
		}
		subscriptionID := strconv.Itoa(int(id))
		createdEeSubscription := &models.CreatedEeSubscription{
			EeSubscription: &eesubscription,
		}

		udmSelf.UdmUePool.Range(func(key, value interface{}) bool {
			ue := value.(*udm_context.UdmUeContext)
			if ue.ExternalGroupID == ueIdentity {
				ue.EeSubscriptions[subscriptionID] = &eesubscription
			}
			return true
		})
		return createdEeSubscription, nil
	// represents any UEs
	case ueIdentity == "anyUE":
		id, err := udmSelf.EeSubscriptionIDGenerator.Allocate()
		if err != nil {
			problemDetails := &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Cause:  "UNSPECIFIED_NF_FAILURE",
			}
			return nil, problemDetails
		}
		subscriptionID := strconv.Itoa(int(id))
		createdEeSubscription := &models.CreatedEeSubscription{
			EeSubscription: &eesubscription,
		}
		udmSelf.UdmUePool.Range(func(key, value interface{}) bool {
			ue := value.(*udm_context.UdmUeContext)
			ue.EeSubscriptions[subscriptionID] = &eesubscription
			return true
		})
		return createdEeSubscription, nil
	default:
		problemDetails := &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "MANDATORY_IE_INCORRECT",
			InvalidParams: []models.InvalidParam{
				{
					Param:  "ueIdentity",
					Reason: "incorrect format",
				},
			},
		}
		return nil, problemDetails
	}
}

// TODO: complete this procedure based on TS 29503 5.5
func (s *nudrService) DeleteEeSubscriptionProcedure(ueIdentity string, subscriptionID string) {
	udmSelf := udm_context.GetSelf()

	switch {
	case strings.HasPrefix(ueIdentity, "msisdn-"):
		fallthrough
	case strings.HasPrefix(ueIdentity, "extid-"):
		if ue, ok := udmSelf.UdmUeFindByGpsi(ueIdentity); ok {
			delete(ue.EeSubscriptions, subscriptionID)
		}
	case strings.HasPrefix(ueIdentity, "extgroupid-"):
		udmSelf.UdmUePool.Range(func(key, value interface{}) bool {
			ue := value.(*udm_context.UdmUeContext)
			if ue.ExternalGroupID == ueIdentity {
				delete(ue.EeSubscriptions, subscriptionID)
			}
			return true
		})
	case ueIdentity == "anyUE":
		udmSelf.UdmUePool.Range(func(key, value interface{}) bool {
			ue := value.(*udm_context.UdmUeContext)
			delete(ue.EeSubscriptions, subscriptionID)
			return true
		})
	}
	if id, err := strconv.ParseInt(subscriptionID, 10, 64); err != nil {
		logger.EeLog.Warnf("subscriptionID covert type error: %+v", err)
	} else {
		udmSelf.EeSubscriptionIDGenerator.FreeID(id)
	}
}

// TODO: complete this procedure based on TS 29503 5.5
func (s *nudrService) UpdateEeSubscriptionProcedure(ueIdentity string, subscriptionID string,
	patchList []models.PatchItem,
) *models.ProblemDetails {
	udmSelf := udm_context.GetSelf()

	switch {
	case strings.HasPrefix(ueIdentity, "msisdn-"):
		fallthrough
	case strings.HasPrefix(ueIdentity, "extid-"):
		if ue, ok := udmSelf.UdmUeFindByGpsi(ueIdentity); ok {
			if _, ok := ue.EeSubscriptions[subscriptionID]; ok {
				for _, patchItem := range patchList {
					logger.EeLog.Debugf("patch item: %+v", patchItem)
					// TODO: patch the Eesubscription
				}
				return nil
			} else {
				problemDetails := &models.ProblemDetails{
					Status: http.StatusNotFound,
					Cause:  "SUBSCRIPTION_NOT_FOUND",
				}
				return problemDetails
			}
		} else {
			problemDetails := &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "SUBSCRIPTION_NOT_FOUND",
			}
			return problemDetails
		}
	case strings.HasPrefix(ueIdentity, "extgroupid-"):
		udmSelf.UdmUePool.Range(func(key, value interface{}) bool {
			ue := value.(*udm_context.UdmUeContext)
			if ue.ExternalGroupID == ueIdentity {
				if _, ok := ue.EeSubscriptions[subscriptionID]; ok {
					for _, patchItem := range patchList {
						logger.EeLog.Debugf("patch item: %+v", patchItem)
						// TODO: patch the Eesubscription
					}
				}
			}
			return true
		})
		return nil
	case ueIdentity == "anyUE":
		udmSelf.UdmUePool.Range(func(key, value interface{}) bool {
			ue := value.(*udm_context.UdmUeContext)
			if _, ok := ue.EeSubscriptions[subscriptionID]; ok {
				for _, patchItem := range patchList {
					logger.EeLog.Debugf("patch item: %+v", patchItem)
					// TODO: patch the Eesubscription
				}
			}
			return true
		})
		return nil
	default:
		problemDetails := &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "MANDATORY_IE_INCORRECT",
			InvalidParams: []models.InvalidParam{
				{
					Param:  "ueIdentity",
					Reason: "incorrect format",
				},
			},
		}
		return problemDetails
	}
}

func (s *nudrService) ConfirmAuthDataProcedure(authEvent models.AuthEvent, supi string) (problemDetails *models.ProblemDetails) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return pd
	}
	var createAuthParam Nudr_DataRepository.CreateAuthenticationStatusParamOpts
	optInterface := optional.NewInterface(authEvent)
	createAuthParam.AuthEvent = optInterface

	client, err := s.createUDMClientToUDR(supi)
	if err != nil {
		return openapi.ProblemDetailsSystemFailure(err.Error())
	}

	resp, err := client.AuthenticationStatusDocumentApi.CreateAuthenticationStatus(
		ctx, supi, &createAuthParam)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}

		logger.UeauLog.Errorln("ConfirmAuth err:", err.Error())
		return problemDetails
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.UeauLog.Errorf("CreateAuthenticationStatus response body cannot close: %+v", rspCloseErr)
		}
	}()

	return nil
}

func (s *nudrService) GenerateAuthDataProcedure(authInfoRequest models.AuthenticationInfoRequest, supiOrSuci string) (
	response *models.AuthenticationInfoResult, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	logger.UeauLog.Traceln("In GenerateAuthDataProcedure")

	response = &models.AuthenticationInfoResult{}
	rand.Seed(time.Now().UnixNano())
	supi, err := suci.ToSupi(supiOrSuci, udm_context.GetSelf().SuciProfiles)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusForbidden,
			Cause:  authenticationRejected,
			Detail: err.Error(),
		}

		logger.UeauLog.Errorln("suciToSupi error: ", err.Error())
		return nil, problemDetails
	}

	logger.UeauLog.Tracef("supi conversion => [%s]", supi)

	client, err := s.createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}
	authSubs, res, err := client.AuthenticationDataDocumentApi.QueryAuthSubsData(ctx, supi, nil)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusForbidden,
			Cause:  authenticationRejected,
			Detail: err.Error(),
		}

		switch res.StatusCode {
		case http.StatusNotFound:
			logger.UeauLog.Warnf("Return from UDR QueryAuthSubsData error")
		default:
			logger.UeauLog.Errorln("Return from UDR QueryAuthSubsData error")
		}
		return nil, problemDetails
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryAuthSubsData response body cannot close: %+v", rspCloseErr)
		}
	}()

	/*
		K, RAND, CK, IK: 128 bits (16 bytes) (hex len = 32)
		SQN, AK: 48 bits (6 bytes) (hex len = 12) TS33.102 - 6.3.2
		AMF: 16 bits (2 bytes) (hex len = 4) TS33.102 - Annex H
	*/

	hasK, hasOP, hasOPC := false, false, false
	var kStr, opStr, opcStr string
	var k, op, opc []byte

	if authSubs.PermanentKey != nil {
		kStr = authSubs.PermanentKey.PermanentKeyValue
		if len(kStr) == keyStrLen {
			k, err = hex.DecodeString(kStr)
			if err != nil {
				logger.UeauLog.Errorln("err:", err)
			} else {
				hasK = true
			}
		} else {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusForbidden,
				Cause:  authenticationRejected,
			}

			logger.UeauLog.Errorln("kStr length is ", len(kStr))
			return nil, problemDetails
		}
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusForbidden,
			Cause:  authenticationRejected,
		}

		logger.UeauLog.Errorln("Nil PermanentKey")
		return nil, problemDetails
	}

	if authSubs.Milenage != nil {
		if authSubs.Milenage.Op != nil && authSubs.Milenage.Op.OpValue != "" {
			opStr = authSubs.Milenage.Op.OpValue
			if len(opStr) == opStrLen {
				op, err = hex.DecodeString(opStr)
				if err != nil {
					logger.UeauLog.Errorln("err:", err)
				} else {
					hasOP = true
				}
			} else {
				logger.UeauLog.Errorln("opStr length is ", len(opStr))
			}
		} else {
			logger.UeauLog.Infoln("Nil Op")
		}
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusForbidden,
			Cause:  authenticationRejected,
		}

		logger.UeauLog.Infoln("Nil Milenage")
		return nil, problemDetails
	}

	if authSubs.Opc != nil && authSubs.Opc.OpcValue != "" {
		opcStr = authSubs.Opc.OpcValue
		if len(opcStr) == opcStrLen {
			opc, err = hex.DecodeString(opcStr)
			if err != nil {
				logger.UeauLog.Errorln("err:", err)
			} else {
				hasOPC = true
			}
		} else {
			logger.UeauLog.Errorln("opcStr length is ", len(opcStr))
		}
	} else {
		logger.UeauLog.Infoln("Nil Opc")
	}

	if !hasOPC && !hasOP {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusForbidden,
			Cause:  authenticationRejected,
		}

		return nil, problemDetails
	}

	if !hasOPC {
		if hasK && hasOP {
			opc, err = milenage.GenerateOPC(k, op)
			if err != nil {
				logger.UeauLog.Errorln("milenage GenerateOPC err:", err)
			}
		} else {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusForbidden,
				Cause:  authenticationRejected,
			}

			logger.UeauLog.Errorln("Unable to derive OPC")
			return nil, problemDetails
		}
	}

	sqnStr := s.strictHex(authSubs.SequenceNumber, 12)
	logger.UeauLog.Traceln("sqnStr", sqnStr)
	sqn, err := hex.DecodeString(sqnStr)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusForbidden,
			Cause:  authenticationRejected,
			Detail: err.Error(),
		}

		logger.UeauLog.Errorln("err:", err)
		return nil, problemDetails
	}

	logger.UeauLog.Tracef("K=[%x], sqn=[%x], OP=[%x], OPC=[%x]", k, sqn, op, opc)

	RAND := make([]byte, 16)
	_, err = cryptoRand.Read(RAND)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusForbidden,
			Cause:  authenticationRejected,
			Detail: err.Error(),
		}

		logger.UeauLog.Errorln("err:", err)
		return nil, problemDetails
	}

	amfStr := s.strictHex(authSubs.AuthenticationManagementField, 4)
	logger.UeauLog.Traceln("amfStr", amfStr)
	AMF, err := hex.DecodeString(amfStr)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusForbidden,
			Cause:  authenticationRejected,
			Detail: err.Error(),
		}

		logger.UeauLog.Errorln("err:", err)
		return nil, problemDetails
	}

	logger.UeauLog.Tracef("RAND=[%x], AMF=[%x]", RAND, AMF)

	// re-synchronization
	if authInfoRequest.ResynchronizationInfo != nil {
		logger.UeauLog.Infof("Authentication re-synchronization")

		Auts, deCodeErr := hex.DecodeString(authInfoRequest.ResynchronizationInfo.Auts)
		if deCodeErr != nil {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusForbidden,
				Cause:  authenticationRejected,
				Detail: deCodeErr.Error(),
			}

			logger.UeauLog.Errorln("err:", deCodeErr)
			return nil, problemDetails
		}

		randHex, deCodeErr := hex.DecodeString(authInfoRequest.ResynchronizationInfo.Rand)
		if deCodeErr != nil {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusForbidden,
				Cause:  authenticationRejected,
				Detail: deCodeErr.Error(),
			}

			logger.UeauLog.Errorln("err:", deCodeErr)
			return nil, problemDetails
		}

		SQNms, macS := s.aucSQN(opc, k, Auts, randHex)
		if reflect.DeepEqual(macS, Auts[6:]) {
			_, err = cryptoRand.Read(RAND)
			if err != nil {
				problemDetails = &models.ProblemDetails{
					Status: http.StatusForbidden,
					Cause:  authenticationRejected,
					Detail: deCodeErr.Error(),
				}

				logger.UeauLog.Errorln("err:", deCodeErr)
				return nil, problemDetails
			}

			// increment sqn authSubs.SequenceNumber
			bigSQN := big.NewInt(0)
			sqnStr = hex.EncodeToString(SQNms)
			logger.UeauLog.Tracef("SQNstr=[%s]", sqnStr)
			bigSQN.SetString(sqnStr, 16)

			bigInc := big.NewInt(ind + 1)

			bigP := big.NewInt(SqnMAx)
			bigSQN = bigInc.Add(bigSQN, bigInc)
			bigSQN = bigSQN.Mod(bigSQN, bigP)
			sqnStr = fmt.Sprintf("%x", bigSQN)
			sqnStr = s.strictHex(sqnStr, 12)
		} else {
			logger.UeauLog.Errorln("Re-Sync MAC failed ", supiOrSuci)
			// Check if suci
			suciPart := strings.Split(supiOrSuci, "-")
			if suciPart[suci.PrefixPlace] == suci.PrefixSUCI &&
				suciPart[suci.SupiTypePlace] == suci.SupiTypeIMSI &&
				suciPart[suci.SchemePlace] != suci.NullScheme {
				// Get SuciProfile index and write public key
				keyIndex, err1 := strconv.Atoi(suciPart[suci.HNPublicKeyIDPlace])
				if err1 != nil {
					logger.UeauLog.Errorln("Re-Sync Failed UDM Public Key HNPublicKeyIDPlace parse Error")
				} else if keyIndex < 1 {
					logger.UeauLog.Errorf("Re-Sync Failed UDM Public Key HNPublicKeyIDPlace keyIndex[%d] < 1",
						keyIndex)
				} else {
					logger.UeauLog.Errorln("Re-Sync Failed UDM Public Key ",
						udm_context.GetSelf().SuciProfiles[keyIndex-1].PublicKey)
				}
			}
			logger.UeauLog.Errorln("MACS ", macS)
			logger.UeauLog.Errorln("Auts[6:] ", Auts[6:])
			logger.UeauLog.Errorln("Sqn ", SQNms)
			problemDetails = &models.ProblemDetails{
				Status: http.StatusForbidden,
				Cause:  "modification is rejected",
			}
			return nil, problemDetails
		}
	}

	// increment sqn
	bigSQN := big.NewInt(0)
	sqn, err = hex.DecodeString(sqnStr)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusForbidden,
			Cause:  authenticationRejected,
			Detail: err.Error(),
		}

		logger.UeauLog.Errorln("err:", err)
		return nil, problemDetails
	}

	bigSQN.SetString(sqnStr, 16)

	bigInc := big.NewInt(1)
	bigSQN = bigInc.Add(bigSQN, bigInc)

	SQNheStr := fmt.Sprintf("%x", bigSQN)
	SQNheStr = s.strictHex(SQNheStr, 12)
	patchItemArray := []models.PatchItem{
		{
			Op:    models.PatchOperation_REPLACE,
			Path:  "/sequenceNumber",
			Value: SQNheStr,
		},
	}

	var rsp *http.Response

	rsp, err = client.AuthenticationDataDocumentApi.ModifyAuthentication(
		ctx, supi, patchItemArray)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusForbidden,
			Cause:  "modification is rejected ",
			Detail: err.Error(),
		}

		logger.UeauLog.Errorln("update sqn error:", err)
		return nil, problemDetails
	}
	defer func() {
		if rspCloseErr := rsp.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("ModifyAuthentication response body cannot close: %+v", rspCloseErr)
		}
	}()

	// Run milenage
	macA, macS := make([]byte, 8), make([]byte, 8)
	CK, IK := make([]byte, 16), make([]byte, 16)
	RES := make([]byte, 8)
	AK, AKstar := make([]byte, 6), make([]byte, 6)

	// Generate macA, macS
	err = milenage.F1(opc, k, RAND, sqn, AMF, macA, macS)
	if err != nil {
		logger.UeauLog.Errorln("milenage F1 err:", err)
	}

	// Generate RES, CK, IK, AK, AKstar
	// RES == XRES (expected RES) for server
	err = milenage.F2345(opc, k, RAND, RES, CK, IK, AK, AKstar)
	if err != nil {
		logger.UeauLog.Errorln("milenage F2345 err:", err)
	}
	logger.UeauLog.Tracef("milenage RES=[%s]", hex.EncodeToString(RES))

	// Generate AUTN
	logger.UeauLog.Tracef("SQN=[%x], AK=[%x]", sqn, AK)
	logger.UeauLog.Tracef("AMF=[%x], macA=[%x]", AMF, macA)
	SQNxorAK := make([]byte, 6)
	for i := 0; i < len(sqn); i++ {
		SQNxorAK[i] = sqn[i] ^ AK[i]
	}
	logger.UeauLog.Tracef("SQN xor AK=[%x]", SQNxorAK)
	AUTN := append(append(SQNxorAK, AMF...), macA...)
	logger.UeauLog.Tracef("AUTN=[%x]", AUTN)

	var av models.AuthenticationVector
	if authSubs.AuthenticationMethod == models.AuthMethod__5_G_AKA {
		response.AuthType = models.AuthType__5_G_AKA

		// derive XRES*
		key := append(CK, IK...)
		FC := ueauth.FC_FOR_RES_STAR_XRES_STAR_DERIVATION
		P0 := []byte(authInfoRequest.ServingNetworkName)
		P1 := RAND
		P2 := RES

		kdfValForXresStar, err := ueauth.GetKDFValue(
			key, FC, P0, ueauth.KDFLen(P0), P1, ueauth.KDFLen(P1), P2, ueauth.KDFLen(P2))
		if err != nil {
			logger.UeauLog.Errorf("Get kdfValForXresStar err: %+v", err)
		}
		xresStar := kdfValForXresStar[len(kdfValForXresStar)/2:]
		logger.UeauLog.Tracef("xresStar=[%x]", xresStar)

		// derive Kausf
		FC = ueauth.FC_FOR_KAUSF_DERIVATION
		P0 = []byte(authInfoRequest.ServingNetworkName)
		P1 = SQNxorAK
		kdfValForKausf, err := ueauth.GetKDFValue(key, FC, P0, ueauth.KDFLen(P0), P1, ueauth.KDFLen(P1))
		if err != nil {
			logger.UeauLog.Errorf("Get kdfValForKausf err: %+v", err)
		}
		logger.UeauLog.Tracef("Kausf=[%x]", kdfValForKausf)

		// Fill in rand, xresStar, autn, kausf
		av.Rand = hex.EncodeToString(RAND)
		av.XresStar = hex.EncodeToString(xresStar)
		av.Autn = hex.EncodeToString(AUTN)
		av.Kausf = hex.EncodeToString(kdfValForKausf)
		av.AvType = models.AvType__5_G_HE_AKA
	} else { // EAP-AKA'
		response.AuthType = models.AuthType_EAP_AKA_PRIME

		// derive CK' and IK'
		key := append(CK, IK...)
		FC := ueauth.FC_FOR_CK_PRIME_IK_PRIME_DERIVATION
		P0 := []byte(authInfoRequest.ServingNetworkName)
		P1 := SQNxorAK
		kdfVal, err := ueauth.GetKDFValue(key, FC, P0, ueauth.KDFLen(P0), P1, ueauth.KDFLen(P1))
		if err != nil {
			logger.UeauLog.Errorf("Get kdfVal err: %+v", err)
		}
		logger.UeauLog.Tracef("kdfVal=[%x] (len=%d)", kdfVal, len(kdfVal))

		// For TS 35.208 test set 19 & RFC 5448 test vector 1
		// CK': 0093 962d 0dd8 4aa5 684b 045c 9edf fa04
		// IK': ccfc 230c a74f cc96 c0a5 d611 64f5 a76

		ckPrime := kdfVal[:len(kdfVal)/2]
		ikPrime := kdfVal[len(kdfVal)/2:]
		logger.UeauLog.Tracef("ckPrime=[%x], kPrime=[%x]", ckPrime, ikPrime)

		// Fill in rand, xres, autn, ckPrime, ikPrime
		av.Rand = hex.EncodeToString(RAND)
		av.Xres = hex.EncodeToString(RES)
		av.Autn = hex.EncodeToString(AUTN)
		av.CkPrime = hex.EncodeToString(ckPrime)
		av.IkPrime = hex.EncodeToString(ikPrime)
		av.AvType = models.AvType_EAP_AKA_PRIME
	}

	response.AuthenticationVector = &av
	response.Supi = supi
	return response, nil
}

func (s *nudrService) UpdateProcedure(updateRequest models.PpData, gpsi string) (problemDetails *models.ProblemDetails) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return pd
	}
	clientAPI, err := s.createUDMClientToUDR(gpsi)
	if err != nil {
		return openapi.ProblemDetailsSystemFailure(err.Error())
	}
	res, err := clientAPI.ProvisionedParameterDataDocumentApi.ModifyPpData(ctx, gpsi, nil)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: int32(res.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		return problemDetails
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.PpLog.Errorf("ModifyPpData response body cannot close: %+v", rspCloseErr)
		}
	}()
	return nil
}

func (s *nudrService) GetAmDataProcedure(supi string, plmnID string, supportedFeatures string) (
	response *models.AccessAndMobilitySubscriptionData, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	var queryAmDataParamOpts Nudr_DataRepository.QueryAmDataParamOpts
	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	clientAPI, err := s.createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	accessAndMobilitySubscriptionDataResp, res, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.
		QueryAmData(ctx, supi, plmnID, &queryAmDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Errorf(err.Error())
		} else if err.Error() != res.Status {
			logger.SdmLog.Errorf("Response State: %+v", err.Error())
		} else {
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryAmData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
		}
		udmUe.SetAMSubsriptionData(&accessAndMobilitySubscriptionDataResp)
		return &accessAndMobilitySubscriptionDataResp, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		return nil, problemDetails
	}
}

func (s *nudrService) GetIdTranslationResultProcedure(gpsi string) (response *models.IdTranslationResult,
	problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	var idTranslationResult models.IdTranslationResult
	var getIdentityDataParamOpts Nudr_DataRepository.GetIdentityDataParamOpts

	clientAPI, err := s.createUDMClientToUDR(gpsi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	idTranslationResultResp, res, err := clientAPI.QueryIdentityDataBySUPIOrGPSIDocumentApi.GetIdentityData(
		ctx, gpsi, &getIdentityDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Errorf(err.Error())
		} else if err.Error() != res.Status {
			logger.SdmLog.Errorf("Response State: %+v", err.Error())
		} else {
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}

			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("GetIdentityData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		if idList := idTranslationResultResp; idList.SupiList != nil {
			// GetCorrespondingSupi get corresponding Supi(here IMSI) matching the given Gpsi from the queried SUPI list from UDR
			idTranslationResult.Supi = udm_context.GetCorrespondingSupi(idList)
			idTranslationResult.Gpsi = gpsi

			return &idTranslationResult, nil
		} else {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "USER_NOT_FOUND",
			}

			return nil, problemDetails
		}
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}

		return nil, problemDetails
	}
}

func (s *nudrService) GetSupiProcedure(supi string, plmnID string, dataSetNames []string, supportedFeatures string) (
	response *models.SubscriptionDataSets, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	if len(dataSetNames) < 2 {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "BAD_REQUEST",
			Detail: "datasetNames must have at least 2 elements",
		}
		return nil, problemDetails
	}

	clientAPI, err := s.createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var subscriptionDataSets, subsDataSetBody models.SubscriptionDataSets
	var ueContextInSmfDataResp models.UeContextInSmfData
	pduSessionMap := make(map[string]models.PduSession)
	var pgwInfoArray []models.PgwInfo

	var queryAmDataParamOpts Nudr_DataRepository.QueryAmDataParamOpts
	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	var querySmfSelectDataParamOpts Nudr_DataRepository.QuerySmfSelectDataParamOpts
	var queryTraceDataParamOpts Nudr_DataRepository.QueryTraceDataParamOpts
	var querySmDataParamOpts Nudr_DataRepository.QuerySmDataParamOpts

	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	querySmfSelectDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	udm_context.GetSelf().CreateSubsDataSetsForUe(supi, subsDataSetBody)

	if s.containDataSetName(dataSetNames, string(models.DataSetName_AM)) {
		var body models.AccessAndMobilitySubscriptionData
		udm_context.GetSelf().CreateAccessMobilitySubsDataForUe(supi, body)

		amData, res, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.QueryAmData(
			ctx, supi, plmnID, &queryAmDataParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorf(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorf("Response State: %+v", err.Error())
			} else {
				problemDetails = &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}

				return nil, problemDetails
			}
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QueryAmData response body cannot close: %+v", rspCloseErr)
			}
		}()
		if res.StatusCode == http.StatusOK {
			udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.GetSelf().NewUdmUe(supi)
			}
			udmUe.SetAMSubsriptionData(&amData)
			subscriptionDataSets.AmData = &amData
		} else {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "DATA_NOT_FOUND",
			}

			return nil, problemDetails
		}
	}

	if s.containDataSetName(dataSetNames, string(models.DataSetName_SMF_SEL)) {
		var smfSelSubsbody models.SmfSelectionSubscriptionData
		udm_context.GetSelf().CreateSmfSelectionSubsDataforUe(supi, smfSelSubsbody)

		smfSelData, res, err := clientAPI.SMFSelectionSubscriptionDataDocumentApi.QuerySmfSelectData(ctx,
			supi, plmnID, &querySmfSelectDataParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorln(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorln(err.Error())
			} else {
				problemDetails = &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}

				return nil, problemDetails
			}
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QuerySmfSelectData response body cannot close: %+v", rspCloseErr)
			}
		}()
		if res.StatusCode == http.StatusOK {
			udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.GetSelf().NewUdmUe(supi)
			}
			udmUe.SetSmfSelectionSubsData(&smfSelData)
			subscriptionDataSets.SmfSelData = &smfSelData
		} else {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "DATA_NOT_FOUND",
			}

			return nil, problemDetails
		}
	}

	if s.containDataSetName(dataSetNames, string(models.DataSetName_UEC_SMF)) {
		var UeContextInSmfbody models.UeContextInSmfData
		var querySmfRegListParamOpts Nudr_DataRepository.QuerySmfRegListParamOpts
		querySmfRegListParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
		udm_context.GetSelf().CreateUeContextInSmfDataforUe(supi, UeContextInSmfbody)

		pdusess, res, err := clientAPI.SMFRegistrationsCollectionApi.QuerySmfRegList(
			ctx, supi, &querySmfRegListParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorf(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorf("Response State: %+v", err.Error())
			} else {
				problemDetails = &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}

				return nil, problemDetails
			}
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QuerySmfRegList response body cannot close: %+v", rspCloseErr)
			}
		}()

		for _, element := range pdusess {
			var pduSession models.PduSession
			pduSession.Dnn = element.Dnn
			pduSession.SmfInstanceId = element.SmfInstanceId
			pduSession.PlmnId = element.PlmnId
			pduSessionMap[strconv.Itoa(int(element.PduSessionId))] = pduSession
		}
		ueContextInSmfDataResp.PduSessions = pduSessionMap

		for _, element := range pdusess {
			var pgwInfo models.PgwInfo
			pgwInfo.Dnn = element.Dnn
			pgwInfo.PgwFqdn = element.PgwFqdn
			pgwInfo.PlmnId = element.PlmnId
			pgwInfoArray = append(pgwInfoArray, pgwInfo)
		}
		ueContextInSmfDataResp.PgwInfo = pgwInfoArray

		if res.StatusCode == http.StatusOK {
			udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.GetSelf().NewUdmUe(supi)
			}
			udmUe.UeCtxtInSmfData = &ueContextInSmfDataResp
			subscriptionDataSets.UecSmfData = &ueContextInSmfDataResp
		} else {
			var problemDetails models.ProblemDetails
			problemDetails.Cause = "DATA_NOT_FOUND"
			logger.SdmLog.Errorf(problemDetails.Cause)
		}
	}

	// TODO: UE Context in SMSF Data
	// if containDataSetName(dataSetNames, string(models.DataSetName_UEC_SMSF)) {
	// }

	// TODO: SMS Subscription Data
	// if containDataSetName(dataSetNames, string(models.DataSetName_SMS_SUB)) {
	// }

	if s.containDataSetName(dataSetNames, string(models.DataSetName_SM)) {
		sessionManagementSubscriptionData, res, err := clientAPI.SessionManagementSubscriptionDataApi.
			QuerySmData(ctx, supi, plmnID, &querySmDataParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorf(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorf("Response State: %+v", err.Error())
			} else {
				problemDetails = &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}

				return nil, problemDetails
			}
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QuerySmData response body cannot close: %+v", rspCloseErr)
			}
		}()
		if res.StatusCode == http.StatusOK {
			udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.GetSelf().NewUdmUe(supi)
			}
			smData, _, _, _ := udm_context.GetSelf().ManageSmData(sessionManagementSubscriptionData, "", "")
			udmUe.SetSMSubsData(smData)
			subscriptionDataSets.SmData = sessionManagementSubscriptionData
		} else {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "DATA_NOT_FOUND",
			}

			return nil, problemDetails
		}
	}

	if s.containDataSetName(dataSetNames, string(models.DataSetName_TRACE)) {
		var TraceDatabody models.TraceData
		udm_context.GetSelf().CreateTraceDataforUe(supi, TraceDatabody)

		traceData, res, err := clientAPI.TraceDataDocumentApi.QueryTraceData(
			ctx, supi, plmnID, &queryTraceDataParamOpts)
		if err != nil {
			if res == nil {
				logger.SdmLog.Errorf(err.Error())
			} else if err.Error() != res.Status {
				logger.SdmLog.Errorf("Response State: %+v", err.Error())
			} else {
				problemDetails = &models.ProblemDetails{
					Status: int32(res.StatusCode),
					Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
					Detail: err.Error(),
				}
			}
			return nil, problemDetails
		}
		defer func() {
			if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
				logger.SdmLog.Errorf("QueryTraceData response body cannot close: %+v", rspCloseErr)
			}
		}()
		if res.StatusCode == http.StatusOK {
			udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
			if !ok {
				udmUe = udm_context.GetSelf().NewUdmUe(supi)
			}
			udmUe.TraceData = &traceData
			udmUe.TraceDataResponse.TraceData = &traceData
			subscriptionDataSets.TraceData = &traceData
		} else {
			problemDetails = &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "DATA_NOT_FOUND",
			}

			return nil, problemDetails
		}
	}

	// TODO: SMS Management Subscription Data
	// if containDataSetName(dataSetNames, string(models.DataSetName_SMS_MNG)) {
	// }

	return &subscriptionDataSets, nil
}

func (s *nudrService) GetSharedDataProcedure(sharedDataIds []string, supportedFeatures string) (
	response []models.SharedData, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	clientAPI, err := s.createUDMClientToUDR("")
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var getSharedDataParamOpts Nudr_DataRepository.GetSharedDataParamOpts
	getSharedDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	sharedDataResp, res, err := clientAPI.RetrievalOfSharedDataApi.GetSharedData(ctx, sharedDataIds,
		&getSharedDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			logger.SdmLog.Warnln(err)
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}

			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("GetShareData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udm_context.GetSelf().SharedSubsDataMap = udm_context.MappingSharedData(sharedDataResp)
		sharedData := udm_context.ObtainRequiredSharedData(sharedDataIds, sharedDataResp)
		return sharedData, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		return nil, problemDetails
	}
}

func (s *nudrService) GetSmDataProcedure(supi string, plmnID string, Dnn string, Snssai string, supportedFeatures string) (
	response interface{}, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	logger.SdmLog.Infof("getSmDataProcedure: SUPI[%s] PLMNID[%s] DNN[%s] SNssai[%s]", supi, plmnID, Dnn, Snssai)

	clientAPI, err := s.createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var querySmDataParamOpts Nudr_DataRepository.QuerySmDataParamOpts
	querySmDataParamOpts.SingleNssai = optional.NewInterface(Snssai)

	sessionManagementSubscriptionDataResp, res, err := clientAPI.SessionManagementSubscriptionDataApi.
		QuerySmData(ctx, supi, plmnID, &querySmDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			logger.SdmLog.Warnln(err)
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}

			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QuerySmData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
		}
		smData, snssaikey, AllDnnConfigsbyDnn, AllDnns := udm_context.GetSelf().ManageSmData(
			sessionManagementSubscriptionDataResp, Snssai, Dnn)
		udmUe.SetSMSubsData(smData)

		rspSMSubDataList := make([]models.SessionManagementSubscriptionData, 0, 4)

		udmUe.SmSubsDataLock.RLock()
		for _, eachSMSubData := range udmUe.SessionManagementSubsData {
			rspSMSubDataList = append(rspSMSubDataList, eachSMSubData)
		}
		udmUe.SmSubsDataLock.RUnlock()

		switch {
		case Snssai == "" && Dnn == "":
			return AllDnns, nil
		case Snssai != "" && Dnn == "":
			udmUe.SmSubsDataLock.RLock()
			defer udmUe.SmSubsDataLock.RUnlock()
			return udmUe.SessionManagementSubsData[snssaikey].DnnConfigurations, nil
		case Snssai == "" && Dnn != "":
			return AllDnnConfigsbyDnn, nil
		case Snssai != "" && Dnn != "":
			return rspSMSubDataList, nil
		default:
			udmUe.SmSubsDataLock.RLock()
			defer udmUe.SmSubsDataLock.RUnlock()
			return udmUe.SessionManagementSubsData, nil
		}
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}

		return nil, problemDetails
	}
}

func (s *nudrService) GetNssaiProcedure(supi string, plmnID string, supportedFeatures string) (
	*models.Nssai, *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	var queryAmDataParamOpts Nudr_DataRepository.QueryAmDataParamOpts
	queryAmDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	var nssaiResp models.Nssai
	clientAPI, err := s.createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	accessAndMobilitySubscriptionDataResp, res, err := clientAPI.AccessAndMobilitySubscriptionDataDocumentApi.
		QueryAmData(ctx, supi, plmnID, &queryAmDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			logger.SdmLog.Warnln(err)
			problemDetails := &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}

			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryAmData response body cannot close: %+v", rspCloseErr)
		}
	}()

	nssaiResp = *accessAndMobilitySubscriptionDataResp.Nssai

	if res.StatusCode == http.StatusOK {
		udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
		}
		udmUe.Nssai = &nssaiResp
		return udmUe.Nssai, nil
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		return nil, problemDetails
	}
}

func (s *nudrService) GetSmfSelectDataProcedure(supi string, plmnID string, supportedFeatures string) (
	response *models.SmfSelectionSubscriptionData, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	var querySmfSelectDataParamOpts Nudr_DataRepository.QuerySmfSelectDataParamOpts
	querySmfSelectDataParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)
	var body models.SmfSelectionSubscriptionData

	clientAPI, err := s.createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	udm_context.GetSelf().CreateSmfSelectionSubsDataforUe(supi, body)

	smfSelectionSubscriptionDataResp, res, err := clientAPI.SMFSelectionSubscriptionDataDocumentApi.
		QuerySmfSelectData(ctx, supi, plmnID, &querySmfSelectDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			logger.SdmLog.Warnln(err)
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			return nil, problemDetails
		}
		return nil, problemDetails
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QuerySmfSelectData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
		}
		udmUe.SetSmfSelectionSubsData(&smfSelectionSubscriptionDataResp)
		return udmUe.SmfSelSubsData, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		return nil, problemDetails
	}
}

func (s *nudrService) SubscribeProcedure(sdmSubscription *models.SdmSubscription, supi string) (
	header http.Header, response *models.SdmSubscription, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, nil, pd
	}
	clientAPI, err := s.createUDMClientToUDR(supi)
	if err != nil {
		return nil, nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	sdmSubscriptionResp, res, err := clientAPI.SDMSubscriptionsCollectionApi.CreateSdmSubscriptions(
		ctx, supi, *sdmSubscription)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			logger.SdmLog.Warnln(err)
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			return nil, nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("CreateSdmSubscriptions response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusCreated {
		header = make(http.Header)
		udmUe, _ := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if udmUe == nil {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
		}
		udmUe.CreateSubscriptiontoNotifChange(sdmSubscriptionResp.SubscriptionId, &sdmSubscriptionResp)
		header.Set("Location", udmUe.GetLocationURI2(udm_context.LocationUriSdmSubscription, supi))
		return header, &sdmSubscriptionResp, nil
	} else if res.StatusCode == http.StatusNotFound {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		return nil, nil, problemDetails
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotImplemented,
			Cause:  "UNSUPPORTED_RESOURCE_URI",
		}
		return nil, nil, problemDetails
	}
}

func (s *nudrService) UnsubscribeProcedure(supi string, subscriptionID string) *models.ProblemDetails {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return pd
	}
	clientAPI, err := s.createUDMClientToUDR(supi)
	if err != nil {
		return openapi.ProblemDetailsSystemFailure(err.Error())
	}

	res, err := clientAPI.SDMSubscriptionDocumentApi.RemovesdmSubscriptions(ctx, supi, subscriptionID)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			logger.SdmLog.Warnln(err)
			problemDetails := &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			return problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("RemovesdmSubscriptions response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusNoContent {
		return nil
	} else {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "USER_NOT_FOUND",
		}
		return problemDetails
	}
}

func (s *nudrService) ModifyProcedure(sdmSubsModification *models.SdmSubsModification, supi string, subscriptionID string) (
	response *models.SdmSubscription, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	clientAPI, err := s.createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	sdmSubscription := models.SdmSubscription{}
	body := Nudr_DataRepository.UpdatesdmsubscriptionsParamOpts{
		SdmSubscription: optional.NewInterface(sdmSubscription),
	}

	res, err := clientAPI.SDMSubscriptionDocumentApi.Updatesdmsubscriptions(
		ctx, supi, subscriptionID, &body)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("Updatesdmsubscriptions response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		return &sdmSubscription, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "USER_NOT_FOUND",
		}

		return nil, problemDetails
	}
}

func (s *nudrService) GetTraceDataProcedure(supi string, plmnID string) (
	response *models.TraceData, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	var body models.TraceData
	var queryTraceDataParamOpts Nudr_DataRepository.QueryTraceDataParamOpts

	clientAPI, err := s.createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	udm_context.GetSelf().CreateTraceDataforUe(supi, body)

	traceDataRes, res, err := clientAPI.TraceDataDocumentApi.QueryTraceData(
		ctx, supi, plmnID, &queryTraceDataParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}

			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryTraceData response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
		}
		udmUe.TraceData = &traceDataRes
		udmUe.TraceDataResponse.TraceData = &traceDataRes

		return udmUe.TraceDataResponse.TraceData, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "USER_NOT_FOUND",
		}

		return nil, problemDetails
	}
}

func (s *nudrService) GetUeContextInSmfDataProcedure(supi string, supportedFeatures string) (
	response *models.UeContextInSmfData, problemDetails *models.ProblemDetails,
) {
	var body models.UeContextInSmfData
	var ueContextInSmfData models.UeContextInSmfData
	var pgwInfoArray []models.PgwInfo
	var querySmfRegListParamOpts Nudr_DataRepository.QuerySmfRegListParamOpts
	querySmfRegListParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	clientAPI, err := s.createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	pduSessionMap := make(map[string]models.PduSession)
	udm_context.GetSelf().CreateUeContextInSmfDataforUe(supi, body)

	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}

	pdusess, res, err := clientAPI.SMFRegistrationsCollectionApi.QuerySmfRegList(
		ctx, supi, &querySmfRegListParamOpts)
	if err != nil {
		if res == nil {
			logger.SdmLog.Infoln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Infoln(err)
		} else {
			logger.SdmLog.Infoln(err)
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}

			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QuerySmfRegList response body cannot close: %+v", rspCloseErr)
		}
	}()

	for _, element := range pdusess {
		var pduSession models.PduSession
		pduSession.Dnn = element.Dnn
		pduSession.SmfInstanceId = element.SmfInstanceId
		pduSession.PlmnId = element.PlmnId
		pduSessionMap[strconv.Itoa(int(element.PduSessionId))] = pduSession
	}
	ueContextInSmfData.PduSessions = pduSessionMap

	for _, element := range pdusess {
		var pgwInfo models.PgwInfo
		pgwInfo.Dnn = element.Dnn
		pgwInfo.PgwFqdn = element.PgwFqdn
		pgwInfo.PlmnId = element.PlmnId
		pgwInfoArray = append(pgwInfoArray, pgwInfo)
	}
	ueContextInSmfData.PgwInfo = pgwInfoArray

	if res.StatusCode == http.StatusOK {
		udmUe, ok := udm_context.GetSelf().UdmUeFindBySupi(supi)
		if !ok {
			udmUe = udm_context.GetSelf().NewUdmUe(supi)
		}
		udmUe.UeCtxtInSmfData = &ueContextInSmfData
		return udmUe.UeCtxtInSmfData, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "DATA_NOT_FOUND",
		}
		return nil, problemDetails
	}
}

// ue_context_managemanet_service
func (s *nudrService) GetAmf3gppAccessProcedure(ueID string, supportedFeatures string) (
	response *models.Amf3GppAccessRegistration, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	var queryAmfContext3gppParamOpts Nudr_DataRepository.QueryAmfContext3gppParamOpts
	queryAmfContext3gppParamOpts.SupportedFeatures = optional.NewString(supportedFeatures)

	clientAPI, err := s.createUDMClientToUDR(ueID)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	amf3GppAccessRegistration, resp, err := clientAPI.AMF3GPPAccessRegistrationDocumentApi.
		QueryAmfContext3gpp(ctx, ueID, &queryAmfContext3gppParamOpts)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		return nil, problemDetails
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryAmfContext3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	return &amf3GppAccessRegistration, nil
}

func (s *nudrService) GetAmfNon3gppAccessProcedure(queryAmfContextNon3gppParamOpts Nudr_DataRepository.
	QueryAmfContextNon3gppParamOpts, ueID string) (response *models.AmfNon3GppAccessRegistration,
	problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	clientAPI, err := s.createUDMClientToUDR(ueID)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	amfNon3GppAccessRegistration, resp, err := clientAPI.AMFNon3GPPAccessRegistrationDocumentApi.
		QueryAmfContextNon3gpp(ctx, ueID, &queryAmfContextNon3gppParamOpts)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		return nil, problemDetails
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("QueryAmfContext3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	return &amfNon3GppAccessRegistration, nil
}

func (s *nudrService) RegistrationAmf3gppAccessProcedure(registerRequest models.Amf3GppAccessRegistration, ueID string) (
	header http.Header, response *models.Amf3GppAccessRegistration, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, nil, pd
	}
	// TODO: EPS interworking with N26 is not supported yet in this stage
	var oldAmf3GppAccessRegContext *models.Amf3GppAccessRegistration
	var ue *udm_context.UdmUeContext

	if udm_context.GetSelf().UdmAmf3gppRegContextExists(ueID) {
		ue, _ = udm_context.GetSelf().UdmUeFindBySupi(ueID)
		oldAmf3GppAccessRegContext = ue.Amf3GppAccessRegistration
	}

	udm_context.GetSelf().CreateAmf3gppRegContext(ueID, registerRequest)

	clientAPI, err := s.createUDMClientToUDR(ueID)
	if err != nil {
		return nil, nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var createAmfContext3gppParamOpts Nudr_DataRepository.CreateAmfContext3gppParamOpts
	optInterface := optional.NewInterface(registerRequest)
	createAmfContext3gppParamOpts.Amf3GppAccessRegistration = optInterface
	resp, err := clientAPI.AMF3GPPAccessRegistrationDocumentApi.CreateAmfContext3gpp(ctx,
		ueID, &createAmfContext3gppParamOpts)
	if err != nil {
		logger.UecmLog.Errorln("CreateAmfContext3gpp error : ", err)
		problemDetails = &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		return nil, nil, problemDetails
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.UecmLog.Errorf("CreateAmfContext3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	// TS 23.502 4.2.2.2.2 14d: UDM initiate a Nudm_UECM_DeregistrationNotification to the old AMF
	// corresponding to the same (e.g. 3GPP) access, if one exists
	if oldAmf3GppAccessRegContext != nil {
		if !ue.SameAsStoredGUAMI3gpp(*oldAmf3GppAccessRegContext.Guami) {
			// Based on TS 23.502 4.2.2.2.2, If the serving NF removal reason indicated by the UDM is Initial Registration,
			// the old AMF invokes the Nsmf_PDUSession_ReleaseSMContext (SM Context ID). Thus we give different
			// dereg cause based on registration parameter from serving AMF
			deregReason := models.DeregistrationReason_UE_REGISTRATION_AREA_CHANGE
			if registerRequest.InitialRegistrationInd {
				deregReason = models.DeregistrationReason_UE_INITIAL_REGISTRATION
			}
			deregistData := models.DeregistrationData{
				DeregReason: deregReason,
				AccessType:  models.AccessType__3_GPP_ACCESS,
			}

			go func() {
				logger.UecmLog.Infof("Send DeregNotify to old AMF GUAMI=%v", oldAmf3GppAccessRegContext.Guami)
				pd := callback.SendOnDeregistrationNotification(ueID,
					oldAmf3GppAccessRegContext.DeregCallbackUri,
					deregistData) // Deregistration Notify Triggered
				if pd != nil {
					logger.UecmLog.Errorf("RegistrationAmf3gppAccess: send DeregNotify fail %v", pd)
				}
			}()
		}
		return nil, &registerRequest, nil
	} else {
		header = make(http.Header)
		udmUe, _ := udm_context.GetSelf().UdmUeFindBySupi(ueID)
		header.Set("Location", udmUe.GetLocationURI(udm_context.LocationUriAmf3GppAccessRegistration))
		return header, &registerRequest, nil
	}
}

func (s *nudrService) RegisterAmfNon3gppAccessProcedure(registerRequest models.AmfNon3GppAccessRegistration, ueID string) (
	header http.Header, response *models.AmfNon3GppAccessRegistration, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, nil, pd
	}
	var oldAmfNon3GppAccessRegContext *models.AmfNon3GppAccessRegistration
	if udm_context.GetSelf().UdmAmfNon3gppRegContextExists(ueID) {
		ue, _ := udm_context.GetSelf().UdmUeFindBySupi(ueID)
		oldAmfNon3GppAccessRegContext = ue.AmfNon3GppAccessRegistration
	}

	udm_context.GetSelf().CreateAmfNon3gppRegContext(ueID, registerRequest)

	clientAPI, err := s.createUDMClientToUDR(ueID)
	if err != nil {
		return nil, nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var createAmfContextNon3gppParamOpts Nudr_DataRepository.CreateAmfContextNon3gppParamOpts
	optInterface := optional.NewInterface(registerRequest)
	createAmfContextNon3gppParamOpts.AmfNon3GppAccessRegistration = optInterface

	resp, err := clientAPI.AMFNon3GPPAccessRegistrationDocumentApi.CreateAmfContextNon3gpp(
		ctx, ueID, &createAmfContextNon3gppParamOpts)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		return nil, nil, problemDetails
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.UecmLog.Errorf("CreateAmfContext3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	// TS 23.502 4.2.2.2.2 14d: UDM initiate a Nudm_UECM_DeregistrationNotification to the old AMF
	// corresponding to the same (e.g. 3GPP) access, if one exists
	if oldAmfNon3GppAccessRegContext != nil {
		deregistData := models.DeregistrationData{
			DeregReason: models.DeregistrationReason_UE_INITIAL_REGISTRATION,
			AccessType:  models.AccessType_NON_3_GPP_ACCESS,
		}
		callback.SendOnDeregistrationNotification(ueID, oldAmfNon3GppAccessRegContext.DeregCallbackUri,
			deregistData) // Deregistration Notify Triggered

		return nil, nil, nil
	} else {
		header = make(http.Header)
		udmUe, _ := udm_context.GetSelf().UdmUeFindBySupi(ueID)
		header.Set("Location", udmUe.GetLocationURI(udm_context.LocationUriAmfNon3GppAccessRegistration))
		return header, &registerRequest, nil
	}
}

func (s *nudrService) UpdateAmf3gppAccessProcedure(request models.Amf3GppAccessRegistrationModification, ueID string) (
	problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return pd
	}
	var patchItemReqArray []models.PatchItem
	currentContext := udm_context.GetSelf().GetAmf3gppRegContext(ueID)
	if currentContext == nil {
		logger.UecmLog.Errorln("[UpdateAmf3gppAccess] Empty Amf3gppRegContext")
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "CONTEXT_NOT_FOUND",
		}
		return problemDetails
	}

	if request.Guami != nil {
		udmUe, _ := udm_context.GetSelf().UdmUeFindBySupi(ueID)
		if udmUe.SameAsStoredGUAMI3gpp(*request.Guami) { // deregistration
			logger.UecmLog.Infoln("UpdateAmf3gppAccess - deregistration")
			request.PurgeFlag = true
		} else {
			logger.UecmLog.Errorln("INVALID_GUAMI")
			problemDetails = &models.ProblemDetails{
				Status: http.StatusForbidden,
				Cause:  "INVALID_GUAMI",
			}
			return problemDetails
		}

		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "guami"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = *request.Guami
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.PurgeFlag {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "purgeFlag"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.PurgeFlag
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.Pei != "" {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "pei"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.Pei
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.ImsVoPs != "" {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "imsVoPs"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.ImsVoPs
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.BackupAmfInfo != nil {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "backupAmfInfo"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.BackupAmfInfo
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	clientAPI, err := s.createUDMClientToUDR(ueID)
	if err != nil {
		return openapi.ProblemDetailsSystemFailure(err.Error())
	}

	resp, err := clientAPI.AMF3GPPAccessRegistrationDocumentApi.AmfContext3gpp(ctx, ueID,
		patchItemReqArray)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}

		return problemDetails
	}

	if request.PurgeFlag {
		udmUe, _ := udm_context.GetSelf().UdmUeFindBySupi(ueID)
		udmUe.Amf3GppAccessRegistration = nil
	}

	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.UecmLog.Errorf("AmfContext3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	return nil
}

func (s *nudrService) UpdateAmfNon3gppAccessProcedure(request models.AmfNon3GppAccessRegistrationModification, ueID string) (
	problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return pd
	}
	var patchItemReqArray []models.PatchItem
	currentContext := udm_context.GetSelf().GetAmfNon3gppRegContext(ueID)
	if currentContext == nil {
		logger.UecmLog.Errorln("[UpdateAmfNon3gppAccess] Empty AmfNon3gppRegContext")
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "CONTEXT_NOT_FOUND",
		}
		return problemDetails
	}

	if request.Guami != nil {
		udmUe, _ := udm_context.GetSelf().UdmUeFindBySupi(ueID)
		if udmUe.SameAsStoredGUAMINon3gpp(*request.Guami) { // deregistration
			logger.UecmLog.Infoln("UpdateAmfNon3gppAccess - deregistration")
			request.PurgeFlag = true
		} else {
			logger.UecmLog.Errorln("INVALID_GUAMI")
			problemDetails = &models.ProblemDetails{
				Status: http.StatusForbidden,
				Cause:  "INVALID_GUAMI",
			}
		}

		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "guami"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = *request.Guami
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.PurgeFlag {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "purgeFlag"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.PurgeFlag
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.Pei != "" {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "pei"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.Pei
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.ImsVoPs != "" {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "imsVoPs"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.ImsVoPs
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	if request.BackupAmfInfo != nil {
		var patchItemTmp models.PatchItem
		patchItemTmp.Path = "/" + "backupAmfInfo"
		patchItemTmp.Op = models.PatchOperation_REPLACE
		patchItemTmp.Value = request.BackupAmfInfo
		patchItemReqArray = append(patchItemReqArray, patchItemTmp)
	}

	clientAPI, err := s.createUDMClientToUDR(ueID)
	if err != nil {
		return openapi.ProblemDetailsSystemFailure(err.Error())
	}

	resp, err := clientAPI.AMFNon3GPPAccessRegistrationDocumentApi.AmfContextNon3gpp(ctx,
		ueID, patchItemReqArray)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		return problemDetails
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.UecmLog.Errorf("AmfContextNon3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	return problemDetails
}

func (s *nudrService) DeregistrationSmfRegistrationsProcedure(ueID string, pduSessionID string) (problemDetails *models.ProblemDetails) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return pd
	}
	clientAPI, err := s.createUDMClientToUDR(ueID)
	if err != nil {
		return openapi.ProblemDetailsSystemFailure(err.Error())
	}

	resp, err := clientAPI.SMFRegistrationDocumentApi.DeleteSmfContext(ctx, ueID, pduSessionID)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		return problemDetails
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.UecmLog.Errorf("DeleteSmfContext response body cannot close: %+v", rspCloseErr)
		}
	}()

	return nil
}

func (s *nudrService) RegistrationSmfRegistrationsProcedure(request *models.SmfRegistration, ueID string, pduSessionID string) (
	header http.Header, response *models.SmfRegistration, problemDetails *models.ProblemDetails,
) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, nil, pd
	}
	contextExisted := false
	udm_context.GetSelf().CreateSmfRegContext(ueID, pduSessionID)
	if !udm_context.GetSelf().UdmSmfRegContextNotExists(ueID) {
		contextExisted = true
	}

	pduID64, err := strconv.ParseInt(pduSessionID, 10, 32)
	if err != nil {
		logger.UecmLog.Errorln(err.Error())
	}
	pduID32 := int32(pduID64)

	var createSmfContextNon3gppParamOpts Nudr_DataRepository.CreateSmfContextNon3gppParamOpts
	optInterface := optional.NewInterface(*request)
	createSmfContextNon3gppParamOpts.SmfRegistration = optInterface

	clientAPI, err := s.createUDMClientToUDR(ueID)
	if err != nil {
		return nil, nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	resp, err := clientAPI.SMFRegistrationDocumentApi.CreateSmfContextNon3gpp(ctx, ueID,
		pduID32, &createSmfContextNon3gppParamOpts)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Status: int32(resp.StatusCode),
			Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
			Detail: err.Error(),
		}
		return nil, nil, problemDetails
	}
	defer func() {
		if rspCloseErr := resp.Body.Close(); rspCloseErr != nil {
			logger.UecmLog.Errorf("CreateSmfContextNon3gpp response body cannot close: %+v", rspCloseErr)
		}
	}()

	if contextExisted {
		return nil, nil, nil
	} else {
		header = make(http.Header)
		udmUe, _ := udm_context.GetSelf().UdmUeFindBySupi(ueID)
		header.Set("Location", udmUe.GetLocationURI(udm_context.LocationUriSmfRegistration))
		return header, request, nil
	}
}

// TS 29.503 5.2.2.7.3
// Modification of a subscription to notifications of shared data change
func (s *nudrService) ModifyForSharedDataProcedure(sdmSubsModification *models.SdmSubsModification, supi string,
	subscriptionID string,
) (response *models.SdmSubscription, problemDetails *models.ProblemDetails) {
	ctx, pd, err := udm_context.GetSelf().GetTokenCtx(models.ServiceName_NUDR_DR, models.NfType_UDR)
	if err != nil {
		return nil, pd
	}
	clientAPI, err := s.createUDMClientToUDR(supi)
	if err != nil {
		return nil, openapi.ProblemDetailsSystemFailure(err.Error())
	}

	var sdmSubscription models.SdmSubscription
	sdmSubs := models.SdmSubscription{}
	body := Nudr_DataRepository.UpdatesdmsubscriptionsParamOpts{
		SdmSubscription: optional.NewInterface(sdmSubs),
	}

	res, err := clientAPI.SDMSubscriptionDocumentApi.Updatesdmsubscriptions(
		ctx, supi, subscriptionID, &body)
	if err != nil {
		if res == nil {
			logger.SdmLog.Warnln(err)
		} else if err.Error() != res.Status {
			logger.SdmLog.Warnln(err)
		} else {
			problemDetails = &models.ProblemDetails{
				Status: int32(res.StatusCode),
				Cause:  err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails).Cause,
				Detail: err.Error(),
			}
			return nil, problemDetails
		}
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.SdmLog.Errorf("Updatesdmsubscriptions response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res.StatusCode == http.StatusOK {
		return &sdmSubscription, nil
	} else {
		problemDetails = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Cause:  "USER_NOT_FOUND",
		}

		return nil, problemDetails
	}
}
