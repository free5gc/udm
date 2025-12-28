package util

import (
	"testing"
)

func TestIsValidSuci(t *testing.T) {
	type args struct {
		suci string
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
				if got := IsValidSuci(tt.args.suci); got != tt.want {
					t.Errorf("IsValidSuci() = %v, want %v (input: %q)", got, tt.want, tt.args.suci)
				}
			})
		}
	}

	// ==========================================
	// 1. IMSI-based SUCI (Type 0)
	// Format: suci-0-<MCC>-<MNC>-<Routing>-<Scheme>-<KeyId>-<Output>
	// Source: TS 23.003 Clause 6.3
	// ==========================================
	t.Run("Check_SUCI_Type0_IMSI", func(t *testing.T) {
		tests := []testCase{
			// Happy Paths
			{
				"Valid Type 0 (Null Scheme)",
				args{"suci-0-208-93-0-0-0-208930000000003"},
				true,
			},
			{
				"Valid Type 0 (Profile A, Long Routing)",
				args{"suci-0-466-92-f001-1-1-ECCOutputHex..."},
				true,
			},
			{
				"Valid Type 0 (3-digit MNC)",
				args{"suci-0-466-092-0-0-0-123456"},
				true,
			},

			// Format Errors
			{
				"Invalid Type 0 (Bad MCC)",
				args{"suci-0-20A-93-0-0-0-123"}, // MCC must be digits
				false,
			},
			{
				"Invalid Type 0 (Bad Routing Ind)",
				args{"suci-0-208-93-GGGG-0-0-123"}, // Routing must be Hex
				false,
			},
			{
				"Invalid Type 0 (Missing Parts)",
				args{"suci-0-208-93-0-0-123"}, // Missing KeyId
				false,
			},
		}
		runTests(t, tests)
	})

	// ==========================================
	// 2. NAI-based SUCI (Type 1)
	// Format: suci-1-<HomeNet>-<Routing>-<Scheme>-<KeyId>-<Output>
	// ==========================================
	t.Run("Check_SUCI_Type1_NAI", func(t *testing.T) {
		tests := []testCase{
			{
				"Valid Type 1 (Standard)",
				args{"suci-1-factory.local-0-0-0-user1"},
				true,
			},
			{
				"Invalid Type 1 (Bad Hex)",
				args{"suci-1-domain-Z-0-0-output"}, // Routing must be Hex
				false,
			},
		}
		runTests(t, tests)
	})

	// ==========================================
	// 3. Security & Edge Cases
	// ==========================================
	t.Run("Check_Security_EdgeCases", func(t *testing.T) {
		tests := []testCase{
			{"Empty String", args{""}, false},
			{"Wrong Prefix", args{"suciX-0-208-93-0-0-0-1"}, false},
			{"Unknown Type (Type 9)", args{"suci-9-208-93-0-0-0-1"}, false}, // Currently only 0 and 1 supported

			// [Security] Null Byte Injection
			{"Null Byte Injection", args{"suci-0-208-93\x00-0-0-0-1"}, false},

			// Fuzzing garbage
			{"Garbage String", args{"suci-0-garbage-data"}, false},
		}
		runTests(t, tests)
	})
}
