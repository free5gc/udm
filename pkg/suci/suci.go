package suci

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/curve25519"

	"github.com/free5gc/udm/internal/logger"
)

// suci-0(SUPI type: IMSI)-mcc-mnc-routingIndicator-protectionScheme-homeNetworkPublicKeyID-schemeOutput.
// TODO: suci-1(SUPI type: NAI)-homeNetworkID-routingIndicator-protectionScheme-homeNetworkPublicKeyID-schemeOutput.

const (
	PrefixIMSI     = "imsi-"
	PrefixSUCI     = "suci"
	SupiTypeIMSI   = "0"
	NullScheme     = "0"
	ProfileAScheme = "1"
	ProfileBScheme = "2"
)

var (
	// Network and identification patterns.
	mccRegex      = `(?P<mcc>\d{3})`                                         // Mobile Country Code; 3 digits
	mncRegex      = `(?P<mnc>\d{2,3})`                                       // Mobile Network Code; 2 or 3 digits
	imsiTypeRegex = fmt.Sprintf("(?P<imsiType>0-%s-%s)", mccRegex, mncRegex) // MCC-MNC

	// The Home Network Identifier consists of a string of
	// characters with a variable length representing a domain name
	// as specified in Section 2.2 of RFC 7542
	naiTypeRegex = "(?P<naiType>1-.*)"

	supiTypeRegex = fmt.Sprintf("(?P<supi_type>%s|%s)", // SUPI type; 0 = IMSI, 1 = NAI (for n3gpp)
		imsiTypeRegex,
		naiTypeRegex)

	routingIndicatorRegex = `(?P<routing_indicator>\d{1,4})`                         // Routing Indicator, used by the AUSF to find the appropriate UDM when SUCI is encrypted 1-4 digits
	protectionSchemeRegex = `(?P<protection_scheme_id>(?:[0-2]))`                    // Protection Scheme ID; 0 = NULL Scheme (unencrypted), 1 = Profile A, 2 = Profile B
	publicKeyIDRegex      = `(?P<public_key_id>(?:\d{1,2}|1\d{2}|2[0-4]\d|25[0-5]))` // Public Key ID; 1-255
	schemeOutputRegex     = `(?P<scheme_output>[A-Fa-f0-9]+)`                        // Scheme Output; unbounded hex string (safe from ReDoS due to bounded length of SUCI)
	suciRegex             = regexp.MustCompile(fmt.Sprintf("^suci-%s-%s-%s-%s-%s$",  // Subscription Concealed Identifier (SUCI) Encrypted SUPI as sent by the UE to the AMF; 3GPP TS 29.503 - Annex C
		supiTypeRegex,
		routingIndicatorRegex,
		protectionSchemeRegex,
		publicKeyIDRegex,
		schemeOutputRegex,
	))
)

type Suci struct {
	SupiType         string // 0 for IMSI, 1 for NAI
	Mcc              string // 3 digits
	Mnc              string // 2-3 digits
	HomeNetworkId    string // variable-length string
	RoutingIndicator string // 1-4 digits
	ProtectionScheme string // 0-2
	PublicKeyID      string // 1-255
	SchemeOutput     string // hex string
}

func ParseSuci(input string) *Suci {
	matches := suciRegex.FindStringSubmatch(input)
	if matches == nil {
		return nil
	}

	// The indices correspond to the order of the regex groups in the pattern
	return &Suci{
		SupiType:         matches[1], // First capture group
		Mcc:              matches[3], // Third capture group
		Mnc:              matches[4], // Fourth capture group
		HomeNetworkId:    matches[5], // Fifth capture group
		RoutingIndicator: matches[6], // Sixth capture group
		ProtectionScheme: matches[7], // Seventh capture group
		PublicKeyID:      matches[8], // Eigth capture group
		SchemeOutput:     matches[9], // Nineth capture group
	}
}

type SuciProfile struct {
	ProtectionScheme string `yaml:"ProtectionScheme,omitempty"`
	PrivateKey       string `yaml:"PrivateKey,omitempty"`
	PublicKey        string `yaml:"PublicKey,omitempty"`
}

// profile A.
const (
	ProfileAMacKeyLen = 32 // octets
	ProfileAEncKeyLen = 16 // octets
	ProfileAIcbLen    = 16 // octets
	ProfileAMacLen    = 8  // octets
	ProfileAHashLen   = 32 // octets
)

