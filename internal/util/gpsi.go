package util

import (
	"regexp"
	"strings"
)
// TS 29.571 5.3.2 & TS 23.003 3.3
// MSISDN format validation
var gpsiMsisdnRegex = regexp.MustCompile(`^msisdn-[0-9]{5,15}$`)
// TS 29.571 5.3.2 & TS 23.003 19.7.2
// External Identifier format validation
var gpsiExtIdRegex  = regexp.MustCompile(`^extid-.+@.+$`)

// IsValidGpsi valid GPSI format (TS 29.571 5.3.2)
func IsValidGpsi(gpsi string) bool {
    if len(gpsi) == 0 {
        return false
    }

    if strings.HasPrefix(gpsi, "msisdn-") {
        return gpsiMsisdnRegex.MatchString(gpsi)
    }

    if strings.HasPrefix(gpsi, "extid-") {
        // External Identifier should not contain null byte
        if strings.Contains(gpsi, "\x00") {
            return false
        }
        return gpsiExtIdRegex.MatchString(gpsi)
    }

    return false
}