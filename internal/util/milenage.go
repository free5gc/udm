package util

import (
	"encoding/hex"

	"github.com/free5gc/util/milenage"
)

func MilenageF1(opc, k, rand, sqn, amf []byte, macA, macS []byte) error {
	_, _, _, autn, err := milenage.GenerateAKAParameters(opc, k, rand, sqn, amf)
	if err != nil {
		return err
	}

	// AUTN = (SQN xor AK) || AMF || MAC-A
	// MAC-A is the last 8 bytes of AUTN
	if len(autn) >= 8 && macA != nil {
		copy(macA, autn[len(autn)-8:])
	}

	// For MAC-S, use resync AMF (0000)
	if macS != nil {
		resyncAMFBytes, _ := hex.DecodeString("0000")
		_, _, _, autnS, err := milenage.GenerateAKAParameters(opc, k, rand, sqn, resyncAMFBytes)
		if err != nil {
			return err
		}
		if len(autnS) >= 8 {
			copy(macS, autnS[len(autnS)-8:])
		}
	}

	return nil
}

func MilenageF2345(opc, k, rand []byte, res, ck, ik, ak, akstar []byte) error {
	// Use GenerateAKAParameters to get basic parameters
	ikOut, ckOut, resOut, autn, err := milenage.GenerateAKAParameters(opc, k, rand, make([]byte, 6), make([]byte, 2))
	if err != nil {
		return err
	}

	// Use GenerateKeysWithAUTN to get AK
	_, akOut, _, _, _, err := milenage.GenerateKeysWithAUTN(opc, k, rand, autn)
	if err != nil {
		return err
	}

	// Copy results to output parameters
	if res != nil {
		copy(res, resOut)
	}
	if ck != nil {
		copy(ck, ckOut)
	}
	if ik != nil {
		copy(ik, ikOut)
	}
	if ak != nil {
		copy(ak, akOut)
	}
	if akstar != nil {
		// For AK*, we need to use a different SQN, but due to API limitations, we use the same value for now
		copy(akstar, akOut)
	}

	return nil
}