// profile B.
const (
	ProfileBMacKeyLen = 32 // octets
	ProfileBEncKeyLen = 16 // octets
	ProfileBIcbLen    = 16 // octets
	ProfileBMacLen    = 8  // octets
	ProfileBHashLen   = 32 // octets
)

func HmacSha256(input, macKey []byte, macLen int) ([]byte, error) {
	h := hmac.New(sha256.New, macKey)
	if _, err := h.Write(input); err != nil {
		return nil, fmt.Errorf("HMAC SHA256 error %+v", err)
	}
	macVal := h.Sum(nil)
	macTag := macVal[:macLen]
	// fmt.Printf("macVal: %x\nmacTag: %x\n", macVal, macTag)
	return macTag, nil
}

func Aes128ctr(input, encKey, icb []byte) ([]byte, error) {
	output := make([]byte, len(input))
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("AES128 CTR error %+v", err)
	}
	stream := cipher.NewCTR(block, icb)
	stream.XORKeyStream(output, input)
	// fmt.Printf("aes input: %x %x %x\naes output: %x\n", input, encKey, icb, output)
	return output, nil
}

func AnsiX963KDF(sharedKey, publicKey []byte, profileEncKeyLen, profileMacKeyLen, profileHashLen int) []byte {
	var counter uint32 = 0x00000001
	var kdfKey []byte
	kdfRounds := int(math.Ceil(float64(profileEncKeyLen+profileMacKeyLen) / float64(profileHashLen)))
	for i := 1; i <= kdfRounds; i++ {
		counterBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(counterBytes, counter)
		// fmt.Printf("counterBytes: %x\n", counterBytes)
		tmpK := sha256.Sum256(append(append(sharedKey, counterBytes...), publicKey...))
		sliceK := tmpK[:]
		kdfKey = append(kdfKey, sliceK...)
		// fmt.Printf("kdfKey in round %d: %x\n", i, kdfKey)
		counter++
	}
	return kdfKey
}

func swapNibbles(input []byte) []byte {
	output := make([]byte, len(input))
	for i, b := range input {
		output[i] = bits.RotateLeft8(b, 4)
	}
	return output
}

func calcSchemeResult(decryptPlainText []byte, supiType string) string {
	var schemeResult string
	if supiType == SupiTypeIMSI {
		schemeResult = hex.EncodeToString(swapNibbles(decryptPlainText))
		if schemeResult[len(schemeResult)-1] == 'f' {
			schemeResult = schemeResult[:len(schemeResult)-1]
		}
	} else {
		schemeResult = hex.EncodeToString(decryptPlainText)
	}
	return schemeResult
}

func profileA(input, supiType, privateKey string) (string, error) {
	logger.SuciLog.Infoln("SuciToSupi Profile A")
	s, hexDecodeErr := hex.DecodeString(input)
	if hexDecodeErr != nil {
		logger.SuciLog.Errorln("hex DecodeString error")
		return "", hexDecodeErr
	}

	// for X25519(profile A), q (The number of elements in the field Fq) = 2^255 - 19
	// len(pubkey) is therefore ceil((log2q)/8+1) = 32octets
	ProfileAPubKeyLen := 32
	if len(s) < ProfileAPubKeyLen+ProfileAMacLen {
		logger.SuciLog.Errorln("len of input data is too short!")
		return "", fmt.Errorf("suci input too short\n")
	}

	decryptMac := s[len(s)-ProfileAMacLen:]
	decryptPublicKey := s[:ProfileAPubKeyLen]
	decryptCipherText := s[ProfileAPubKeyLen : len(s)-ProfileAMacLen]
	// fmt.Printf("dePub: %x\ndeCiph: %x\ndeMac: %x\n", decryptPublicKey, decryptCipherText, decryptMac)

	// test data from TS33.501 Annex C.4
	// aHNPriv, _ := hex.DecodeString("c53c2208b61860b06c62e5406a7b330c2b577aa5558981510d128247d38bd1d")
	var aHNPriv []byte
	if aHNPrivTmp, err := hex.DecodeString(privateKey); err != nil {
		return "", err
	} else {
		aHNPriv = aHNPrivTmp
	}
	var decryptSharedKey []byte
	if decryptSharedKeyTmp, err := curve25519.X25519(aHNPriv, decryptPublicKey); err != nil {
		return "", err
	} else {
		decryptSharedKey = decryptSharedKeyTmp
	}
	// fmt.Printf("deShared: %x\n", decryptSharedKey)

	kdfKey := AnsiX963KDF(decryptSharedKey, decryptPublicKey, ProfileAEncKeyLen, ProfileAMacKeyLen, ProfileAHashLen)
	decryptEncKey := kdfKey[:ProfileAEncKeyLen]
	decryptIcb := kdfKey[ProfileAEncKeyLen : ProfileAEncKeyLen+ProfileAIcbLen]
	decryptMacKey := kdfKey[len(kdfKey)-ProfileAMacKeyLen:]
	// fmt.Printf("\ndeEncKey(size%d): %x\ndeMacKey: %x\ndeIcb: %x\n", len(decryptEncKey), decryptEncKey, decryptMacKey,
	// decryptIcb)

	decryptMacTag, err := HmacSha256(decryptCipherText, decryptMacKey, ProfileAMacLen)
	if err != nil {
		logger.SuciLog.Errorln("decryption MAC error")
		return "", err
	}
	if hmac.Equal(decryptMacTag, decryptMac) {
		logger.SuciLog.Infoln("decryption MAC match")
	} else {
		logger.SuciLog.Errorln("decryption MAC failed")
		return "", fmt.Errorf("decryption MAC failed\n")
	}

	decryptPlainText, err := Aes128ctr(decryptCipherText, decryptEncKey, decryptIcb)
	if err != nil {
		logger.SuciLog.Errorln("decryptPlainText error")
		return "", err
	}

	return calcSchemeResult(decryptPlainText, supiType), nil
}

