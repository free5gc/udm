package util

import (
	"testing"
)

func TestIsValidGpsi(t *testing.T) {
	type args struct {
		gpsi string
	}
	type testCase struct {
		name string
		args args
		want bool
	}

	runTests := func(t *testing.T, tests []testCase) {
		t.Helper()
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := IsValidGpsi(tt.args.gpsi); got != tt.want {
					t.Errorf("IsValidGpsi() = %v, want %v (input: %q)", got, tt.want, tt.args.gpsi)
				}
			})
		}
	}

	// MSISDN test
	t.Run("Check_MSISDN", func(t *testing.T) {
		tests := []testCase{
			{"Valid MSISDN (Standard)", args{"msisdn-886912345678"}, true},
			{"Valid MSISDN (Min Length 5)", args{"msisdn-12345"}, true},            // TS 29.571 regex lower bound
			{"Valid MSISDN (Max Length 15)", args{"msisdn-123456789012345"}, true}, // TS 29.571 regex upper bound
			{"Invalid MSISDN (Too short)", args{"msisdn-1234"}, false},
			{"Invalid MSISDN (Too long)", args{"msisdn-1234567890123456"}, false},
			{"Invalid MSISDN (Non-digits)", args{"msisdn-886912abc"}, false},
			{"Invalid MSISDN (Null Byte)", args{"msisdn-123\x00456"}, false},
		}
		runTests(t, tests)
	})

	// external identifier test
	t.Run("Check_ExternalID", func(t *testing.T) {
		tests := []testCase{
			{"Valid ExtID (Standard)", args{"extid-user@domain.com"}, true},
			{"Valid ExtID (Subdomain)", args{"extid-sensor1@factory.iot.org"}, true},
			{"Invalid ExtID (Missing @)", args{"extid-user-domain.com"}, false}, // TS 23.003: MUST have @
			{"Invalid ExtID (Missing LocalId)", args{"extid-@domain.com"}, false},
			{"Invalid ExtID (Missing DomainId)", args{"extid-user@"}, false},
			{"Invalid ExtID (Empty content)", args{"extid-"}, false},
			// [Security] Null Byte Injection check
			{"Security: ExtID with Null Byte", args{"extid-user\x00@domain.com"}, false},
		}
		runTests(t, tests)
	})

	// other invalid cases
	t.Run("Check_General_Invalid", func(t *testing.T) {
		tests := []testCase{
			{"Empty String", args{""}, false},
			{"Unknown Prefix", args{"unknown-12345"}, false},
			{"Just Prefix (msisdn-)", args{"msisdn-"}, false},
			{"Just Prefix (extid-)", args{"extid-"}, false},
			// [Security] Fuzzing / Garbage
			{"Security: Raw Null Bytes", args{"\x00\x00\x00"}, false},
			{"Security: Random String", args{"not_a_gpsi"}, false},
		}
		runTests(t, tests)
	})
}
