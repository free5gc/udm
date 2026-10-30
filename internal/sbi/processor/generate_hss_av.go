package processor

// Nudm_UEAU GetHssAv for IMS-AKA (TS 29.503 clause 5.4.2.4,
// POST /{supi}/hss-security-information/ims-aka/generate-av): the operation an
// HSS uses to obtain IMS-AKA authentication vectors {RAND, XRES, AUTN, CK, IK}
// from the UDM in the HSS/UDM coexistence architecture of TS 23.632.
//
// The vectors are UMTS AVs (TS 33.102 clause 6.3.2) built as
// GenerateAuthDataProcedure builds 5G ones, except that:
//   - the AMF separation bit is cleared (it is set only for E-UTRAN/5G AVs);
//   - there are no 5G derivations (XRES*, K_AUSF): XRES, CK and IK are returned;
//   - the SQN of the last vector + 1 is written back to the UDR before
//     answering, so consecutive requests get fresh vectors without a result
//     confirmation (there is no AUSF in this procedure).

import (
	cryptoRand "crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	Nudr_DataRepository "github.com/free5gc/openapi/udr/DR"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/util/metrics/sbi"
	"github.com/free5gc/util/milenage"
)

const maxHssVectors = 5 // HssAuthenticationVectors maxItems (TS 29.503)

func hssAvReject(c *gin.Context, status int, cause, detail string) {
	pd := &models.ProblemDetails{Status: int32(status), Cause: cause, Detail: detail}
	c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
	c.JSON(status, pd)
}

