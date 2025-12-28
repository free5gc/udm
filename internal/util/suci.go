package util

import (
	"regexp"
	"strings"
)

// TS 29.571 5.3.2 & TS 23.003 Clause 6.3
// Regex for IMSI-based SUCI (Type 0)
// Format: suci-0-<MCC>-<MNC>-<RoutingInd>-<Scheme>-<KeyId>-<Output>
// Example: suci-0-208-93-0-0-0-1234567890 (Null Scheme)
var suciImsiRegex = regexp.MustCompile(`^suci-0-[0-9]{3}-[0-9]{2,3}-[0-9a-fA-F]{1,4}-` +
	`[0-9a-fA-F]{1,2}-[0-9a-fA-F]{1,2}-.+$`)

// Regex for NAI-based SUCI (Type 1)
// Format: suci-1-<HomeNetworkId>-<RoutingInd>-<Scheme>-<KeyId>-<Output>
var suciNaiRegex = regexp.MustCompile(`^suci-1-.+-[0-9a-fA-F]{1,4}-[0-9a-fA-F]{1,2}-[0-9a-fA-F]{1,2}-.+$`)

// IsValidSuci checks if the given string is a valid SUCI
func IsValidSuci(suci string) bool {
	if len(suci) == 0 {
		return false
	}

	// must start with "suci-"
	if !strings.HasPrefix(suci, "suci-") {
		return false
	}

	// prevent null byte injection
	if strings.Contains(suci, "\x00") {
		return false
	}

	// validate IMSI-based SUCI (Type 0)
	if strings.HasPrefix(suci, "suci-0-") {
		return suciImsiRegex.MatchString(suci)
	}

	// validate NAI-based SUCI (Type 1)
	if strings.HasPrefix(suci, "suci-1-") {
		return suciNaiRegex.MatchString(suci)
	}

	return false
}