var (
	InvalidPointError = fmt.Errorf("crypto/elliptic: attempted operation on invalid point")
)

func checkOnCurve(curve elliptic.Curve, x, y *big.Int) error {
	// (0, 0) is the point at infinity by convention. It's ok to operate on it,
	// although IsOnCurve is documented to return false for it. See Issue 37294.
	if x.Sign() == 0 && y.Sign() == 0 {
		return nil
	}

	if !curve.IsOnCurve(x, y) {
		return InvalidPointError
	}

	return nil
}

func profileB(input, supiType, privateKey string) (string, error) {
	logger.SuciLog.Infoln("SuciToSupi Profile B")
	s, hexDecodeErr := hex.DecodeString(input)
	if hexDecodeErr != nil || len(s) < 1 {
		logger.SuciLog.Errorln("hex DecodeString error")
		return "", hexDecodeErr
	}

	var ProfileBPubKeyLen int // p256, module q = 2^256 - 2^224 + 2^192 + 2^96 - 1
	var uncompressed bool
	if s[0] == 0x02 || s[0] == 0x03 {
		ProfileBPubKeyLen = 33 // ceil(log(2, q)/8) + 1 = 33
		uncompressed = false
	} else if s[0] == 0x04 {
		ProfileBPubKeyLen = 65 // 2*ceil(log(2, q)/8) + 1 = 65
		uncompressed = true
	} else {
		logger.SuciLog.Errorln("input error")
		return "", fmt.Errorf("suci input error\n")
	}

	if len(s) < ProfileBPubKeyLen+ProfileBMacLen {
		logger.SuciLog.Errorln("len of input data is too short!")
		return "", fmt.Errorf("suci input too short\n")
	}
	decryptPublicKey := s[:ProfileBPubKeyLen]
	decryptMac := s[len(s)-ProfileBMacLen:]
	decryptCipherText := s[ProfileBPubKeyLen : len(s)-ProfileBMacLen]

	// test data from TS33.501 Annex C.4
	// bHNPriv, _ := hex.DecodeString("F1AB1074477EBCC7F554EA1C5FC368B1616730155E0041AC447D6301975FECDA")
	var bHNPriv []byte
	if bHNPrivTmp, err := hex.DecodeString(privateKey); err != nil {
		return "", err
	} else {
		bHNPriv = bHNPrivTmp
	}

	var xUncompressed, yUncompressed *big.Int
	if uncompressed {
		xUncompressed = new(big.Int).SetBytes(decryptPublicKey[1:(ProfileBPubKeyLen/2 + 1)])
		yUncompressed = new(big.Int).SetBytes(decryptPublicKey[(ProfileBPubKeyLen/2 + 1):])
	} else {
		xUncompressed, yUncompressed = elliptic.UnmarshalCompressed(elliptic.P256(), decryptPublicKey)
		if xUncompressed == nil || yUncompressed == nil {
			logger.SuciLog.Errorln("Uncompressed key has invalid point")
			return "", fmt.Errorf("Key uncompression error\n")
		}
	}

	if err := checkOnCurve(elliptic.P256(), xUncompressed, yUncompressed); err != nil {
		return "", err
	}

	// x-coordinate is the shared key
	decryptSharedKeyTmp, _ := elliptic.P256().ScalarMult(xUncompressed, yUncompressed, bHNPriv)
	decryptSharedKey := FillFrontZero(decryptSharedKeyTmp, len(xUncompressed.Bytes()))

	decryptPublicKeyForKDF := decryptPublicKey
	if uncompressed {
		decryptPublicKeyForKDF = elliptic.MarshalCompressed(elliptic.P256(), xUncompressed, yUncompressed)
	}

	kdfKey := AnsiX963KDF(decryptSharedKey, decryptPublicKeyForKDF, ProfileBEncKeyLen, ProfileBMacKeyLen,
		ProfileBHashLen)
	decryptEncKey := kdfKey[:ProfileBEncKeyLen]
	decryptIcb := kdfKey[ProfileBEncKeyLen : ProfileBEncKeyLen+ProfileBIcbLen]
	decryptMacKey := kdfKey[len(kdfKey)-ProfileBMacKeyLen:]

	decryptMacTag, err := HmacSha256(decryptCipherText, decryptMacKey, ProfileBMacLen)
	if err != nil {
		logger.SuciLog.Errorln("decryption MAC error")
		return "", err
	}

	if hmac.Equal(decryptMacTag, decryptMac) {
		logger.SuciLog.Infoln("decryption MAC match")
	} else {
		logger.SuciLog.Errorln("decryption MAC failed")
		return "", fmt.Errorf("decryption MAC failed\n")
	}

	decryptPlainText, err := Aes128ctr(decryptCipherText, decryptEncKey, decryptIcb)
	if err != nil {
		logger.SuciLog.Errorln("decryptPlainText MAC error")
		return "", err
	}

	return calcSchemeResult(decryptPlainText, supiType), nil
}