// GenerateHssAvProcedure serves GetHssAv for hssAuthType "ims-aka".
func (p *Processor) GenerateHssAvProcedure(
	c *gin.Context,
	req models.Udm_UEAU_HssAuthenticationInfoRequest,
	supi string,
	hssAuthType string,
) {
	if hssAuthType != string(models.Udm_UEAU_HssAuthTypeInUri_IMS_AKA) {
		hssAvReject(c, http.StatusNotImplemented, "UNSUPPORTED_AUTH_TYPE",
			"only ims-aka is implemented, got "+hssAuthType)
		return
	}
	n := int(req.NumOfRequestedVectors)
	if n < 1 {
		n = 1
	}
	if n > maxHssVectors {
		n = maxHssVectors
	}

	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	client, err := p.Consumer().CreateUDMClientToUDR(supi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	var query Nudr_DataRepository.QueryAuthSubsDataRequest
	query.UeId = &supi
	authSubs, err := client.AuthenticationDataDocumentApi.QueryAuthSubsData(ctx, &query)
	if err != nil {
		if apiError, ok := err.(openapi.GenericOpenAPIError); ok {
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, http.StatusText(apiError.ErrorStatus))
			c.Data(apiError.ErrorStatus, "application/json", apiError.RawBody)
			return
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	if authSubs == nil || authSubs.Udr_DR_AuthenticationSubscription == nil {
		hssAvReject(c, http.StatusInternalServerError, "SYSTEM_FAILURE",
			"UDR returned an empty authentication subscription")
		return
	}
	subs := authSubs.Udr_DR_AuthenticationSubscription

	k, errK := hex.DecodeString(subs.EncPermanentKey)
	opc, errO := hex.DecodeString(subs.EncOpcKey)
	if errK != nil || errO != nil || len(k) != keyStrLen/2 || len(opc) != opcStrLen/2 {
		hssAvReject(c, http.StatusForbidden, authenticationRejected, "K or OPc missing or malformed")
		return
	}
	if subs.SequenceNumber == nil {
		hssAvReject(c, http.StatusForbidden, authenticationRejected, "SequenceNumber is missing")
		return
	}
	sqnStr := p.strictHex(subs.SequenceNumber.Sqn, 12)
	amf, err := hex.DecodeString(p.strictHex(subs.AuthenticationManagementField, 4))
	if err != nil {
		hssAvReject(c, http.StatusForbidden, authenticationRejected, err.Error())
		return
	}
	// TS 33.102 Annex H: the separation bit (bit 0 of AMF) is set only for AVs
	// used in E-UTRAN/5G; IMS-AKA uses UMTS AVs.
	amf[0] &^= 0x80

	// Re-synchronisation (TS 33.102 clause 6.3.5), as in GenerateAuthDataProcedure.
	if rs := req.ResynchronizationInfo; rs != nil {
		auts, errA := hex.DecodeString(rs.Auts)
		rnd, errR := hex.DecodeString(rs.Rand)
		if errA != nil || errR != nil {
			hssAvReject(c, http.StatusForbidden, authenticationRejected, "malformed resynchronizationInfo")
			return
		}
		sqnMS, macS, errS := p.aucSQN(opc, k, auts, rnd)
		if errS != nil || subtle.ConstantTimeCompare(macS, auts[6:]) != 1 {
			hssAvReject(c, http.StatusForbidden, authenticationRejected, "re-synchronisation MAC-S failure")
			return
		}
		next := new(big.Int).SetBytes(sqnMS)
		next.Add(next, big.NewInt(ind+1))
		next.Mod(next, big.NewInt(SqnMAx))
		sqnStr = p.strictHex(fmt.Sprintf("%x", next), 12)
		logger.UeauLog.Infof("GetHssAv re-synchronisation for %s, SQN -> %s", supi, sqnStr)
	}

	// Vectors use SQN, SQN+1, ...; SQN+n is stored before answering.
	base := new(big.Int)
	base.SetString(sqnStr, 16)
	stored := new(big.Int).Add(base, big.NewInt(int64(n)))
	stored.Mod(stored, big.NewInt(SqnMAx))
	patch := []models.PatchItem{{
		Op:    models.PatchOperation_REPLACE,
		Path:  "/sequenceNumber",
		Value: models.Udr_DR_SequenceNumber{Sqn: p.strictHex(fmt.Sprintf("%x", stored), 12)},
	}}
	var modify Nudr_DataRepository.ModifyAuthenticationSubscriptionRequest
	modify.UeId = &supi
	modify.RequestBody = patch
	if _, err = client.AuthenticationSubscriptionDocumentApi.ModifyAuthenticationSubscription(ctx, &modify); err != nil {
		hssAvReject(c, http.StatusInternalServerError, "SYSTEM_FAILURE", "update sqn: "+err.Error())
		return
	}

	// Udm_UEAU_HssAuthenticationVectors is generated as an empty struct (the
	// schema is a oneOf), so the array is built from the typed AV items.
	vectors := make([]models.Udm_UEAU_AvImsGbaEapAka, 0, n)
	for i := 0; i < n; i++ {
		s := new(big.Int).Add(base, big.NewInt(int64(i)))
		s.Mod(s, big.NewInt(SqnMAx))
		sqn, _ := hex.DecodeString(p.strictHex(fmt.Sprintf("%x", s), 12))
		rnd := make([]byte, 16)
		if _, err = cryptoRand.Read(rnd); err != nil {
			hssAvReject(c, http.StatusInternalServerError, "SYSTEM_FAILURE", err.Error())
			return
		}
		ik, ck, xres, autn, errM := milenage.GenerateAKAParameters(opc, k, rnd, sqn, amf)
		if errM != nil {
			hssAvReject(c, http.StatusInternalServerError, "SYSTEM_FAILURE", errM.Error())
			return
		}
		vectors = append(vectors, models.Udm_UEAU_AvImsGbaEapAka{
			AvType: models.Udm_UEAU_HssAvType_IMS_AKA,
			Rand:   hex.EncodeToString(rnd),
			Xres:   hex.EncodeToString(xres),
			Autn:   hex.EncodeToString(autn),
			Ck:     hex.EncodeToString(ck),
			Ik:     hex.EncodeToString(ik),
		})
	}
	logger.UeauLog.Infof("GetHssAv ims-aka: %d vector(s) for %s from SQN %x", n, supi, base)
	c.JSON(http.StatusOK, gin.H{"hssAuthenticationVectors": vectors})
}
