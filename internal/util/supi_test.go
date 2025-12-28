package util

import (
	"testing"
)

func TestIsValidSupi(t *testing.T) {
	type Args struct {
		supi string
	}

	type testCase struct {
		name string
		Args Args
		Want bool
	}
	runTests := func(t *testing.T, tests []testCase) {
		t.Helper()
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := IsValidSupi(tt.Args.supi); got != tt.Want {
					t.Errorf("IsValidSupi() = %v, Want %v (input: %q)", got, tt.Want, tt.Args.supi)
				}
			})
		}
	}
	// IMSI test
	t.Run("Check_IMSI", func(t *testing.T) {
		tests := []testCase{
			{"Valid IMSI (15 digits)", Args{"imsi-208930000000003"}, true},
			{"Valid IMSI (5 digits)", Args{"imsi-12345"}, true},
			{"Invalid IMSI (Too short)", Args{"imsi-1234"}, false},
			{"Invalid IMSI (Too long)", Args{"imsi-1234567890123456"}, false},
			{"Invalid IMSI (Non-digits)", Args{"imsi-20893abc000003"}, false},
			{"Invalid IMSI (Null Byte)", Args{"imsi-123\x00456"}, false},
		}
		runTests(t, tests)
	})

	// NAI test
	t.Run("Check_NAI", func(t *testing.T) {
		tests := []testCase{
			{"Valid NAI (Standard)", Args{"nai-user@realm.com"}, true},
			{"Valid NAI (3GPP Style)", Args{"nai-type0.rid123.schid0.userid1@5gc.mnc001.mcc001.org"}, true},
			{"Invalid NAI (Missing @)", Args{"nai-userrealm.com"}, false},
			{"Invalid NAI (Missing Realm)", Args{"nai-user@"}, false},
			{"Invalid NAI (Missing User)", Args{"nai-@realm"}, false},
			{"Security: NAI with Null Byte", Args{"nai-user\x00@realm"}, false},
		}
		runTests(t, tests)
	})

	// GCI/GLI test
	t.Run("Check_GCI_GLI", func(t *testing.T) {
		tests := []testCase{
			{"Valid GCI", Args{"gci-cable-mac-1234"}, true},
			{"Valid GLI", Args{"gli-fiber-line-5678"}, true},
			{"Invalid GCI (Empty body)", Args{"gci-"}, false},
			{"Invalid GLI (Empty body)", Args{"gli-"}, false},
		}
		runTests(t, tests)
	})

	// General invalid test
	t.Run("Check_General_Invalid", func(t *testing.T) {
		tests := []testCase{
			{"Empty String", Args{""}, false},
			{"Unknown Prefix", Args{"unknown-12345"}, false},
			{"Just Prefix", Args{"imsi-"}, false},
			{"Security: Raw Null Bytes", Args{"\x00\x00\x00"}, false},
			{"Security: Garbage", Args{"fuzzing_payload"}, false},
		}
		runTests(t, tests)
	})

}