func FillFrontZero(input *big.Int, length int) []byte {
	if len(input.Bytes()) >= length {
		return input.Bytes()
	}
	result := make([]byte, length)
	inputBytes := input.Bytes()
	copy(result[length-len(inputBytes):], input.Bytes())
	return result
}

func ToSupi(suci string, suciProfiles []SuciProfile) (string, error) {
	parsedSuci := ParseSuci(suci)
	if parsedSuci == nil {
		return "", fmt.Errorf("unknown suciPrefix [%+v]", parsedSuci)
	}

	logger.SuciLog.Infof("scheme %s\n", parsedSuci.ProtectionScheme)
	scheme := parsedSuci.ProtectionScheme
	mccMnc := parsedSuci.Mcc + parsedSuci.Mnc

	supiPrefix := PrefixIMSI
	if strings.HasPrefix(parsedSuci.SupiType, SupiTypeIMSI) {
		logger.SuciLog.Infof("SUPI type is IMSI\n")
	} else {
		logger.SuciLog.Infof("SUPI type is NAI\n")
		return "", fmt.Errorf("unsupported suciType NAI")
	}

	if scheme == NullScheme { // NULL scheme
		return supiPrefix + mccMnc + parsedSuci.SchemeOutput, nil
	}

	// (HNPublicKeyID-1) is the index of "suciProfiles" slices
	keyIndex, err := strconv.Atoi(parsedSuci.PublicKeyID)
	if err != nil {
		return "", fmt.Errorf("parse HNPublicKeyID error: %+v", err)
	}
	if keyIndex < 1 || keyIndex > len(suciProfiles) {
		return "", fmt.Errorf("keyIndex(%d) out of range(%d)", keyIndex, len(suciProfiles))
	}

	protectScheme := suciProfiles[keyIndex-1].ProtectionScheme
	privateKey := suciProfiles[keyIndex-1].PrivateKey

	if scheme != protectScheme {
		return "", fmt.Errorf("protect Scheme mismatch [%s:%s]", scheme, protectScheme)
	}

	if scheme == ProfileAScheme {
		if profileAResult, err := profileA(parsedSuci.SchemeOutput, SupiTypeIMSI, privateKey); err != nil {
			return "", err
		} else {
			return supiPrefix + mccMnc + profileAResult, nil
		}
	} else if scheme == ProfileBScheme {
		if profileBResult, err := profileB(parsedSuci.SchemeOutput, SupiTypeIMSI, privateKey); err != nil {
			return "", err
		} else {
			return supiPrefix + mccMnc + profileBResult, nil
		}
	} else {
		return "", fmt.Errorf("protect Scheme (%s) is not supported", scheme)
	}
}
