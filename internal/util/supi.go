package util

import (
	"regexp"
	"strings"
)

// TS 29.571 5.3.2 & TS 23.003
// SUPI format validation
var supiImsiRegex = regexp.MustCompile(`^imsi-[0-9]{5,15}$`)
// TS 29.571 5.3.2 & TS 23.003 28.7.2 
// NAI format validation
var supiNaiRegex = regexp.MustCompile(`^nai-.+@.+$`)
// TS 29.571 5.3.2 & TS 23.003 28.15.2(gci) & TS 23.003 28.16.2(gli)
// GCI/GLI format validation
var supiGciGliRegex = regexp.MustCompile(`^(gci|gli)-.+$`)

// IsValidSupi checks if the given SUPI is valid according to 3GPP specifications
func IsValidSupi(supi string) bool {
	if len(supi) == 0 {
		return false
	}

	if strings.HasPrefix(supi, "imsi-") {
		return supiImsiRegex.MatchString(supi)
	}

	if strings.HasPrefix(supi, "nai-") {
		if strings.Contains(supi, "\x00") {
            return false
        }
		return supiNaiRegex.MatchString(supi)
	}
	if strings.HasPrefix(supi, "gci-") || strings.HasPrefix(supi, "gli-") {
		return supiGciGliRegex.MatchString(supi)
	}

	return false
}